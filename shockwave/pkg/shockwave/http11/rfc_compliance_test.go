package http11

import (
	"fmt"
	"strings"
	"testing"
)

// RFC 7230 Compliance Tests - Message Syntax and Routing

// TestRFC7230_3_1_1_RequestLine tests request line format compliance
// RFC 7230 Section 3.1.1: Request Line
func TestRFC7230_3_1_1_RequestLine(t *testing.T) {
	tests := []struct {
		name    string
		request string
		valid   bool
	}{
		{
			name:    "Valid GET request",
			request: "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			valid:   true,
		},
		{
			name:    "Valid POST with path",
			request: "POST /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n",
			valid:   true,
		},
		{
			name:    "Valid with query string",
			request: "GET /search?q=test HTTP/1.1\r\nHost: example.com\r\n\r\n",
			valid:   true,
		},
		{
			name:    "Valid OPTIONS with asterisk",
			request: "OPTIONS * HTTP/1.1\r\nHost: example.com\r\n\r\n",
			valid:   true,
		},
		{
			name:    "Invalid - no HTTP version",
			request: "GET /\r\nHost: example.com\r\n\r\n",
			valid:   false,
		},
		{
			name:    "Invalid - no path",
			request: "GET HTTP/1.1\r\nHost: example.com\r\n\r\n",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			_, err := parser.Parse(strings.NewReader(tt.request))

			if tt.valid && err != nil {
				t.Errorf("Expected valid request, got error: %v", err)
			}

			if !tt.valid && err == nil {
				t.Error("Expected invalid request, but parsed successfully")
			}
		})
	}
}

// TestRFC7230_3_2_HeaderFields tests header field format compliance
// RFC 7230 Section 3.2: Header Fields
func TestRFC7230_3_2_HeaderFields(t *testing.T) {
	tests := []struct {
		name    string
		request string
		valid   bool
	}{
		{
			name: "Valid headers",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"User-Agent: Test/1.0\r\n" +
				"\r\n",
			valid: true,
		},
		{
			name: "Header with leading whitespace in value (obs-fold)",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"X-Custom:   value\r\n" +
				"\r\n",
			valid: true, // Should be trimmed
		},
		{
			name: "Header with trailing whitespace in value",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"X-Custom: value   \r\n" +
				"\r\n",
			valid: true, // Should be trimmed
		},
		{
			name: "Header with tab whitespace",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"X-Custom:\tvalue\r\n" +
				"\r\n",
			valid: true, // Should be trimmed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			if tt.valid && err != nil {
				t.Errorf("Expected valid request, got error: %v", err)
			}

			if !tt.valid && err == nil {
				t.Error("Expected invalid request, but parsed successfully")
			}

			// Verify whitespace trimming for valid requests
			if tt.valid && err == nil && strings.Contains(tt.name, "whitespace") {
				customHeader := req.GetHeaderString("X-Custom")
				if strings.HasPrefix(customHeader, " ") || strings.HasPrefix(customHeader, "\t") {
					t.Error("Leading whitespace not trimmed from header value")
				}
				if strings.HasSuffix(customHeader, " ") || strings.HasSuffix(customHeader, "\t") {
					t.Error("Trailing whitespace not trimmed from header value")
				}
			}
		})
	}
}

// TestRFC7230_3_3_1_TransferEncoding tests Transfer-Encoding compliance
// RFC 7230 Section 3.3.1: Transfer-Encoding
func TestRFC7230_3_3_1_TransferEncoding(t *testing.T) {
	tests := []struct {
		name     string
		request  string
		isChunked bool
	}{
		{
			name: "Chunked transfer encoding",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Transfer-Encoding: chunked\r\n" +
				"\r\n",
			isChunked: true,
		},
		{
			name: "No transfer encoding",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: 10\r\n" +
				"\r\n",
			isChunked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if req.IsChunked() != tt.isChunked {
				t.Errorf("IsChunked() = %v, want %v", req.IsChunked(), tt.isChunked)
			}
		})
	}
}

