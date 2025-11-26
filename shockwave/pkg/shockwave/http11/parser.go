package http11

import (
	"bytes"
	"io"
	"sync"
)

// tmpBufPool provides pooled temporary buffers for reading requests.
// This eliminates 4KB allocation per request.
var tmpBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 4096)
		return &buf
	},
}

// Parser implements zero-allocation HTTP/1.1 request parsing.
// Uses a state machine approach for incremental parsing.
//
// Design:
// - Single-pass parsing (no backtracking)
// - Zero allocations for requests with ≤32 headers
// - Byte-by-byte state machine for streaming data
// - RFC 7230 compliant
// - Supports HTTP pipelining (multiple requests on same connection)
//
// Allocation behavior: 0 allocs/op for typical requests
type Parser struct {
	// Internal buffer for request line and headers
	// Maximum 8KB per RFC recommendations
	buf []byte

	// Unread buffer for pipelining support
	// Stores excess bytes read beyond current request boundary
	// Used for next Parse() call to enable HTTP keep-alive pipelining
	unreadBuf []byte
}

// NewParser creates a new HTTP/1.1 parser.
func NewParser() *Parser {
	return &Parser{
		buf: make([]byte, 0, MaxRequestLineSize+MaxHeadersSize),
	}
}

// Parse parses an HTTP/1.1 request from the reader.
// Returns a Request object or an error if parsing fails.
//
// The returned Request contains zero-copy slices referencing the internal buffer.
// The Request is valid until the next call to Parse() or until the Parser is discarded.
//
// IMPORTANT: The returned Request is from a pool. The caller MUST call PutRequest(req)
// when done to return it to the pool.
//
// For requests with ≤32 headers, this performs zero allocations after warmup.
//
// Supports HTTP pipelining: If previous Parse() read multiple requests, this will
// process the next request from the unread buffer before reading from the reader.
//
// Allocation behavior: 0 allocs/op for typical requests (after pool warmup)
func (p *Parser) Parse(r io.Reader) (*Request, error) {
	// Read request line and headers into buffer
	// We read until we find \r\n\r\n (end of headers)
	p.buf = p.buf[:0] // Reset buffer

	// For pipelining: combine unread buffer with reader
	// This allows processing multiple requests sent in one TCP packet
	var reader io.Reader
	if len(p.unreadBuf) > 0 {
		reader = io.MultiReader(bytes.NewReader(p.unreadBuf), r)
		p.unreadBuf = nil // Will be repopulated if more data after this request
	} else {
		reader = r
	}

	if err := p.readUntilHeadersEnd(reader); err != nil {
		return nil, err
	}

	// Get request object from pool (eliminates 11KB allocation)
	req := GetRequest()

	// Initialize fields
	req.Proto = http11Proto
	req.ProtoMajor = ProtoHTTP11Major
	req.ProtoMinor = ProtoHTTP11Minor
	req.buf = p.buf // Zero-copy reference

	// Parse request line
	pos, err := p.parseRequestLine(req, p.buf)
	if err != nil {
		// Return to pool on error
		PutRequest(req)
		return nil, err
	}

	// Parse headers
	if err := p.parseHeaders(req, p.buf[pos:]); err != nil {
		// Return to pool on error
		PutRequest(req)
		return nil, err
	}

	// Setup body reader if needed
	// P1 FIX #1: Pass unreadBuf for chunked/body reading
	// The unreadBuf may contain body data that was read along with headers
	bodyReader := r
	if len(p.unreadBuf) > 0 {
		bodyReader = io.MultiReader(bytes.NewReader(p.unreadBuf), r)
		p.unreadBuf = nil
	}

	if err := p.setupBodyReader(req, bodyReader); err != nil {
		// Return to pool on error
		PutRequest(req)
		return nil, err
	}

	return req, nil
}

// readUntilHeadersEnd reads from the reader until we find \r\n\r\n
// This marks the end of the headers section.
// Uses pooled buffer to eliminate 4KB allocation per request.
func (p *Parser) readUntilHeadersEnd(r io.Reader) error {
	// Get pooled buffer
	tmpBufPtr := tmpBufPool.Get().(*[]byte)
	defer tmpBufPool.Put(tmpBufPtr)
	tmpBuf := *tmpBufPtr

	foundEnd := false

	for !foundEnd {
		n, err := r.Read(tmpBuf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			if err == io.EOF {
				return ErrUnexpectedEOF
			}
			continue
		}

		// Append to buffer
		p.buf = append(p.buf, tmpBuf[:n]...)

		// Check for \r\n\r\n
		if len(p.buf) >= 4 {
			// Look for \r\n\r\n in the last read + 3 previous bytes
			searchStart := len(p.buf) - n - 3
			if searchStart < 0 {
				searchStart = 0
			}

			idx := bytes.Index(p.buf[searchStart:], []byte("\r\n\r\n"))
			if idx != -1 {
				foundEnd = true
				// actualIdx is position right after \r\n\r\n
				actualIdx := searchStart + idx + 4

				// HTTP Pipelining support: Save excess bytes beyond actualIdx
				// These bytes belong to the next request in the pipeline
				if actualIdx < len(p.buf) {
					// Copy excess bytes to unreadBuf for next Parse() call
					excessLen := len(p.buf) - actualIdx
					p.unreadBuf = make([]byte, excessLen)
					copy(p.unreadBuf, p.buf[actualIdx:])
				}

				// Trim to just the current request headers (before body/next request)
				p.buf = p.buf[:actualIdx]
			}
		}

		// Safety check: don't exceed maximum size
		if len(p.buf) > MaxRequestLineSize+MaxHeadersSize {
			return ErrHeadersTooLarge
		}

		if err == io.EOF {
			break
		}
	}

	if !foundEnd {
		return ErrUnexpectedEOF
	}

	return nil
}

