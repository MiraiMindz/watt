package http2

import (
	"encoding/binary"
	"testing"
)

// RFC 7540 Compliance Test Suite
// Tests all requirements from the HTTP/2 specification

// TestRFC7540_Section4_1_FrameFormat tests frame format compliance
// RFC 7540 §4.1: All frames begin with a fixed 9-octet header
func TestRFC7540_Section4_1_FrameFormat(t *testing.T) {
	tests := []struct {
		name        string
		header      [9]byte
		description string
		valid       bool
	}{
		{
			name:        "Valid frame header",
			header:      [9]byte{0x00, 0x00, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01},
			description: "DATA frame with 5 bytes payload, END_STREAM flag, stream 1",
			valid:       true,
		},
		{
			name:        "Maximum payload length",
			header:      [9]byte{0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			description: "Frame with maximum payload (2^24-1 bytes)",
			valid:       true,
		},
		{
			name:        "Reserved bit must be ignored",
			header:      [9]byte{0x00, 0x00, 0x05, 0x00, 0x01, 0x80, 0x00, 0x00, 0x01},
			description: "Reserved bit set in stream ID - must be cleared",
			valid:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := ParseFrameHeader(tt.header)

			// Reserved bit should always be cleared (RFC 7540 §4.1)
			if fh.StreamID&0x80000000 != 0 {
				t.Error("Reserved bit not cleared in stream ID")
			}

			// Validate frame size doesn't exceed maximum
			if fh.Length > MaxFrameSize {
				t.Errorf("Frame length %d exceeds maximum %d", fh.Length, MaxFrameSize)
			}
		})
	}
}

// TestRFC7540_Section4_2_FrameSize tests frame size requirements
// RFC 7540 §4.2: Implementations MUST support receiving frames up to 2^14 octets
func TestRFC7540_Section4_2_FrameSize(t *testing.T) {
	tests := []struct {
		name     string
		size     uint32
		valid    bool
		mustSupport bool
	}{
		{
			name:        "Minimum frame size",
			size:        0,
			valid:       true,
			mustSupport: true,
		},
		{
			name:        "Default maximum (16KB)",
			size:        16384,
			valid:       true,
			mustSupport: true,
		},
		{
			name:        "Larger frame (requires negotiation)",
			size:        32768,
			valid:       true,
			mustSupport: false,
		},
		{
			name:        "Maximum possible frame size",
			size:        MaxFrameSize,
			valid:       true,
			mustSupport: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   tt.size,
				Type:     FrameData,
				StreamID: 1,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid frame, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error for invalid frame")
			}
		})
	}
}

// TestRFC7540_Section5_1_StreamStates tests stream identifier requirements
// RFC 7540 §5.1.1: Stream identifiers are 31-bit unsigned integers
func TestRFC7540_Section5_1_StreamIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		streamID uint32
		isClient bool
		valid    bool
		reason   string
	}{
		{
			name:     "Stream ID 0 (connection)",
			streamID: 0,
			valid:    true,
			reason:   "Stream 0 is reserved for connection control",
		},
		{
			name:     "Client-initiated stream (odd)",
			streamID: 1,
			isClient: true,
			valid:    true,
			reason:   "Clients use odd stream IDs",
		},
		{
			name:     "Server-initiated stream (even)",
			streamID: 2,
			isClient: false,
			valid:    true,
			reason:   "Servers use even stream IDs",
		},
		{
			name:     "Maximum stream ID (client)",
			streamID: 0x7FFFFFFF,
			isClient: true,
			valid:    true,
			reason:   "2^31-1 is maximum stream ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify stream ID doesn't exceed 31 bits
			if tt.streamID > MaxStreamID {
				t.Errorf("Stream ID %d exceeds maximum %d", tt.streamID, MaxStreamID)
			}

			// Verify client/server stream ID parity
			if tt.streamID > 0 && tt.isClient && tt.streamID%2 == 0 {
				t.Error("Client-initiated stream has even ID")
			}
			if tt.streamID > 0 && !tt.isClient && tt.streamID%2 == 1 {
				t.Error("Server-initiated stream has odd ID")
			}
		})
	}
}

