package http11

import (
	"io"
	"net/url"
)

// Request represents an HTTP/1.1 request.
// Designed for zero-allocation parsing and pooling.
//
// CRITICAL: All byte slices (methodBytes, pathBytes, queryBytes, protoBytes)
// are zero-copy references into the request buffer. They are only valid
// during the request lifetime. Do NOT store these slices beyond the handler
// execution or use them after the request is returned to the pool.
//
// For safe string access that persists, use Method(), Path(), etc. which
// return strings (1 allocation each, but safe to store).
type Request struct {
	// Method as numeric ID for O(1) switching
	// Use MethodString() to get the string representation
	MethodID uint8

	// Request-Line components (zero-copy slices into buffer)
	// WARNING: These slices are only valid during request lifetime
	// They reference the internal buffer which is pooled and reused
	methodBytes []byte // e.g., "GET"
	pathBytes   []byte // e.g., "/api/users"
	queryBytes  []byte // e.g., "id=123&name=foo" (without '?')
	protoBytes  []byte // e.g., "HTTP/1.1"

	// Parsed URL (lazy allocation)
	// Only allocated if ParsedURL() is called
	// Use PathBytes() to avoid this allocation
	pathParsed *url.URL

	// Headers (inline storage, zero heap allocations for ≤32)
	Header Header

	// Body reader
	// nil if no body present
	// Will be io.LimitReader for Content-Length
	// or chunkedReader for Transfer-Encoding: chunked
	Body io.Reader

	// Protocol information
	Proto      string // Always "HTTP/1.1" for this engine
	ProtoMajor int    // Always 1
	ProtoMinor int    // Always 1

	// Content information
	ContentLength int64 // -1 if unknown, >=0 if specified

	// Transfer encoding
	// nil for identity encoding
	// ["chunked"] for chunked encoding
	TransferEncoding []string

	// Connection control
	// true if "Connection: close" header present
	// or if HTTP/1.0 without "Connection: keep-alive"
	Close bool

	// RemoteAddr is the network address of the client
	RemoteAddr string

	// Internal buffer reference (for zero-copy safety)
	// This buffer is pooled and will be reused after request completes
	// All zero-copy slices reference this buffer
	buf []byte
}

// Method returns the HTTP method as a string.
// Uses pre-compiled constants for zero allocations.
//
// Allocation behavior: 0 allocs/op
func (r *Request) Method() string {
	return MethodString(r.MethodID)
}

// MethodBytes returns the HTTP method as a byte slice.
// This is a zero-copy reference into the request buffer.
// WARNING: Only valid during request lifetime.
//
// Allocation behavior: 0 allocs/op
func (r *Request) MethodBytes() []byte {
	return r.methodBytes
}

// Path returns the request path as a string.
// This allocates a string from the byte slice.
// For zero-allocation access, use PathBytes().
//
// Allocation behavior: 1 alloc/op
func (r *Request) Path() string {
	return string(r.pathBytes)
}

// PathBytes returns the request path as a byte slice.
// This is a zero-copy reference into the request buffer.
// WARNING: Only valid during request lifetime.
//
// Allocation behavior: 0 allocs/op
func (r *Request) PathBytes() []byte {
	return r.pathBytes
}

// Query returns the query string as a string.
// This allocates a string from the byte slice.
// For zero-allocation access, use QueryBytes().
//
// Allocation behavior: 1 alloc/op
func (r *Request) Query() string {
	return string(r.queryBytes)
}

// QueryBytes returns the query string as a byte slice (without the '?').
// This is a zero-copy reference into the request buffer.
// WARNING: Only valid during request lifetime.
//
// Allocation behavior: 0 allocs/op
func (r *Request) QueryBytes() []byte {
	return r.queryBytes
}

// ParsedURL returns the parsed URL.
// This is lazily allocated only when called.
// The result is cached for subsequent calls.
//
// Use PathBytes() or QueryBytes() if you don't need URL parsing
// to avoid this allocation.
//
// Allocation behavior: Multiple allocs/op on first call, 0 on subsequent
func (r *Request) ParsedURL() (*url.URL, error) {
	if r.pathParsed == nil {
		// Build full URL string for parsing
		// Format: path?query
		var urlStr string
		if len(r.queryBytes) > 0 {
			urlStr = string(r.pathBytes) + "?" + string(r.queryBytes)
		} else {
			urlStr = string(r.pathBytes)
		}

		var err error
		r.pathParsed, err = url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
	}
	return r.pathParsed, nil
}

// GetHeader retrieves a header value by name (case-insensitive).
// Returns nil if not found.
//
// Allocation behavior: 0 allocs/op
func (r *Request) GetHeader(name []byte) []byte {
	return r.Header.Get(name)
}

