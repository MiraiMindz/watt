package http2

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// Stream represents an HTTP/2 stream (RFC 7540 Section 5)
// Streams are independent, bidirectional sequences of frames between client and server.
type Stream struct {
	id            uint32
	state         StreamState
	stateMu       sync.RWMutex

	// Flow control
	sendWindow    int32 // How much we can send
	recvWindow    int32 // How much we can receive
	windowMu      sync.Mutex

	// Priority
	weight        uint8  // 1-256, stored as 0-255
	dependency    uint32 // Stream this depends on
	exclusive     bool   // Exclusive dependency
	priorityMu    sync.RWMutex

	// Data
	recvBuf       []byte    // Received data buffer
	recvBufMu     sync.Mutex
	recvCond      *sync.Cond // Signal when data arrives
	recvClosed    bool       // Receive side closed

	sendBuf       []byte    // Send data buffer
	sendBufMu     sync.Mutex
	sendClosed    bool       // Send side closed

	// Buffer limits (security hardening)
	maxBufferSize int64 // Maximum buffer size (default: 1MB)
	conn          *Connection // Parent connection for buffer tracking

	// Headers
	requestHeaders  []HeaderField
	responseHeaders []HeaderField
	trailers        []HeaderField
	headersMu       sync.RWMutex

	// Lifecycle
	ctx           context.Context
	cancel        context.CancelFunc
	created       time.Time
	lastActivity  time.Time
	activityMu    sync.RWMutex

	// Error handling
	err           error
	errMu         sync.RWMutex
	resetCode     ErrorCode
}

// StreamState represents the state of an HTTP/2 stream (RFC 7540 Section 5.1)
type StreamState uint8

const (
	StateIdle StreamState = iota
	StateReservedLocal
	StateReservedRemote
	StateOpen
	StateHalfClosedLocal
	StateHalfClosedRemote
	StateClosed
)

func (s StreamState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateReservedLocal:
		return "reserved(local)"
	case StateReservedRemote:
		return "reserved(remote)"
	case StateOpen:
		return "open"
	case StateHalfClosedLocal:
		return "half-closed(local)"
	case StateHalfClosedRemote:
		return "half-closed(remote)"
	case StateClosed:
		return "closed"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// NewStream creates a new stream with the given ID and initial window size
// Now uses object pooling to reduce allocations from 6 to ~2
func NewStream(id uint32, initialWindowSize int32) *Stream {
	return getPooledStream(id, initialWindowSize)
}

// SetMaxBufferSize sets the maximum buffer size for this stream
func (s *Stream) SetMaxBufferSize(size int64) {
	s.sendBufMu.Lock()
	s.maxBufferSize = size
	s.sendBufMu.Unlock()
}

// ID returns the stream identifier
func (s *Stream) ID() uint32 {
	return s.id
}

// State returns the current stream state
func (s *Stream) State() StreamState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

// setState changes the stream state (internal, caller must hold stateMu)
func (s *Stream) setState(newState StreamState) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	// Validate state transition per RFC 7540 Section 5.1
	valid := s.isValidTransition(s.state, newState)
	if !valid {
		return fmt.Errorf("invalid state transition: %s -> %s", s.state, newState)
	}

	s.state = newState
	s.updateActivity()

	return nil
}

// isValidTransition checks if a state transition is valid per RFC 7540
func (s *Stream) isValidTransition(from, to StreamState) bool {
	// RFC 7540 Section 5.1 state transition diagram
	switch from {
	case StateIdle:
		// idle can transition to:
		// - open (HEADERS)
		// - reserved(local) (PUSH_PROMISE)
		// - reserved(remote) (PUSH_PROMISE from peer)
		// - half-closed(local) (HEADERS with END_STREAM)
		// - half-closed(remote) (HEADERS with END_STREAM from peer)
		// - closed (RST_STREAM or connection error)
		return to == StateOpen || to == StateReservedLocal || to == StateReservedRemote ||
			to == StateHalfClosedLocal || to == StateHalfClosedRemote || to == StateClosed
	case StateReservedLocal:
		return to == StateHalfClosedRemote || to == StateClosed
	case StateReservedRemote:
		return to == StateHalfClosedLocal || to == StateClosed
	case StateOpen:
		return to == StateHalfClosedLocal || to == StateHalfClosedRemote || to == StateClosed
	case StateHalfClosedLocal:
		return to == StateClosed
	case StateHalfClosedRemote:
		return to == StateClosed
	case StateClosed:
		return to == StateClosed // Can stay closed
	default:
		return false
	}
}

// Open transitions the stream to open state (for sending/receiving HEADERS)
func (s *Stream) Open() error {
	return s.setState(StateOpen)
}

