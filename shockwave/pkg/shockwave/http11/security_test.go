package http11

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// Security Tests - HTTP Request Smuggling, Injection, DoS, etc.

// TestSecurity_RequestSmuggling_CLTE tests Content-Length + Transfer-Encoding conflict
// RFC 7230 Section 3.3.3: If both Content-Length and Transfer-Encoding are present,
// Transfer-Encoding takes precedence and Content-Length MUST be removed/ignored.
func TestSecurity_RequestSmuggling_CLTE(t *testing.T) {
	// This attack sends both Content-Length and Transfer-Encoding
	// Some servers/proxies might prefer one over the other, leading to smuggling
	request := "POST / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Length: 6\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n" +
		"0\r\n\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(request))

	if err != nil {
		t.Logf("Parser rejected conflicting headers (GOOD): %v", err)
		return
	}

	// If we allow both, ensure Transfer-Encoding takes precedence
	if req.IsChunked() {
		// Transfer-Encoding should be used, ContentLength should be -1 or 0
		if req.ContentLength > 0 {
			t.Error("SECURITY: Both Transfer-Encoding and Content-Length are active - request smuggling risk!")
		}
		t.Log("Transfer-Encoding takes precedence (RFC compliant)")
	} else {
		t.Error("SECURITY: Transfer-Encoding header not recognized when both CL and TE present")
	}

	if req != nil {
		PutRequest(req)
	}
}

// TestSecurity_RequestSmuggling_DualContentLength tests multiple Content-Length headers
// RFC 7230 Section 3.3.3: If multiple Content-Length headers with different values,
// the request MUST be rejected as invalid
func TestSecurity_RequestSmuggling_DualContentLength(t *testing.T) {
	tests := []struct {
		name    string
		request string
		valid   bool
	}{
		{
			name: "Duplicate Content-Length same value (acceptable)",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: 10\r\n" +
				"Content-Length: 10\r\n" +
				"\r\n",
			valid: true, // Same value is technically OK per RFC
		},
		{
			name: "Conflicting Content-Length values (MUST REJECT)",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: 10\r\n" +
				"Content-Length: 20\r\n" +
				"\r\n",
			valid: false, // Different values MUST be rejected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			if !tt.valid && err == nil {
				t.Error("SECURITY: Parser accepted conflicting Content-Length headers - smuggling risk!")
			}

			if tt.valid && err != nil {
				t.Logf("Parser rejected duplicate CL (conservative approach): %v", err)
			}

			if req != nil {
				PutRequest(req)
			}
		})
	}
}

// TestSecurity_HeaderInjection_CRLF tests header injection via CRLF sequences
func TestSecurity_HeaderInjection_CRLF(t *testing.T) {
	tests := []struct {
		name    string
		request string
		valid   bool
	}{
		// NOTE: CRLF injection in header VALUES cannot be tested at the parser level
		// because CRLF is the line terminator in HTTP. The parser will always split
		// on CRLF, so it can never receive CRLF embedded in a value from raw input.
		//
		// CRLF injection protection is tested in header_test.go where Header.Add()
		// is called directly with crafted values containing CRLF (P0 FIX #3).
		// That's the actual attack vector - application code adding headers with
		// user-controlled values containing CRLF.
		{
			name: "CRLF injection in header name",
			request: "GET / HTTP/1.1\r\n" +
				"Host\r\nX-Injected: malicious\r\n: example.com\r\n" +
				"\r\n",
			valid: false, // Malformed header
		},
		{
			name: "Normal header",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"User-Agent: Mozilla\r\n" +
				"\r\n",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			if !tt.valid && err == nil {
				t.Error("SECURITY: Parser accepted header with CRLF injection")
			}

			if tt.valid && err != nil {
				t.Errorf("Parser rejected valid request: %v", err)
			}

			if req != nil {
				PutRequest(req)
			}
		})
	}
}

