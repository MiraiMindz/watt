package http2

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Test frame header parsing (zero allocations)
func TestParseFrameHeader(t *testing.T) {
	tests := []struct {
		name   string
		input  [9]byte
		want   FrameHeader
	}{
		{
			name:  "DATA frame",
			input: [9]byte{0x00, 0x00, 0x0A, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01},
			want: FrameHeader{
				Length:   10,
				Type:     FrameData,
				Flags:    FlagDataEndStream,
				StreamID: 1,
			},
		},
		{
			name:  "HEADERS frame with priority",
			input: [9]byte{0x00, 0x00, 0x14, 0x01, 0x25, 0x00, 0x00, 0x00, 0x03},
			want: FrameHeader{
				Length:   20,
				Type:     FrameHeaders,
				Flags:    FlagHeadersEndStream | FlagHeadersEndHeaders | FlagHeadersPriority,
				StreamID: 3,
			},
		},
		{
			name:  "SETTINGS frame",
			input: [9]byte{0x00, 0x00, 0x0C, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: FrameHeader{
				Length:   12,
				Type:     FrameSettings,
				Flags:    0,
				StreamID: 0,
			},
		},
		{
			name:  "PING frame with ACK",
			input: [9]byte{0x00, 0x00, 0x08, 0x06, 0x01, 0x00, 0x00, 0x00, 0x00},
			want: FrameHeader{
				Length:   8,
				Type:     FramePing,
				Flags:    FlagPingAck,
				StreamID: 0,
			},
		},
		{
			name:  "Maximum length frame",
			input: [9]byte{0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			want: FrameHeader{
				Length:   16777215, // 2^24 - 1
				Type:     FrameData,
				Flags:    0,
				StreamID: 1,
			},
		},
		{
			name:  "Reserved bit cleared",
			input: [9]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x01}, // Reserved bit set
			want: FrameHeader{
				Length:   0,
				Type:     FrameData,
				Flags:    0,
				StreamID: 1, // Reserved bit should be cleared
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFrameHeader(tt.input)

			if got.Length != tt.want.Length {
				t.Errorf("Length = %d, want %d", got.Length, tt.want.Length)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Flags != tt.want.Flags {
				t.Errorf("Flags = %v, want %v", got.Flags, tt.want.Flags)
			}
			if got.StreamID != tt.want.StreamID {
				t.Errorf("StreamID = %d, want %d", got.StreamID, tt.want.StreamID)
			}
		})
	}
}

// Test frame header writing
func TestWriteFrameHeader(t *testing.T) {
	tests := []struct {
		name string
		fh   FrameHeader
		want [9]byte
	}{
		{
			name: "DATA frame",
			fh: FrameHeader{
				Length:   10,
				Type:     FrameData,
				Flags:    FlagDataEndStream,
				StreamID: 1,
			},
			want: [9]byte{0x00, 0x00, 0x0A, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01},
		},
		{
			name: "SETTINGS frame",
			fh: FrameHeader{
				Length:   12,
				Type:     FrameSettings,
				Flags:    0,
				StreamID: 0,
			},
			want: [9]byte{0x00, 0x00, 0x0C, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf [9]byte
			n := WriteFrameHeader(buf[:], tt.fh)

			if n != 9 {
				t.Errorf("WriteFrameHeader returned %d bytes, want 9", n)
			}

			if !bytes.Equal(buf[:], tt.want[:]) {
				t.Errorf("WriteFrameHeader = %v, want %v", buf, tt.want)
			}
		})
	}
}

// Test frame header validation
func TestFrameHeaderValidation(t *testing.T) {
	tests := []struct {
		name    string
		fh      FrameHeader
		wantErr bool
	}{
		{
			name: "Valid DATA frame",
			fh: FrameHeader{
				Length:   100,
				Type:     FrameData,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: false,
		},
		{
			name: "DATA frame with stream ID 0 (invalid)",
			fh: FrameHeader{
				Length:   100,
				Type:     FrameData,
				Flags:    0,
				StreamID: 0,
			},
			wantErr: true,
		},
		{
			name: "Valid SETTINGS frame",
			fh: FrameHeader{
				Length:   12, // 2 settings
				Type:     FrameSettings,
				Flags:    0,
				StreamID: 0,
			},
			wantErr: false,
		},
		{
			name: "SETTINGS frame with non-zero stream ID (invalid)",
			fh: FrameHeader{
				Length:   12,
				Type:     FrameSettings,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: true,
		},
		{
			name: "SETTINGS frame with invalid length (invalid)",
			fh: FrameHeader{
				Length:   5, // Not multiple of 6
				Type:     FrameSettings,
				Flags:    0,
				StreamID: 0,
			},
			wantErr: true,
		},
		{
			name: "SETTINGS ACK with zero length (valid)",
			fh: FrameHeader{
				Length:   0,
				Type:     FrameSettings,
				Flags:    FlagSettingsAck,
				StreamID: 0,
			},
			wantErr: false,
		},
		{
			name: "SETTINGS ACK with non-zero length (invalid)",
			fh: FrameHeader{
				Length:   6,
				Type:     FrameSettings,
				Flags:    FlagSettingsAck,
				StreamID: 0,
			},
			wantErr: true,
		},
		{
			name: "Valid PING frame",
			fh: FrameHeader{
				Length:   8,
				Type:     FramePing,
				Flags:    0,
				StreamID: 0,
			},
			wantErr: false,
		},
		{
			name: "PING frame with non-zero stream ID (invalid)",
			fh: FrameHeader{
				Length:   8,
				Type:     FramePing,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: true,
		},
		{
			name: "PING frame with invalid length (invalid)",
			fh: FrameHeader{
				Length:   7,
				Type:     FramePing,
				Flags:    0,
				StreamID: 0,
			},
			wantErr: true,
		},
		{
			name: "Valid PRIORITY frame",
			fh: FrameHeader{
				Length:   5,
				Type:     FramePriority,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: false,
		},
		{
			name: "PRIORITY frame with invalid length (invalid)",
			fh: FrameHeader{
				Length:   4,
				Type:     FramePriority,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: true,
		},
		{
			name: "Valid RST_STREAM frame",
			fh: FrameHeader{
				Length:   4,
				Type:     FrameRSTStream,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: false,
		},
		{
			name: "RST_STREAM frame with invalid length (invalid)",
			fh: FrameHeader{
				Length:   3,
				Type:     FrameRSTStream,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: true,
		},
		{
			name: "Valid WINDOW_UPDATE frame (connection)",
			fh: FrameHeader{
				Length:   4,
				Type:     FrameWindowUpdate,
				Flags:    0,
				StreamID: 0,
			},
			wantErr: false,
		},
		{
			name: "Valid WINDOW_UPDATE frame (stream)",
			fh: FrameHeader{
				Length:   4,
				Type:     FrameWindowUpdate,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: false,
		},
		{
			name: "WINDOW_UPDATE frame with invalid length (invalid)",
			fh: FrameHeader{
				Length:   5,
				Type:     FrameWindowUpdate,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: true,
		},
		{
			name: "Valid GOAWAY frame",
			fh: FrameHeader{
				Length:   8,
				Type:     FrameGoAway,
				Flags:    0,
				StreamID: 0,
			},
			wantErr: false,
		},
		{
			name: "GOAWAY frame with non-zero stream ID (invalid)",
			fh: FrameHeader{
				Length:   8,
				Type:     FrameGoAway,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: true,
		},
		{
			name: "Frame too large (invalid)",
			fh: FrameHeader{
				Length:   MaxFrameSize + 1,
				Type:     FrameData,
				Flags:    0,
				StreamID: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fh.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test DATA frame parsing
func TestParseDataFrame(t *testing.T) {
	tests := []struct {
		name    string
		fh      FrameHeader
		payload []byte
		want    *DataFrame
		wantErr bool
	}{
		{
			name: "Simple DATA frame",
			fh: FrameHeader{
				Length:   5,
				Type:     FrameData,
				Flags:    FlagDataEndStream,
				StreamID: 1,
			},
			payload: []byte("hello"),
			want: &DataFrame{
				FrameHeader: FrameHeader{Length: 5, Type: FrameData, Flags: FlagDataEndStream, StreamID: 1},
				Data:        []byte("hello"),
			},
			wantErr: false,
		},
		{
			name: "DATA frame with padding",
			fh: FrameHeader{
				Length:   10,
				Type:     FrameData,
				Flags:    FlagDataPadded,
				StreamID: 1,
			},
			payload: append([]byte{3}, append([]byte("hello"), []byte{0, 0, 0}...)...), // Pad length 3 + "hello" + 3 bytes padding
			want: &DataFrame{
				FrameHeader: FrameHeader{Length: 10, Type: FrameData, Flags: FlagDataPadded, StreamID: 1},
				Data:        []byte("hello"),
				PadLength:   3,
			},
			wantErr: false,
		},
		{
			name: "DATA frame with excessive padding (invalid)",
			fh: FrameHeader{
				Length:   5,
				Type:     FrameData,
				Flags:    FlagDataPadded,
				StreamID: 1,
			},
			payload: []byte{10, 0, 0, 0, 0}, // Pad length 10 but only 4 bytes remaining
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDataFrame(tt.fh, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDataFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if !bytes.Equal(got.Data, tt.want.Data) {
				t.Errorf("Data = %v, want %v", got.Data, tt.want.Data)
			}
			if got.PadLength != tt.want.PadLength {
				t.Errorf("PadLength = %d, want %d", got.PadLength, tt.want.PadLength)
			}
		})
	}
}

// Test HEADERS frame parsing
func TestParseHeadersFrame(t *testing.T) {
	tests := []struct {
		name    string
		fh      FrameHeader
		payload []byte
		want    *HeadersFrame
		wantErr bool
	}{
		{
			name: "Simple HEADERS frame",
			fh: FrameHeader{
				Length:   10,
				Type:     FrameHeaders,
				Flags:    FlagHeadersEndHeaders,
				StreamID: 1,
			},
			payload: []byte{0x82, 0x86, 0x84, 0x41, 0x0f, 0x77, 0x77, 0x77, 0x2e, 0x65}, // HPACK encoded headers
			want: &HeadersFrame{
				FrameHeader: FrameHeader{Length: 10, Type: FrameHeaders, Flags: FlagHeadersEndHeaders, StreamID: 1},
				HeaderBlock: []byte{0x82, 0x86, 0x84, 0x41, 0x0f, 0x77, 0x77, 0x77, 0x2e, 0x65},
			},
			wantErr: false,
		},
		{
			name: "HEADERS frame with priority",
			fh: FrameHeader{
				Length:   15,
				Type:     FrameHeaders,
				Flags:    FlagHeadersPriority | FlagHeadersEndHeaders,
				StreamID: 1,
			},
			payload: func() []byte {
				buf := make([]byte, 15)
				// Exclusive bit + stream dependency
				binary.BigEndian.PutUint32(buf[0:4], 0x80000003) // Exclusive, depends on stream 3
				buf[4] = 16                                       // Weight (16+1 = 17)
				// Header block
				copy(buf[5:], []byte{0x82, 0x86, 0x84, 0x41, 0x0f, 0x77, 0x77, 0x77, 0x2e, 0x65})
				return buf
			}(),
			want: &HeadersFrame{
				FrameHeader:      FrameHeader{Length: 15, Type: FrameHeaders, Flags: FlagHeadersPriority | FlagHeadersEndHeaders, StreamID: 1},
				Exclusive:        true,
				StreamDependency: 3,
				Weight:           16,
				HeaderBlock:      []byte{0x82, 0x86, 0x84, 0x41, 0x0f, 0x77, 0x77, 0x77, 0x2e, 0x65},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHeadersFrame(tt.fh, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHeadersFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if !bytes.Equal(got.HeaderBlock, tt.want.HeaderBlock) {
				t.Errorf("HeaderBlock = %v, want %v", got.HeaderBlock, tt.want.HeaderBlock)
			}
			if got.Exclusive != tt.want.Exclusive {
				t.Errorf("Exclusive = %v, want %v", got.Exclusive, tt.want.Exclusive)
			}
			if got.StreamDependency != tt.want.StreamDependency {
				t.Errorf("StreamDependency = %d, want %d", got.StreamDependency, tt.want.StreamDependency)
			}
			if got.Weight != tt.want.Weight {
				t.Errorf("Weight = %d, want %d", got.Weight, tt.want.Weight)
			}
		})
	}
}

// Test SETTINGS frame parsing
func TestParseSettingsFrame(t *testing.T) {
	tests := []struct {
		name    string
		fh      FrameHeader
		payload []byte
		want    *SettingsFrame
		wantErr bool
	}{
		{
			name: "SETTINGS ACK",
			fh: FrameHeader{
				Length:   0,
				Type:     FrameSettings,
				Flags:    FlagSettingsAck,
				StreamID: 0,
			},
			payload: []byte{},
			want: &SettingsFrame{
				FrameHeader: FrameHeader{Length: 0, Type: FrameSettings, Flags: FlagSettingsAck, StreamID: 0},
			},
			wantErr: false,
		},
		{
			name: "SETTINGS with parameters",
			fh: FrameHeader{
				Length:   12,
				Type:     FrameSettings,
				Flags:    0,
				StreamID: 0,
			},
			payload: func() []byte {
				buf := make([]byte, 12)
				// Setting 1: HEADER_TABLE_SIZE = 4096
				binary.BigEndian.PutUint16(buf[0:2], uint16(SettingHeaderTableSize))
				binary.BigEndian.PutUint32(buf[2:6], 4096)
				// Setting 2: MAX_CONCURRENT_STREAMS = 100
				binary.BigEndian.PutUint16(buf[6:8], uint16(SettingMaxConcurrentStreams))
				binary.BigEndian.PutUint32(buf[8:12], 100)
				return buf
			}(),
			want: &SettingsFrame{
				FrameHeader: FrameHeader{Length: 12, Type: FrameSettings, Flags: 0, StreamID: 0},
				Settings: []Setting{
					{ID: SettingHeaderTableSize, Value: 4096},
					{ID: SettingMaxConcurrentStreams, Value: 100},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSettingsFrame(tt.fh, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSettingsFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(got.Settings) != len(tt.want.Settings) {
				t.Errorf("Settings count = %d, want %d", len(got.Settings), len(tt.want.Settings))
				return
			}

			for i := range got.Settings {
				if got.Settings[i].ID != tt.want.Settings[i].ID {
					t.Errorf("Setting[%d].ID = %v, want %v", i, got.Settings[i].ID, tt.want.Settings[i].ID)
				}
				if got.Settings[i].Value != tt.want.Settings[i].Value {
					t.Errorf("Setting[%d].Value = %d, want %d", i, got.Settings[i].Value, tt.want.Settings[i].Value)
				}
			}
		})
	}
}

// Test PING frame parsing
func TestParsePingFrame(t *testing.T) {
	tests := []struct {
		name    string
		fh      FrameHeader
		payload []byte
		want    *PingFrame
		wantErr bool
	}{
		{
			name: "PING frame",
			fh: FrameHeader{
				Length:   8,
				Type:     FramePing,
				Flags:    0,
				StreamID: 0,
			},
			payload: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			want: &PingFrame{
				FrameHeader: FrameHeader{Length: 8, Type: FramePing, Flags: 0, StreamID: 0},
				Data:        [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			},
			wantErr: false,
		},
		{
			name: "PING ACK frame",
			fh: FrameHeader{
				Length:   8,
				Type:     FramePing,
				Flags:    FlagPingAck,
				StreamID: 0,
			},
			payload: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			want: &PingFrame{
				FrameHeader: FrameHeader{Length: 8, Type: FramePing, Flags: FlagPingAck, StreamID: 0},
				Data:        [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePingFrame(tt.fh, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePingFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Data != tt.want.Data {
				t.Errorf("Data = %v, want %v", got.Data, tt.want.Data)
			}
		})
	}
}

// Test WINDOW_UPDATE frame parsing
func TestParseWindowUpdateFrame(t *testing.T) {
	tests := []struct {
		name    string
		fh      FrameHeader
		payload []byte
		want    *WindowUpdateFrame
		wantErr bool
	}{
		{
			name: "WINDOW_UPDATE for connection",
			fh: FrameHeader{
				Length:   4,
				Type:     FrameWindowUpdate,
				Flags:    0,
				StreamID: 0,
			},
			payload: func() []byte {
				buf := make([]byte, 4)
				binary.BigEndian.PutUint32(buf, 1024)
				return buf
			}(),
			want: &WindowUpdateFrame{
				FrameHeader:         FrameHeader{Length: 4, Type: FrameWindowUpdate, Flags: 0, StreamID: 0},
				WindowSizeIncrement: 1024,
			},
			wantErr: false,
		},
		{
			name: "WINDOW_UPDATE for stream",
			fh: FrameHeader{
				Length:   4,
				Type:     FrameWindowUpdate,
				Flags:    0,
				StreamID: 1,
			},
			payload: func() []byte {
				buf := make([]byte, 4)
				binary.BigEndian.PutUint32(buf, 65535)
				return buf
			}(),
			want: &WindowUpdateFrame{
				FrameHeader:         FrameHeader{Length: 4, Type: FrameWindowUpdate, Flags: 0, StreamID: 1},
				WindowSizeIncrement: 65535,
			},
			wantErr: false,
		},
		{
			name: "WINDOW_UPDATE with zero increment (invalid)",
			fh: FrameHeader{
				Length:   4,
				Type:     FrameWindowUpdate,
				Flags:    0,
				StreamID: 1,
			},
			payload: []byte{0x00, 0x00, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseWindowUpdateFrame(tt.fh, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWindowUpdateFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.WindowSizeIncrement != tt.want.WindowSizeIncrement {
				t.Errorf("WindowSizeIncrement = %d, want %d", got.WindowSizeIncrement, tt.want.WindowSizeIncrement)
			}
		})
	}
}

// Test GOAWAY frame parsing
func TestParseGoAwayFrame(t *testing.T) {
	tests := []struct {
		name    string
		fh      FrameHeader
		payload []byte
		want    *GoAwayFrame
		wantErr bool
	}{
		{
			name: "GOAWAY without debug data",
			fh: FrameHeader{
				Length:   8,
				Type:     FrameGoAway,
				Flags:    0,
				StreamID: 0,
			},
			payload: func() []byte {
				buf := make([]byte, 8)
				binary.BigEndian.PutUint32(buf[0:4], 7) // Last stream ID = 7
				binary.BigEndian.PutUint32(buf[4:8], uint32(ErrCodeNo))
				return buf
			}(),
			want: &GoAwayFrame{
				FrameHeader:  FrameHeader{Length: 8, Type: FrameGoAway, Flags: 0, StreamID: 0},
				LastStreamID: 7,
				ErrorCode:    ErrCodeNo,
			},
			wantErr: false,
		},
		{
			name: "GOAWAY with debug data",
			fh: FrameHeader{
				Length:   13,
				Type:     FrameGoAway,
				Flags:    0,
				StreamID: 0,
			},
			payload: func() []byte {
				buf := make([]byte, 13)
				binary.BigEndian.PutUint32(buf[0:4], 7) // Last stream ID = 7
				binary.BigEndian.PutUint32(buf[4:8], uint32(ErrCodeProtocol))
				copy(buf[8:], []byte("debug"))
				return buf
			}(),
			want: &GoAwayFrame{
				FrameHeader:  FrameHeader{Length: 13, Type: FrameGoAway, Flags: 0, StreamID: 0},
				LastStreamID: 7,
				ErrorCode:    ErrCodeProtocol,
				DebugData:    []byte("debug"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGoAwayFrame(tt.fh, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGoAwayFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.LastStreamID != tt.want.LastStreamID {
				t.Errorf("LastStreamID = %d, want %d", got.LastStreamID, tt.want.LastStreamID)
			}
			if got.ErrorCode != tt.want.ErrorCode {
				t.Errorf("ErrorCode = %v, want %v", got.ErrorCode, tt.want.ErrorCode)
			}
			if !bytes.Equal(got.DebugData, tt.want.DebugData) {
				t.Errorf("DebugData = %v, want %v", got.DebugData, tt.want.DebugData)
			}
		})
	}
}

// Test RST_STREAM frame parsing
func TestParseRSTStreamFrame(t *testing.T) {
	fh := FrameHeader{
		Length:   4,
		Type:     FrameRSTStream,
		Flags:    0,
		StreamID: 1,
	}

	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(ErrCodeCancel))

	got, err := ParseRSTStreamFrame(fh, payload)
	if err != nil {
		t.Fatalf("ParseRSTStreamFrame() error = %v", err)
	}

	if got.ErrorCode != ErrCodeCancel {
		t.Errorf("ErrorCode = %v, want %v", got.ErrorCode, ErrCodeCancel)
	}
}

// Test PRIORITY frame parsing
func TestParsePriorityFrame(t *testing.T) {
	fh := FrameHeader{
		Length:   5,
		Type:     FramePriority,
		Flags:    0,
		StreamID: 1,
	}

	payload := make([]byte, 5)
	binary.BigEndian.PutUint32(payload[0:4], 0x80000003) // Exclusive, depends on stream 3
	payload[4] = 16                                       // Weight 16+1 = 17

	got, err := ParsePriorityFrame(fh, payload)
	if err != nil {
		t.Fatalf("ParsePriorityFrame() error = %v", err)
	}

	if !got.Exclusive {
		t.Error("Exclusive = false, want true")
	}
	if got.StreamDependency != 3 {
		t.Errorf("StreamDependency = %d, want 3", got.StreamDependency)
	}
	if got.Weight != 16 {
		t.Errorf("Weight = %d, want 16", got.Weight)
	}
}
