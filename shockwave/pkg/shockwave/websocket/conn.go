package websocket

import (
	"crypto/rand"
	"io"
	"net"
	"sync"
	"time"
	"unicode/utf8"
)

// MessageType represents the type of a WebSocket message.
type MessageType int

const (
	// TextMessage denotes a text data message (UTF-8 encoded)
	TextMessage MessageType = 1

	// BinaryMessage denotes a binary data message
	BinaryMessage MessageType = 2

	// CloseMessage denotes a close control message
	CloseMessage MessageType = 8

	// PingMessage denotes a ping control message
	PingMessage MessageType = 9

	// PongMessage denotes a pong control message
	PongMessage MessageType = 10
)

// Conn represents a WebSocket connection.
// It handles fragmentation, control frames, and provides a simple message-based API.
type Conn struct {
	conn       net.Conn
	isServer   bool
	subprotocol string

	// Frame I/O
	frameReader *FrameReader
	frameWriter *FrameWriter
	writeMu     sync.Mutex // Serialize writes

	// Read state for fragmentation
	readMu          sync.Mutex
	readRemaining   []byte      // Remaining data from previous Read
	readMessage     []byte      // Buffer for assembling fragmented messages
	readMessageType MessageType // Type of message being assembled
	readFinal       bool        // True if current message is complete

	// Write state
	writeBuf []byte // Reusable write buffer

	// Close state
	closeOnce sync.Once
	closeSent bool
	closeErr  error

	// Ping/Pong handlers
	pingHandler func(appData string) error
	pongHandler func(appData string) error

	// Configuration
	readDeadline  time.Time
	writeDeadline time.Time
	maxMessageSize int64
}

// newConn creates a new WebSocket connection.
func newConn(netConn net.Conn, isServer bool, readBufSize, writeBufSize int, subprotocol string) *Conn {
	conn := &Conn{
		conn:           netConn,
		isServer:       isServer,
		subprotocol:    subprotocol,
		frameReader:    NewFrameReader(netConn),
		frameWriter:    NewFrameWriter(netConn),
		maxMessageSize: 32 * 1024 * 1024, // 32MB default
		writeBuf:       make([]byte, 0, writeBufSize),
	}

	// Set default ping/pong handlers
	conn.pingHandler = conn.defaultPingHandler
	conn.pongHandler = func(appData string) error { return nil }

	return conn
}

// ReadMessage reads the next data message from the connection.
// It handles fragmentation automatically and returns complete messages.
// Returns the message type (TextMessage or BinaryMessage) and the message payload.
//
// Control frames (Ping, Pong, Close) are handled automatically.
func (c *Conn) ReadMessage() (MessageType, []byte, error) {
	for {
		frame, err := c.frameReader.ReadFrame()
		if err != nil {
			return 0, nil, err
		}

		// Client frames must be masked, server frames must not be masked (RFC 6455 5.1)
		if c.isServer && !frame.Masked {
			c.Close()
			return 0, nil, ErrMaskRequired
		}
		if !c.isServer && frame.Masked {
			c.Close()
			return 0, nil, ErrMaskNotAllowed
		}

		// Handle control frames
		if frame.IsControl() {
			if err := c.handleControlFrame(frame); err != nil {
				return 0, nil, err
			}
			continue // Control frames are handled, continue reading
		}

		// Handle data frames
		msgType, data, err, needMore := c.handleDataFrame(frame)
		if needMore {
			continue // Fragmented message, continue reading
		}
		return msgType, data, err
	}
}

