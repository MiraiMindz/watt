//go:build ultraperf
// +build ultraperf

package client

// Ultra Performance Configuration
// Target: ~4,000 B/op total client memory
// Use case: Maximum throughput, latency-sensitive applications
//
// Build: go build -tags ultraperf .

// Header limits optimized for ultra performance
const (
	// MaxHeaders is the maximum number of headers we can store inline
	// Set to 16 for high performance
	// Covers ~70% of HTTP responses without map overflow
	MaxHeaders = 16

	// MaxHeaderName is the maximum length of a header name
	MaxHeaderName = 64

	// MaxHeaderValue is the maximum length of a header value for inline storage
	// Set to 64 bytes - covers almost all common headers
	MaxHeaderValue = 64

	// MaxStatusLine is the maximum size of the status line
	MaxStatusLine = 256
)

// Memory calculation:
// - names: [16][64]byte = 1,024 bytes
// - values: [16][64]byte = 1,024 bytes
// - overhead: 32 bytes
// Total ClientHeaders: ~2,080 bytes
// Total ClientResponse: ~200 bytes
// Target total: ~4,000 B/op
