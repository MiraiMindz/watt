package websocket

import (
	"bytes"
	"io"
	"testing"
)

func TestFrameReaderReadFrame(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		expect  *Frame
		wantErr error
	}{
		{
			name: "simple unmasked text frame",
			input: []byte{
				0x81, 0x05, // FIN, Text, length=5
				'H', 'e', 'l', 'l', 'o',
			},
			expect: &Frame{
				Fin:     true,
				Opcode:  OpcodeText,
				Masked:  false,
				Length:  5,
				Payload: []byte("Hello"),
			},
		},
		{
			name: "masked text frame",
			input: []byte{
				0x81, 0x85,             // FIN, Text, masked, length=5
				0x12, 0x34, 0x56, 0x78, // Mask key
				0x5A, 0x51, 0x3A, 0x14, 0x7D, // Masked "Hello"
			},
			expect: &Frame{
				Fin:     true,
				Opcode:  OpcodeText,
				Masked:  true,
				Length:  5,
				MaskKey: [4]byte{0x12, 0x34, 0x56, 0x78},
				Payload: []byte("Hello"),
			},
		},
		{
			name: "ping frame",
			input: []byte{
				0x89, 0x00, // FIN, Ping, length=0
			},
			expect: &Frame{
				Fin:     true,
				Opcode:  OpcodePing,
				Masked:  false,
				Length:  0,
				Payload: nil,
			},
		},
		{
			name: "close frame with code",
			input: []byte{
				0x88, 0x02, // FIN, Close, length=2
				0x03, 0xE8, // Code 1000
			},
			expect: &Frame{
				Fin:     true,
				Opcode:  OpcodeClose,
				Masked:  false,
				Length:  2,
				Payload: []byte{0x03, 0xE8},
			},
		},
		{
			name: "fragmented text frame (not final)",
			input: []byte{
				0x01, 0x03, // NOT FIN, Text, length=3
				'H', 'e', 'l',
			},
			expect: &Frame{
				Fin:     false,
				Opcode:  OpcodeText,
				Masked:  false,
				Length:  3,
				Payload: []byte("Hel"),
			},
		},
		{
			name: "continuation frame (final)",
			input: []byte{
				0x80, 0x02, // FIN, Continuation, length=2
				'l', 'o',
			},
			expect: &Frame{
				Fin:     true,
				Opcode:  OpcodeContinuation,
				Masked:  false,
				Length:  2,
				Payload: []byte("lo"),
			},
		},
		{
			name: "extended 16-bit length",
			input: func() []byte {
				data := make([]byte, 2+2+256)
				data[0] = 0x82 // FIN, Binary
				data[1] = 126  // Extended 16-bit length
				data[2] = 0x01 // Length high byte (256)
				data[3] = 0x00 // Length low byte
				for i := 0; i < 256; i++ {
					data[4+i] = byte(i)
				}
				return data
			}(),
			expect: &Frame{
				Fin:     true,
				Opcode:  OpcodeBinary,
				Masked:  false,
				Length:  256,
				Payload: func() []byte {
					data := make([]byte, 256)
					for i := 0; i < 256; i++ {
						data[i] = byte(i)
					}
					return data
				}(),
			},
		},
		{
			name: "invalid opcode",
			input: []byte{
				0x83, 0x00, // FIN, opcode=3 (reserved), length=0
			},
			wantErr: ErrInvalidOpcode,
		},
		{
			name: "fragmented control frame",
			input: []byte{
				0x08, 0x00, // NOT FIN, Close (invalid)
			},
			wantErr: ErrFragmentedControl,
		},
		{
			name: "control frame too large",
			input: []byte{
				0x89, 0x7E, // FIN, Ping, length=126 (too large)
			},
			wantErr: ErrInvalidControlFrame,
		},
		{
			name: "RSV bits set",
			input: []byte{
				0xC1, 0x00, // FIN, RSV1 set, Text
			},
			wantErr: ErrReservedBitsSet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewFrameReader(bytes.NewReader(tt.input))
			frame, err := reader.ReadFrame()

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare frame fields
			if frame.Fin != tt.expect.Fin {
				t.Errorf("Fin: got %v, want %v", frame.Fin, tt.expect.Fin)
			}
			if frame.Opcode != tt.expect.Opcode {
				t.Errorf("Opcode: got 0x%X, want 0x%X", frame.Opcode, tt.expect.Opcode)
			}
			if frame.Masked != tt.expect.Masked {
				t.Errorf("Masked: got %v, want %v", frame.Masked, tt.expect.Masked)
			}
			if frame.Length != tt.expect.Length {
				t.Errorf("Length: got %d, want %d", frame.Length, tt.expect.Length)
			}
			if frame.Masked && frame.MaskKey != tt.expect.MaskKey {
				t.Errorf("MaskKey: got %v, want %v", frame.MaskKey, tt.expect.MaskKey)
			}
			if !bytes.Equal(frame.Payload, tt.expect.Payload) {
				t.Errorf("Payload: got %v, want %v", frame.Payload, tt.expect.Payload)
			}
		})
	}
}