// CloseLocal closes the local (sending) side of the stream
func (s *Stream) CloseLocal() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	switch s.state {
	case StateIdle:
		// RFC 7540: Sending HEADERS with END_STREAM transitions idle -> half-closed(local)
		s.state = StateHalfClosedLocal
	case StateOpen:
		s.state = StateHalfClosedLocal
	case StateHalfClosedRemote:
		s.state = StateClosed
		s.cancel()
	default:
		return fmt.Errorf("cannot close local in state %s", s.state)
	}

	s.sendBufMu.Lock()
	s.sendClosed = true
	s.sendBufMu.Unlock()

	s.updateActivity()
	return nil
}

// CloseRemote closes the remote (receiving) side of the stream
func (s *Stream) CloseRemote() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	switch s.state {
	case StateIdle:
		// RFC 7540: Receiving HEADERS with END_STREAM transitions idle -> half-closed(remote)
		s.state = StateHalfClosedRemote
	case StateOpen:
		s.state = StateHalfClosedRemote
	case StateHalfClosedLocal:
		s.state = StateClosed
		s.cancel()
	default:
		return fmt.Errorf("cannot close remote in state %s", s.state)
	}

	s.recvBufMu.Lock()
	s.recvClosed = true
	s.recvCond.Broadcast()
	s.recvBufMu.Unlock()

	s.updateActivity()
	return nil
}

// Reset resets the stream with the given error code
func (s *Stream) Reset(code ErrorCode) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.state = StateClosed
	s.resetCode = code
	s.cancel()

	// Unblock any waiting readers
	s.recvBufMu.Lock()
	s.recvClosed = true
	s.recvCond.Broadcast()
	s.recvBufMu.Unlock()

	s.sendBufMu.Lock()
	s.sendClosed = true
	s.sendBufMu.Unlock()

	return nil
}

// IsClosed returns true if the stream is in closed state
func (s *Stream) IsClosed() bool {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state == StateClosed
}

// SendWindow returns the current send window size
func (s *Stream) SendWindow() int32 {
	s.windowMu.Lock()
	defer s.windowMu.Unlock()
	return s.sendWindow
}

// RecvWindow returns the current receive window size
func (s *Stream) RecvWindow() int32 {
	s.windowMu.Lock()
	defer s.windowMu.Unlock()
	return s.recvWindow
}

// IncrementSendWindow increases the send window by the given amount
func (s *Stream) IncrementSendWindow(increment int32) error {
	s.windowMu.Lock()
	defer s.windowMu.Unlock()

	if increment <= 0 {
		return fmt.Errorf("window increment must be positive: %d", increment)
	}

	// Check for overflow (RFC 7540 Section 6.9.1)
	if int64(s.sendWindow)+int64(increment) > MaxWindowSize {
		return fmt.Errorf("window size overflow")
	}

	s.sendWindow += increment
	return nil
}

// IncrementRecvWindow increases the receive window by the given amount
func (s *Stream) IncrementRecvWindow(increment int32) error {
	s.windowMu.Lock()
	defer s.windowMu.Unlock()

	if increment <= 0 {
		return fmt.Errorf("window increment must be positive: %d", increment)
	}

	// Check for overflow (RFC 7540 Section 6.9.1)
	if int64(s.recvWindow)+int64(increment) > MaxWindowSize {
		return fmt.Errorf("window size overflow")
	}

	s.recvWindow += increment
	return nil
}

// ConsumeSendWindow decreases the send window (when sending data)
func (s *Stream) ConsumeSendWindow(amount int32) error {
	s.windowMu.Lock()
	defer s.windowMu.Unlock()

	if amount < 0 {
		return fmt.Errorf("negative window consumption: %d", amount)
	}

	if s.sendWindow < amount {
		return fmt.Errorf("insufficient send window: have %d, need %d", s.sendWindow, amount)
	}

	s.sendWindow -= amount
	return nil
}

// ConsumeRecvWindow decreases the receive window (when receiving data)
func (s *Stream) ConsumeRecvWindow(amount int32) error {
	s.windowMu.Lock()
	defer s.windowMu.Unlock()

	if amount < 0 {
		return fmt.Errorf("negative window consumption: %d", amount)
	}

	if s.recvWindow < amount {
		return fmt.Errorf("insufficient receive window: have %d, need %d", s.recvWindow, amount)
	}

	s.recvWindow -= amount
	return nil
}

