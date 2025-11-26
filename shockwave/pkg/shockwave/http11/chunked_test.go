package http11

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// TestChunkedReader_Simple tests basic chunked transfer encoding
func TestChunkedReader_Simple(t *testing.T) {
	// Example from RFC 7230 §4.1
	input := "4\r\nWiki\r\n5\r\npedia\r\n0\r\n\r\n"
	expected := "Wikipedia"

	cr := NewChunkedReader(strings.NewReader(input))
	output, err := io.ReadAll(cr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(output) != expected {
		t.Errorf("Got %q, want %q", string(output), expected)
	}
}

// TestChunkedReader_ComplexExample tests a more complex chunked example
func TestChunkedReader_ComplexExample(t *testing.T) {
	// Wikipedia in chunks
	input := "4\r\nWiki\r\n5\r\npedia\r\nE\r\n in\r\n\r\nchunks.\r\n0\r\n\r\n"
	expected := "Wikipedia in\r\n\r\nchunks."

	cr := NewChunkedReader(strings.NewReader(input))
	output, err := io.ReadAll(cr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(output) != expected {
		t.Errorf("Got %q, want %q", string(output), expected)
	}
}

// TestChunkedReader_WithChunkExtensions tests chunk extensions (should be ignored)
func TestChunkedReader_WithChunkExtensions(t *testing.T) {
	// Chunk extensions after size (should be ignored per our security policy)
	input := "4;name=value\r\nWiki\r\n5;foo=bar\r\npedia\r\n0\r\n\r\n"
	expected := "Wikipedia"

	cr := NewChunkedReader(strings.NewReader(input))
	output, err := io.ReadAll(cr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(output) != expected {
		t.Errorf("Got %q, want %q", string(output), expected)
	}
}

// TestChunkedReader_EmptyBody tests zero-length chunked body
func TestChunkedReader_EmptyBody(t *testing.T) {
	input := "0\r\n\r\n"
	expected := ""

	cr := NewChunkedReader(strings.NewReader(input))
	output, err := io.ReadAll(cr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(output) != expected {
		t.Errorf("Got %q, want %q", string(output), expected)
	}
}

// TestChunkedReader_SingleChunk tests single chunk
func TestChunkedReader_SingleChunk(t *testing.T) {
	input := "D\r\nHello, World!\r\n0\r\n\r\n"
	expected := "Hello, World!"

	cr := NewChunkedReader(strings.NewReader(input))
	output, err := io.ReadAll(cr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(output) != expected {
		t.Errorf("Got %q, want %q", string(output), expected)
	}
}

// TestChunkedReader_IncrementalRead tests reading chunks incrementally
func TestChunkedReader_IncrementalRead(t *testing.T) {
	input := "4\r\nWiki\r\n5\r\npedia\r\n0\r\n\r\n"

	cr := NewChunkedReader(strings.NewReader(input))

	// Read 3 bytes at a time
	var result []byte
	buf := make([]byte, 3)

	for {
		n, err := cr.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	expected := "Wikipedia"
	if string(result) != expected {
		t.Errorf("Got %q, want %q", string(result), expected)
	}
}

// TestChunkedReader_HexCases tests various hex formats
func TestChunkedReader_HexCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Lowercase hex",
			input:    "a\r\n0123456789\r\n0\r\n\r\n",
			expected: "0123456789",
		},
		{
			name:     "Uppercase hex",
			input:    "A\r\n0123456789\r\n0\r\n\r\n",
			expected: "0123456789",
		},
		{
			name:     "Mixed case hex",
			input:    "aB\r\n" + strings.Repeat("x", 171) + "\r\n0\r\n\r\n",
			expected: strings.Repeat("x", 171),
		},
		{
			name:     "Large chunk (1000 bytes)",
			input:    "3e8\r\n" + strings.Repeat("y", 1000) + "\r\n0\r\n\r\n",
			expected: strings.Repeat("y", 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := NewChunkedReader(strings.NewReader(tt.input))
			output, err := io.ReadAll(cr)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if string(output) != tt.expected {
				t.Errorf("Got %d bytes, want %d bytes", len(output), len(tt.expected))
			}
		})
	}
}

// TestChunkedReader_Errors tests error conditions
func TestChunkedReader_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Invalid hex digit",
			input: "G\r\ndata\r\n0\r\n\r\n",
		},
		{
			name:  "Missing CRLF after size",
			input: "4\nWiki\r\n0\r\n\r\n",
		},
		{
			name:  "Missing CRLF after chunk",
			input: "4\r\nWiki\n0\r\n\r\n",
		},
		{
			name:  "Incomplete chunk",
			input: "4\r\nWi",
		},
		{
			name:  "Missing final CRLF",
			input: "4\r\nWiki\r\n0\r\n",
		},
		{
			name:  "Chunk size mismatch (too much data)",
			input: "2\r\nWiki\r\n0\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := NewChunkedReader(strings.NewReader(tt.input))
			_, err := io.ReadAll(cr)

			if err == nil {
				t.Error("Expected error, got nil")
			}

			if err != ErrChunkedEncoding && err != io.ErrUnexpectedEOF {
				// Both are acceptable error types
				t.Logf("Got error: %v", err)
			}
		})
	}
}

