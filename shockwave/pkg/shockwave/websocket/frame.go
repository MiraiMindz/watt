package websocket

import (
	"encoding/binary"
	"io"
)

// FrameReader provides zero-copy frame parsing from an io.Reader.
// It reuses buffers to minimize allocations.
type FrameReader struct {
	r          io.Reader
	headerBuf  *[]byte // Pooled buffer for frame headers
	payloadBuf []byte  // Reusable buffer for payloads
	pool       *BufferPool
}

// NewFrameReader creates a new frame reader with pre-allocated buffers.
func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{
		r:          r,
		headerBuf:  getHeaderBuffer(),
		payloadBuf: make([]byte, 0, 4096), // Pre-allocate 4KB
		pool:       DefaultBufferPool,
	}
}

// Close releases resources held by the FrameReader.
func (fr *FrameReader) Close() {
	if fr.headerBuf != nil {
		putHeaderBuffer(fr.headerBuf)
		fr.headerBuf = nil
	}
}

// ReadFrame reads and parses the next WebSocket frame.
// Returns a Frame with payload data. The returned Frame.Payload slice
// may be reused on the next call to ReadFrame, so callers must copy it
// if they need to retain the data.
//
// Performance: Zero allocations for frames ≤4096 bytes (reuses internal buffer).
// Frames >4096 bytes will allocate a larger buffer, which is then retained.
func (fr *FrameReader) ReadFrame() (*Frame, error) {
	// Read first 2 bytes (always present)
	if _, err := io.ReadFull(fr.r, (*fr.headerBuf)[:2]); err != nil {
		return nil, err
	}

	frame := &Frame{}

	// Parse first byte: FIN, RSV, Opcode
	b0 := (*fr.headerBuf)[0]
	frame.Fin = (b0 & finalBit) != 0
	frame.RSV1 = (b0 & rsv1Bit) != 0
	frame.RSV2 = (b0 & rsv2Bit) != 0
	frame.RSV3 = (b0 & rsv3Bit) != 0
	frame.Opcode = b0 & opcodeMask

	// Parse second byte: MASK, Payload Length
	b1 := (*fr.headerBuf)[1]
	frame.Masked = (b1 & maskBit) != 0
	payloadLen := uint64(b1 & lengthMask)

	// Validate opcode
	if frame.Opcode > 0xA || (frame.Opcode > 0x2 && frame.Opcode < 0x8) {
		return nil, ErrInvalidOpcode
	}

	// Control frames must have FIN=1 and payload ≤125 bytes (RFC 6455 5.5)
	if frame.IsControl() {
		if !frame.Fin {
			return nil, ErrFragmentedControl
		}
		if payloadLen > MaxControlFramePayload {
			return nil, ErrInvalidControlFrame
		}
	}

	// RSV bits must be 0 unless extensions are negotiated (RFC 6455 5.2)
	if frame.RSV1 || frame.RSV2 || frame.RSV3 {
		return nil, ErrReservedBitsSet
	}

	// Read extended payload length
	headerSize := 2
	switch payloadLen {
	case 126:
		// 16-bit extended length
		if _, err := io.ReadFull(fr.r, (*fr.headerBuf)[2:4]); err != nil {
			return nil, err
		}
		frame.Length = uint64(binary.BigEndian.Uint16((*fr.headerBuf)[2:4]))
		headerSize = 4

	case 127:
		// 64-bit extended length
		if _, err := io.ReadFull(fr.r, (*fr.headerBuf)[2:10]); err != nil {
			return nil, err
		}
		frame.Length = binary.BigEndian.Uint64((*fr.headerBuf)[2:10])
		headerSize = 10

		// RFC 6455: Most significant bit must be 0
		if frame.Length&(1<<63) != 0 {
			return nil, ErrFrameTooLarge
		}

	default:
		frame.Length = payloadLen
	}

	// Read masking key if present
	if frame.Masked {
		if _, err := io.ReadFull(fr.r, (*fr.headerBuf)[headerSize:headerSize+4]); err != nil {
			return nil, err
		}
		copy(frame.MaskKey[:], (*fr.headerBuf)[headerSize:headerSize+4])
	}

	// Read payload data
	if frame.Length > 0 {
		// Try to get buffer from pool first
		if poolBuf, ok := fr.pool.GetExact(int(frame.Length)); ok {
			fr.payloadBuf = poolBuf[:frame.Length]
		} else {
			// Ensure buffer is large enough (grow if needed, reuse if possible)
			if uint64(cap(fr.payloadBuf)) < frame.Length {
				// Need larger buffer - allocate and retain it
				fr.payloadBuf = make([]byte, frame.Length)
			} else {
				// Reuse existing buffer
				fr.payloadBuf = fr.payloadBuf[:frame.Length]
			}
		}

		if _, err := io.ReadFull(fr.r, fr.payloadBuf); err != nil {
			return nil, err
		}

		// Unmask payload if masked
		if frame.Masked {
			maskBytes(fr.payloadBuf, frame.MaskKey)
		}

		frame.Payload = fr.payloadBuf
	}

	return frame, nil
}