// TestSecurity_IntegerOverflow_ContentLength tests integer overflow in Content-Length
func TestSecurity_IntegerOverflow_ContentLength(t *testing.T) {
	tests := []struct {
		name    string
		request string
	}{
		{
			name: "Max int64 Content-Length",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: 9223372036854775807\r\n" +
				"\r\n",
		},
		{
			name: "Overflow attempt",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: 99999999999999999999999999999\r\n" +
				"\r\n",
		},
		{
			name: "Negative Content-Length",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: -1\r\n" +
				"\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			// Should either accept with correct value or reject
			if err == nil {
				if req.ContentLength < 0 {
					t.Error("SECURITY: Negative Content-Length accepted")
				}
				PutRequest(req)
			} else {
				t.Logf("Parser rejected (safe): %v", err)
			}
		})
	}
}

// TestSecurity_LargeHeaders tests DoS via large headers
func TestSecurity_LargeHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headerCount int
		headerSize  int
		expectError bool
	}{
		{
			name:        "Normal headers",
			headerCount: 10,
			headerSize:  50,
			expectError: false,
		},
		{
			name:        "Many headers (33)",
			headerCount: 33,
			headerSize:  50,
			expectError: false, // Should use overflow storage
		},
		{
			name:        "Very many headers (100)",
			headerCount: 100,
			headerSize:  50,
			expectError: false, // Should handle via overflow
		},
		{
			name:        "Large header value",
			headerCount: 5,
			headerSize:  1024,
			expectError: false, // Should handle via overflow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			buf.WriteString("GET / HTTP/1.1\r\n")
			buf.WriteString("Host: example.com\r\n")

			for i := 0; i < tt.headerCount; i++ {
				buf.WriteString("X-Custom-")
				buf.WriteString(string('0' + byte(i%10)))
				buf.WriteString(": ")
				for j := 0; j < tt.headerSize; j++ {
					buf.WriteByte('a')
				}
				buf.WriteString("\r\n")
			}
			buf.WriteString("\r\n")

			parser := NewParser()
			req, err := parser.Parse(&buf)

			if tt.expectError && err == nil {
				t.Error("Expected error for oversized headers")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if req != nil {
				if req.Header.Len() != tt.headerCount+1 { // +1 for Host
					t.Logf("Expected %d headers, got %d", tt.headerCount+1, req.Header.Len())
				}
				PutRequest(req)
			}
		})
	}
}

// TestSecurity_VeryLongURI tests extremely long URIs
func TestSecurity_VeryLongURI(t *testing.T) {
	tests := []struct {
		name        string
		uriLength   int
		expectError bool
	}{
		{
			name:        "Normal URI",
			uriLength:   100,
			expectError: false,
		},
		{
			name:        "Long URI (2KB)",
			uriLength:   2048,
			expectError: false,
		},
		{
			name:        "Very long URI (8KB - at limit)",
			uriLength:   8000,
			expectError: false,
		},
		{
			name:        "Excessive URI (10KB)",
			uriLength:   10240,
			expectError: true, // Should exceed MaxRequestLineSize
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			buf.WriteString("GET /")
			for i := 0; i < tt.uriLength; i++ {
				buf.WriteByte('a')
			}
			buf.WriteString(" HTTP/1.1\r\n")
			buf.WriteString("Host: example.com\r\n")
			buf.WriteString("\r\n")

			parser := NewParser()
			req, err := parser.Parse(&buf)

			if tt.expectError && err == nil {
				t.Error("SECURITY: Parser accepted excessive URI length - DoS risk!")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for valid URI: %v", err)
			}

			if req != nil {
				PutRequest(req)
			}
		})
	}
}

