package qpack

import (
	"bytes"
	"testing"
)

func TestHuffmanEncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple", "test"},
		{"with spaces", "content-type"},
		{"mixed case", "Content-Type"},
		{"path", "/api/v1/users"},
		{"authority", "example.com:443"},
		{"numbers", "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := HuffmanEncode([]byte(tt.input))

			// Decode
			decoded, err := HuffmanDecode(encoded)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}

			// Verify
			if string(decoded) != tt.input {
				t.Errorf("Roundtrip failed: got %q, want %q", string(decoded), tt.input)
			}

			t.Logf("Input: %d bytes, Encoded: %d bytes, Ratio: %.2f",
				len(tt.input), len(encoded), float64(len(tt.input))/float64(len(encoded)))
		})
	}
}

func TestHuffmanEncodeString(t *testing.T) {
	input := "www.example.com"
	encoded := HuffmanEncodeString(input)

	if len(encoded) == 0 {
		t.Error("Encoded string is empty")
	}

	decoded, err := HuffmanDecodeString(encoded)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if decoded != input {
		t.Errorf("Roundtrip failed: got %q, want %q", decoded, input)
	}
}

func TestHuffmanEncodedLength(t *testing.T) {
	tests := []string{
		"content-type",
		"application/json",
		"/api/v1/users",
		"www.example.com",
	}

	for _, s := range tests {
		encoded := HuffmanEncode([]byte(s))
		estimatedLen := HuffmanEncodedLength([]byte(s))

		if estimatedLen != len(encoded) {
			t.Errorf("Length estimation failed for %q: estimated %d, actual %d",
				s, estimatedLen, len(encoded))
		}
	}
}

func TestShouldHuffmanEncode(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"short", false},    // Too short
		{"content-type: application/json", true}, // Should benefit
		{"abcdefghijklmnop", true}, // Long enough
	}

	for _, tt := range tests {
		result := ShouldHuffmanEncode([]byte(tt.input))
		if result != tt.expected {
			t.Errorf("ShouldHuffmanEncode(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestHuffmanEncodeWriter(t *testing.T) {
	input := "test data"
	var buf bytes.Buffer

	writer := NewHuffmanEncodeWriter(&buf)
	_, err := writer.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}

	err = writer.Flush()
	if err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	// Decode and verify
	decoded, err := HuffmanDecode(buf.Bytes())
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if string(decoded) != input {
		t.Errorf("Roundtrip failed: got %q, want %q", string(decoded), input)
	}
}

func TestHuffmanCompressionRatio(t *testing.T) {
	// Test with common HTTP header names and values
	testStrings := []string{
		":method",
		":path",
		":scheme",
		":authority",
		"content-type",
		"content-length",
		"user-agent",
		"accept",
		"accept-encoding",
		"accept-language",
		"application/json",
		"text/html",
		"https",
		"/api/v1/users",
		"example.com",
	}

	totalOriginal := 0
	totalEncoded := 0

	for _, s := range testStrings {
		encoded := HuffmanEncode([]byte(s))
		totalOriginal += len(s)
		totalEncoded += len(encoded)

		t.Logf("%20s: %3d -> %3d bytes (%.2f%%)",
			s, len(s), len(encoded), 100.0*float64(len(encoded))/float64(len(s)))
	}

	ratio := float64(totalOriginal) / float64(totalEncoded)
	t.Logf("\nTotal: %d -> %d bytes, compression ratio: %.2fx",
		totalOriginal, totalEncoded, ratio)

	if ratio < 1.0 {
		t.Error("Huffman encoding should provide compression, got expansion")
	}
}

func TestHuffmanInvalidData(t *testing.T) {
	// Test decoding invalid data
	invalid := []byte{0xFF, 0xFF, 0xFF} // Invalid code sequence
	_, err := HuffmanDecode(invalid)

	// Should either decode successfully (with padding) or return error
	// Both are acceptable depending on implementation
	if err != nil {
		t.Logf("Decode returned error (expected): %v", err)
	}
}

func BenchmarkHuffmanEncode(b *testing.B) {
	data := []byte("content-type: application/json; charset=utf-8")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = HuffmanEncode(data)
	}
}

func BenchmarkHuffmanDecode(b *testing.B) {
	data := []byte("content-type: application/json; charset=utf-8")
	encoded := HuffmanEncode(data)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = HuffmanDecode(encoded)
	}
}

func BenchmarkHuffmanEncodeShort(b *testing.B) {
	data := []byte("test")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = HuffmanEncode(data)
	}
}

func BenchmarkHuffmanEncodeLong(b *testing.B) {
	data := bytes.Repeat([]byte("content-type: application/json; "), 10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = HuffmanEncode(data)
	}
}

func BenchmarkShouldHuffmanEncode(b *testing.B) {
	data := []byte("content-type: application/json")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ShouldHuffmanEncode(data)
	}
}