// SetPriority sets the stream priority
// Returns error if stream tries to depend on itself (RFC 7540 Section 5.3.1)
func (s *Stream) SetPriority(weight uint8, dependency uint32, exclusive bool) error {
	// RFC 7540 Section 5.3.1: A stream cannot depend on itself
	if s.id == dependency {
		return ErrStreamSelfDependency
	}

	s.priorityMu.Lock()
	defer s.priorityMu.Unlock()

	s.weight = weight
	s.dependency = dependency
	s.exclusive = exclusive
	s.updateActivity()

	return nil
}

// Priority returns the stream priority information
func (s *Stream) Priority() (weight uint8, dependency uint32, exclusive bool) {
	s.priorityMu.RLock()
	defer s.priorityMu.RUnlock()

	return s.weight, s.dependency, s.exclusive
}

// Write writes data to the stream (adds to send buffer)
// Enforces buffer size limits to prevent DoS attacks
func (s *Stream) Write(data []byte) (int, error) {
	s.sendBufMu.Lock()
	defer s.sendBufMu.Unlock()

	if s.sendClosed {
		return 0, io.EOF
	}

	dataLen := int64(len(data))

	// Check stream buffer size limit (security hardening)
	newSize := int64(len(s.sendBuf)) + dataLen
	if newSize > s.maxBufferSize {
		return 0, ErrBufferSizeExceeded
	}

	// Check connection-level buffer limit (security hardening)
	if s.conn != nil {
		if err := s.conn.trackBufferGrowth(dataLen); err != nil {
			return 0, err
		}
	}

	s.sendBuf = append(s.sendBuf, data...)
	s.updateActivity()

	return len(data), nil
}

// Read reads data from the stream (blocking if no data available)
func (s *Stream) Read(p []byte) (int, error) {
	s.recvBufMu.Lock()
	defer s.recvBufMu.Unlock()

	// Wait for data or close
	for len(s.recvBuf) == 0 && !s.recvClosed {
		s.recvCond.Wait()
	}

	if len(s.recvBuf) == 0 && s.recvClosed {
		return 0, io.EOF
	}

	n := copy(p, s.recvBuf)
	s.recvBuf = s.recvBuf[n:]
	s.updateActivity()

	// Track buffer shrinkage at connection level
	if s.conn != nil && n > 0 {
		s.conn.trackBufferShrink(int64(n))
	}

	return n, nil
}

// ReceiveData adds received data to the stream buffer
// Enforces buffer size limits to prevent DoS attacks
func (s *Stream) ReceiveData(data []byte) error {
	s.recvBufMu.Lock()
	defer s.recvBufMu.Unlock()

	if s.recvClosed {
		return fmt.Errorf("stream receive side closed")
	}

	dataLen := int64(len(data))

	// Check stream buffer size limit (security hardening)
	newSize := int64(len(s.recvBuf)) + dataLen
	if newSize > s.maxBufferSize {
		return ErrBufferSizeExceeded
	}

	// Check connection-level buffer limit (security hardening)
	if s.conn != nil {
		if err := s.conn.trackBufferGrowth(dataLen); err != nil {
			return err
		}
	}

	s.recvBuf = append(s.recvBuf, data...)
	s.recvCond.Broadcast()
	s.updateActivity()

	return nil
}

// SetRequestHeaders sets the request headers
func (s *Stream) SetRequestHeaders(headers []HeaderField) {
	s.headersMu.Lock()
	defer s.headersMu.Unlock()

	s.requestHeaders = headers
	s.updateActivity()
}

// SetResponseHeaders sets the response headers
func (s *Stream) SetResponseHeaders(headers []HeaderField) {
	s.headersMu.Lock()
	defer s.headersMu.Unlock()

	s.responseHeaders = headers
	s.updateActivity()
}

// SetTrailers sets the trailers
func (s *Stream) SetTrailers(trailers []HeaderField) {
	s.headersMu.Lock()
	defer s.headersMu.Unlock()

	s.trailers = trailers
	s.updateActivity()
}

// RequestHeaders returns the request headers
func (s *Stream) RequestHeaders() []HeaderField {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()

	return s.requestHeaders
}

// ResponseHeaders returns the response headers
func (s *Stream) ResponseHeaders() []HeaderField {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()

	return s.responseHeaders
}

// Trailers returns the trailers
func (s *Stream) Trailers() []HeaderField {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()

	return s.trailers
}

// Context returns the stream context
func (s *Stream) Context() context.Context {
	return s.ctx
}

// SetError sets an error on the stream
func (s *Stream) SetError(err error) {
	s.errMu.Lock()
	defer s.errMu.Unlock()

	if s.err == nil {
		s.err = err
	}
}

// Error returns the stream error
func (s *Stream) Error() error {
	s.errMu.RLock()
	defer s.errMu.RUnlock()

	return s.err
}