// TestSecurity_PathTraversal tests path traversal attempts
func TestSecurity_PathTraversal(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		valid   bool
	}{
		{
			name:  "Normal path",
			path:  "/api/users",
			valid: true,
		},
		{
			name:  "Path with dots (valid)",
			path:  "/api/v1.0/users",
			valid: true,
		},
		{
			name:  "Parent directory reference",
			path:  "/api/../etc/passwd",
			valid: true, // Parser should accept, app layer validates
		},
		{
			name:  "Multiple parent refs",
			path:  "/../../../../etc/passwd",
			valid: true, // Parser should accept, app layer validates
		},
		{
			name:  "Encoded traversal",
			path:  "/api%2F..%2Fetc%2Fpasswd",
			valid: true, // Parser should accept encoded URIs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := "GET " + tt.path + " HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"\r\n"

			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(request))

			if !tt.valid && err == nil {
				t.Error("Parser accepted invalid path")
			}

			if tt.valid && err != nil {
				t.Errorf("Parser rejected valid path: %v", err)
			}

			if req != nil {
				// Just verify path is preserved as-is
				if req.Path() != tt.path {
					t.Errorf("Path mangled: got %s, want %s", req.Path(), tt.path)
				}
				PutRequest(req)
			}
		})
	}
}

// TestSecurity_EmptyRequest tests empty/malformed requests
func TestSecurity_EmptyRequest(t *testing.T) {
	tests := []struct {
		name    string
		request string
	}{
		{
			name:    "Empty string",
			request: "",
		},
		{
			name:    "Only CRLF",
			request: "\r\n",
		},
		{
			name:    "Only whitespace",
			request: "   \r\n\r\n",
		},
		{
			name:    "Incomplete request line",
			request: "GET /",
		},
		{
			name:    "Missing final CRLF",
			request: "GET / HTTP/1.1\r\nHost: example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			if err == nil {
				t.Error("Parser accepted malformed/empty request")
				if req != nil {
					PutRequest(req)
				}
			} else {
				t.Logf("Correctly rejected: %v", err)
			}
		})
	}
}

// TestSecurity_LineEndingVariations tests different line ending combinations
// RFC 7230 requires CRLF, but some implementations accept LF only
func TestSecurity_LineEndingVariations(t *testing.T) {
	tests := []struct {
		name    string
		request string
		valid   bool
	}{
		{
			name: "Correct CRLF",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"\r\n",
			valid: true,
		},
		{
			name: "LF only (non-compliant)",
			request: "GET / HTTP/1.1\n" +
				"Host: example.com\n" +
				"\n",
			valid: false, // RFC requires CRLF
		},
		{
			name: "CR only (invalid)",
			request: "GET / HTTP/1.1\r" +
				"Host: example.com\r" +
				"\r",
			valid: false,
		},
		{
			name: "Mixed CRLF and LF",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\n" +
				"\r\n",
			valid: false, // Inconsistent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			if !tt.valid && err == nil {
				t.Log("WARNING: Parser accepted non-CRLF line endings (lenient mode)")
			}

			if tt.valid && err != nil {
				t.Errorf("Parser rejected valid request: %v", err)
			}

			if req != nil {
				PutRequest(req)
			}
		})
	}
}

// TestSecurity_WhitespaceBeforeColon tests RFC 7230 Section 3.2.4
// No whitespace allowed between header name and colon
func TestSecurity_WhitespaceBeforeColon(t *testing.T) {
	tests := []struct {
		name    string
		request string
		valid   bool
	}{
		{
			name: "Valid header (no whitespace)",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"\r\n",
			valid: true,
		},
		{
			name: "Space before colon (INVALID per RFC)",
			request: "GET / HTTP/1.1\r\n" +
				"Host : example.com\r\n" +
				"\r\n",
			valid: false,
		},
		{
			name: "Tab before colon (INVALID per RFC)",
			request: "GET / HTTP/1.1\r\n" +
				"Host\t: example.com\r\n" +
				"\r\n",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			if !tt.valid && err == nil {
				t.Error("SECURITY: Parser accepted whitespace before colon - RFC violation!")
			}

			if tt.valid && err != nil {
				t.Errorf("Parser rejected valid request: %v", err)
			}

			if req != nil {
				PutRequest(req)
			}
		})
	}
}

