package quic

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"
)

// TLSConn wraps a QUIC connection to provide a net.Conn interface for crypto/tls
// It maps TLS records to QUIC CRYPTO frames at the appropriate encryption level
type TLSConn struct {
	conn *Connection

	// Separate buffers for each encryption level
	initialBuffer   *cryptoBuffer
	handshakeBuffer *cryptoBuffer
	appDataBuffer   *cryptoBuffer

	// Current encryption level for reading and writing
	readLevel  EncryptionLevel
	writeLevel EncryptionLevel

	// Synchronization
	mu        sync.RWMutex
	readCond  *sync.Cond
	writeCond *sync.Cond

	// Close state
	closed    bool
	closeErr  error
	closeCond *sync.Cond
}

// cryptoBuffer buffers CRYPTO frame data for a specific encryption level
type cryptoBuffer struct {
	data   []byte
	offset uint64 // Next expected offset
	mu     sync.Mutex
}

func newCryptoBuffer() *cryptoBuffer {
	return &cryptoBuffer{
		data: make([]byte, 0, 4096),
	}
}

func (cb *cryptoBuffer) Write(data []byte, offset uint64) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if this is the next expected data
	if offset != cb.offset {
		// Out of order data - would need buffering in production
		// For now, return error
		return ErrProtocolViolation
	}

	cb.data = append(cb.data, data...)
	cb.offset += uint64(len(data))
	return nil
}

func (cb *cryptoBuffer) Read(p []byte) (int, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if len(cb.data) == 0 {
		return 0, nil // No data available
	}

	n := copy(p, cb.data)
	cb.data = cb.data[n:]
	return n, nil
}

func (cb *cryptoBuffer) Available() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return len(cb.data)
}

// NewTLSConn creates a new TLS connection wrapper
func NewTLSConn(conn *Connection) *TLSConn {
	tc := &TLSConn{
		conn:            conn,
		initialBuffer:   newCryptoBuffer(),
		handshakeBuffer: newCryptoBuffer(),
		appDataBuffer:   newCryptoBuffer(),
		readLevel:       EncryptionLevelInitial,
		writeLevel:      EncryptionLevelInitial,
	}

	tc.readCond = sync.NewCond(&tc.mu)
	tc.writeCond = sync.NewCond(&tc.mu)
	tc.closeCond = sync.NewCond(&tc.mu)

	return tc
}

// Read implements net.Conn.Read
// Reads CRYPTO frame data from the current read encryption level
func (tc *TLSConn) Read(p []byte) (int, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for {
		if tc.closed {
			if tc.closeErr != nil {
				return 0, tc.closeErr
			}
			return 0, io.EOF
		}

		// Get buffer for current read level
		buf := tc.getReadBuffer()

		// Try to read from buffer
		if buf.Available() > 0 {
			n, _ := buf.Read(p)
			return n, nil
		}

		// No data available, wait for more
		tc.readCond.Wait()
	}
}

// Write implements net.Conn.Write
// Writes TLS records as CRYPTO frames at the current write encryption level
func (tc *TLSConn) Write(p []byte) (int, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.closed {
		return 0, io.ErrClosedPipe
	}

	// Create CRYPTO frame with this data
	frame := &CryptoFrame{
		Offset: tc.getCryptoOffset(tc.writeLevel),
		Data:   make([]byte, len(p)),
	}
	copy(frame.Data, p)

	// Send frame at current write level
	tc.mu.Unlock()
	err := tc.conn.sendCryptoFrame(frame, tc.writeLevel)
	tc.mu.Lock()

	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close implements net.Conn.Close
func (tc *TLSConn) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.closed {
		return tc.closeErr
	}

	tc.closed = true
	tc.readCond.Broadcast()
	tc.writeCond.Broadcast()
	tc.closeCond.Broadcast()

	return nil
}

// LocalAddr implements net.Conn.LocalAddr
func (tc *TLSConn) LocalAddr() net.Addr {
	return tc.conn.localAddr
}

// RemoteAddr implements net.Conn.RemoteAddr
func (tc *TLSConn) RemoteAddr() net.Addr {
	return tc.conn.remoteAddr
}

// SetDeadline implements net.Conn.SetDeadline
func (tc *TLSConn) SetDeadline(t time.Time) error {
	tc.SetReadDeadline(t)
	tc.SetWriteDeadline(t)
	return nil
}

// SetReadDeadline implements net.Conn.SetReadDeadline
func (tc *TLSConn) SetReadDeadline(t time.Time) error {
	// TODO: Implement deadline support
	return nil
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline
func (tc *TLSConn) SetWriteDeadline(t time.Time) error {
	// TODO: Implement deadline support
	return nil
}

// HandleCryptoFrame processes an incoming CRYPTO frame
func (tc *TLSConn) HandleCryptoFrame(frame *CryptoFrame, level EncryptionLevel) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Get buffer for this encryption level
	var buf *cryptoBuffer
	switch level {
	case EncryptionLevelInitial:
		buf = tc.initialBuffer
	case EncryptionLevelHandshake:
		buf = tc.handshakeBuffer
	case EncryptionLevelApplication:
		buf = tc.appDataBuffer
	default:
		return ErrProtocolViolation
	}

	// Write data to buffer
	if err := buf.Write(frame.Data, frame.Offset); err != nil {
		return err
	}

	// Notify readers
	tc.readCond.Broadcast()

	return nil
}

// SetReadLevel sets the encryption level for reading CRYPTO frames
func (tc *TLSConn) SetReadLevel(level EncryptionLevel) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.readLevel = level
	tc.readCond.Broadcast()
}

// SetWriteLevel sets the encryption level for writing CRYPTO frames
func (tc *TLSConn) SetWriteLevel(level EncryptionLevel) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.writeLevel = level
}

// getReadBuffer returns the buffer for the current read level
func (tc *TLSConn) getReadBuffer() *cryptoBuffer {
	switch tc.readLevel {
	case EncryptionLevelInitial:
		return tc.initialBuffer
	case EncryptionLevelHandshake:
		return tc.handshakeBuffer
	case EncryptionLevelApplication:
		return tc.appDataBuffer
	default:
		return tc.initialBuffer
	}
}

// getCryptoOffset returns the current offset for CRYPTO frames at a level
func (tc *TLSConn) getCryptoOffset(level EncryptionLevel) uint64 {
	switch level {
	case EncryptionLevelInitial:
		return tc.initialBuffer.offset
	case EncryptionLevelHandshake:
		return tc.handshakeBuffer.offset
	case EncryptionLevelApplication:
		return tc.appDataBuffer.offset
	default:
		return 0
	}
}

// TLSConfig creates a TLS configuration for QUIC
func (conn *Connection) TLSConfig(base *tls.Config) *tls.Config {
	config := base.Clone()

	// QUIC requires TLS 1.3
	config.MinVersion = tls.VersionTLS13
	config.MaxVersion = tls.VersionTLS13

	// Set ALPN for HTTP/3
	if len(config.NextProtos) == 0 {
		config.NextProtos = []string{"h3"}
	}

	return config
}
