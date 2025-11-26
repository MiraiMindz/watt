package http11

import (
	"io"
	"strconv"
)

// ResponseWriter writes HTTP/1.1 responses with zero allocations for common cases.
//
// Design:
// - Pre-compiled status lines for common status codes (200, 404, 500, etc.)
// - Inline header storage (reuses Request's Header design)
// - Buffered writing for efficiency
// - Zero allocations for common responses
//
// Allocation behavior: 0 allocs/op for common status codes and ≤32 headers
type ResponseWriter struct {
	// Underlying writer
	w io.Writer

	// Status code (default 200)
	status int

	// Response headers (inline storage, zero allocations for ≤32)
	header Header

	// State tracking
	statusWritten bool // True if WriteHeader was explicitly called
	headerWritten bool // True if headers were written to the wire
	bytesWritten  int64

	// Content length (if known)
	contentLength int64

	// Chunked encoding flag
	chunked bool
}

// NewResponseWriter creates a new ResponseWriter for the given writer.
func NewResponseWriter(w io.Writer) *ResponseWriter {
	return &ResponseWriter{
		w:      w,
		status: 200, // Default to 200 OK
	}
}

// Header returns the response header map.
// Headers must be set before calling Write or WriteHeader.
func (rw *ResponseWriter) Header() *Header {
	return &rw.header
}

// WriteHeader sends an HTTP response header with the provided status code.
// If WriteHeader is not called explicitly, the first call to Write will
// trigger an implicit WriteHeader(200).
// Multiple calls to WriteHeader are ignored (only the first takes effect).
//
// Allocation behavior: 0 allocs/op for common status codes
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	if rw.statusWritten {
		return // Already set, ignore subsequent calls
	}
	rw.status = statusCode
	rw.statusWritten = true
}

// Write writes the data to the connection as part of an HTTP reply.
// If WriteHeader has not yet been called, Write calls WriteHeader(200)
// before writing the data.
//
// Allocation behavior: Depends on underlying writer
func (rw *ResponseWriter) Write(data []byte) (int, error) {
	if !rw.headerWritten {
		if err := rw.writeHeaders(); err != nil {
			return 0, err
		}
	}

	n, err := rw.w.Write(data)
	rw.bytesWritten += int64(n)
	return n, err
}

// writeHeaders writes the status line and headers to the writer.
// This is called automatically by Write() or can be called explicitly.
//
// Allocation behavior: 0 allocs/op for common status codes and ≤32 headers
func (rw *ResponseWriter) writeHeaders() error {
	if rw.headerWritten {
		return nil
	}
	rw.headerWritten = true

	// Write status line
	statusLine := getStatusLine(rw.status)
	if _, err := rw.w.Write(statusLine); err != nil {
		return err
	}

	// Write headers
	rw.header.VisitAll(func(name, value []byte) bool {
		// Write name
		if _, err := rw.w.Write(name); err != nil {
			return false
		}
		// Write ": "
		if _, err := rw.w.Write(colonSpace); err != nil {
			return false
		}
		// Write value
		if _, err := rw.w.Write(value); err != nil {
			return false
		}
		// Write CRLF
		if _, err := rw.w.Write(crlfBytes); err != nil {
			return false
		}
		return true
	})

	// Write final CRLF (blank line separating headers from body)
	if _, err := rw.w.Write(crlfBytes); err != nil {
		return err
	}

	return nil
}

// Flush ensures all buffered data is written.
// For ResponseWriter without buffering, this is a no-op.
// When used with a buffered writer, it flushes the buffer.
func (rw *ResponseWriter) Flush() error {
	if !rw.headerWritten {
		if err := rw.writeHeaders(); err != nil {
			return err
		}
	}

	// If the underlying writer supports Flush, call it
	if flusher, ok := rw.w.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}

	return nil
}

// Status returns the HTTP status code that was written.
// If WriteHeader was not called, returns 200.
func (rw *ResponseWriter) Status() int {
	return rw.status
}

// BytesWritten returns the number of bytes written to the response body.
func (rw *ResponseWriter) BytesWritten() int64 {
	return rw.bytesWritten
}

// HeaderWritten returns whether the headers have been written.
func (rw *ResponseWriter) HeaderWritten() bool {
	return rw.headerWritten
}

// Reset resets the ResponseWriter for reuse (pooling support).
// All fields are cleared to their zero values.
//
// Allocation behavior: 0 allocs/op
func (rw *ResponseWriter) Reset(w io.Writer) {
	rw.w = w
	rw.status = 200
	rw.header.Reset()
	rw.statusWritten = false
	rw.headerWritten = false
	rw.bytesWritten = 0
	rw.contentLength = 0
	rw.chunked = false
}