// ReadMessageInto reads the next data message into a caller-provided buffer.
// This allows zero-allocation reading when the caller manages buffers.
// Returns (messageType, bytesRead, error).
// If buf is too small, returns ErrMessageTooLarge.
//
// Performance: Only 1 allocation (for the Frame header) instead of 2.
// Use with buffer pooling for true zero-allocation reading.
//
// Example:
//   buf := make([]byte, 4096)
//   msgType, n, err := conn.ReadMessageInto(buf)
//   data := buf[:n]
func (c *Conn) ReadMessageInto(buf []byte) (MessageType, int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	// Reset assembly state
	c.readMessageType = 0
	bytesRead := 0

	for {
		// Read frame into provided buffer (offset by bytes already read)
		if bytesRead >= len(buf) {
			c.Close()
			return 0, 0, ErrMessageTooLarge
		}

		frame, n, err := c.frameReader.ReadFrameInto(buf[bytesRead:])
		if err != nil {
			return 0, 0, err
		}

		// Validate masking
		if c.isServer && !frame.Masked {
			c.Close()
			return 0, 0, ErrMaskRequired
		}
		if !c.isServer && frame.Masked {
			c.Close()
			return 0, 0, ErrMaskNotAllowed
		}

		// Handle control frames
		if frame.IsControl() {
			if err := c.handleControlFrame(frame); err != nil {
				return 0, 0, err
			}
			continue // Control frames don't affect data assembly
		}

		// Handle data frames
		if frame.Opcode == OpcodeContinuation {
			if c.readMessageType == 0 {
				c.Close()
				return 0, 0, ErrProtocolViolation
			}
		} else {
			if c.readMessageType != 0 {
				c.Close()
				return 0, 0, ErrProtocolViolation
			}
			c.readMessageType = MessageType(frame.Opcode)
		}

		bytesRead += n

		// Check message size
		if int64(bytesRead) > c.maxMessageSize {
			c.Close()
			return 0, 0, ErrMessageTooLarge
		}

		// If final frame, return
		if frame.Fin {
			msgType := c.readMessageType
			c.readMessageType = 0

			// Validate UTF-8 for text messages
			if msgType == TextMessage && !utf8.Valid(buf[:bytesRead]) {
				c.Close()
				return 0, 0, ErrInvalidUTF8
			}

			return msgType, bytesRead, nil
		}

		// Fragmented, continue reading
	}
}

// handleDataFrame processes data frames (Text, Binary, Continuation).
// Handles fragmentation by assembling continuation frames.
// Returns (msgType, data, error, needMore) where needMore=true means continue reading.
func (c *Conn) handleDataFrame(frame *Frame) (MessageType, []byte, error, bool) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	// Check for protocol violations
	if frame.Opcode == OpcodeContinuation {
		// Continuation frame without a prior fragment
		if c.readMessageType == 0 {
			c.Close()
			return 0, nil, ErrProtocolViolation, false
		}
	} else {
		// Data frame while already assembling a message
		if c.readMessageType != 0 {
			c.Close()
			return 0, nil, ErrProtocolViolation, false
		}

		// Start new message
		c.readMessageType = MessageType(frame.Opcode)
		c.readMessage = c.readMessage[:0] // Reset buffer
	}

	// Append payload to message buffer
	if len(frame.Payload) > 0 {
		// Check message size limit
		if int64(len(c.readMessage)+len(frame.Payload)) > c.maxMessageSize {
			c.Close()
			return 0, nil, ErrMessageTooLarge, false
		}

		c.readMessage = append(c.readMessage, frame.Payload...)
	}

	// If this is the final frame, return the complete message
	if frame.Fin {
		msgType := c.readMessageType
		payload := c.readMessage

		// Validate UTF-8 for text messages (RFC 6455 8.1)
		if msgType == TextMessage && !utf8.Valid(payload) {
			c.Close()
			return 0, nil, ErrInvalidUTF8, false
		}

		// Make a copy of the payload (since our buffer will be reused)
		result := make([]byte, len(payload))
		copy(result, payload)

		// Reset state for next message
		c.readMessageType = 0
		c.readMessage = c.readMessage[:0]

		return msgType, result, nil, false
	}

	// Fragmented message, need to continue reading
	return 0, nil, nil, true
}

// handleControlFrame processes control frames (Ping, Pong, Close).
func (c *Conn) handleControlFrame(frame *Frame) error {
	switch frame.Opcode {
	case OpcodePing:
		// Respond with Pong
		if err := c.pingHandler(string(frame.Payload)); err != nil {
			return err
		}

	case OpcodePong:
		// Pong received
		if err := c.pongHandler(string(frame.Payload)); err != nil {
			return err
		}

	case OpcodeClose:
		// Parse close frame
		var closeCode uint16
		var closeReason string

		if len(frame.Payload) >= 2 {
			closeCode = uint16(frame.Payload[0])<<8 | uint16(frame.Payload[1])
			if len(frame.Payload) > 2 {
				closeReason = string(frame.Payload[2:])

				// Validate UTF-8 in close reason
				if !utf8.ValidString(closeReason) {
					return ErrInvalidUTF8
				}
			}

			// Validate close code (RFC 6455 7.4.1)
			if !isValidCloseCode(closeCode) {
				return ErrInvalidCloseCode
			}
		}

		// Send close response if we haven't already
		if !c.closeSent {
			c.WriteControl(CloseMessage, frame.Payload)
			c.closeSent = true
		}

		// Close the connection
		return io.EOF
	}

	return nil
}

