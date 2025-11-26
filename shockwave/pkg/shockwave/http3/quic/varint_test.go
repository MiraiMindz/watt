package quic

import (
	"bytes"
	"testing"
)

func TestVarintEncoding(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		want  []byte
	}{
		{"1-byte max", 63, []byte{0x3F}},
		{"2-byte min", 64, []byte{0x40, 0x40}},
		{"2-byte max", 16383, []byte{0x7F, 0xFF}},
		{"4-byte min", 16384, []byte{0x80, 0x00, 0x40, 0x00}},
		{"4-byte max", 1073741823, []byte{0xBF, 0xFF, 0xFF, 0xFF}},
		{"8-byte min", 1073741824, []byte{0xC0, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00}},
		{"8-byte max", 4611686018427387903, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"42", 42, []byte{0x2A}},
		{"1000", 1000, []byte{0x43, 0xE8}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test putVarint
			buf := make([]byte, 8)
			n := putVarint(buf, tt.value)
			if n != len(tt.want) {
				t.Errorf("putVarint() length = %d, want %d", n, len(tt.want))
			}
			if !bytes.Equal(buf[:n], tt.want) {
				t.Errorf("putVarint() = %x, want %x", buf[:n], tt.want)
			}

			// Test appendVarint
			buf2, err := appendVarint(nil, tt.value)
			if err != nil {
				t.Fatalf("appendVarint() error = %v", err)
			}
			if !bytes.Equal(buf2, tt.want) {
				t.Errorf("appendVarint() = %x, want %x", buf2, tt.want)
			}
		})
	}
}

func TestVarintDecoding(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  uint64
		wantN int
	}{
		{"1-byte", []byte{0x3F}, 63, 1},
		{"2-byte", []byte{0x7F, 0xFF}, 16383, 2},
		{"4-byte", []byte{0xBF, 0xFF, 0xFF, 0xFF}, 1073741823, 4},
		{"8-byte", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, 4611686018427387903, 8},
		{"zero", []byte{0x00}, 0, 1},
		{"42", []byte{0x2A}, 42, 1},
		{"1000", []byte{0x43, 0xE8}, 1000, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := parseVarint(tt.input)
			if err != nil {
				t.Fatalf("parseVarint() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("parseVarint() value = %d, want %d", got, tt.want)
			}
			if n != tt.wantN {
				t.Errorf("parseVarint() n = %d, want %d", n, tt.wantN)
			}
		})
	}
}

func TestVarintRoundTrip(t *testing.T) {
	values := []uint64{
		0, 1, 42, 63, 64, 100, 1000, 10000,
		MaxVarint1, MaxVarint1 + 1,
		MaxVarint2, MaxVarint2 + 1,
		MaxVarint4, MaxVarint4 + 1,
		MaxVarint8,
	}

	for _, v := range values {
		buf := make([]byte, 8)
		n := putVarint(buf, v)
		if n == 0 {
			t.Fatalf("putVarint(%d) failed", v)
		}

		got, n2, err := parseVarint(buf[:n])
		if err != nil {
			t.Fatalf("parseVarint() error = %v for value %d", err, v)
		}
		if got != v {
			t.Errorf("Round trip failed: got %d, want %d", got, v)
		}
		if n2 != n {
			t.Errorf("Length mismatch: encoded %d bytes, decoded %d bytes", n, n2)
		}
	}
}

func TestVarintTooLarge(t *testing.T) {
	buf := make([]byte, 8)
	n := putVarint(buf, MaxVarint8+1)
	if n != 0 {
		t.Errorf("putVarint(too large) should return 0, got %d", n)
	}

	_, err := appendVarint(nil, MaxVarint8+1)
	if err != ErrVarintTooLarge {
		t.Errorf("appendVarint(too large) error = %v, want %v", err, ErrVarintTooLarge)
	}
}

func TestVarintTruncated(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"empty", []byte{}},
		{"2-byte truncated", []byte{0x40}},
		{"4-byte truncated", []byte{0x80, 0x00}},
		{"8-byte truncated", []byte{0xC0, 0x00, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseVarint(tt.input)
			if err != ErrVarintTrunc {
				t.Errorf("parseVarint() error = %v, want %v", err, ErrVarintTrunc)
			}
		})
	}
}

func TestConnectionID(t *testing.T) {
	tests := []struct {
		name string
		cid  ConnectionID
	}{
		{"empty", ConnectionID{}},
		{"8-byte", ConnectionID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
		{"20-byte", ConnectionID{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			0x11, 0x12, 0x13, 0x14,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test length
			if tt.cid.Len() != len(tt.cid) {
				t.Errorf("Len() = %d, want %d", tt.cid.Len(), len(tt.cid))
			}

			// Test IsEmpty
			isEmpty := len(tt.cid) == 0
			if tt.cid.IsEmpty() != isEmpty {
				t.Errorf("IsEmpty() = %v, want %v", tt.cid.IsEmpty(), isEmpty)
			}

			// Test Equal
			if !tt.cid.Equal(tt.cid) {
				t.Error("Equal() should return true for same CID")
			}

			other := make(ConnectionID, len(tt.cid))
			copy(other, tt.cid)
			if !tt.cid.Equal(other) {
				t.Error("Equal() should return true for copy")
			}

			if len(tt.cid) > 0 {
				other[0] ^= 0xFF
				if tt.cid.Equal(other) {
					t.Error("Equal() should return false for different CID")
				}
			}
		})
	}
}

func TestConnectionIDEncoding(t *testing.T) {
	tests := []struct {
		name string
		cid  ConnectionID
	}{
		{"empty", ConnectionID{}},
		{"1-byte", ConnectionID{0x42}},
		{"8-byte", ConnectionID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := appendConnectionID(nil, tt.cid)

			// First byte should be length
			if buf[0] != byte(len(tt.cid)) {
				t.Errorf("Length byte = %d, want %d", buf[0], len(tt.cid))
			}

			// Parse it back
			parsed, n, err := parseConnectionID(buf)
			if err != nil {
				t.Fatalf("parseConnectionID() error = %v", err)
			}

			if !parsed.Equal(tt.cid) {
				t.Errorf("parseConnectionID() = %x, want %x", parsed, tt.cid)
			}

			if n != 1+len(tt.cid) {
				t.Errorf("parseConnectionID() n = %d, want %d", n, 1+len(tt.cid))
			}
		})
	}
}

func BenchmarkVarintEncode(b *testing.B) {
	values := []struct {
		name  string
		value uint64
	}{
		{"1-byte", 42},
		{"2-byte", 1000},
		{"4-byte", 100000},
		{"8-byte", 10000000000},
	}

	for _, v := range values {
		b.Run(v.name, func(b *testing.B) {
			buf := make([]byte, 8)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = putVarint(buf, v.value)
			}
		})
	}
}

func BenchmarkVarintDecode(b *testing.B) {
	values := []struct {
		name string
		data []byte
	}{
		{"1-byte", []byte{0x2A}},
		{"2-byte", []byte{0x43, 0xE8}},
		{"4-byte", []byte{0x80, 0x01, 0x86, 0xA0}},
		{"8-byte", []byte{0xC0, 0x00, 0x00, 0x02, 0x54, 0x0B, 0xE4, 0x00}},
	}

	for _, v := range values {
		b.Run(v.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _, _ = parseVarint(v.data)
			}
		})
	}
}
