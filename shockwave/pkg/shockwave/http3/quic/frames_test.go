package quic

import (
	"bytes"
	"testing"
)

func TestPingFrame(t *testing.T) {
	frame := &PingFrame{}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	if len(buf) != 1 || buf[0] != byte(FrameTypePing) {
		t.Errorf("Encoded PING = %x, want [01]", buf)
	}

	// Decode
	parsed, n, err := ParseFrame(buf)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	if n != len(buf) {
		t.Errorf("ParseFrame() consumed %d bytes, want %d", n, len(buf))
	}

	if parsed.Type() != FrameTypePing {
		t.Errorf("Type = %v, want %v", parsed.Type(), FrameTypePing)
	}
}

func TestCryptoFrame(t *testing.T) {
	frame := &CryptoFrame{
		Offset: 100,
		Data:   []byte("crypto data"),
	}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Decode
	parsed, n, err := ParseFrame(buf)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	if n != len(buf) {
		t.Errorf("ParseFrame() consumed %d bytes, want %d", n, len(buf))
	}

	crypto, ok := parsed.(*CryptoFrame)
	if !ok {
		t.Fatalf("Parsed frame is not CryptoFrame")
	}

	if crypto.Offset != frame.Offset {
		t.Errorf("Offset = %d, want %d", crypto.Offset, frame.Offset)
	}

	if !bytes.Equal(crypto.Data, frame.Data) {
		t.Errorf("Data = %x, want %x", crypto.Data, frame.Data)
	}
}

func TestStreamFrame(t *testing.T) {
	tests := []struct {
		name   string
		frame  *StreamFrame
		wantFin bool
		wantOff bool
	}{
		{
			name: "no offset, no fin",
			frame: &StreamFrame{
				StreamID: 4,
				Offset:   0,
				Data:     []byte("hello"),
				Fin:      false,
			},
			wantFin: false,
			wantOff: false,
		},
		{
			name: "with offset, no fin",
			frame: &StreamFrame{
				StreamID: 8,
				Offset:   100,
				Data:     []byte("world"),
				Fin:      false,
			},
			wantFin: false,
			wantOff: true,
		},
		{
			name: "with fin, no offset",
			frame: &StreamFrame{
				StreamID: 12,
				Offset:   0,
				Data:     []byte("final"),
				Fin:      true,
			},
			wantFin: true,
			wantOff: false,
		},
		{
			name: "with fin and offset",
			frame: &StreamFrame{
				StreamID: 16,
				Offset:   500,
				Data:     []byte("done"),
				Fin:      true,
			},
			wantFin: true,
			wantOff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			buf, err := tt.frame.AppendTo(nil)
			if err != nil {
				t.Fatalf("AppendTo() error = %v", err)
			}

			// Verify frame type byte
			frameTypeByte := buf[0]
			hasFin := (frameTypeByte & StreamFrameFlagFIN) != 0
			hasOff := (frameTypeByte & StreamFrameFlagOFF) != 0

			if hasFin != tt.wantFin {
				t.Errorf("FIN flag = %v, want %v", hasFin, tt.wantFin)
			}
			if hasOff != tt.wantOff {
				t.Errorf("OFF flag = %v, want %v", hasOff, tt.wantOff)
			}

			// Decode
			parsed, n, err := ParseFrame(buf)
			if err != nil {
				t.Fatalf("ParseFrame() error = %v", err)
			}

			if n != len(buf) {
				t.Errorf("ParseFrame() consumed %d bytes, want %d", n, len(buf))
			}

			stream, ok := parsed.(*StreamFrame)
			if !ok {
				t.Fatalf("Parsed frame is not StreamFrame")
			}

			if stream.StreamID != tt.frame.StreamID {
				t.Errorf("StreamID = %d, want %d", stream.StreamID, tt.frame.StreamID)
			}
			if stream.Offset != tt.frame.Offset {
				t.Errorf("Offset = %d, want %d", stream.Offset, tt.frame.Offset)
			}
			if stream.Fin != tt.frame.Fin {
				t.Errorf("Fin = %v, want %v", stream.Fin, tt.frame.Fin)
			}
			if !bytes.Equal(stream.Data, tt.frame.Data) {
				t.Errorf("Data = %x, want %x", stream.Data, tt.frame.Data)
			}
		})
	}
}