// WriteMessage writes a complete message to the connection.
// The message is sent as a single frame (or fragmented if too large).
func (c *Conn) WriteMessage(messageType MessageType, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.writeDeadline != (time.Time{}) {
		c.conn.SetWriteDeadline(c.writeDeadline)
	}

	// Validate UTF-8 for text messages
	if messageType == TextMessage && !utf8.Valid(data) {
		return ErrInvalidUTF8
	}

	// Get opcode
	var opcode byte
	switch messageType {
	case TextMessage:
		opcode = OpcodeText
	case BinaryMessage:
		opcode = OpcodeBinary
	default:
		return ErrInvalidOpcode
	}

	// Get mask key for client→server frames
	var maskKey *[4]byte
	if !c.isServer {
		var key [4]byte
		if _, err := rand.Read(key[:]); err != nil {
			return err
		}
		maskKey = &key
	}

	// TODO: Fragment large messages (for now, send as single frame)
	return c.frameWriter.WriteFrame(opcode, true, data, maskKey)
}

// WriteControl writes a control frame (Ping, Pong, Close).
func (c *Conn) WriteControl(messageType MessageType, data []byte) error {
	if len(data) > MaxControlFramePayload {
		return ErrInvalidControlFrame
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.writeDeadline != (time.Time{}) {
		c.conn.SetWriteDeadline(c.writeDeadline)
	}

	var opcode byte
	switch messageType {
	case CloseMessage:
		opcode = OpcodeClose
		c.closeSent = true
	case PingMessage:
		opcode = OpcodePing
	case PongMessage:
		opcode = OpcodePong
	default:
		return ErrInvalidOpcode
	}

	// Get mask key for client→server frames
	var maskKey *[4]byte
	if !c.isServer {
		var key [4]byte
		if _, err := rand.Read(key[:]); err != nil {
			return err
		}
		maskKey = &key
	}

	return c.frameWriter.WriteControlFrame(opcode, data, maskKey)
}

// WritePing sends a Ping frame.
func (c *Conn) WritePing(data []byte) error {
	return c.WriteControl(PingMessage, data)
}

// WritePong sends a Pong frame.
func (c *Conn) WritePong(data []byte) error {
	return c.WriteControl(PongMessage, data)
}

// Close sends a Close frame and closes the underlying connection.
func (c *Conn) Close() error {
	c.closeOnce.Do(func() {
		// Send close frame if we haven't already
		if !c.closeSent {
			payload := make([]byte, 2)
			payload[0] = byte(CloseNormalClosure >> 8)
			payload[1] = byte(CloseNormalClosure & 0xFF)
			c.WriteControl(CloseMessage, payload)
		}

		// Close underlying connection
		c.closeErr = c.conn.Close()
	})
	return c.closeErr
}

// CloseWithCode sends a Close frame with a specific code and reason.
func (c *Conn) CloseWithCode(code uint16, reason string) error {
	if !isValidCloseCode(code) {
		return ErrInvalidCloseCode
	}

	payload := make([]byte, 2+len(reason))
	payload[0] = byte(code >> 8)
	payload[1] = byte(code)
	copy(payload[2:], reason)

	c.closeOnce.Do(func() {
		c.WriteControl(CloseMessage, payload)
		c.closeErr = c.conn.Close()
	})
	return c.closeErr
}

// SetReadDeadline sets the read deadline on the underlying connection.
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = t
	return c.conn.SetWriteDeadline(t)
}

// SetPingHandler sets the handler for Ping frames.
// The default handler sends a Pong frame.
func (c *Conn) SetPingHandler(handler func(appData string) error) {
	c.pingHandler = handler
}

// SetPongHandler sets the handler for Pong frames.
func (c *Conn) SetPongHandler(handler func(appData string) error) {
	c.pongHandler = handler
}

// defaultPingHandler is the default handler for Ping frames.
func (c *Conn) defaultPingHandler(appData string) error {
	return c.WritePong([]byte(appData))
}

// SetMaxMessageSize sets the maximum size of received messages.
// Default is 32MB.
func (c *Conn) SetMaxMessageSize(size int64) {
	c.maxMessageSize = size
}

// Subprotocol returns the negotiated subprotocol.
func (c *Conn) Subprotocol() string {
	return c.subprotocol
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Helper functions

// isValidCloseCode validates close status codes per RFC 6455 7.4.1.
func isValidCloseCode(code uint16) bool {
	switch code {
	case 1000, 1001, 1002, 1003, 1007, 1008, 1009, 1010, 1011:
		return true
	case 1004, 1005, 1006, 1015: // Reserved, not sent on wire
		return false
	default:
		// 3000-3999: registered
		// 4000-4999: private use
		return code >= 3000 && code <= 4999
	}
}
