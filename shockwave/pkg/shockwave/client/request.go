package client

import (
	"context"
	"io"
)

// ClientRequest represents an optimized HTTP client request with zero-allocation design.
//
// Design principles:
// - Pooled via sync.Pool for reuse (0 allocs per request)
// - Inline header storage for ≤32 headers (0 allocs)
// - Method ID for O(1) switching
// - Pre-compiled constants where possible
// - Zero-copy byte slices for most operations
type ClientRequest struct {
	// HTTP Method
	methodID    uint8  // Method ID for O(1) switching
	methodBytes []byte // Pre-compiled method bytes (zero-copy reference)
	methodStr   string // Method string (for custom methods)

	// URL components (inline storage)
	// Using fixed-size arrays to avoid allocations
	schemeBytes [8]byte   // "http" or "https"
	schemeLen   uint8     // Length of scheme
	hostBytes   [128]byte // Host name (128 bytes is enough for most hosts)
	hostLen     uint8     // Length of host
	portBytes   [6]byte   // Port number as string
	portLen     uint8     // Length of port
	pathBytes   [256]byte // Request path (256 bytes covers most paths)
	pathLen     uint16    // Length of path
	queryBytes  [256]byte // Query string (256 bytes for query params)
	queryLen    uint16    // Length of query

	// Headers (inline storage via ClientHeaders)
	headers *ClientHeaders

	// Body
	body       io.Reader
	bodyLength int64 // -1 if unknown, >= 0 if known

	// Protocol version
	proto      string // "HTTP/1.1"
	protoMajor int    // 1
	protoMinor int    // 1

	// Timeouts
	dialTimeout    int64 // nanoseconds
	requestTimeout int64 // nanoseconds

	// Context
	ctx context.Context

	// Buffer for building request (reused across requests)
	buf []byte

	// Cached hostPort buffer (avoids allocation in Do())
	// 160 bytes is enough for "host:port" (128 + 1 + 6 + padding)
	hostPortBuf [160]byte
	hostPortLen uint8
}

// Reset clears the request for reuse.
// Called before returning to pool and when acquiring from pool.
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) Reset() {
	r.methodID = methodIDGET
	r.methodBytes = nil
	r.methodStr = ""

	r.schemeLen = 0
	r.hostLen = 0
	r.portLen = 0
	r.pathLen = 0
	r.queryLen = 0

	if r.headers != nil {
		r.headers.Reset()
	}

	r.body = nil
	r.bodyLength = -1

	r.proto = http11String
	r.protoMajor = 1
	r.protoMinor = 1

	r.dialTimeout = 0
	r.requestTimeout = 0

	r.ctx = nil

	// Reset buffer length but keep capacity
	if r.buf != nil {
		r.buf = r.buf[:0]
	}

	// Reset hostPort cache
	r.hostPortLen = 0
}

// SetMethod sets the HTTP method.
// Uses pre-compiled constants for common methods (0 allocs).
//
// Allocation behavior: 0 allocs/op for known methods
func (r *ClientRequest) SetMethod(method string) {
	r.methodID = methodToID(method)
	r.methodBytes = methodToBytes(method)

	// If method is unknown, store the string
	if r.methodID == methodIDUnknown {
		r.methodStr = method
	}
}

// SetURL sets the complete URL.
// Parses and stores components inline for zero allocations.
//
// Allocation behavior: 0 allocs/op if all components fit inline
func (r *ClientRequest) SetURL(scheme, host, port, path, query string) {
	// Set scheme
	if len(scheme) <= len(r.schemeBytes) {
		copy(r.schemeBytes[:], scheme)
		r.schemeLen = uint8(len(scheme))
	}

	// Set host (truncate if too long)
	hostLen := len(host)
	if hostLen > len(r.hostBytes) {
		hostLen = len(r.hostBytes)
	}
	copy(r.hostBytes[:], host[:hostLen])
	r.hostLen = uint8(hostLen)

	// Set port
	if len(port) <= len(r.portBytes) {
		copy(r.portBytes[:], port)
		r.portLen = uint8(len(port))
	}

	// Set path (truncate if too long)
	pathLen := len(path)
	if pathLen > len(r.pathBytes) {
		pathLen = len(r.pathBytes)
	}
	copy(r.pathBytes[:], path[:pathLen])
	r.pathLen = uint16(pathLen)

	// Set query (truncate if too long)
	queryLen := len(query)
	if queryLen > len(r.queryBytes) {
		queryLen = len(r.queryBytes)
	}
	copy(r.queryBytes[:], query[:queryLen])
	r.queryLen = uint16(queryLen)
}

// SetPath sets just the path component.
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) SetPath(path string) {
	pathLen := len(path)
	if pathLen > len(r.pathBytes) {
		pathLen = len(r.pathBytes)
	}
	copy(r.pathBytes[:], path[:pathLen])
	r.pathLen = uint16(pathLen)
}

// SetHost sets the host component.
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) SetHost(host string) {
	hostLen := len(host)
	if hostLen > len(r.hostBytes) {
		hostLen = len(r.hostBytes)
	}
	copy(r.hostBytes[:], host[:hostLen])
	r.hostLen = uint8(hostLen)
}