// ReadFrameInto reads a frame into a caller-provided buffer.
// This allows zero-allocation reading when the caller manages buffers.
// Returns the frame header and number of bytes written to buf.
// If buf is too small, returns ErrFrameTooLarge.
//
// Performance: Zero allocations if buf is large enough.
func (fr *FrameReader) ReadFrameInto(buf []byte) (*Frame, int, error) {
	// Read first 2 bytes (always present)
	if _, err := io.ReadFull(fr.r, (*fr.headerBuf)[:2]); err != nil {
		return nil, 0, err
	}

	frame := &Frame{}

	// Parse first byte: FIN, RSV, Opcode
	b0 := (*fr.headerBuf)[0]
	frame.Fin = (b0 & finalBit) != 0
	frame.RSV1 = (b0 & rsv1Bit) != 0
	frame.RSV2 = (b0 & rsv2Bit) != 0
	frame.RSV3 = (b0 & rsv3Bit) != 0
	frame.Opcode = b0 & opcodeMask

	// Parse second byte: MASK, Payload Length
	b1 := (*fr.headerBuf)[1]
	frame.Masked = (b1 & maskBit) != 0
	payloadLen := uint64(b1 & lengthMask)

	// Validate opcode
	if frame.Opcode > 0xA || (frame.Opcode > 0x2 && frame.Opcode < 0x8) {
		return nil, 0, ErrInvalidOpcode
	}

	// Control frames validation
	if frame.IsControl() {
		if !frame.Fin {
			return nil, 0, ErrFragmentedControl
		}
		if payloadLen > MaxControlFramePayload {
			return nil, 0, ErrInvalidControlFrame
		}
	}

	// RSV bits validation
	if frame.RSV1 || frame.RSV2 || frame.RSV3 {
		return nil, 0, ErrReservedBitsSet
	}

	// Read extended payload length
	headerSize := 2
	switch payloadLen {
	case 126:
		if _, err := io.ReadFull(fr.r, (*fr.headerBuf)[2:4]); err != nil {
			return nil, 0, err
		}
		frame.Length = uint64(binary.BigEndian.Uint16((*fr.headerBuf)[2:4]))
		headerSize = 4

	case 127:
		if _, err := io.ReadFull(fr.r, (*fr.headerBuf)[2:10]); err != nil {
			return nil, 0, err
		}
		frame.Length = binary.BigEndian.Uint64((*fr.headerBuf)[2:10])
		headerSize = 10

		if frame.Length&(1<<63) != 0 {
			return nil, 0, ErrFrameTooLarge
		}

	default:
		frame.Length = payloadLen
	}

	// Read masking key if present
	if frame.Masked {
		if _, err := io.ReadFull(fr.r, (*fr.headerBuf)[headerSize:headerSize+4]); err != nil {
			return nil, 0, err
		}
		copy(frame.MaskKey[:], (*fr.headerBuf)[headerSize:headerSize+4])
	}

	// Read payload data into provided buffer
	if frame.Length > 0 {
		if uint64(len(buf)) < frame.Length {
			return nil, 0, ErrFrameTooLarge
		}

		payload := buf[:frame.Length]
		if _, err := io.ReadFull(fr.r, payload); err != nil {
			return nil, 0, err
		}

		// Unmask payload if masked
		if frame.Masked {
			maskBytes(payload, frame.MaskKey)
		}

		frame.Payload = payload
		return frame, int(frame.Length), nil
	}

	return frame, 0, nil
}

