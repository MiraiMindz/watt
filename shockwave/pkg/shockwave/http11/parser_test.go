package http11

import (
	"bytes"
	"strings"
	"testing"
)

// Test valid requests

func TestParseSimpleGET(t *testing.T) {
	input := "GET / HTTP/1.1\r\n\r\n"
	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.MethodID != MethodGET {
		t.Errorf("MethodID = %d, want %d", req.MethodID, MethodGET)
	}
	if string(req.pathBytes) != "/" {
		t.Errorf("pathBytes = %q, want %q", req.pathBytes, "/")
	}
	if req.queryBytes != nil {
		t.Errorf("queryBytes = %q, want nil", req.queryBytes)
	}
	if req.Proto != "HTTP/1.1" {
		t.Errorf("Proto = %q, want %q", req.Proto, "HTTP/1.1")
	}
}

func TestParseGETWithPath(t *testing.T) {
	input := "GET /api/users HTTP/1.1\r\n\r\n"
	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.MethodID != MethodGET {
		t.Errorf("MethodID = %d, want %d", req.MethodID, MethodGET)
	}
	if string(req.pathBytes) != "/api/users" {
		t.Errorf("pathBytes = %q, want %q", req.pathBytes, "/api/users")
	}
}

func TestParseGETWithQuery(t *testing.T) {
	input := "GET /search?q=test&limit=10 HTTP/1.1\r\n\r\n"
	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if string(req.pathBytes) != "/search" {
		t.Errorf("pathBytes = %q, want %q", req.pathBytes, "/search")
	}
	if string(req.queryBytes) != "q=test&limit=10" {
		t.Errorf("queryBytes = %q, want %q", req.queryBytes, "q=test&limit=10")
	}
}

func TestParsePOST(t *testing.T) {
	input := "POST /api/users HTTP/1.1\r\n\r\n"
	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.MethodID != MethodPOST {
		t.Errorf("MethodID = %d, want %d", req.MethodID, MethodPOST)
	}
}

func TestParseAllMethods(t *testing.T) {
	methods := []struct {
		name     string
		methodID uint8
	}{
		{"GET", MethodGET},
		{"POST", MethodPOST},
		{"PUT", MethodPUT},
		{"DELETE", MethodDELETE},
		{"PATCH", MethodPATCH},
		{"HEAD", MethodHEAD},
		{"OPTIONS", MethodOPTIONS},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			input := m.name + " / HTTP/1.1\r\n\r\n"
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if req.MethodID != m.methodID {
				t.Errorf("MethodID = %d, want %d", req.MethodID, m.methodID)
			}
		})
	}
}

func TestParseWithSingleHeader(t *testing.T) {
	input := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.Header.Len() != 1 {
		t.Errorf("Header.Len() = %d, want 1", req.Header.Len())
	}

	host := req.GetHeaderString("Host")
	if host != "example.com" {
		t.Errorf("Host header = %q, want %q", host, "example.com")
	}
}

func TestParseWithMultipleHeaders(t *testing.T) {
	input := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: test-client\r\n" +
		"Accept: */*\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.Header.Len() != 3 {
		t.Errorf("Header.Len() = %d, want 3", req.Header.Len())
	}

	expectedHeaders := map[string]string{
		"Host":       "example.com",
		"User-Agent": "test-client",
		"Accept":     "*/*",
	}

	for name, expectedValue := range expectedHeaders {
		value := req.GetHeaderString(name)
		if value != expectedValue {
			t.Errorf("%s header = %q, want %q", name, value, expectedValue)
		}
	}
}

func TestParseHeaderWithSpaces(t *testing.T) {
	// RFC 7230 allows optional whitespace after colon
	input := "GET / HTTP/1.1\r\n" +
		"Host:   example.com  \r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	host := req.GetHeaderString("Host")
	if host != "example.com" {
		t.Errorf("Host header = %q, want %q (whitespace not trimmed)", host, "example.com")
	}
}

func TestParseContentLength(t *testing.T) {
	input := "POST / HTTP/1.1\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n" +
		"Hello, World!"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.ContentLength != 13 {
		t.Errorf("ContentLength = %d, want 13", req.ContentLength)
	}

	if !req.HasBody() {
		t.Error("HasBody() = false, want true")
	}
}

func TestParseTransferEncodingChunked(t *testing.T) {
	input := "POST / HTTP/1.1\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !req.IsChunked() {
		t.Error("IsChunked() = false, want true")
	}

	if !req.HasBody() {
		t.Error("HasBody() = false, want true")
	}
}

func TestParseConnectionClose(t *testing.T) {
	input := "GET / HTTP/1.1\r\n" +
		"Connection: close\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !req.Close {
		t.Error("Close = false, want true")
	}
}

