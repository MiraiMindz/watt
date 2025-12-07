package client

import (
	"bufio"
	"io"
	"strconv"
)

// ClientResponse represents an optimized HTTP client response with zero-allocation parsing.
//
// Design principles:
// - Pooled via sync.Pool for reuse
// - Inline header storage for ≤32 headers
// - Zero-copy byte slices for status line components
// - Minimal allocations for common cases
type ClientResponse struct {
	// Status line components (inline storage)
	protoBytes [16]byte // "HTTP/1.1"
	protoLen   uint8
	statusCode int          // 200, 404, etc.
	statusBytes [64]byte     // "OK", "Not Found", etc.
	statusLen  uint8

	// Cached string conversions (lazy-initialized to avoid allocations)
	protoString  string
	statusString string

	// Headers (inline storage)
	headers *ClientHeaders

	// Body reader
	body io.ReadCloser

	// Content information
	contentLength    int64 // -1 if unknown
	isChunked        bool  // true if Transfer-Encoding: chunked

	// Connection info
	close bool // true if connection should be closed after this response

	// Reference to the connection (for returning to pool)
	conn *PooledConn

	// Buffer for parsing (reused)
	buf []byte
}

// Reset clears the response for reuse.
//
// Allocation behavior: 0 allocs/op
func (r *ClientResponse) Reset() {
	r.protoLen = 0
	r.statusCode = 0
	r.statusLen = 0
	r.protoString = ""
	r.statusString = ""

	if r.headers != nil {
		PutHeaders(r.headers)
		r.headers = nil
	}

	r.body = nil
	r.contentLength = -1
	r.isChunked = false
	r.close = false
	r.conn = nil

	// Reset buffer length but keep capacity
	if r.buf != nil {
		r.buf = r.buf[:0]
	}
}

// ParseStatusLine parses the HTTP status line.
// Format: "HTTP/1.1 200 OK\r\n"
//
// Allocation behavior: 0 allocs/op
func (r *ClientResponse) ParseStatusLine(line []byte) error {
	// Find first space (after protocol)
	spaceIdx1 := -1
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' {
			spaceIdx1 = i
			break
		}
	}
	if spaceIdx1 == -1 {
		return ErrInvalidURL // Reuse error
	}

	// Copy protocol
	if spaceIdx1 <= cap(r.protoBytes) {
		copy(r.protoBytes[:], line[:spaceIdx1])
		r.protoLen = uint8(spaceIdx1)
	}

	// Find second space (after status code)
	spaceIdx2 := -1
	for i := spaceIdx1 + 1; i < len(line); i++ {
		if line[i] == ' ' {
			spaceIdx2 = i
			break
		}
	}

	// Parse status code
	var statusCodeEnd int
	if spaceIdx2 == -1 {
		// No status text (e.g., "HTTP/1.1 200\r\n")
		statusCodeEnd = len(line)
		// Trim CRLF
		if statusCodeEnd >= 2 && line[statusCodeEnd-2] == '\r' {
			statusCodeEnd -= 2
		} else if statusCodeEnd >= 1 && line[statusCodeEnd-1] == '\n' {
			statusCodeEnd -= 1
		}
	} else {
		statusCodeEnd = spaceIdx2
	}

	statusCodeBytes := line[spaceIdx1+1 : statusCodeEnd]
	code, err := parseIntFast(statusCodeBytes)
	if err != nil {
		return err
	}
	r.statusCode = code

	// Copy status text (if present)
	if spaceIdx2 != -1 && spaceIdx2+1 < len(line) {
		statusText := line[spaceIdx2+1:]
		// Trim CRLF
		if len(statusText) >= 2 && statusText[len(statusText)-2] == '\r' {
			statusText = statusText[:len(statusText)-2]
		} else if len(statusText) >= 1 && statusText[len(statusText)-1] == '\n' {
			statusText = statusText[:len(statusText)-1]
		}

		if len(statusText) <= cap(r.statusBytes) {
			copy(r.statusBytes[:], statusText)
			r.statusLen = uint8(len(statusText))
		}
	}

	return nil
}