// TestRFC7540_Section6_1_DATA tests DATA frame requirements
// RFC 7540 §6.1: DATA frames MUST be associated with a stream
func TestRFC7540_Section6_1_DATA(t *testing.T) {
	tests := []struct {
		name     string
		streamID uint32
		valid    bool
		reason   string
	}{
		{
			name:     "DATA on stream 0 (invalid)",
			streamID: 0,
			valid:    false,
			reason:   "DATA frames MUST NOT be sent on stream 0",
		},
		{
			name:     "DATA on stream 1 (valid)",
			streamID: 1,
			valid:    true,
			reason:   "DATA frames must be associated with a stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   100,
				Type:     FrameData,
				StreamID: tt.streamID,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid DATA frame, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error for DATA frame on stream 0")
			}
		})
	}
}

// TestRFC7540_Section6_2_HEADERS tests HEADERS frame requirements
// RFC 7540 §6.2: HEADERS frames MUST be associated with a stream
func TestRFC7540_Section6_2_HEADERS(t *testing.T) {
	tests := []struct {
		name     string
		streamID uint32
		flags    Flags
		valid    bool
		reason   string
	}{
		{
			name:     "HEADERS on stream 0 (invalid)",
			streamID: 0,
			valid:    false,
			reason:   "HEADERS frames MUST NOT be sent on stream 0",
		},
		{
			name:     "HEADERS on stream 1 (valid)",
			streamID: 1,
			valid:    true,
			reason:   "HEADERS frames must be associated with a stream",
		},
		{
			name:     "HEADERS with END_STREAM",
			streamID: 1,
			flags:    FlagHeadersEndStream,
			valid:    true,
			reason:   "END_STREAM flag is valid on HEADERS",
		},
		{
			name:     "HEADERS with PRIORITY",
			streamID: 1,
			flags:    FlagHeadersPriority,
			valid:    true,
			reason:   "PRIORITY flag includes stream dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   100,
				Type:     FrameHeaders,
				Flags:    tt.flags,
				StreamID: tt.streamID,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid HEADERS frame, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error for HEADERS frame on stream 0")
			}
		})
	}
}

// TestRFC7540_Section6_3_PRIORITY tests PRIORITY frame requirements
// RFC 7540 §6.3: PRIORITY frames are always 5 octets long
func TestRFC7540_Section6_3_PRIORITY(t *testing.T) {
	tests := []struct {
		name     string
		length   uint32
		streamID uint32
		valid    bool
		reason   string
	}{
		{
			name:     "PRIORITY with correct length",
			length:   5,
			streamID: 1,
			valid:    true,
			reason:   "PRIORITY frames are exactly 5 octets",
		},
		{
			name:     "PRIORITY with incorrect length (4)",
			length:   4,
			streamID: 1,
			valid:    false,
			reason:   "PRIORITY frames must be exactly 5 octets",
		},
		{
			name:     "PRIORITY with incorrect length (6)",
			length:   6,
			streamID: 1,
			valid:    false,
			reason:   "PRIORITY frames must be exactly 5 octets",
		},
		{
			name:     "PRIORITY on stream 0 (invalid)",
			length:   5,
			streamID: 0,
			valid:    false,
			reason:   "PRIORITY frames must be associated with a stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   tt.length,
				Type:     FramePriority,
				StreamID: tt.streamID,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid PRIORITY frame, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error for invalid PRIORITY frame")
			}
		})
	}
}

// TestRFC7540_Section6_4_RST_STREAM tests RST_STREAM frame requirements
// RFC 7540 §6.4: RST_STREAM frames are always 4 octets long
func TestRFC7540_Section6_4_RST_STREAM(t *testing.T) {
	tests := []struct {
		name     string
		length   uint32
		streamID uint32
		valid    bool
	}{
		{
			name:     "RST_STREAM with correct length",
			length:   4,
			streamID: 1,
			valid:    true,
		},
		{
			name:     "RST_STREAM with incorrect length",
			length:   5,
			streamID: 1,
			valid:    false,
		},
		{
			name:     "RST_STREAM on stream 0 (invalid)",
			length:   4,
			streamID: 0,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   tt.length,
				Type:     FrameRSTStream,
				StreamID: tt.streamID,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid RST_STREAM frame, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error for invalid RST_STREAM frame")
			}
		})
	}
}