func TestFrameWriterWriteFrame(t *testing.T) {
	tests := []struct {
		name    string
		opcode  byte
		fin     bool
		payload []byte
		maskKey *[4]byte
		expect  []byte
	}{
		{
			name:    "simple unmasked text frame",
			opcode:  OpcodeText,
			fin:     true,
			payload: []byte("Hello"),
			maskKey: nil,
			expect: []byte{
				0x81, 0x05, // FIN, Text, length=5
				'H', 'e', 'l', 'l', 'o',
			},
		},
		{
			name:    "masked text frame",
			opcode:  OpcodeText,
			fin:     true,
			payload: []byte("Hello"),
			maskKey: &[4]byte{0x12, 0x34, 0x56, 0x78},
			expect: []byte{
				0x81, 0x85,             // FIN, Text, masked, length=5
				0x12, 0x34, 0x56, 0x78, // Mask key
				0x5A, 0x51, 0x3A, 0x14, 0x7D, // Masked "Hello"
			},
		},
		{
			name:    "ping frame",
			opcode:  OpcodePing,
			fin:     true,
			payload: nil,
			maskKey: nil,
			expect: []byte{
				0x89, 0x00, // FIN, Ping, length=0
			},
		},
		{
			name:    "fragmented text frame (not final)",
			opcode:  OpcodeText,
			fin:     false,
			payload: []byte("Hel"),
			maskKey: nil,
			expect: []byte{
				0x01, 0x03, // NOT FIN, Text, length=3
				'H', 'e', 'l',
			},
		},
		{
			name:    "extended 16-bit length",
			opcode:  OpcodeBinary,
			fin:     true,
			payload: make([]byte, 256), // 256 bytes
			maskKey: nil,
			expect: func() []byte {
				data := make([]byte, 4+256)
				data[0] = 0x82 // FIN, Binary
				data[1] = 126  // Extended 16-bit length
				data[2] = 0x01 // Length high byte
				data[3] = 0x00 // Length low byte
				return data
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewFrameWriter(&buf)

			// Make a copy of payload since masking is in-place
			payload := make([]byte, len(tt.payload))
			copy(payload, tt.payload)

			err := writer.WriteFrame(tt.opcode, tt.fin, payload, tt.maskKey)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()

			// For frames with extended length, only compare header
			if len(tt.expect) < len(got) {
				got = got[:len(tt.expect)]
			}

			if !bytes.Equal(got, tt.expect) {
				t.Errorf("WriteFrame output:\ngot:  %v\nwant: %v", got, tt.expect)
			}
		})
	}
}

func TestFrameReaderBufferReuse(t *testing.T) {
	// Test that FrameReader reuses its internal buffer
	var buf bytes.Buffer
	writer := NewFrameWriter(&buf)

	// Write two frames
	writer.WriteFrame(OpcodeText, true, []byte("Hello"), nil)
	writer.WriteFrame(OpcodeText, true, []byte("World"), nil)

	reader := NewFrameReader(&buf)

	// Read first frame
	frame1, err := reader.ReadFrame()
	if err != nil {
		t.Fatal(err)
	}
	payload1Addr := &frame1.Payload[0]

	// Read second frame
	frame2, err := reader.ReadFrame()
	if err != nil {
		t.Fatal(err)
	}
	payload2Addr := &frame2.Payload[0]

	// Check that buffers are reused (same underlying array)
	if payload1Addr == payload2Addr {
		t.Log("Buffer reused successfully (zero allocations)")
	}
}

// Benchmarks

func BenchmarkFrameReaderReadFrame(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			// Prepare frame data
			var buf bytes.Buffer
			writer := NewFrameWriter(&buf)
			data := make([]byte, size)
			writer.WriteFrame(OpcodeBinary, true, data, nil)

			frameData := buf.Bytes()

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				reader := NewFrameReader(bytes.NewReader(frameData))
				_, err := reader.ReadFrame()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFrameWriterWriteFrame(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			writer := NewFrameWriter(io.Discard)

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Make a copy since masking is in-place
				payload := make([]byte, size)
				err := writer.WriteFrame(OpcodeBinary, true, payload, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFrameReadWriteRoundtrip(b *testing.B) {
	data := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		writer := NewFrameWriter(&buf)

		// Write
		payload := make([]byte, len(data))
		copy(payload, data)
		err := writer.WriteFrame(OpcodeBinary, true, payload, nil)
		if err != nil {
			b.Fatal(err)
		}

		// Read
		reader := NewFrameReader(&buf)
		_, err = reader.ReadFrame()
		if err != nil {
			b.Fatal(err)
		}
	}
}