// TestRFC7230_3_3_2_ContentLength tests Content-Length compliance
// RFC 7230 Section 3.3.2: Content-Length
func TestRFC7230_3_3_2_ContentLength(t *testing.T) {
	tests := []struct {
		name          string
		request       string
		expectLength  int64
		expectError   bool
	}{
		{
			name: "Valid Content-Length",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: 100\r\n" +
				"\r\n",
			expectLength: 100,
			expectError:  false,
		},
		{
			name: "Zero Content-Length",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: 0\r\n" +
				"\r\n",
			expectLength: 0,
			expectError:  false,
		},
		{
			name: "Invalid Content-Length (non-numeric)",
			request: "POST / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Length: abc\r\n" +
				"\r\n",
			expectLength: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid Content-Length")
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if req.ContentLength != tt.expectLength {
				t.Errorf("ContentLength = %d, want %d", req.ContentLength, tt.expectLength)
			}
		})
	}
}

// TestRFC7230_5_4_Host tests Host header requirement
// RFC 7230 Section 5.4: Host
func TestRFC7230_5_4_Host(t *testing.T) {
	// HTTP/1.1 requires Host header
	requestWithHost := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(requestWithHost))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	host := req.GetHeaderString("Host")
	if host != "example.com" {
		t.Errorf("Host header = %s, want example.com", host)
	}
}

// TestRFC7230_6_1_ConnectionHeader tests Connection header
// RFC 7230 Section 6.1: Connection
func TestRFC7230_6_1_ConnectionHeader(t *testing.T) {
	tests := []struct {
		name        string
		request     string
		expectClose bool
	}{
		{
			name: "Connection: close",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Connection: close\r\n" +
				"\r\n",
			expectClose: true,
		},
		{
			name: "Connection: keep-alive",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Connection: keep-alive\r\n" +
				"\r\n",
			expectClose: false,
		},
		{
			name: "No Connection header (default keep-alive for HTTP/1.1)",
			request: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"\r\n",
			expectClose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(tt.request))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if req.Close != tt.expectClose {
				t.Errorf("Request.Close = %v, want %v", req.Close, tt.expectClose)
			}
		})
	}
}

// TestRFC7231_4_Methods tests HTTP method compliance
// RFC 7231 Section 4: Request Methods
func TestRFC7231_4_Methods(t *testing.T) {
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			request := method + " / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"\r\n"

			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(request))
			if err != nil {
				t.Fatalf("Parse error for %s: %v", method, err)
			}

			if req.Method() != method {
				t.Errorf("Method() = %s, want %s", req.Method(), method)
			}
		})
	}
}

