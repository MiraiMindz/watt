package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
)

// RFC 6455 WebSocket Protocol Constants

// Frame opcodes as defined in RFC 6455 Section 5.2
const (
	OpcodeContinuation = 0x0
	OpcodeText         = 0x1
	OpcodeBinary       = 0x2
	// 0x3-0x7 reserved for further non-control frames
	OpcodeClose = 0x8
	OpcodePing  = 0x9
	OpcodePong  = 0xA
	// 0xB-0xF reserved for further control frames
)

// Close status codes as defined in RFC 6455 Section 7.4
const (
	CloseNormalClosure           = 1000
	CloseGoingAway              = 1001
	CloseProtocolError          = 1002
	CloseUnsupportedData        = 1003
	CloseNoStatusReceived       = 1005 // Reserved, not sent
	CloseAbnormalClosure        = 1006 // Reserved, not sent
	CloseInvalidFramePayload    = 1007
	ClosePolicyViolation        = 1008
	CloseMessageTooBig          = 1009
	CloseMandatoryExtension     = 1010
	CloseInternalServerErr      = 1011
	CloseTLSHandshake           = 1015 // Reserved, not sent
)

// Frame header masks
const (
	finalBit   = 1 << 7
	rsv1Bit    = 1 << 6
	rsv2Bit    = 1 << 5
	rsv3Bit    = 1 << 4
	opcodeMask = 0x0F
	maskBit    = 1 << 7
	lengthMask = 0x7F
)

// WebSocket protocol constants
const (
	// MaxFrameHeaderSize is the maximum size of a frame header (2 + 8 + 4 = 14 bytes)
	MaxFrameHeaderSize = 14

	// MaxControlFramePayload is the maximum payload size for control frames (RFC 6455 5.5)
	MaxControlFramePayload = 125

	// WebSocket protocol GUID for handshake (RFC 6455 Section 1.3)
	websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

// Errors
var (
	ErrInvalidOpcode        = errors.New("websocket: invalid opcode")
	ErrInvalidControlFrame  = errors.New("websocket: invalid control frame")
	ErrFragmentedControl    = errors.New("websocket: control frame cannot be fragmented")
	ErrReservedBitsSet      = errors.New("websocket: reserved bits must be 0")
	ErrMaskRequired         = errors.New("websocket: client frames must be masked")
	ErrMaskNotAllowed       = errors.New("websocket: server frames must not be masked")
	ErrFrameTooLarge        = errors.New("websocket: frame too large")
	ErrInvalidUTF8          = errors.New("websocket: invalid UTF-8 in text frame")
	ErrInvalidCloseCode     = errors.New("websocket: invalid close code")
	ErrProtocolViolation    = errors.New("websocket: protocol violation")
	ErrMessageTooLarge      = errors.New("websocket: message too large")
)

// Frame represents a WebSocket frame as defined in RFC 6455 Section 5.2
//
// Frame format:
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-------+-+-------------+-------------------------------+
// |F|R|R|R| opcode|M| Payload len |    Extended payload length    |
// |I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
// |N|V|V|V|       |S|             |   (if payload len==126/127)   |
// | |1|2|3|       |K|             |                               |
// +-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
// |     Extended payload length continued, if payload len == 127  |
// + - - - - - - - - - - - - - - - +-------------------------------+
// |                               |Masking-key, if MASK set to 1  |
// +-------------------------------+-------------------------------+
// | Masking-key (continued)       |          Payload Data         |
// +-------------------------------- - - - - - - - - - - - - - - - +
// :                     Payload Data continued ...                :
// + - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
// |                     Payload Data continued ...                |
// +---------------------------------------------------------------+
type Frame struct {
	Fin     bool   // Final fragment flag
	RSV1    bool   // Reserved bit 1 (must be 0 unless extension negotiated)
	RSV2    bool   // Reserved bit 2 (must be 0 unless extension negotiated)
	RSV3    bool   // Reserved bit 3 (must be 0 unless extension negotiated)
	Opcode  byte   // Frame opcode
	Masked  bool   // Mask flag
	Length  uint64 // Payload length
	MaskKey [4]byte // Masking key (if masked)
	Payload []byte // Payload data (unmasked)
}

// IsControl returns true if the frame is a control frame (Close, Ping, Pong)
func (f *Frame) IsControl() bool {
	return f.Opcode >= 0x8
}

// IsData returns true if the frame is a data frame (Text, Binary, Continuation)
func (f *Frame) IsData() bool {
	return f.Opcode < 0x8
}

// ComputeAcceptKey computes the Sec-WebSocket-Accept value for the handshake.
// RFC 6455 Section 1.3: base64(SHA1(key + GUID))
func ComputeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte(websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// maskBytes applies XOR masking to data in place using the 4-byte mask key.
// This is the masking algorithm defined in RFC 6455 Section 5.3.
// Performance optimized for zero allocations.
//
// On amd64 with AVX2, uses SIMD instructions for 32-byte chunks.
// Otherwise, uses scalar implementation with 8-byte chunks.
var maskBytes = maskBytesDefault

// maskBytesDefault is the default scalar implementation.
func maskBytesDefault(data []byte, maskKey [4]byte) {
	// Fast path: process 8 bytes at a time using uint64
	// This is significantly faster than byte-by-byte masking
	if len(data) >= 8 {
		// Expand mask key to 8 bytes
		mask64 := uint64(maskKey[0]) |
			uint64(maskKey[1])<<8 |
			uint64(maskKey[2])<<16 |
			uint64(maskKey[3])<<24 |
			uint64(maskKey[0])<<32 |
			uint64(maskKey[1])<<40 |
			uint64(maskKey[2])<<48 |
			uint64(maskKey[3])<<56

		// Process 8 bytes at a time
		i := 0
		for ; i+8 <= len(data); i += 8 {
			// XOR 8 bytes at once
			val := uint64(data[i]) |
				uint64(data[i+1])<<8 |
				uint64(data[i+2])<<16 |
				uint64(data[i+3])<<24 |
				uint64(data[i+4])<<32 |
				uint64(data[i+5])<<40 |
				uint64(data[i+6])<<48 |
				uint64(data[i+7])<<56
			val ^= mask64

			// Write back
			data[i] = byte(val)
			data[i+1] = byte(val >> 8)
			data[i+2] = byte(val >> 16)
			data[i+3] = byte(val >> 24)
			data[i+4] = byte(val >> 32)
			data[i+5] = byte(val >> 40)
			data[i+6] = byte(val >> 48)
			data[i+7] = byte(val >> 56)
		}

		// Process remaining bytes
		for ; i < len(data); i++ {
			data[i] ^= maskKey[i%4]
		}
	} else {
		// Slow path: process byte by byte for small data
		for i := 0; i < len(data); i++ {
			data[i] ^= maskKey[i%4]
		}
	}
}
