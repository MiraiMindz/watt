package quic

import (
	"errors"
	"io"
	"sync"
)

// Stream types (RFC 9000 Section 2.1)
// Stream IDs encode type and initiator:
//   Bits: | 0 (initiator) | 1 (direction) |
//   - Bit 0: 0=client-initiated, 1=server-initiated
//   - Bit 1: 0=bidirectional, 1=unidirectional

const (
	streamTypeBidiMask = 0x02
	streamTypeServerMask = 0x01
)

var (
	ErrStreamClosed = errors.New("quic: stream closed")
	ErrStreamReset  = errors.New("quic: stream reset")
	ErrFlowControl  = errors.New("quic: flow control limit exceeded")
)

// StreamType represents the type of stream
type StreamType uint8

const (
	StreamTypeBidirectional StreamType = iota
	StreamTypeUnidirectional
)

// Stream represents a QUIC stream
type Stream struct {
	id         uint64
	conn       *Connection
	streamType StreamType

	// Send state
	sendMu      sync.Mutex
	sendBuf     []byte
	sendOffset  uint64
	sendClosed  bool
	sendMaxData uint64 // Flow control limit

	// Receive state
	recvMu         sync.Mutex
	recvBuf        []byte
	recvOffset     uint64
	recvFinalSize  uint64
	recvClosed     bool
	recvFin        bool
	recvMaxData    uint64 // Flow control limit we've advertised
	recvFrames     map[uint64][]byte // Out-of-order frames

	// Error state
	resetCode uint64
	stopCode  uint64
	resetErr  error
}

// newStream creates a new stream
func newStream(id uint64, conn *Connection, maxData uint64) *Stream {
	streamType := StreamTypeBidirectional
	if id&streamTypeBidiMask != 0 {
		streamType = StreamTypeUnidirectional
	}

	return &Stream{
		id:          id,
		conn:        conn,
		streamType:  streamType,
		sendMaxData: maxData,
		recvMaxData: maxData,
		recvFrames:  make(map[uint64][]byte),
	}
}

// ID returns the stream ID
func (s *Stream) ID() uint64 {
	return s.id
}

// IsClientInitiated returns true if the stream was initiated by the client
func (s *Stream) IsClientInitiated() bool {
	return s.id&streamTypeServerMask == 0
}

// IsBidirectional returns true if the stream is bidirectional
func (s *Stream) IsBidirectional() bool {
	return s.id&streamTypeBidiMask == 0
}

// Read reads data from the stream
func (s *Stream) Read(p []byte) (int, error) {
	s.recvMu.Lock()
	defer s.recvMu.Unlock()

	// Check for errors
	if s.resetErr != nil {
		return 0, s.resetErr
	}

	// If no data and stream is closed, return EOF
	if len(s.recvBuf) == 0 && s.recvClosed {
		return 0, io.EOF
	}

	// If no data available, would need to block/wait
	// For now, return what we have
	if len(s.recvBuf) == 0 {
		return 0, nil
	}

	// Copy available data
	n := copy(p, s.recvBuf)
	s.recvBuf = s.recvBuf[n:]
	s.recvOffset += uint64(n)

	return n, nil
}

// Write writes data to the stream
func (s *Stream) Write(p []byte) (int, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if s.sendClosed {
		return 0, ErrStreamClosed
	}

	// Check flow control
	if s.sendOffset+uint64(len(p)) > s.sendMaxData {
		return 0, ErrFlowControl
	}

	// Buffer data for sending
	s.sendBuf = append(s.sendBuf, p...)

	// Create STREAM frame
	frame := &StreamFrame{
		StreamID: s.id,
		Offset:   s.sendOffset,
		Data:     make([]byte, len(p)),
		Fin:      false,
	}
	copy(frame.Data, p)

	// Queue frame for transmission
	if s.conn != nil {
		s.conn.queueFrame(frame)
	}

	s.sendOffset += uint64(len(p))

	return len(p), nil
}

// Close closes the stream for writing
func (s *Stream) Close() error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if s.sendClosed {
		return nil
	}

	s.sendClosed = true

	// Send STREAM frame with FIN bit
	frame := &StreamFrame{
		StreamID: s.id,
		Offset:   s.sendOffset,
		Data:     nil,
		Fin:      true,
	}

	if s.conn != nil {
		s.conn.queueFrame(frame)
	}

	return nil
}

// Reset resets the stream with an error code
func (s *Stream) Reset(errorCode uint64) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if s.sendClosed {
		return nil
	}

	s.sendClosed = true
	s.resetCode = errorCode

	// Send RESET_STREAM frame
	frame := &ResetStreamFrame{
		StreamID:  s.id,
		ErrorCode: errorCode,
		FinalSize: s.sendOffset,
	}

	if s.conn != nil {
		s.conn.queueFrame(frame)
	}

	return nil
}