// GetHeaderString retrieves a header value as a string (case-insensitive).
// Returns empty string if not found.
//
// Allocation behavior: 1 alloc/op (string conversion)
func (r *Request) GetHeaderString(name string) string {
	return r.Header.GetString([]byte(name))
}

// HasHeader checks if a header exists (case-insensitive).
//
// Allocation behavior: 0 allocs/op
func (r *Request) HasHeader(name []byte) bool {
	return r.Header.Has(name)
}

// IsGET returns true if the request method is GET.
// Allocation behavior: 0 allocs/op
func (r *Request) IsGET() bool {
	return r.MethodID == MethodGET
}

// IsPOST returns true if the request method is POST.
// Allocation behavior: 0 allocs/op
func (r *Request) IsPOST() bool {
	return r.MethodID == MethodPOST
}

// IsPUT returns true if the request method is PUT.
// Allocation behavior: 0 allocs/op
func (r *Request) IsPUT() bool {
	return r.MethodID == MethodPUT
}

// IsDELETE returns true if the request method is DELETE.
// Allocation behavior: 0 allocs/op
func (r *Request) IsDELETE() bool {
	return r.MethodID == MethodDELETE
}

// IsPATCH returns true if the request method is PATCH.
// Allocation behavior: 0 allocs/op
func (r *Request) IsPATCH() bool {
	return r.MethodID == MethodPATCH
}

// IsHEAD returns true if the request method is HEAD.
// Allocation behavior: 0 allocs/op
func (r *Request) IsHEAD() bool {
	return r.MethodID == MethodHEAD
}

// IsOPTIONS returns true if the request method is OPTIONS.
// Allocation behavior: 0 allocs/op
func (r *Request) IsOPTIONS() bool {
	return r.MethodID == MethodOPTIONS
}

// HasBody returns true if the request has a body.
// Checks for Content-Length > 0 or Transfer-Encoding: chunked.
//
// Allocation behavior: 0 allocs/op
func (r *Request) HasBody() bool {
	return r.ContentLength > 0 || len(r.TransferEncoding) > 0
}

// IsChunked returns true if the request uses chunked transfer encoding.
//
// Allocation behavior: 0 allocs/op
func (r *Request) IsChunked() bool {
	if len(r.TransferEncoding) == 0 {
		return false
	}
	// Check last encoding (per RFC 7230, chunked must be last)
	lastEncoding := r.TransferEncoding[len(r.TransferEncoding)-1]
	return lastEncoding == "chunked"
}

// ShouldClose returns true if the connection should be closed after this request.
//
// Allocation behavior: 0 allocs/op
func (r *Request) ShouldClose() bool {
	return r.Close
}

// Reset clears the request for reuse (when returning to pool).
// All fields are reset to zero values.
// This enables efficient object pooling.
//
// Allocation behavior: 0 allocs/op
func (r *Request) Reset() {
	r.MethodID = 0
	r.methodBytes = nil
	r.pathBytes = nil
	r.queryBytes = nil
	r.protoBytes = nil
	r.pathParsed = nil
	r.Header.Reset()
	r.Body = nil
	r.Proto = ""
	r.ProtoMajor = 0
	r.ProtoMinor = 0
	r.ContentLength = 0
	r.TransferEncoding = nil
	r.Close = false
	r.RemoteAddr = ""
	r.buf = nil
}

// Clone creates a shallow copy of the request.
// This is useful when you need to store the request beyond its lifetime.
//
// IMPORTANT: This performs string conversions for path/query to ensure
// they remain valid after the original buffer is reused.
//
// The Body reader is NOT cloned - the clone will have Body = nil.
// If you need the body, read it before cloning or use io.TeeReader.
//
// Allocation behavior: Multiple allocations (strings, url.URL, etc.)
func (r *Request) Clone() *Request {
	clone := &Request{
		MethodID:         r.MethodID,
		methodBytes:      []byte(r.Method()), // Allocate new slice with string data
		pathBytes:        []byte(r.Path()),   // Allocate new slice
		queryBytes:       []byte(r.Query()),  // Allocate new slice
		protoBytes:       []byte(r.Proto),    // Allocate new slice
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding, // Shallow copy (slice header)
		Close:            r.Close,
		RemoteAddr:       r.RemoteAddr,
		Body:             nil, // Don't clone body reader
		buf:              nil, // Don't reference original buffer
	}

	// Clone headers (this will allocate)
	r.Header.VisitAll(func(name, value []byte) bool {
		clone.Header.Add(name, value)
		return true
	})

	// Clone parsed URL if present
	if r.pathParsed != nil {
		parsed, _ := r.ParsedURL()
		if parsed != nil {
			clone.pathParsed = &url.URL{
				Scheme:   parsed.Scheme,
				Host:     parsed.Host,
				Path:     parsed.Path,
				RawQuery: parsed.RawQuery,
			}
		}
	}

	return clone
}