// parseRequestLine parses "METHOD /path?query HTTP/1.1\r\n"
// Returns the position after the request line.
//
// Format: METHOD SP Request-URI SP HTTP-Version CRLF
// Example: GET /index.html HTTP/1.1\r\n
//
// Allocation behavior: 0 allocs/op
func (p *Parser) parseRequestLine(req *Request, buf []byte) (int, error) {
	// Find end of request line (\r\n)
	lineEnd := bytes.Index(buf, []byte("\r\n"))
	if lineEnd == -1 {
		return 0, ErrInvalidRequestLine
	}

	line := buf[:lineEnd]

	// P0 FIX #5: Excessive URI Length DoS Protection
	// RFC 7230 recommends 8KB limit for request line
	// This prevents memory exhaustion attacks
	if len(line) > MaxRequestLineSize {
		return 0, ErrRequestLineTooLarge
	}

	// Parse METHOD
	spaceIdx := bytes.IndexByte(line, ' ')
	if spaceIdx == -1 {
		return 0, ErrInvalidRequestLine
	}

	methodBytes := line[:spaceIdx]
	req.MethodID = ParseMethodID(methodBytes)
	if req.MethodID == MethodUnknown {
		return 0, ErrInvalidMethod
	}
	req.methodBytes = methodBytes

	// Parse Request-URI (path + optional query)
	line = line[spaceIdx+1:]
	spaceIdx = bytes.IndexByte(line, ' ')
	if spaceIdx == -1 {
		return 0, ErrInvalidRequestLine
	}

	uriBytes := line[:spaceIdx]

	// P0 FIX #5: Additional URI length check
	// Prevent extremely long URIs that could cause DoS
	if len(uriBytes) > MaxURILength {
		return 0, ErrURITooLong
	}

	// Split path and query
	queryIdx := bytes.IndexByte(uriBytes, '?')
	if queryIdx != -1 {
		req.pathBytes = uriBytes[:queryIdx]
		req.queryBytes = uriBytes[queryIdx+1:]
	} else {
		req.pathBytes = uriBytes
		req.queryBytes = nil
	}

	// Validate path (must start with / or be *)
	if len(req.pathBytes) == 0 {
		return 0, ErrInvalidPath
	}
	if req.pathBytes[0] != '/' && req.pathBytes[0] != '*' {
		return 0, ErrInvalidPath
	}

	// Parse HTTP-Version
	line = line[spaceIdx+1:]
	req.protoBytes = line

	// Validate HTTP/1.1
	if !bytes.Equal(line, http11Bytes) {
		return 0, ErrInvalidProtocol
	}

	return lineEnd + 2, nil // +2 for \r\n
}

// parseHeaders parses HTTP headers.
// Format: Name: Value\r\n
// Headers end with \r\n\r\n (extra blank line)
//
// Allocation behavior: 0 allocs/op for ≤32 headers
func (p *Parser) parseHeaders(req *Request, buf []byte) error {
	pos := 0

	// P0 FIX #1 & #2: Track special headers for smuggling prevention
	var hasContentLength bool
	var hasTransferEncoding bool
	var contentLengthValue int64 = -1

	// P2 FIX #4: Track Host header (RFC 7230 §5.4 - MUST have exactly one)
	var hasHost bool

	for {
		// Check for end of headers (\r\n)
		if pos >= len(buf) {
			break
		}

		// Empty line marks end of headers
		if pos+1 < len(buf) && buf[pos] == '\r' && buf[pos+1] == '\n' {
			break
		}

		// Find end of header line
		lineEnd := bytes.Index(buf[pos:], []byte("\r\n"))
		if lineEnd == -1 {
			return ErrInvalidHeader
		}
		lineEnd += pos

		line := buf[pos:lineEnd]

		// Find colon separator
		colonIdx := bytes.IndexByte(line, ':')
		if colonIdx == -1 {
			return ErrInvalidHeader
		}

		name := line[:colonIdx]
		value := line[colonIdx+1:]

		// P0 FIX #4: Whitespace Before Colon Protection
		// RFC 7230 §3.2: No whitespace is allowed between header field name and colon
		// Examples that should be rejected: "Host : example.com" or "Host\t: example.com"
		if colonIdx > 0 && (line[colonIdx-1] == ' ' || line[colonIdx-1] == '\t') {
			return ErrInvalidHeader
		}

		// Trim leading/trailing whitespace from value (per RFC 7230)
		value = trimLeadingSpace(value)
		value = trimTrailingSpace(value)

		// Validate header name (no spaces or tabs allowed)
		if bytes.IndexByte(name, ' ') != -1 || bytes.IndexByte(name, '\t') != -1 {
			return ErrInvalidHeader
		}

		// Add header
		if err := req.Header.Add(name, value); err != nil {
			return err
		}

		// Process special headers with smuggling checks
		if err := p.processSpecialHeader(req, name, value, &hasContentLength, &hasTransferEncoding, &contentLengthValue, &hasHost); err != nil {
			return err
		}

		// Move to next line
		pos = lineEnd + 2 // +2 for \r\n
	}

	// P0 FIX #1: HTTP Request Smuggling - CL.TE Attack Protection
	// RFC 7230 §3.3.3: If a message has both Transfer-Encoding and Content-Length,
	// the request MUST be rejected as malformed
	if hasContentLength && hasTransferEncoding {
		return ErrContentLengthWithTransferEncoding
	}

	return nil
}