// TestRFC7231_6_StatusCodes tests status code handling
// RFC 7231 Section 6: Response Status Codes
func TestRFC7231_6_StatusCodes(t *testing.T) {
	statusCodes := []struct {
		code int
		text string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{304, "Not Modified"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
	}

	for _, sc := range statusCodes {
		t.Run(sc.text, func(t *testing.T) {
			mockConn := newMockConn("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
			config := DefaultConnectionConfig()

			handler := func(req *Request, rw *ResponseWriter) error {
				rw.WriteHeader(sc.code)
				return nil
			}

			conn := NewConnection(mockConn, config, handler)
			defer conn.Close()

			conn.Serve()

			response := mockConn.GetWritten()
			expectedStatus := fmt.Sprintf("HTTP/1.1 %d", sc.code)
			if !strings.Contains(response, expectedStatus) {
				t.Errorf("Response missing status code: %s", expectedStatus)
			}

			// Also verify status text is present
			if !strings.Contains(response, sc.text) {
				t.Errorf("Response missing status text: %s", sc.text)
			}
		})
	}
}

// TestRFC7232_2_3_ETag tests ETag header handling
// RFC 7232 Section 2.3: ETag
func TestRFC7232_2_3_ETag(t *testing.T) {
	request := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"If-None-Match: \"abc123\"\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(request))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	ifNoneMatch := req.GetHeaderString("If-None-Match")
	if ifNoneMatch != `"abc123"` {
		t.Errorf("If-None-Match = %s, want \"abc123\"", ifNoneMatch)
	}

	// Test response with ETag
	mockConn := newMockConn(request)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		rw.Header().Set([]byte("ETag"), []byte(`"abc123"`))
		rw.WriteHeader(304) // Not Modified
		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	conn.Serve()

	response := mockConn.GetWritten()
	if !strings.Contains(response, `ETag: "abc123"`) {
		t.Error("Response missing ETag header")
	}
}

// TestRFC7233_RangeRequests tests Range request handling
// RFC 7233: Range Requests
func TestRFC7233_RangeRequests(t *testing.T) {
	request := "GET /file.txt HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Range: bytes=0-1023\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(request))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	rangeHeader := req.GetHeaderString("Range")
	if rangeHeader != "bytes=0-1023" {
		t.Errorf("Range header = %s, want bytes=0-1023", rangeHeader)
	}
}

// TestRFC7234_CacheControl tests Cache-Control header
// RFC 7234: Caching
func TestRFC7234_CacheControl(t *testing.T) {
	request := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Cache-Control: no-cache\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(request))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	cacheControl := req.GetHeaderString("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("Cache-Control = %s, want no-cache", cacheControl)
	}

	// Test response with Cache-Control
	mockConn := newMockConn(request)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		rw.Header().Set([]byte("Cache-Control"), []byte("max-age=3600"))
		return rw.WriteText(200, []byte("OK"))
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	conn.Serve()

	response := mockConn.GetWritten()
	if !strings.Contains(response, "Cache-Control: max-age=3600") {
		t.Error("Response missing Cache-Control header")
	}
}

// TestRFC7235_Authentication tests authentication headers
// RFC 7235: Authentication
func TestRFC7235_Authentication(t *testing.T) {
	request := "GET /protected HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Authorization: Bearer token123\r\n" +
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(request))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	auth := req.GetHeaderString("Authorization")
	if auth != "Bearer token123" {
		t.Errorf("Authorization = %s, want Bearer token123", auth)
	}

	// Test 401 response with WWW-Authenticate
	mockConn := newMockConn("GET /protected HTTP/1.1\r\nHost: example.com\r\n\r\n")
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		rw.Header().Set([]byte("WWW-Authenticate"), []byte("Bearer realm=\"api\""))
		return rw.WriteError(401, "Unauthorized")
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	conn.Serve()

	response := mockConn.GetWritten()
	if !strings.Contains(response, "WWW-Authenticate: Bearer realm=") {
		t.Error("Response missing WWW-Authenticate header")
	}
}

// TestRFC_CaseInsensitiveHeaders tests case-insensitive header names
// RFC 7230 Section 3.2: Header field names are case-insensitive
func TestRFC_CaseInsensitiveHeaders(t *testing.T) {
	request := "GET / HTTP/1.1\r\n" +
		"host: example.com\r\n" +  // lowercase
		"Content-Type: text/html\r\n" +  // mixed case
		"ACCEPT: */*\r\n" +  // uppercase
		"\r\n"

	parser := NewParser()
	req, err := parser.Parse(strings.NewReader(request))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// All these should work
	tests := []struct {
		header string
		value  string
	}{
		{"Host", "example.com"},
		{"host", "example.com"},
		{"HOST", "example.com"},
		{"Content-Type", "text/html"},
		{"content-type", "text/html"},
		{"CONTENT-TYPE", "text/html"},
		{"Accept", "*/*"},
		{"accept", "*/*"},
		{"ACCEPT", "*/*"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			value := req.GetHeaderString(tt.header)
			if value != tt.value {
				t.Errorf("GetHeaderString(%s) = %s, want %s", tt.header, value, tt.value)
			}
		})
	}
}

// TestRFC_URIEncoding tests URI encoding handling
func TestRFC_URIEncoding(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "Simple path",
			uri:      "/api/users",
			expected: "/api/users",
		},
		{
			name:     "Path with query",
			uri:      "/search?q=golang&page=1",
			expected: "/search",
		},
		{
			name:     "Path with encoded spaces",
			uri:      "/path%20with%20spaces",
			expected: "/path%20with%20spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := "GET " + tt.uri + " HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"\r\n"

			parser := NewParser()
			req, err := parser.Parse(strings.NewReader(request))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if req.Path() != tt.expected {
				t.Errorf("Path() = %s, want %s", req.Path(), tt.expected)
			}
		})
	}
}