// SetHeader sets a request header.
// Uses inline storage for ≤32 headers.
//
// Allocation behavior: 0 allocs/op for ≤32 headers
func (r *ClientRequest) SetHeader(name, value string) {
	if r.headers == nil {
		r.headers = GetHeaders()
	}
	r.headers.SetString(name, value)
}

// SetBody sets the request body and content length.
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) SetBody(body io.Reader, length int64) {
	r.body = body
	r.bodyLength = length
}

// SetContext sets the request context.
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) SetContext(ctx context.Context) {
	r.ctx = ctx
}

// GetMethod returns the method bytes.
// Zero-copy for known methods.
//
// Allocation behavior: 0 allocs/op for known methods
func (r *ClientRequest) GetMethod() []byte {
	if r.methodBytes != nil {
		return r.methodBytes
	}
	return []byte(r.methodStr)
}

// GetMethodString returns the method as a string.
//
// Allocation behavior: 0 allocs/op for known methods, 1 alloc for unknown
func (r *ClientRequest) GetMethodString() string {
	if r.methodID != methodIDUnknown {
		// Return pre-compiled string
		switch r.methodID {
		case methodIDGET:
			return methodGETString
		case methodIDPOST:
			return methodPOSTString
		case methodIDPUT:
			return methodPUTString
		case methodIDDELETE:
			return methodDELETEString
		case methodIDPATCH:
			return methodPATCHString
		case methodIDHEAD:
			return methodHEADString
		case methodIDOPTIONS:
			return methodOPTIONSString
		case methodIDCONNECT:
			return methodCONNECTString
		case methodIDTRACE:
			return methodTRACEString
		}
	}
	return r.methodStr
}

// GetHost returns the host as a byte slice (zero-copy).
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) GetHost() []byte {
	return r.hostBytes[:r.hostLen]
}

// GetPath returns the path as a byte slice (zero-copy).
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) GetPath() []byte {
	return r.pathBytes[:r.pathLen]
}

// GetQuery returns the query string as a byte slice (zero-copy).
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) GetQuery() []byte {
	return r.queryBytes[:r.queryLen]
}

// BuildRequestLine builds the HTTP request line into the buffer.
// Format: "METHOD /path?query HTTP/1.1\r\n"
//
// Allocation behavior: 0 allocs/op (uses pre-allocated buffer)
func (r *ClientRequest) BuildRequestLine() []byte {
	buf := r.buf[:0]

	// Method
	buf = append(buf, r.GetMethod()...)
	buf = append(buf, spaceBytes...)

	// Path
	buf = append(buf, r.GetPath()...)

	// Query (if present)
	if r.queryLen > 0 {
		buf = append(buf, questionBytes...)
		buf = append(buf, r.GetQuery()...)
	}

	buf = append(buf, spaceBytes...)

	// Protocol
	buf = append(buf, http11Bytes...)
	buf = append(buf, crlfBytes...)

	r.buf = buf
	return buf
}

// BuildHeaders builds the HTTP headers into the buffer.
//
// Allocation behavior: 0 allocs/op for inline headers
func (r *ClientRequest) BuildHeaders() []byte {
	if r.headers == nil {
		return r.buf
	}

	// Write headers using WriteTo method
	r.buf = r.headers.WriteTo(r.buf)
	return r.buf
}

// BuildRequest builds the complete HTTP request.
// Returns a byte slice containing the full request (without body).
//
// Allocation behavior: 0 allocs/op (reuses buffer)
func (r *ClientRequest) BuildRequest() []byte {
	// Reset buffer
	r.buf = r.buf[:0]

	// Ensure we have enough capacity
	if cap(r.buf) < DefaultBufferSize {
		r.buf = make([]byte, 0, DefaultBufferSize)
	}

	// Build request line
	r.BuildRequestLine()

	// Build headers
	r.BuildHeaders()

	// Final CRLF
	r.buf = append(r.buf, crlfBytes...)

	return r.buf
}

// buildHostPort builds "host:port" string and caches it.
// Returns zero-copy byte slice. Zero allocations.
//
// Allocation behavior: 0 allocs/op
func (r *ClientRequest) buildHostPort() []byte {
	// Build into cached buffer
	pos := uint8(0)

	// Copy host
	n := copy(r.hostPortBuf[pos:], r.hostBytes[:r.hostLen])
	pos += uint8(n)

	// Add colon
	if pos < uint8(len(r.hostPortBuf)) {
		r.hostPortBuf[pos] = ':'
		pos++
	}

	// Add port (explicit or default based on scheme)
	if r.portLen > 0 {
		n = copy(r.hostPortBuf[pos:], r.portBytes[:r.portLen])
		pos += uint8(n)
	} else {
		// Default port based on scheme
		if r.schemeLen == 5 && r.schemeBytes[0] == 'h' && r.schemeBytes[1] == 't' &&
			r.schemeBytes[2] == 't' && r.schemeBytes[3] == 'p' && r.schemeBytes[4] == 's' {
			copy(r.hostPortBuf[pos:], "443")
			pos += 3
		} else {
			copy(r.hostPortBuf[pos:], "80")
			pos += 2
		}
	}

	r.hostPortLen = pos
	return r.hostPortBuf[:pos]
}