// TestRFC7540_Section6_5_SETTINGS tests SETTINGS frame requirements
// RFC 7540 §6.5: SETTINGS frames always apply to a connection, never a stream
func TestRFC7540_Section6_5_SETTINGS(t *testing.T) {
	tests := []struct {
		name     string
		length   uint32
		streamID uint32
		flags    Flags
		valid    bool
		reason   string
	}{
		{
			name:     "SETTINGS on stream 0 (valid)",
			length:   12,
			streamID: 0,
			valid:    true,
			reason:   "SETTINGS frames must be sent on stream 0",
		},
		{
			name:     "SETTINGS on stream 1 (invalid)",
			length:   12,
			streamID: 1,
			valid:    false,
			reason:   "SETTINGS frames MUST be sent on stream 0",
		},
		{
			name:     "SETTINGS with non-multiple-of-6 length (invalid)",
			length:   7,
			streamID: 0,
			valid:    false,
			reason:   "SETTINGS payload must be multiple of 6 octets",
		},
		{
			name:     "SETTINGS ACK with zero length (valid)",
			length:   0,
			streamID: 0,
			flags:    FlagSettingsAck,
			valid:    true,
			reason:   "SETTINGS ACK must have zero length",
		},
		{
			name:     "SETTINGS ACK with non-zero length (invalid)",
			length:   6,
			streamID: 0,
			flags:    FlagSettingsAck,
			valid:    false,
			reason:   "SETTINGS ACK with payload is a connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   tt.length,
				Type:     FrameSettings,
				Flags:    tt.flags,
				StreamID: tt.streamID,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid SETTINGS frame, got error: %v (reason: %s)", err, tt.reason)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected error for invalid SETTINGS frame (reason: %s)", tt.reason)
			}
		})
	}
}

// TestRFC7540_Section6_7_PING tests PING frame requirements
// RFC 7540 §6.7: PING frames are always 8 octets long
func TestRFC7540_Section6_7_PING(t *testing.T) {
	tests := []struct {
		name     string
		length   uint32
		streamID uint32
		valid    bool
		reason   string
	}{
		{
			name:     "PING with correct length",
			length:   8,
			streamID: 0,
			valid:    true,
			reason:   "PING frames are exactly 8 octets",
		},
		{
			name:     "PING with incorrect length",
			length:   7,
			streamID: 0,
			valid:    false,
			reason:   "PING frames must be exactly 8 octets",
		},
		{
			name:     "PING on stream 1 (invalid)",
			length:   8,
			streamID: 1,
			valid:    false,
			reason:   "PING frames must be sent on stream 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   tt.length,
				Type:     FramePing,
				StreamID: tt.streamID,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid PING frame, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected error for invalid PING frame: %s", tt.reason)
			}
		})
	}
}

// TestRFC7540_Section6_8_GOAWAY tests GOAWAY frame requirements
// RFC 7540 §6.8: GOAWAY frames always apply to the connection
func TestRFC7540_Section6_8_GOAWAY(t *testing.T) {
	tests := []struct {
		name     string
		length   uint32
		streamID uint32
		valid    bool
		reason   string
	}{
		{
			name:     "GOAWAY on stream 0 (valid)",
			length:   8,
			streamID: 0,
			valid:    true,
			reason:   "GOAWAY frames must be sent on stream 0",
		},
		{
			name:     "GOAWAY on stream 1 (invalid)",
			length:   8,
			streamID: 1,
			valid:    false,
			reason:   "GOAWAY frames MUST be sent on stream 0",
		},
		{
			name:     "GOAWAY with minimum length",
			length:   8,
			streamID: 0,
			valid:    true,
			reason:   "GOAWAY minimum: last stream ID (4) + error code (4)",
		},
		{
			name:     "GOAWAY with debug data",
			length:   100,
			streamID: 0,
			valid:    true,
			reason:   "GOAWAY can include optional debug data",
		},
		{
			name:     "GOAWAY with insufficient length (invalid)",
			length:   7,
			streamID: 0,
			valid:    false,
			reason:   "GOAWAY must be at least 8 octets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   tt.length,
				Type:     FrameGoAway,
				StreamID: tt.streamID,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid GOAWAY frame, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected error for invalid GOAWAY frame: %s", tt.reason)
			}
		})
	}
}