func TestParse32Headers(t *testing.T) {
	var input strings.Builder
	input.WriteString("GET / HTTP/1.1\r\n")

	// Add exactly 32 headers (max inline)
	for i := 0; i < 32; i++ {
		input.WriteString("X-Header-")
		input.WriteString(string(rune('0' + i%10)))
		input.WriteString(": value")
		input.WriteString(string(rune('0' + i%10)))
		input.WriteString("\r\n")
	}
	input.WriteString("\r\n")

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input.String()))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.Header.Len() != 32 {
		t.Errorf("Header.Len() = %d, want 32", req.Header.Len())
	}
}

// Test malformed requests

func TestParseInvalidMethod(t *testing.T) {
	input := "INVALID / HTTP/1.1\r\n\r\n"
	parser := NewParser()
	_, err := parser.Parse(strings.NewReader(input))
	if err != ErrInvalidMethod {
		t.Errorf("Parse error = %v, want %v", err, ErrInvalidMethod)
	}
}

func TestParseMissingCRLF(t *testing.T) {
	input := "GET / HTTP/1.1\n\n" // Should be \r\n
	parser := NewParser()
	_, err := parser.Parse(strings.NewReader(input))
	if err == nil {
		t.Error("Parse should fail with missing CRLF, but succeeded")
	}
}

func TestParseInvalidRequestLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Missing path", "GET HTTP/1.1\r\n\r\n"},
		{"Missing protocol", "GET /\r\n\r\n"},
		{"Empty", "\r\n\r\n"},
		{"Only method", "GET\r\n\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			_, err := parser.Parse(strings.NewReader(tt.input))
			if err == nil {
				t.Errorf("Parse should fail for %q, but succeeded", tt.input)
			}
		})
	}
}

func TestParseInvalidPath(t *testing.T) {
	// Path must start with / or *
	input := "GET invalid HTTP/1.1\r\n\r\n"
	parser := NewParser()
	_, err := parser.Parse(strings.NewReader(input))
	if err != ErrInvalidPath {
		t.Errorf("Parse error = %v, want %v", err, ErrInvalidPath)
	}
}

func TestParseInvalidProtocol(t *testing.T) {
	input := "GET / HTTP/2.0\r\n\r\n"
	parser := NewParser()
	_, err := parser.Parse(strings.NewReader(input))
	if err != ErrInvalidProtocol {
		t.Errorf("Parse error = %v, want %v", err, ErrInvalidProtocol)
	}
}

func TestParseInvalidHeader(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Missing colon", "GET / HTTP/1.1\r\nInvalidHeader\r\n\r\n"},
		{"Space in name", "GET / HTTP/1.1\r\nInvalid Name: value\r\n\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			_, err := parser.Parse(strings.NewReader(tt.input))
			if err != ErrInvalidHeader {
				t.Errorf("Parse error = %v, want %v", err, ErrInvalidHeader)
			}
		})
	}
}

func TestParseInvalidContentLength(t *testing.T) {
	input := "POST / HTTP/1.1\r\n" +
		"Content-Length: invalid\r\n" +
		"\r\n"

	parser := NewParser()
	_, err := parser.Parse(strings.NewReader(input))
	if err != ErrInvalidContentLength {
		t.Errorf("Parse error = %v, want %v", err, ErrInvalidContentLength)
	}
}

// RFC 7230 compliance tests

func TestRFC7230_RequestLine(t *testing.T) {
	// RFC 7230 Section 3.1.1: Request Line
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"Valid GET", "GET /index.html HTTP/1.1\r\n\r\n", true},
		{"Valid with query", "GET /search?q=test HTTP/1.1\r\n\r\n", true},
		{"Valid asterisk", "OPTIONS * HTTP/1.1\r\n\r\n", true},
		{"Valid absolute path", "GET /pub/WWW/TheProject.html HTTP/1.1\r\n\r\n", true},
		{"Invalid - no leading slash", "GET index.html HTTP/1.1\r\n\r\n", false},
		{"Invalid - wrong protocol", "GET / HTTP/2.0\r\n\r\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			_, err := parser.Parse(strings.NewReader(tt.input))

			if tt.valid && err != nil {
				t.Errorf("Parse failed for valid input: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Parse succeeded for invalid input")
			}
		})
	}
}