// handleStreamFrame processes an incoming STREAM frame
func (s *Stream) handleStreamFrame(frame *StreamFrame) error {
	s.recvMu.Lock()
	defer s.recvMu.Unlock()

	// Check flow control
	endOffset := frame.Offset + uint64(len(frame.Data))
	if endOffset > s.recvMaxData {
		return ErrFlowControl
	}

	// Check if this is the expected offset
	if frame.Offset == s.recvOffset {
		// In-order frame, append to buffer
		s.recvBuf = append(s.recvBuf, frame.Data...)
		s.recvOffset += uint64(len(frame.Data))

		// Check for buffered out-of-order frames
		for {
			if data, ok := s.recvFrames[s.recvOffset]; ok {
				s.recvBuf = append(s.recvBuf, data...)
				s.recvOffset += uint64(len(data))
				delete(s.recvFrames, s.recvOffset-uint64(len(data)))
			} else {
				break
			}
		}
	} else if frame.Offset > s.recvOffset {
		// Out-of-order frame, buffer it
		s.recvFrames[frame.Offset] = make([]byte, len(frame.Data))
		copy(s.recvFrames[frame.Offset], frame.Data)
	}
	// else: duplicate frame, ignore

	// Handle FIN
	if frame.Fin {
		s.recvFin = true
		s.recvFinalSize = frame.Offset + uint64(len(frame.Data))

		// If we've received all data, close the stream
		if s.recvOffset >= s.recvFinalSize {
			s.recvClosed = true
		}
	}

	return nil
}

// handleResetStream processes a RESET_STREAM frame
func (s *Stream) handleResetStream(frame *ResetStreamFrame) error {
	s.recvMu.Lock()
	defer s.recvMu.Unlock()

	s.recvClosed = true
	s.resetErr = ErrStreamReset
	s.resetCode = frame.ErrorCode

	return nil
}

// handleStopSending processes a STOP_SENDING frame
func (s *Stream) handleStopSending(frame *StopSendingFrame) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	s.stopCode = frame.ErrorCode

	// Reset the stream
	return s.Reset(frame.ErrorCode)
}

// updateSendMaxData updates the flow control limit
func (s *Stream) updateSendMaxData(maxData uint64) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if maxData > s.sendMaxData {
		s.sendMaxData = maxData
	}
}

// StreamManager manages all streams for a connection
type StreamManager struct {
	mu      sync.RWMutex
	streams map[uint64]*Stream

	// Stream limits
	maxStreamsBidi uint64
	maxStreamsUni  uint64

	// Next stream IDs
	nextBidiClient uint64
	nextBidiServer uint64
	nextUniClient  uint64
	nextUniServer  uint64

	conn *Connection
}

// newStreamManager creates a new stream manager
func newStreamManager(conn *Connection) *StreamManager {
	return &StreamManager{
		streams:        make(map[uint64]*Stream),
		maxStreamsBidi: 100,
		maxStreamsUni:  100,
		nextBidiClient: 0, // Client-initiated bidirectional
		nextBidiServer: 1, // Server-initiated bidirectional
		nextUniClient:  2, // Client-initiated unidirectional
		nextUniServer:  3, // Server-initiated unidirectional
		conn:           conn,
	}
}

// OpenStream opens a new stream
func (sm *StreamManager) OpenStream(bidirectional bool, isClient bool) (*Stream, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var streamID uint64
	if bidirectional {
		if isClient {
			streamID = sm.nextBidiClient
			sm.nextBidiClient += 4
		} else {
			streamID = sm.nextBidiServer
			sm.nextBidiServer += 4
		}

		// Check limit
		if streamID/4 >= sm.maxStreamsBidi {
			return nil, errors.New("quic: max bidirectional streams exceeded")
		}
	} else {
		if isClient {
			streamID = sm.nextUniClient
			sm.nextUniClient += 4
		} else {
			streamID = sm.nextUniServer
			sm.nextUniServer += 4
		}

		// Check limit
		if streamID/4 >= sm.maxStreamsUni {
			return nil, errors.New("quic: max unidirectional streams exceeded")
		}
	}

	stream := newStream(streamID, sm.conn, 1024*1024) // 1MB default
	sm.streams[streamID] = stream

	return stream, nil
}

// GetStream gets an existing stream or creates it if it doesn't exist
func (sm *StreamManager) GetStream(streamID uint64) *Stream {
	sm.mu.RLock()
	stream, exists := sm.streams[streamID]
	sm.mu.RUnlock()

	if exists {
		return stream
	}

	// Create new stream for peer-initiated streams
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check after acquiring write lock
	if stream, exists := sm.streams[streamID]; exists {
		return stream
	}

	stream = newStream(streamID, sm.conn, 1024*1024)
	sm.streams[streamID] = stream

	return stream
}

// CloseStream closes and removes a stream
func (sm *StreamManager) CloseStream(streamID uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.streams, streamID)
}

// UpdateMaxStreams updates the maximum number of streams
func (sm *StreamManager) UpdateMaxStreams(maxStreams uint64, bidirectional bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if bidirectional {
		sm.maxStreamsBidi = maxStreams
	} else {
		sm.maxStreamsUni = maxStreams
	}
}

// GetAllStreams returns all active streams
func (sm *StreamManager) GetAllStreams() []*Stream {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	streams := make([]*Stream, 0, len(sm.streams))
	for _, stream := range sm.streams {
		streams = append(streams, stream)
	}

	return streams
}