func TestAckFrame(t *testing.T) {
	tests := []struct {
		name string
		frame *AckFrame
	}{
		{
			name: "single range",
			frame: &AckFrame{
				LargestAcked: 100,
				AckDelay:     50,
				Ranges: []AckRange{
					{Gap: 0, Length: 10},
				},
			},
		},
		{
			name: "multiple ranges",
			frame: &AckFrame{
				LargestAcked: 200,
				AckDelay:     100,
				Ranges: []AckRange{
					{Gap: 0, Length: 5},
					{Gap: 2, Length: 3},
					{Gap: 1, Length: 4},
				},
			},
		},
		{
			name: "with ECN",
			frame: &AckFrame{
				LargestAcked: 150,
				AckDelay:     75,
				Ranges: []AckRange{
					{Gap: 0, Length: 8},
				},
				ECN: &ECNCounts{
					ECT0: 10,
					ECT1: 5,
					CE:   2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			buf, err := tt.frame.AppendTo(nil)
			if err != nil {
				t.Fatalf("AppendTo() error = %v", err)
			}

			// Verify frame type
			if tt.frame.ECN != nil {
				if buf[0] != byte(FrameTypeAckECN) {
					t.Errorf("Frame type = 0x%02x, want 0x03", buf[0])
				}
			} else {
				if buf[0] != byte(FrameTypeAck) {
					t.Errorf("Frame type = 0x%02x, want 0x02", buf[0])
				}
			}

			// Decode
			parsed, n, err := ParseFrame(buf)
			if err != nil {
				t.Fatalf("ParseFrame() error = %v", err)
			}

			if n != len(buf) {
				t.Errorf("ParseFrame() consumed %d bytes, want %d", n, len(buf))
			}

			ack, ok := parsed.(*AckFrame)
			if !ok {
				t.Fatalf("Parsed frame is not AckFrame")
			}

			if ack.LargestAcked != tt.frame.LargestAcked {
				t.Errorf("LargestAcked = %d, want %d", ack.LargestAcked, tt.frame.LargestAcked)
			}
			if ack.AckDelay != tt.frame.AckDelay {
				t.Errorf("AckDelay = %d, want %d", ack.AckDelay, tt.frame.AckDelay)
			}
			if len(ack.Ranges) != len(tt.frame.Ranges) {
				t.Fatalf("Ranges count = %d, want %d", len(ack.Ranges), len(tt.frame.Ranges))
			}

			for i := range ack.Ranges {
				if ack.Ranges[i].Gap != tt.frame.Ranges[i].Gap {
					t.Errorf("Range[%d].Gap = %d, want %d", i, ack.Ranges[i].Gap, tt.frame.Ranges[i].Gap)
				}
				if ack.Ranges[i].Length != tt.frame.Ranges[i].Length {
					t.Errorf("Range[%d].Length = %d, want %d", i, ack.Ranges[i].Length, tt.frame.Ranges[i].Length)
				}
			}

			if tt.frame.ECN != nil {
				if ack.ECN == nil {
					t.Fatal("ECN is nil")
				}
				if ack.ECN.ECT0 != tt.frame.ECN.ECT0 {
					t.Errorf("ECN.ECT0 = %d, want %d", ack.ECN.ECT0, tt.frame.ECN.ECT0)
				}
				if ack.ECN.ECT1 != tt.frame.ECN.ECT1 {
					t.Errorf("ECN.ECT1 = %d, want %d", ack.ECN.ECT1, tt.frame.ECN.ECT1)
				}
				if ack.ECN.CE != tt.frame.ECN.CE {
					t.Errorf("ECN.CE = %d, want %d", ack.ECN.CE, tt.frame.ECN.CE)
				}
			}
		})
	}
}

func TestConnectionCloseFrame(t *testing.T) {
	tests := []struct {
		name  string
		frame *ConnectionCloseFrame
	}{
		{
			name: "QUIC error",
			frame: &ConnectionCloseFrame{
				ErrorCode:    0x01,
				FrameType:    0x06,
				ReasonPhrase: []byte("internal error"),
				IsAppError:   false,
			},
		},
		{
			name: "Application error",
			frame: &ConnectionCloseFrame{
				ErrorCode:    0x100,
				ReasonPhrase: []byte("user requested"),
				IsAppError:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			buf, err := tt.frame.AppendTo(nil)
			if err != nil {
				t.Fatalf("AppendTo() error = %v", err)
			}

			// Decode
			parsed, n, err := ParseFrame(buf)
			if err != nil {
				t.Fatalf("ParseFrame() error = %v", err)
			}

			if n != len(buf) {
				t.Errorf("ParseFrame() consumed %d bytes, want %d", n, len(buf))
			}

			cc, ok := parsed.(*ConnectionCloseFrame)
			if !ok {
				t.Fatalf("Parsed frame is not ConnectionCloseFrame")
			}

			if cc.ErrorCode != tt.frame.ErrorCode {
				t.Errorf("ErrorCode = 0x%x, want 0x%x", cc.ErrorCode, tt.frame.ErrorCode)
			}
			if !tt.frame.IsAppError && cc.FrameType != tt.frame.FrameType {
				t.Errorf("FrameType = 0x%x, want 0x%x", cc.FrameType, tt.frame.FrameType)
			}
			if !bytes.Equal(cc.ReasonPhrase, tt.frame.ReasonPhrase) {
				t.Errorf("ReasonPhrase = %s, want %s", cc.ReasonPhrase, tt.frame.ReasonPhrase)
			}
			if cc.IsAppError != tt.frame.IsAppError {
				t.Errorf("IsAppError = %v, want %v", cc.IsAppError, tt.frame.IsAppError)
			}
		})
	}
}

func TestDatagramFrame(t *testing.T) {
	frame := &DatagramFrame{
		Data: []byte("unreliable datagram data"),
	}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Verify frame type
	if buf[0] != byte(FrameTypeDatagramLen) {
		t.Errorf("Frame type = 0x%02x, want 0x31", buf[0])
	}

	// Note: ParseFrame doesn't support DATAGRAM yet
	// This test just verifies encoding
}

func TestResetStreamFrame(t *testing.T) {
	frame := &ResetStreamFrame{
		StreamID:  4,
		ErrorCode: 0x100,
		FinalSize: 1234,
	}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Verify frame type
	if buf[0] != byte(FrameTypeResetStream) {
		t.Errorf("Frame type = 0x%02x, want 0x04", buf[0])
	}
}

func TestStopSendingFrame(t *testing.T) {
	frame := &StopSendingFrame{
		StreamID:  8,
		ErrorCode: 0x200,
	}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Verify frame type
	if buf[0] != byte(FrameTypeStopSending) {
		t.Errorf("Frame type = 0x%02x, want 0x05", buf[0])
	}
}

func TestMaxDataFrame(t *testing.T) {
	frame := &MaxDataFrame{
		MaximumData: 1000000,
	}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Verify frame type
	if buf[0] != byte(FrameTypeMaxData) {
		t.Errorf("Frame type = 0x%02x, want 0x10", buf[0])
	}
}

func TestMaxStreamDataFrame(t *testing.T) {
	frame := &MaxStreamDataFrame{
		StreamID:    4,
		MaximumData: 500000,
	}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Verify frame type
	if buf[0] != byte(FrameTypeMaxStreamData) {
		t.Errorf("Frame type = 0x%02x, want 0x11", buf[0])
	}
}

func BenchmarkStreamFrameEncode(b *testing.B) {
	frame := &StreamFrame{
		StreamID: 4,
		Offset:   1000,
		Data:     make([]byte, 1024),
		Fin:      false,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.AppendTo(nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamFrameDecode(b *testing.B) {
	frame := &StreamFrame{
		StreamID: 4,
		Offset:   1000,
		Data:     make([]byte, 1024),
		Fin:      false,
	}

	buf, _ := frame.AppendTo(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := ParseFrame(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAckFrameEncode(b *testing.B) {
	frame := &AckFrame{
		LargestAcked: 1000,
		AckDelay:     100,
		Ranges: []AckRange{
			{Gap: 0, Length: 10},
			{Gap: 2, Length: 5},
			{Gap: 1, Length: 3},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.AppendTo(nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
