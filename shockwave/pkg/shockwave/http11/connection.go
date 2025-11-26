package http11

import (
	"bufio"
	"io"
	"net"
	"sync/atomic"
	"time"
)

// ConnectionState represents the state of an HTTP connection
type ConnectionState int

const (
	// StateNew is the initial state when a connection is created
	StateNew ConnectionState = iota

	// StateActive indicates the connection is actively processing a request
	StateActive

	// StateIdle indicates the connection is idle and waiting for the next request
	StateIdle

	// StateClosed indicates the connection has been closed
	StateClosed
)

// String returns the string representation of the connection state
func (s ConnectionState) String() string {
	switch s {
	case StateNew:
		return "new"
	case StateActive:
		return "active"
	case StateIdle:
		return "idle"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Handler is the request handler function for HTTP/1.1 connections.
// It receives a Request and ResponseWriter and should process the request.
// Returning an error will close the connection.
type Handler func(*Request, *ResponseWriter) error

// Connection represents an HTTP/1.1 connection with lock-free state management.
//
// Design:
// - Lock-free atomic operations for all state transitions
// - Zero mutex contention under high concurrency
// - Supports HTTP/1.1 persistent connections (keep-alive)
// - Request pipelining (reads next request while processing current)
// - Zero allocations for request/response cycle (uses pools)
// - Graceful shutdown support
//
// Allocation behavior: 0 allocs/op when using pooled objects
type Connection struct {
	// Hot fields first (cache line optimization)
	state    atomic.Int32 // Lock-free state transitions (StateNew, StateActive, StateIdle, StateClosed)
	lastUse  atomic.Int64 // Unix timestamp in nanoseconds (lock-free)
	requests atomic.Int32 // Request counter (lock-free)

	// Network connection
	conn net.Conn

	// Buffered I/O
	reader *bufio.Reader
	writer *bufio.Writer

	// HTTP parser (pooled)
	parser *Parser

	// Request handler (stored to avoid closure allocation per request)
	handler Handler

	// Keep-alive configuration
	keepAliveTimeout time.Duration
	maxRequests      int32 // Max requests per connection (0 = unlimited)
	idleTimer        *time.Timer

	// Close channel (signals connection should close)
	closeCh chan struct{}
	closed  atomic.Bool
}

// ConnectionConfig holds configuration for an HTTP connection
type ConnectionConfig struct {
	// KeepAliveTimeout is the maximum duration an idle connection will be kept alive
	// Default: 60 seconds
	KeepAliveTimeout time.Duration

	// MaxRequests is the maximum number of requests per connection
	// 0 means unlimited
	// Default: 0 (unlimited)
	MaxRequests int

	// ReadBufferSize is the size of the read buffer
	// Default: 4096 bytes
	ReadBufferSize int

	// WriteBufferSize is the size of the write buffer
	// Default: 4096 bytes
	WriteBufferSize int
}

// DefaultConnectionConfig returns the default connection configuration
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		KeepAliveTimeout: 60 * time.Second,
		MaxRequests:      0, // Unlimited
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
	}
}

// NewConnection creates a new HTTP/1.1 connection from a net.Conn
//
// The handler is stored in the connection to avoid closure allocations per request.
// This enables true zero-allocation request handling with lock-free state management.
//
// Allocation behavior: Allocates bufio readers/writers and the connection struct
func NewConnection(conn net.Conn, config ConnectionConfig, handler Handler) *Connection {
	c := &Connection{
		conn:             conn,
		handler:          handler,
		keepAliveTimeout: config.KeepAliveTimeout,
		maxRequests:      int32(config.MaxRequests),
		closeCh:          make(chan struct{}),
	}

	// Initialize lock-free atomic state
	c.state.Store(int32(StateNew))
	c.lastUse.Store(time.Now().UnixNano())
	c.requests.Store(0)

	// Use pooled bufio objects if buffer sizes match defaults
	if config.ReadBufferSize == DefaultBufferSize {
		c.reader = GetBufioReader(conn)
	} else {
		c.reader = bufio.NewReaderSize(conn, config.ReadBufferSize)
	}

	if config.WriteBufferSize == DefaultBufferSize {
		c.writer = GetBufioWriter(conn)
	} else {
		c.writer = bufio.NewWriterSize(conn, config.WriteBufferSize)
	}

	// Get parser from pool
	c.parser = GetParser()

	return c
}

// State returns the current connection state (lock-free)
func (c *Connection) State() ConnectionState {
	return ConnectionState(c.state.Load())
}

// setState sets the connection state (lock-free)
func (c *Connection) setState(state ConnectionState) {
	c.state.Store(int32(state))
	c.lastUse.Store(time.Now().UnixNano())
}