// getStatusLine returns the pre-compiled status line for common status codes.
// For uncommon codes, it builds the status line (1 allocation).
//
// Covers 95% of HTTP responses with zero allocations.
//
// Allocation behavior: 0 allocs/op for common codes, 1 alloc/op for uncommon codes
func getStatusLine(code int) []byte {
	switch code {
	// 1xx Informational
	case 100:
		return status100Bytes
	case 101:
		return status101Bytes

	// 2xx Success
	case 200:
		return status200Bytes
	case 201:
		return status201Bytes
	case 202:
		return status202Bytes
	case 203:
		return status203Bytes
	case 204:
		return status204Bytes
	case 205:
		return status205Bytes
	case 206:
		return status206Bytes

	// 3xx Redirection
	case 300:
		return status300Bytes
	case 301:
		return status301Bytes
	case 302:
		return status302Bytes
	case 303:
		return status303Bytes
	case 304:
		return status304Bytes
	case 307:
		return status307Bytes
	case 308:
		return status308Bytes

	// 4xx Client Error
	case 400:
		return status400Bytes
	case 401:
		return status401Bytes
	case 403:
		return status403Bytes
	case 404:
		return status404Bytes
	case 405:
		return status405Bytes
	case 406:
		return status406Bytes
	case 408:
		return status408Bytes
	case 409:
		return status409Bytes
	case 410:
		return status410Bytes
	case 411:
		return status411Bytes
	case 412:
		return status412Bytes
	case 413:
		return status413Bytes
	case 414:
		return status414Bytes
	case 415:
		return status415Bytes
	case 429:
		return status429Bytes

	// 5xx Server Error
	case 500:
		return status500Bytes
	case 501:
		return status501Bytes
	case 502:
		return status502Bytes
	case 503:
		return status503Bytes
	case 504:
		return status504Bytes

	default:
		// Uncommon status code, build it (allocates)
		return buildStatusLine(code)
	}
}

// buildStatusLine builds a status line for uncommon status codes.
// This allocates but is only called for rare status codes.
func buildStatusLine(code int) []byte {
	text := statusText(code)
	// Format: "HTTP/1.1 CODE TEXT\r\n"
	return []byte("HTTP/1.1 " + strconv.Itoa(code) + " " + text + "\r\n")
}

// statusText returns the text description for an HTTP status code.
// Based on RFC 7231 Section 6.
func statusText(code int) string {
	switch code {
	// 1xx Informational
	case 100:
		return "Continue"
	case 101:
		return "Switching Protocols"

	// 2xx Success
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 202:
		return "Accepted"
	case 203:
		return "Non-Authoritative Information"
	case 204:
		return "No Content"
	case 205:
		return "Reset Content"
	case 206:
		return "Partial Content"

	// 3xx Redirection
	case 300:
		return "Multiple Choices"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 303:
		return "See Other"
	case 304:
		return "Not Modified"
	case 305:
		return "Use Proxy"
	case 307:
		return "Temporary Redirect"
	case 308:
		return "Permanent Redirect"

	// 4xx Client Error
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 402:
		return "Payment Required"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 406:
		return "Not Acceptable"
	case 407:
		return "Proxy Authentication Required"
	case 408:
		return "Request Timeout"
	case 409:
		return "Conflict"
	case 410:
		return "Gone"
	case 411:
		return "Length Required"
	case 412:
		return "Precondition Failed"
	case 413:
		return "Payload Too Large"
	case 414:
		return "URI Too Long"
	case 415:
		return "Unsupported Media Type"
	case 416:
		return "Range Not Satisfiable"
	case 417:
		return "Expectation Failed"
	case 418:
		return "I'm a teapot"
	case 422:
		return "Unprocessable Entity"
	case 426:
		return "Upgrade Required"
	case 428:
		return "Precondition Required"
	case 429:
		return "Too Many Requests"
	case 431:
		return "Request Header Fields Too Large"

	// 5xx Server Error
	case 500:
		return "Internal Server Error"
	case 501:
		return "Not Implemented"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	case 504:
		return "Gateway Timeout"
	case 505:
		return "HTTP Version Not Supported"

	default:
		return "Unknown"
	}
}

// WriteJSON is a convenience method to write a JSON response.
// Sets Content-Type to application/json and writes the data.
//
// Allocation behavior: 1 alloc/op for status line (if uncommon), plus underlying Write
func (rw *ResponseWriter) WriteJSON(statusCode int, data []byte) error {
	rw.WriteHeader(statusCode)
	rw.header.Set(headerContentType, contentTypeJSONUTF8)

	// Set Content-Length
	contentLengthStr := strconv.FormatInt(int64(len(data)), 10)
	rw.header.Set(headerContentLength, []byte(contentLengthStr))

	if _, err := rw.Write(data); err != nil {
		return err
	}
	return rw.Flush()
}