// ParseHeader parses a single header line and adds it to the headers.
// Format: "Name: Value\r\n"
//
// Allocation behavior: 0 allocs/op for ≤32 headers
func (r *ClientResponse) ParseHeader(line []byte) error {
	// Find colon
	colonIdx := -1
	for i := 0; i < len(line); i++ {
		if line[i] == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx == -1 {
		return nil // Skip invalid header
	}

	name := line[:colonIdx]
	value := line[colonIdx+1:]

	// Trim leading space from value
	for len(value) > 0 && value[0] == ' ' {
		value = value[1:]
	}

	// Trim CRLF from value
	if len(value) >= 2 && value[len(value)-2] == '\r' {
		value = value[:len(value)-2]
	} else if len(value) >= 1 && value[len(value)-1] == '\n' {
		value = value[:len(value)-1]
	}

	// Lazy initialize headers
	if r.headers == nil {
		r.headers = GetHeaders()
	}
	r.headers.Add(name, value)

	return nil
}

// ParseResponse parses an HTTP response from a bufio.Reader.
// This is the main parsing entry point.
//
// Allocation behavior: ~2-4 allocs/op (mainly from bufio.ReadLine)
func (r *ClientResponse) ParseResponse(br *bufio.Reader) error {
	// Read status line
	statusLine, err := br.ReadBytes('\n')
	if err != nil {
		return err
	}

	if err := r.ParseStatusLine(statusLine); err != nil {
		return err
	}

	// Read headers
	for {
		line, err := br.ReadBytes('\n')
		if err != nil {
			return err
		}

		// Empty line marks end of headers
		if len(line) <= 2 {
			break
		}

		if err := r.ParseHeader(line); err != nil {
			return err
		}
	}

	// Process special headers
	r.processHeaders()

	return nil
}

// ParseResponseOptimized parses an HTTP response from an OptimizedReader.
// This uses zero-allocation line reading for better performance.
//
// Allocation behavior: 0-1 allocs/op (zero-copy line reading)
func (r *ClientResponse) ParseResponseOptimized(or *OptimizedReader) error {
	// Read status line (zero-copy)
	statusLine, err := or.ReadLine()
	if err != nil {
		return err
	}

	if err := r.ParseStatusLine(statusLine); err != nil {
		return err
	}

	// Read headers (zero-copy)
	for {
		line, err := or.ReadLine()
		if err != nil {
			return err
		}

		// Empty line marks end of headers
		if len(line) <= 2 {
			break
		}

		if err := r.ParseHeader(line); err != nil {
			return err
		}
	}

	// Process special headers
	r.processHeaders()

	return nil
}

// processHeaders extracts important header values
func (r *ClientResponse) processHeaders() {
	if r.headers == nil {
		return
	}

	// Content-Length
	if clBytes := r.headers.Get(headerContentLength); clBytes != nil {
		if cl, err := parseIntFast(clBytes); err == nil {
			r.contentLength = int64(cl)
		}
	}

	// Transfer-Encoding
	if teBytes := r.headers.Get(headerTransferEncoding); teBytes != nil {
		if bytesEqual(teBytes, headerChunked) {
			r.isChunked = true
		}
	}

	// Connection
	if connBytes := r.headers.Get(headerConnection); connBytes != nil {
		if bytesEqualCaseInsensitive(connBytes, headerClose) {
			r.close = true
		}
	}
}

// StatusCode returns the HTTP status code.
//
// Allocation behavior: 0 allocs/op
func (r *ClientResponse) StatusCode() int {
	return r.statusCode
}

// Status returns the status text as a string.
//
// Allocation behavior: 0 allocs/op after first call (cached)
func (r *ClientResponse) Status() string {
	if r.statusString == "" && r.statusLen > 0 {
		r.statusString = string(r.statusBytes[:r.statusLen])
	}
	return r.statusString
}

// Proto returns the protocol version as a string.
//
// Allocation behavior: 0 allocs/op after first call (cached)
func (r *ClientResponse) Proto() string {
	if r.protoString == "" && r.protoLen > 0 {
		r.protoString = string(r.protoBytes[:r.protoLen])
	}
	return r.protoString
}

// GetHeader returns a header value by name.
//
// Allocation behavior: 0 allocs/op
func (r *ClientResponse) GetHeader(name string) string {
	if r.headers == nil {
		return ""
	}
	return r.headers.GetString(name)
}

// Body returns the response body reader.
//
// Allocation behavior: 0 allocs/op
func (r *ClientResponse) Body() io.ReadCloser {
	return r.body
}

// SetBody sets the response body reader.
//
// Allocation behavior: 0 allocs/op
func (r *ClientResponse) SetBody(body io.ReadCloser) {
	r.body = body
}

// ContentLength returns the Content-Length header value.
//
// Allocation behavior: 0 allocs/op
func (r *ClientResponse) ContentLength() int64 {
	return r.contentLength
}

// Close closes the response body and returns the connection to the pool.
//
// Allocation behavior: 0 allocs/op
func (r *ClientResponse) Close() error {
	var err error

	// Close body if present
	if r.body != nil {
		err = r.body.Close()
		r.body = nil
	}

	// Return connection to pool
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}

	return err
}

// parseIntFast parses an integer from a byte slice without allocation.
// Optimized for common HTTP status codes and header values.
//
// Allocation behavior: 0 allocs/op
func parseIntFast(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, strconv.ErrSyntax
	}

	// Fast path for 3-digit numbers (HTTP status codes)
	if len(b) == 3 {
		if b[0] >= '0' && b[0] <= '9' &&
			b[1] >= '0' && b[1] <= '9' &&
			b[2] >= '0' && b[2] <= '9' {
			return int(b[0]-'0')*100 + int(b[1]-'0')*10 + int(b[2]-'0'), nil
		}
	}

	// General case
	n := 0
	for i := 0; i < len(b); i++ {
		if b[i] < '0' || b[i] > '9' {
			return 0, strconv.ErrSyntax
		}
		n = n*10 + int(b[i]-'0')
	}
	return n, nil
}