func TestRFC7230_HeaderFields(t *testing.T) {
	// RFC 7230 Section 3.2: Header Fields
	tests := []struct {
		name          string
		input         string
		valid         bool
		expectedValue string // For valid cases
	}{
		{
			"Valid simple header",
			"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			true,
			"example.com",
		},
		{
			"Valid with OWS",
			"GET / HTTP/1.1\r\nHost:  example.com  \r\n\r\n",
			true,
			"example.com",
		},
		{
			"Valid multiline simulation",
			"GET / HTTP/1.1\r\nHost: example.com\r\nAccept: */*\r\n\r\n",
			true,
			"example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.input))

			if tt.valid && err != nil {
				t.Errorf("Parse failed for valid input: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Parse succeeded for invalid input")
			}

			if tt.valid && req != nil && tt.expectedValue != "" {
				host := req.GetHeaderString("Host")
				if host != tt.expectedValue {
					t.Errorf("Host header = %q, want %q", host, tt.expectedValue)
				}
			}
		})
	}
}

func TestRFC7230_CaseInsensitiveHeaders(t *testing.T) {
	// RFC 7230: Header field names are case-insensitive
	input := "GET / HTTP/1.1\r\n" +
		"content-type: application/json\r\n" +
		"Content-Length: 0\r\n" +
		"ACCEPT: */*\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should be accessible with any case
	tests := [][]string{
		{"Content-Type", "content-type", "CONTENT-TYPE"},
		{"Content-Length", "content-length", "CONTENT-LENGTH"},
		{"Accept", "accept", "ACCEPT"},
	}

	for _, variations := range tests {
		for _, name := range variations {
			val := req.GetHeaderString(name)
			if val == "" {
				t.Errorf("Header %q not found (case-insensitive lookup failed)", name)
			}
		}
	}
}

// Benchmarks

func BenchmarkParseSimpleGET(b *testing.B) {
	input := []byte("GET / HTTP/1.1\r\n\r\n")

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))

	for i := 0; i < b.N; i++ {
		parser := NewParser()
		_, err := parser.Parse(bytes.NewReader(input))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseGETWithHeaders(b *testing.B) {
	input := []byte("GET /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: benchmark\r\n" +
		"Accept: */*\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n")

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))

	for i := 0; i < b.N; i++ {
		parser := NewParser()
		_, err := parser.Parse(bytes.NewReader(input))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePOSTWithBody(b *testing.B) {
	input := []byte("POST /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Length: 13\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Hello, World!")

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))

	for i := 0; i < b.N; i++ {
		parser := NewParser()
		_, err := parser.Parse(bytes.NewReader(input))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse16Headers(b *testing.B) {
	var input bytes.Buffer
	input.WriteString("GET / HTTP/1.1\r\n")
	for i := 0; i < 16; i++ {
		input.WriteString("X-Header-")
		input.WriteByte(byte('0' + i%10))
		input.WriteString(": value")
		input.WriteByte(byte('0' + i%10))
		input.WriteString("\r\n")
	}
	input.WriteString("\r\n")

	inputBytes := input.Bytes()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(inputBytes)))

	for i := 0; i < b.N; i++ {
		parser := NewParser()
		_, err := parser.Parse(bytes.NewReader(inputBytes))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse32Headers(b *testing.B) {
	var input bytes.Buffer
	input.WriteString("GET / HTTP/1.1\r\n")
	for i := 0; i < 32; i++ {
		input.WriteString("X-Header-")
		input.WriteByte(byte('0' + i%10))
		input.WriteString(": value")
		input.WriteByte(byte('0' + i%10))
		input.WriteString("\r\n")
	}
	input.WriteString("\r\n")

	inputBytes := input.Bytes()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(inputBytes)))

	for i := 0; i < b.N; i++ {
		parser := NewParser()
		_, err := parser.Parse(bytes.NewReader(inputBytes))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Additional tests for 100% coverage

func TestParserParseContentLengthOverflow(t *testing.T) {
	parser := NewParser()

	// Content-Length that would overflow int64
	input := "POST / HTTP/1.1\r\n" +
		"Content-Length: 99999999999999999999999999999\r\n" +
		"\r\n"

	_, err := parser.Parse(strings.NewReader(input))
	if err == nil {
		t.Error("Expected error for Content-Length overflow")
	}
}

func TestParserChunkedTransferEncoding(t *testing.T) {
	parser := NewParser()

	input := "POST / HTTP/1.1\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n"

	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !req.IsChunked() {
		t.Error("Request should be chunked")
	}

	if len(req.TransferEncoding) == 0 {
		t.Error("TransferEncoding should not be empty")
	}
}

func TestParserConnectionClose(t *testing.T) {
	parser := NewParser()

	input := "GET / HTTP/1.1\r\n" +
		"Connection: close\r\n" +
		"\r\n"

	req, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !req.Close {
		t.Error("Request.Close should be true")
	}
}