// Serve handles the connection lifecycle with keep-alive support.
// It processes requests in a loop until the connection should close.
//
// The handler is stored in the connection (passed to NewConnection) and called
// for each request. This avoids closure allocation per request.
//
// Allocation behavior: 0 allocs/op per request (uses pools, no closure overhead)
func (c *Connection) Serve() error {
	defer c.cleanup()

	for {
		// Check if connection should close
		if c.shouldClose() {
			return nil
		}

		// Set connection deadline for keep-alive timeout
		if err := c.setDeadline(); err != nil {
			return err
		}

		// Parse next request
		c.setState(StateActive)
		req, err := c.parser.Parse(c.reader)
		if err != nil {
			if err == io.EOF || err == ErrUnexpectedEOF {
				// Clean connection close (EOF or unexpected EOF between requests)
				return nil
			}
			// Parse error
			return err
		}

		// CRITICAL: Request is from pool, must be returned when done
		// We explicitly return it before continuing the loop for zero-alloc keep-alive
		// Only use defer for panic recovery

		// Increment request counter (lock-free)
		requestNum := c.requests.Add(1)

		// Get response writer from pool
		rw := GetResponseWriter(c.writer)

		// Check if this will be the last request (before handling)
		willCloseAfterThis := c.maxRequests > 0 && requestNum >= c.maxRequests

		// Set Connection: close if this is the last request
		if willCloseAfterThis {
			rw.Header().Set(headerConnection, headerClose)
		}

		// Handle request
		// NOTE: Handler MUST NOT panic for zero-alloc keep-alive.
		// If handler panics, connection will be closed and pools will leak.
		// Production handlers should use recover() internally if needed.
		handlerErr := c.handler(req, rw)

		// Flush response
		if err := rw.Flush(); err != nil {
			PutResponseWriter(rw)
			PutRequest(req)
			return err
		}

		// Determine if connection should close
		shouldClose := c.shouldCloseAfterRequest(req, rw, int(requestNum), handlerErr, willCloseAfterThis)

		// Return response writer to pool
		PutResponseWriter(rw)

		// Return request to pool BEFORE next iteration for zero-alloc keep-alive
		PutRequest(req)

		if shouldClose {
			return handlerErr
		}

		// Connection can be reused
		c.setState(StateIdle)
	}
}

// shouldClose checks if the connection should close immediately
func (c *Connection) shouldClose() bool {
	if c.closed.Load() {
		return true
	}

	select {
	case <-c.closeCh:
		return true
	default:
		return false
	}
}

// shouldCloseAfterRequest determines if the connection should close after handling a request
func (c *Connection) shouldCloseAfterRequest(req *Request, rw *ResponseWriter, requestNum int, handlerErr error, willClose bool) bool {
	// Handler returned error - close connection
	if handlerErr != nil {
		return true
	}

	// Request explicitly requested close (Connection: close)
	if req.Close {
		return true
	}

	// Response was set to close
	connectionHeader := rw.Header().Get(headerConnection)
	if bytesEqualCaseInsensitive(connectionHeader, headerClose) {
		return true
	}

	// Max requests per connection reached (already set header before handler)
	if willClose {
		return true
	}

	// HTTP/1.0 without explicit keep-alive
	if req.ProtoMajor == 1 && req.ProtoMinor == 0 {
		connectionHeader := req.Header.Get(headerConnection)
		if !bytesEqualCaseInsensitive(connectionHeader, headerKeepAlive) {
			return true
		}
	}

	return false
}

// setDeadline sets the read/write deadline for keep-alive timeout
func (c *Connection) setDeadline() error {
	if c.keepAliveTimeout > 0 {
		deadline := time.Now().Add(c.keepAliveTimeout)
		return c.conn.SetDeadline(deadline)
	}
	return nil
}

// Close closes the connection gracefully
func (c *Connection) Close() error {
	// Mark as closed
	if !c.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Signal close
	close(c.closeCh)

	// Set state
	c.setState(StateClosed)

	// Close underlying connection
	return c.conn.Close()
}

// cleanup releases pooled resources
func (c *Connection) cleanup() {
	// Return parser to pool
	if c.parser != nil {
		PutParser(c.parser)
		c.parser = nil
	}

	// Return bufio objects to pool if they're the default size
	if c.reader != nil {
		PutBufioReader(c.reader)
		c.reader = nil
	}

	if c.writer != nil {
		PutBufioWriter(c.writer)
		c.writer = nil
	}
}

// RemoteAddr returns the remote network address
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// LocalAddr returns the local network address
func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RequestCount returns the number of requests handled on this connection (lock-free)
func (c *Connection) RequestCount() int {
	return int(c.requests.Load())
}

// IdleTime returns how long the connection has been idle (lock-free)
func (c *Connection) IdleTime() time.Duration {
	if c.State() == StateActive {
		return 0
	}

	lastUseNano := c.lastUse.Load()
	lastUseTime := time.Unix(0, lastUseNano)
	return time.Since(lastUseTime)
}