// updateActivity updates the last activity timestamp (caller must hold appropriate lock)
func (s *Stream) updateActivity() {
	s.activityMu.Lock()
	defer s.activityMu.Unlock()

	s.lastActivity = time.Now()
}

// LastActivity returns the last activity time
func (s *Stream) LastActivity() time.Time {
	s.activityMu.RLock()
	defer s.activityMu.RUnlock()

	return s.lastActivity
}

// Age returns the age of the stream since creation
func (s *Stream) Age() time.Duration {
	return time.Since(s.created)
}

// IdleTime returns how long the stream has been idle
func (s *Stream) IdleTime() time.Duration {
	s.activityMu.RLock()
	defer s.activityMu.RUnlock()

	return time.Since(s.lastActivity)
}

// ====== Stream Object Pooling ======
// Eliminates 6 allocations and 680 bytes per stream creation

var (
	// Stream object pool
	streamPool = sync.Pool{
		New: func() interface{} {
			return &Stream{}
		},
	}

	// Buffer pools for stream data
	bufferPool4K = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 4096)
			return &buf
		},
	}

	bufferPool16K = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 16384)
			return &buf
		},
	}

	// Header field slice pool (typical: 10-20 headers per request/response)
	headerPool = sync.Pool{
		New: func() interface{} {
			headers := make([]HeaderField, 0, 16)
			return &headers
		},
	}
)

// getPooledStream retrieves a stream from the pool and initializes it
// This function performs ~2 allocations (context + sync.Cond) vs 6 in NewStream
func getPooledStream(id uint32, initialWindowSize int32) *Stream {
	s := streamPool.Get().(*Stream)

	// Initialize/reset fields
	ctx, cancel := context.WithCancel(context.Background())

	s.id = id
	s.state = StateIdle
	s.sendWindow = initialWindowSize
	s.recvWindow = initialWindowSize
	s.weight = 15
	s.maxBufferSize = 1024 * 1024
	s.ctx = ctx
	s.cancel = cancel
	s.created = time.Now()
	s.lastActivity = time.Now()

	// Reset buffer slices (keep capacity, reset length)
	s.recvBuf = s.recvBuf[:0]
	s.sendBuf = s.sendBuf[:0]

	// Reset header slices
	s.requestHeaders = s.requestHeaders[:0]
	s.responseHeaders = s.responseHeaders[:0]
	s.trailers = s.trailers[:0]

	// Reset flags
	s.recvClosed = false
	s.sendClosed = false
	s.err = nil
	s.resetCode = 0
	s.dependency = 0
	s.exclusive = false
	s.conn = nil

	// Create condition variable if needed
	if s.recvCond == nil {
		s.recvCond = sync.NewCond(&s.recvBufMu)
	}

	return s
}

// putPooledStream returns a stream to the pool after cleanup
// Must be called when stream is fully closed to avoid memory leaks
func putPooledStream(s *Stream) {
	// Cancel context
	if s.cancel != nil {
		s.cancel()
	}

	// Clear large buffers to avoid keeping them in pool
	// Small buffers (< 4KB) are kept for reuse
	if cap(s.recvBuf) > 4096 {
		s.recvBuf = nil
	}
	if cap(s.sendBuf) > 4096 {
		s.sendBuf = nil
	}

	// Clear header slices (keep small capacity)
	if cap(s.requestHeaders) > 32 {
		s.requestHeaders = nil
	}
	if cap(s.responseHeaders) > 32 {
		s.responseHeaders = nil
	}
	if cap(s.trailers) > 32 {
		s.trailers = nil
	}

	// Return to pool
	streamPool.Put(s)
}

// getPooledBuffer retrieves a buffer from the appropriate pool
func getPooledBuffer(minSize int) *[]byte {
	if minSize <= 4096 {
		return bufferPool4K.Get().(*[]byte)
	}
	return bufferPool16K.Get().(*[]byte)
}

// putPooledBuffer returns a buffer to the appropriate pool
func putPooledBuffer(buf *[]byte, originalSize int) {
	// Clear buffer
	*buf = (*buf)[:0]

	// Return to appropriate pool
	if originalSize <= 4096 {
		bufferPool4K.Put(buf)
	} else {
		bufferPool16K.Put(buf)
	}
}

// getPooledHeaders retrieves a header slice from pool
func getPooledHeaders() *[]HeaderField {
	return headerPool.Get().(*[]HeaderField)
}

// putPooledHeaders returns a header slice to pool
func putPooledHeaders(headers *[]HeaderField) {
	// Clear slice
	*headers = (*headers)[:0]
	headerPool.Put(headers)
}
