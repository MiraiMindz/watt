package websocket

import (
	"bytes"
	"testing"
)

func TestComputeAcceptKey(t *testing.T) {
	tests := []struct {
		key    string
		expect string
	}{
		{
			// Example from RFC 6455
			key:    "dGhlIHNhbXBsZSBub25jZQ==",
			expect: "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=",
		},
		{
			key:    "x3JJHMbDL1EzLkh9GBhXDw==",
			expect: "HSmrc0sMlYUkAGmm5OPpG2HaGWk=",
		},
	}

	for _, tt := range tests {
		got := ComputeAcceptKey(tt.key)
		if got != tt.expect {
			t.Errorf("ComputeAcceptKey(%q) = %q, want %q", tt.key, got, tt.expect)
		}
	}
}

func TestMaskBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		maskKey [4]byte
		expect  []byte
	}{
		{
			name:    "simple 4 bytes",
			data:    []byte{0x00, 0x11, 0x22, 0x33},
			maskKey: [4]byte{0xAA, 0xBB, 0xCC, 0xDD},
			expect:  []byte{0xAA, 0xAA, 0xEE, 0xEE},
		},
		{
			name:    "longer than mask",
			data:    []byte{0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF},
			maskKey: [4]byte{0x12, 0x34, 0x56, 0x78},
			expect:  []byte{0x12, 0x34, 0x56, 0x78, 0xED, 0xCB, 0xA9, 0x87},
		},
		{
			name:    "empty data",
			data:    []byte{},
			maskKey: [4]byte{0x12, 0x34, 0x56, 0x78},
			expect:  []byte{},
		},
		{
			name:    "single byte",
			data:    []byte{0xFF},
			maskKey: [4]byte{0x12, 0x34, 0x56, 0x78},
			expect:  []byte{0xED},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy since masking is in-place
			data := make([]byte, len(tt.data))
			copy(data, tt.data)

			maskBytes(data, tt.maskKey)

			if !bytes.Equal(data, tt.expect) {
				t.Errorf("maskBytes(%v, %v) = %v, want %v",
					tt.data, tt.maskKey, data, tt.expect)
			}
		})
	}
}

func TestMaskBytesInverse(t *testing.T) {
	// Test that masking twice returns original data
	original := []byte("Hello, WebSocket!")
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

	data := make([]byte, len(original))
	copy(data, original)

	// Mask once
	maskBytes(data, maskKey)

	// Should be different
	if bytes.Equal(data, original) {
		t.Error("maskBytes did not modify data")
	}

	// Mask again
	maskBytes(data, maskKey)

	// Should be back to original
	if !bytes.Equal(data, original) {
		t.Errorf("Double masking did not restore original: got %v, want %v",
			data, original)
	}
}

func TestFrameIsControl(t *testing.T) {
	tests := []struct {
		opcode byte
		expect bool
	}{
		{OpcodeContinuation, false},
		{OpcodeText, false},
		{OpcodeBinary, false},
		{OpcodeClose, true},
		{OpcodePing, true},
		{OpcodePong, true},
	}

	for _, tt := range tests {
		frame := &Frame{Opcode: tt.opcode}
		if got := frame.IsControl(); got != tt.expect {
			t.Errorf("Frame{Opcode: 0x%X}.IsControl() = %v, want %v",
				tt.opcode, got, tt.expect)
		}
	}
}

func TestFrameIsData(t *testing.T) {
	tests := []struct {
		opcode byte
		expect bool
	}{
		{OpcodeContinuation, true},
		{OpcodeText, true},
		{OpcodeBinary, true},
		{OpcodeClose, false},
		{OpcodePing, false},
		{OpcodePong, false},
	}

	for _, tt := range tests {
		frame := &Frame{Opcode: tt.opcode}
		if got := frame.IsData(); got != tt.expect {
			t.Errorf("Frame{Opcode: 0x%X}.IsData() = %v, want %v",
				tt.opcode, got, tt.expect)
		}
	}
}

func TestIsValidCloseCode(t *testing.T) {
	tests := []struct {
		code  uint16
		valid bool
	}{
		{1000, true},  // Normal closure
		{1001, true},  // Going away
		{1002, true},  // Protocol error
		{1003, true},  // Unsupported data
		{1004, false}, // Reserved
		{1005, false}, // No status received (reserved)
		{1006, false}, // Abnormal closure (reserved)
		{1007, true},  // Invalid frame payload
		{1008, true},  // Policy violation
		{1009, true},  // Message too big
		{1010, true},  // Mandatory extension
		{1011, true},  // Internal server error
		{1015, false}, // TLS handshake (reserved)
		{3000, true},  // Registered (3000-3999)
		{3999, true},  // Registered (3000-3999)
		{4000, true},  // Private use (4000-4999)
		{4999, true},  // Private use (4000-4999)
		{5000, false}, // Invalid
		{999, false},  // Invalid
	}

	for _, tt := range tests {
		got := isValidCloseCode(tt.code)
		if got != tt.valid {
			t.Errorf("isValidCloseCode(%d) = %v, want %v", tt.code, got, tt.valid)
		}
	}
}

// Benchmarks

func BenchmarkMaskBytes(b *testing.B) {
	sizes := []int{16, 64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			data := make([]byte, size)
			maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				maskBytes(data, maskKey)
			}
		})
	}
}

func BenchmarkComputeAcceptKey(b *testing.B) {
	key := "dGhlIHNhbXBsZSBub25jZQ=="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ComputeAcceptKey(key)
	}
}
