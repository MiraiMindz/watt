package http11

import "errors"

// Parser errors - Pre-allocated for zero runtime allocation
var (
	// ErrInvalidRequestLine indicates the request line is malformed
	// Request line format: METHOD PATH PROTOCOL\r\n
	ErrInvalidRequestLine = errors.New("http11: invalid request line")

	// ErrInvalidMethod indicates an unsupported or malformed HTTP method
	ErrInvalidMethod = errors.New("http11: invalid HTTP method")

	// ErrInvalidPath indicates the request path is malformed
	ErrInvalidPath = errors.New("http11: invalid request path")

	// ErrInvalidProtocol indicates an unsupported protocol version
	// Only HTTP/1.1 is supported by this engine
	ErrInvalidProtocol = errors.New("http11: invalid or unsupported protocol version")

	// ErrInvalidHeader indicates a malformed header
	// Headers must be in format: Name: Value\r\n
	ErrInvalidHeader = errors.New("http11: invalid HTTP header")

	// ErrHeaderTooLarge indicates a header name or value exceeds size limits
	// Limits: name ≤64 bytes, value ≤256 bytes
	ErrHeaderTooLarge = errors.New("http11: header name or value too large")

	// ErrTooManyHeaders indicates more than 32 headers without overflow buffer
	ErrTooManyHeaders = errors.New("http11: too many headers (>32 without overflow)")

	// ErrRequestLineTooLarge indicates the request line exceeds 8KB
	ErrRequestLineTooLarge = errors.New("http11: request line too large")

	// ErrHeadersTooLarge indicates total headers size exceeds 8KB
	ErrHeadersTooLarge = errors.New("http11: headers too large")

	// ErrChunkedEncoding indicates an error parsing chunked transfer encoding
	ErrChunkedEncoding = errors.New("http11: chunked encoding error")

	// ErrInvalidContentLength indicates Content-Length header is malformed
	ErrInvalidContentLength = errors.New("http11: invalid Content-Length")

	// P0 FIX #1: HTTP Request Smuggling - CL.TE Attack Protection
	// ErrContentLengthWithTransferEncoding indicates a request has both headers
	// RFC 7230 §3.3.3: This MUST be rejected to prevent smuggling attacks
	ErrContentLengthWithTransferEncoding = errors.New("http11: request has both Content-Length and Transfer-Encoding (RFC 7230 violation)")

	// P0 FIX #2: HTTP Request Smuggling - Duplicate Content-Length Protection
	// ErrDuplicateContentLength indicates multiple Content-Length headers with different values
	// RFC 7230 §3.3.3: This MUST be rejected to prevent smuggling attacks
	ErrDuplicateContentLength = errors.New("http11: duplicate Content-Length headers with different values (RFC 7230 violation)")

	// P0 FIX #5: Excessive URI Length DoS Protection
	// ErrURITooLong indicates the URI exceeds the maximum allowed length
	// This prevents memory exhaustion attacks
	ErrURITooLong = errors.New("http11: URI too long")

	// ErrUnexpectedEOF indicates unexpected end of input
	ErrUnexpectedEOF = errors.New("http11: unexpected EOF")

	// ErrBufferTooSmall indicates the provided buffer is too small
	ErrBufferTooSmall = errors.New("http11: buffer too small")
)

// Connection errors
var (
	// ErrConnectionClosed indicates the connection has been closed
	ErrConnectionClosed = errors.New("http11: connection closed")

	// ErrTimeout indicates a read or write timeout occurred
	ErrTimeout = errors.New("http11: timeout")

	// ErrMaxRequestsExceeded indicates max requests per connection exceeded
	ErrMaxRequestsExceeded = errors.New("http11: max requests per connection exceeded")
)

// Response errors
var (
	// ErrHeadersAlreadyWritten indicates WriteHeader was called multiple times
	ErrHeadersAlreadyWritten = errors.New("http11: headers already written")

	// ErrInvalidStatusCode indicates an invalid HTTP status code
	ErrInvalidStatusCode = errors.New("http11: invalid status code")
)
