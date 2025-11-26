//go:build highperf
// +build highperf

package client

// High Performance Configuration
// Target: ~2,800 B/op total client memory
// Use case: General-purpose high-performance APIs, balanced memory/performance
//
// Build: go build -tags highperf .

// Header limits optimized for balanced performance
const (
	// MaxHeaders is the maximum number of headers we can store inline
	// Set to 12 for balanced memory/performance
	// Covers ~60% of HTTP responses without map overflow
	MaxHeaders = 12

	// MaxHeaderName is the maximum length of a header name
	MaxHeaderName = 64

	// MaxHeaderValue is the maximum length of a header value for inline storage
	// Set to 48 bytes - covers most common header values
	MaxHeaderValue = 48

	// MaxStatusLine is the maximum size of the status line
	MaxStatusLine = 192
)

// Memory calculation:
// - names: [12][64]byte = 768 bytes
// - values: [12][48]byte = 576 bytes
// - overhead: 32 bytes
// Total ClientHeaders: ~1,376 bytes
// Total ClientResponse: ~200 bytes
// Target total: ~2,800 B/op