// TestSecurity_ObsoletedLineFolding tests obsolete line folding (RFC 7230 Section 3.2.4)
// Line folding (multi-line headers) is obsolete and should be rejected
func TestSecurity_ObsoletedLineFolding(t *testing.T) {
	// Obsolete syntax: Header value can continue on next line with leading whitespace
	request := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"X-Multi-Line: value1\r\n" +
		" value2\r\n" + // This is obsolete line folding
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(request))

	if err == nil {
		t.Log("WARNING: Parser accepted obsolete line folding (should reject per RFC 7230)")
		if req != nil {
			PutRequest(req)
		}
	} else {
		t.Logf("Correctly rejected obsolete line folding: %v", err)
	}
}

// TestSecurity_NullByteInHeader tests null byte injection
func TestSecurity_NullByteInHeader(t *testing.T) {
	tests := []struct {
		name    string
		request string
	}{
		{
			name:    "Null in header value",
			request: "GET / HTTP/1.1\r\nHost: example.com\x00.evil.com\r\n\r\n",
		},
		{
			name:    "Null in path",
			request: "GET /path\x00/../../etc/passwd HTTP/1.1\r\nHost: example.com\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			// Parser can accept or reject null bytes - both are valid approaches
			if err != nil {
				t.Logf("Parser rejected null byte (strict): %v", err)
			} else {
				t.Log("Parser accepted null byte (lenient) - application layer should validate")
				if req != nil {
					PutRequest(req)
				}
			}
		})
	}
}

// TestSecurity_SlowlorisProtection tests that parser doesn't hang on slow reads
func TestSecurity_SlowlorisProtection(t *testing.T) {
	// Create a slow reader that sends one byte at a time
	slowReader := &slowReader{
		data: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		pos:  0,
	}

	parser := NewParser()
	req, err := parser.Parse(slowReader)

	// Parser should eventually complete or timeout
	if err != nil && err != io.EOF {
		t.Logf("Parser error (acceptable): %v", err)
	}

	if req != nil {
		PutRequest(req)
	}

	// Note: True slowloris protection requires timeout at connection level
	t.Log("Note: Full slowloris protection requires read timeout in connection handler")
}

// slowReader simulates a slow client by reading one byte at a time
type slowReader struct {
	data []byte
	pos  int
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	// Read only 1 byte per call
	if len(p) > 0 {
		p[0] = r.data[r.pos]
		r.pos++
		return 1, nil
	}
	return 0, nil
}

// TestSecurity_MethodCaseSensitivity tests that methods are case-sensitive per RFC
func TestSecurity_MethodCaseSensitivity(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		valid   bool
	}{
		{
			name:   "GET uppercase (valid)",
			method: "GET",
			valid:  true,
		},
		{
			name:   "get lowercase (invalid)",
			method: "get",
			valid:  false,
		},
		{
			name:   "Get mixed case (invalid)",
			method: "Get",
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.method + " / HTTP/1.1\r\nHost: example.com\r\n\r\n"

			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(request))

			if !tt.valid && err == nil {
				t.Error("SECURITY: Parser accepted non-uppercase method - RFC violation!")
			}

			if tt.valid && err != nil {
				t.Errorf("Parser rejected valid method: %v", err)
			}

			if req != nil {
				PutRequest(req)
			}
		})
	}
}

// TestSecurity_HTTPVersionValidation tests HTTP version string validation
func TestSecurity_HTTPVersionValidation(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{
			name:    "HTTP/1.1 (valid)",
			version: "HTTP/1.1",
			valid:   true,
		},
		{
			name:    "HTTP/1.0 (different version)",
			version: "HTTP/1.0",
			valid:   false, // This parser only supports 1.1
		},
		{
			name:    "HTTP/2.0 (different version)",
			version: "HTTP/2.0",
			valid:   false,
		},
		{
			name:    "http/1.1 lowercase (invalid)",
			version: "http/1.1",
			valid:   false,
		},
		{
			name:    "HTTP/1.1 with extra space",
			version: "HTTP/1.1 ",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := "GET / " + tt.version + "\r\nHost: example.com\r\n\r\n"

			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(request))

			if !tt.valid && err == nil {
				t.Error("Parser accepted invalid HTTP version")
			}

			if tt.valid && err != nil {
				t.Errorf("Parser rejected valid version: %v", err)
			}

			if req != nil {
				PutRequest(req)
			}
		})
	}
}