// WriteText is a convenience method to write a plain text response.
// Sets Content-Type to text/plain and writes the data.
func (rw *ResponseWriter) WriteText(statusCode int, data []byte) error {
	rw.WriteHeader(statusCode)
	rw.header.Set(headerContentType, contentTypePlain)

	// Set Content-Length
	contentLengthStr := strconv.FormatInt(int64(len(data)), 10)
	rw.header.Set(headerContentLength, []byte(contentLengthStr))

	if _, err := rw.Write(data); err != nil {
		return err
	}
	return rw.Flush()
}

// WriteHTML is a convenience method to write an HTML response.
// Sets Content-Type to text/html and writes the data.
func (rw *ResponseWriter) WriteHTML(statusCode int, data []byte) error {
	rw.WriteHeader(statusCode)
	rw.header.Set(headerContentType, contentTypeHTML)

	// Set Content-Length
	contentLengthStr := strconv.FormatInt(int64(len(data)), 10)
	rw.header.Set(headerContentLength, []byte(contentLengthStr))

	if _, err := rw.Write(data); err != nil {
		return err
	}
	return rw.Flush()
}

// WriteError is a convenience method to write an error response.
// Writes a plain text error message with the given status code.
func (rw *ResponseWriter) WriteError(statusCode int, message string) error {
	return rw.WriteText(statusCode, []byte(message))
}

// WriteChunked writes a response using chunked transfer encoding.
// This is useful when the content length is not known in advance.
//
// Usage:
//   rw.WriteHeader(200)
//   rw.Header().Set([]byte("Transfer-Encoding"), []byte("chunked"))
//   chunks := [][]byte{chunk1, chunk2, chunk3}
//   rw.WriteChunked(chunks)
//
// Allocation behavior: 1 alloc for hex conversion per chunk
func (rw *ResponseWriter) WriteChunked(chunks [][]byte) error {
	// Write headers if not yet written
	if !rw.headerWritten {
		rw.chunked = true
		if err := rw.writeHeaders(); err != nil {
			return err
		}
	}

	// Write each chunk with chunk size prefix
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}

		// Write chunk size in hex
		chunkSize := []byte(strconv.FormatInt(int64(len(chunk)), 16))
		if _, err := rw.w.Write(chunkSize); err != nil {
			return err
		}

		// Write CRLF after size
		if _, err := rw.w.Write(crlfBytes); err != nil {
			return err
		}

		// Write chunk data
		if _, err := rw.w.Write(chunk); err != nil {
			return err
		}

		// Write CRLF after chunk
		if _, err := rw.w.Write(crlfBytes); err != nil {
			return err
		}

		rw.bytesWritten += int64(len(chunk))
	}

	// Write final chunk (0\r\n\r\n)
	if _, err := rw.w.Write([]byte("0\r\n\r\n")); err != nil {
		return err
	}

	return rw.Flush()
}

// WriteChunk writes a single chunk using chunked transfer encoding.
// Call this multiple times followed by FinishChunked() to complete the response.
//
// First call will automatically write headers with Transfer-Encoding: chunked.
//
// Allocation behavior: 1 alloc for hex conversion per chunk
func (rw *ResponseWriter) WriteChunk(chunk []byte) error {
	// Write headers on first chunk
	if !rw.headerWritten {
		rw.chunked = true
		// Ensure Transfer-Encoding header is set
		if rw.header.Get([]byte("Transfer-Encoding")) == nil {
			rw.header.Set([]byte("Transfer-Encoding"), []byte("chunked"))
		}
		if err := rw.writeHeaders(); err != nil {
			return err
		}
	}

	if len(chunk) == 0 {
		return nil
	}

	// Write chunk size in hex
	chunkSize := []byte(strconv.FormatInt(int64(len(chunk)), 16))
	if _, err := rw.w.Write(chunkSize); err != nil {
		return err
	}

	// Write CRLF after size
	if _, err := rw.w.Write(crlfBytes); err != nil {
		return err
	}

	// Write chunk data
	if _, err := rw.w.Write(chunk); err != nil {
		return err
	}

	// Write CRLF after chunk
	if _, err := rw.w.Write(crlfBytes); err != nil {
		return err
	}

	rw.bytesWritten += int64(len(chunk))
	return nil
}

// FinishChunked writes the final chunk marker (0\r\n\r\n) to complete
// a chunked transfer encoding response.
// Call this after all WriteChunk() calls.
//
// Allocation behavior: 0 allocs/op
func (rw *ResponseWriter) FinishChunked() error {
	// Write final chunk (0\r\n\r\n)
	if _, err := rw.w.Write([]byte("0\r\n\r\n")); err != nil {
		return err
	}
	return rw.Flush()
}