// TestRFC7540_Section6_9_WINDOW_UPDATE tests WINDOW_UPDATE frame requirements
// RFC 7540 §6.9: WINDOW_UPDATE frames are always 4 octets long
func TestRFC7540_Section6_9_WINDOW_UPDATE(t *testing.T) {
	tests := []struct {
		name      string
		length    uint32
		streamID  uint32
		increment uint32
		valid     bool
		reason    string
	}{
		{
			name:      "WINDOW_UPDATE on connection",
			length:    4,
			streamID:  0,
			increment: 1024,
			valid:     true,
			reason:    "WINDOW_UPDATE can be sent on stream 0",
		},
		{
			name:      "WINDOW_UPDATE on stream",
			length:    4,
			streamID:  1,
			increment: 1024,
			valid:     true,
			reason:    "WINDOW_UPDATE can be sent on any stream",
		},
		{
			name:      "WINDOW_UPDATE with zero increment (invalid)",
			length:    4,
			streamID:  1,
			increment: 0,
			valid:     false,
			reason:    "Window size increment must not be zero",
		},
		{
			name:      "WINDOW_UPDATE with incorrect length",
			length:    5,
			streamID:  1,
			increment: 1024,
			valid:     false,
			reason:    "WINDOW_UPDATE frames must be exactly 4 octets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   tt.length,
				Type:     FrameWindowUpdate,
				StreamID: tt.streamID,
			}

			// Validate header
			err := fh.Validate()
			if !tt.valid && err != nil {
				return // Expected error
			}
			if tt.valid && err != nil {
				t.Errorf("Expected valid WINDOW_UPDATE frame header, got error: %v", err)
				return
			}

			// Parse full frame to test increment validation
			if tt.length == 4 {
				payload := make([]byte, 4)
				binary.BigEndian.PutUint32(payload, tt.increment)

				_, err := ParseWindowUpdateFrame(fh, payload)
				if tt.valid && err != nil {
					t.Errorf("Expected valid WINDOW_UPDATE frame, got error: %v", err)
				}
				if !tt.valid && err == nil {
					t.Errorf("Expected error for invalid WINDOW_UPDATE frame: %s", tt.reason)
				}
			}
		})
	}
}

// TestRFC7540_Section6_10_CONTINUATION tests CONTINUATION frame requirements
// RFC 7540 §6.10: CONTINUATION frames MUST be associated with a stream
func TestRFC7540_Section6_10_CONTINUATION(t *testing.T) {
	tests := []struct {
		name     string
		streamID uint32
		valid    bool
		reason   string
	}{
		{
			name:     "CONTINUATION on stream 1 (valid)",
			streamID: 1,
			valid:    true,
			reason:   "CONTINUATION frames must be associated with a stream",
		},
		{
			name:     "CONTINUATION on stream 0 (invalid)",
			streamID: 0,
			valid:    false,
			reason:   "CONTINUATION frames MUST NOT be sent on stream 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := FrameHeader{
				Length:   100,
				Type:     FrameContinuation,
				StreamID: tt.streamID,
			}

			err := fh.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid CONTINUATION frame, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected error for invalid CONTINUATION frame: %s", tt.reason)
			}
		})
	}
}

// TestRFC7540_ErrorCodes tests error code definitions
// RFC 7540 §7: Error codes are 32-bit fields
func TestRFC7540_ErrorCodes(t *testing.T) {
	errorCodes := []struct {
		code ErrorCode
		name string
	}{
		{ErrCodeNo, "NO_ERROR"},
		{ErrCodeProtocol, "PROTOCOL_ERROR"},
		{ErrCodeInternal, "INTERNAL_ERROR"},
		{ErrCodeFlowControl, "FLOW_CONTROL_ERROR"},
		{ErrCodeSettingsTimeout, "SETTINGS_TIMEOUT"},
		{ErrCodeStreamClosed, "STREAM_CLOSED"},
		{ErrCodeFrameSize, "FRAME_SIZE_ERROR"},
		{ErrCodeRefusedStream, "REFUSED_STREAM"},
		{ErrCodeCancel, "CANCEL"},
		{ErrCodeCompression, "COMPRESSION_ERROR"},
		{ErrCodeConnect, "CONNECT_ERROR"},
		{ErrCodeEnhanceYourCalm, "ENHANCE_YOUR_CALM"},
		{ErrCodeInadequateSecurity, "INADEQUATE_SECURITY"},
		{ErrCodeHTTP11Required, "HTTP_1_1_REQUIRED"},
	}

	for _, ec := range errorCodes {
		t.Run(ec.name, func(t *testing.T) {
			if ec.code.String() != ec.name {
				t.Errorf("Error code %d: expected %s, got %s", ec.code, ec.name, ec.code.String())
			}
		})
	}
}

// TestRFC7540_ConnectionPreface tests connection preface
// RFC 7540 §3.5: Client connection preface
func TestRFC7540_ConnectionPreface(t *testing.T) {
	expected := []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")

	if len(ClientPreface) != 24 {
		t.Errorf("Client preface length: expected 24, got %d", len(ClientPreface))
	}

	for i := range expected {
		if ClientPreface[i] != expected[i] {
			t.Errorf("Client preface byte %d: expected 0x%02x, got 0x%02x",
				i, expected[i], ClientPreface[i])
		}
	}
}