// TestChunkedReader_SizeLimits tests chunk size limits
func TestChunkedReader_SizeLimits(t *testing.T) {
	// Try to create a chunk larger than maxChunkSize (16MB)
	// Using hex for 17MB: 0x1100000 = 17825792
	input := "1100000\r\n" + strings.Repeat("x", 100) + "\r\n0\r\n\r\n"

	cr := NewChunkedReader(strings.NewReader(input))
	_, err := io.ReadAll(cr)

	if err != ErrChunkedEncoding {
		t.Errorf("Expected ErrChunkedEncoding for oversized chunk, got %v", err)
	}
}

// TestChunkedReader_WithLimits tests body size limits
func TestChunkedReader_WithLimits(t *testing.T) {
	// Create 200 bytes of data with 100-byte limit
	input := "64\r\n" + strings.Repeat("a", 100) + "\r\n64\r\n" + strings.Repeat("b", 100) + "\r\n0\r\n\r\n"

	cr := NewChunkedReaderWithLimits(strings.NewReader(input), 0, 100)
	_, err := io.ReadAll(cr)

	if err != ErrChunkedEncoding {
		t.Errorf("Expected ErrChunkedEncoding for body size limit exceeded, got %v", err)
	}

	// Verify we can read up to the limit
	input2 := "64\r\n" + strings.Repeat("a", 100) + "\r\n0\r\n\r\n"
	cr2 := NewChunkedReaderWithLimits(strings.NewReader(input2), 0, 100)
	data, err := io.ReadAll(cr2)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(data) != 100 {
		t.Errorf("Expected 100 bytes, got %d", len(data))
	}
}

// TestChunkedReader_TotalRead tests TotalRead() method
func TestChunkedReader_TotalRead(t *testing.T) {
	input := "4\r\nWiki\r\n5\r\npedia\r\n0\r\n\r\n"

	cr := NewChunkedReader(strings.NewReader(input))
	io.ReadAll(cr)

	if cr.TotalRead() != 9 {
		t.Errorf("Expected TotalRead() = 9, got %d", cr.TotalRead())
	}
}

// TestChunkedReader_WithParser tests chunked encoding through the parser
func TestChunkedReader_WithParser(t *testing.T) {
	request := "POST /upload HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n" +
		"4\r\n" +
		"Wiki\r\n" +
		"5\r\n" +
		"pedia\r\n" +
		"0\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(request))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	defer PutRequest(req)

	if !req.IsChunked() {
		t.Fatal("Request should be chunked")
	}

	if req.Body == nil {
		t.Fatal("Request body should not be nil")
	}

	// Read body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Body read error: %v", err)
	}

	expected := "Wikipedia"
	if string(body) != expected {
		t.Errorf("Got body %q, want %q", string(body), expected)
	}
}

// TestChunkedReader_LargeBody tests reading a large chunked body
func TestChunkedReader_LargeBody(t *testing.T) {
	// Create a 1MB body in 1KB chunks
	var buf bytes.Buffer
	chunkSize := 1024
	numChunks := 1024

	for i := 0; i < numChunks; i++ {
		buf.WriteString("400\r\n") // 1024 in hex
		buf.WriteString(strings.Repeat("x", chunkSize))
		buf.WriteString("\r\n")
	}
	buf.WriteString("0\r\n\r\n")

	cr := NewChunkedReader(&buf)
	data, err := io.ReadAll(cr)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSize := chunkSize * numChunks
	if len(data) != expectedSize {
		t.Errorf("Expected %d bytes, got %d bytes", expectedSize, len(data))
	}
}

// BenchmarkChunkedReader_Small benchmarks small chunked transfers
func BenchmarkChunkedReader_Small(b *testing.B) {
	input := "4\r\nWiki\r\n5\r\npedia\r\n0\r\n\r\n"
	buf := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cr := NewChunkedReader(strings.NewReader(input))
		io.ReadFull(cr, buf[:9])
	}
}

// BenchmarkChunkedReader_Large benchmarks large chunked transfers
func BenchmarkChunkedReader_Large(b *testing.B) {
	// 10KB body in 1KB chunks
	var buf bytes.Buffer
	for i := 0; i < 10; i++ {
		buf.WriteString("400\r\n") // 1024 in hex
		buf.WriteString(strings.Repeat("x", 1024))
		buf.WriteString("\r\n")
	}
	buf.WriteString("0\r\n\r\n")

	input := buf.String()
	readBuf := make([]byte, 10240)

	b.ResetTimer()
	b.SetBytes(10240)

	for i := 0; i < b.N; i++ {
		cr := NewChunkedReader(strings.NewReader(input))
		io.ReadFull(cr, readBuf)
	}
}