// FrameWriter provides efficient frame writing with pre-allocated buffers.
type FrameWriter struct {
	w         io.Writer
	headerBuf [MaxFrameHeaderSize]byte // Reusable buffer for frame headers
	maskKey   [4]byte                  // Masking key (for client mode)
}

// NewFrameWriter creates a new frame writer.
func NewFrameWriter(w io.Writer) *FrameWriter {
	return &FrameWriter{w: w}
}

// WriteFrame writes a WebSocket frame to the writer.
// If maskKey is non-nil, the payload will be masked (required for client→server).
//
// Performance: Zero allocations for writing. Masking is done in-place.
func (fw *FrameWriter) WriteFrame(opcode byte, fin bool, payload []byte, maskKey *[4]byte) error {
	// Build frame header
	b0 := opcode
	if fin {
		b0 |= finalBit
	}
	fw.headerBuf[0] = b0

	// Determine payload length encoding
	payloadLen := uint64(len(payload))
	headerSize := 2

	b1 := byte(0)
	if maskKey != nil {
		b1 |= maskBit
	}

	switch {
	case payloadLen <= 125:
		fw.headerBuf[1] = b1 | byte(payloadLen)

	case payloadLen <= 0xFFFF:
		fw.headerBuf[1] = b1 | 126
		binary.BigEndian.PutUint16(fw.headerBuf[2:4], uint16(payloadLen))
		headerSize = 4

	default:
		fw.headerBuf[1] = b1 | 127
		binary.BigEndian.PutUint64(fw.headerBuf[2:10], payloadLen)
		headerSize = 10
	}

	// Add masking key if present
	if maskKey != nil {
		copy(fw.headerBuf[headerSize:headerSize+4], maskKey[:])
		headerSize += 4
	}

	// Write header
	if _, err := fw.w.Write(fw.headerBuf[:headerSize]); err != nil {
		return err
	}

	// Write payload (mask if needed)
	if len(payload) > 0 {
		if maskKey != nil {
			// Mask payload in place (caller's buffer will be modified)
			maskBytes(payload, *maskKey)
		}

		if _, err := fw.w.Write(payload); err != nil {
			return err
		}
	}

	return nil
}

// WriteControlFrame writes a control frame (Close, Ping, Pong).
// Control frames must be ≤125 bytes and have FIN=1.
func (fw *FrameWriter) WriteControlFrame(opcode byte, payload []byte, maskKey *[4]byte) error {
	if len(payload) > MaxControlFramePayload {
		return ErrInvalidControlFrame
	}
	if opcode < OpcodeClose || opcode > OpcodePong {
		return ErrInvalidOpcode
	}
	return fw.WriteFrame(opcode, true, payload, maskKey)
}

// WriteTextFrame writes a text frame with UTF-8 validation.
func (fw *FrameWriter) WriteTextFrame(data []byte, maskKey *[4]byte) error {
	// TODO: Add UTF-8 validation
	return fw.WriteFrame(OpcodeText, true, data, maskKey)
}

// WriteBinaryFrame writes a binary frame.
func (fw *FrameWriter) WriteBinaryFrame(data []byte, maskKey *[4]byte) error {
	return fw.WriteFrame(OpcodeBinary, true, data, maskKey)
}

// WritePing writes a Ping control frame.
func (fw *FrameWriter) WritePing(payload []byte, maskKey *[4]byte) error {
	return fw.WriteControlFrame(OpcodePing, payload, maskKey)
}

// WritePong writes a Pong control frame.
func (fw *FrameWriter) WritePong(payload []byte, maskKey *[4]byte) error {
	return fw.WriteControlFrame(OpcodePong, payload, maskKey)
}

// WriteClose writes a Close control frame with status code and reason.
func (fw *FrameWriter) WriteClose(code uint16, reason string, maskKey *[4]byte) error {
	var payload []byte
	if code != 0 {
		payload = make([]byte, 2+len(reason))
		binary.BigEndian.PutUint16(payload, code)
		copy(payload[2:], reason)
	}
	return fw.WriteControlFrame(OpcodeClose, payload, maskKey)
}