// processSpecialHeader handles headers that affect request state
// (Content-Length, Transfer-Encoding, Connection, Host)
// P0 FIX #1 & #2: Added tracking parameters to prevent HTTP Request Smuggling
// P2 FIX #4: Added Host header tracking
func (p *Parser) processSpecialHeader(req *Request, name, value []byte,
	hasContentLength, hasTransferEncoding *bool, contentLengthValue *int64, hasHost *bool) error {

	// Content-Length
	if bytesEqualCaseInsensitive(name, headerContentLength) {
		contentLength, err := parseContentLength(value)
		if err != nil {
			return ErrInvalidContentLength
		}

		// P0 FIX #2: Duplicate Content-Length Protection
		// RFC 7230 §3.3.3: If multiple Content-Length headers exist,
		// they must all have the same value, otherwise reject
		if *hasContentLength {
			// We've seen Content-Length before
			if *contentLengthValue != contentLength {
				// Different value - this is a smuggling attempt
				return ErrDuplicateContentLength
			}
			// Same value is OK, just ignore
			return nil
		}

		// First Content-Length header
		*hasContentLength = true
		*contentLengthValue = contentLength
		req.ContentLength = contentLength
		return nil
	}

	// Transfer-Encoding
	if bytesEqualCaseInsensitive(name, headerTransferEncoding) {
		// Mark that we've seen Transfer-Encoding
		*hasTransferEncoding = true

		// Parse comma-separated list
		// For now, just check for "chunked"
		if bytesEqualCaseInsensitive(value, headerChunked) {
			req.TransferEncoding = []string{"chunked"}
		}
		return nil
	}

	// Connection
	if bytesEqualCaseInsensitive(name, headerConnection) {
		if bytesEqualCaseInsensitive(value, headerClose) {
			req.Close = true
		}
		return nil
	}

	// P2 FIX #4: Host header detection
	// RFC 7230 §5.4: A server MUST respond with 400 to any HTTP/1.1
	// request message that lacks a Host header field or contains more than one.
	if bytesEqualCaseInsensitive(name, headerHost) {
		if *hasHost {
			// Multiple Host headers - RFC violation
			return ErrInvalidHeader
		}
		*hasHost = true
		return nil
	}

	return nil
}

// setupBodyReader configures the body reader based on Content-Length or Transfer-Encoding
func (p *Parser) setupBodyReader(req *Request, r io.Reader) error {
	// No body
	if req.ContentLength == 0 && len(req.TransferEncoding) == 0 {
		req.Body = nil
		return nil
	}

	// Content-Length body
	if req.ContentLength > 0 {
		req.Body = io.LimitReader(r, req.ContentLength)
		return nil
	}

	// Chunked encoding
	if req.IsChunked() {
		// P1 FIX #1: Implement chunked transfer encoding reader
		// RFC 7230 §4.1 - chunked transfer coding
		req.Body = NewChunkedReader(r)
		return nil
	}

	return nil
}

// Helper functions

// parseContentLength parses Content-Length header value
// Returns -1 on error
func parseContentLength(b []byte) (int64, error) {
	if len(b) == 0 {
		return -1, ErrInvalidContentLength
	}

	var n int64
	for _, c := range b {
		if c < '0' || c > '9' {
			return -1, ErrInvalidContentLength
		}
		n = n*10 + int64(c-'0')

		// Prevent overflow
		if n < 0 {
			return -1, ErrInvalidContentLength
		}
	}
	return n, nil
}

// trimLeadingSpace trims leading spaces and tabs (per RFC 7230)
func trimLeadingSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	return b
}

// trimTrailingSpace trims trailing spaces and tabs (per RFC 7230)
func trimTrailingSpace(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	return b
}
