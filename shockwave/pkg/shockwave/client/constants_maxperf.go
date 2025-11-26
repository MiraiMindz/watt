//go:build maxperf
// +build maxperf

package client

// Maximum Performance Configuration
// Target: ~9,500 B/op total client memory
// Use case: Extreme performance, unlimited memory available
//
// Build: go build -tags maxperf .

// Header limits optimized for maximum performance
const (
	// MaxHeaders is the maximum number of headers we can store inline
	// Set to 32 for maximum performance
	// Covers 99%+ of HTTP responses without map overflow
	MaxHeaders = 32

	// MaxHeaderName is the maximum length of a header name
	MaxHeaderName = 128

	// MaxHeaderValue is the maximum length of a header value for inline storage
	// Set to 128 bytes - handles virtually all header values inline
	MaxHeaderValue = 128

	// MaxStatusLine is the maximum size of the status line
	MaxStatusLine = 512
)

// Memory calculation:
// - names: [32][128]byte = 4,096 bytes
// - values: [32][128]byte = 4,096 bytes
// - overhead: 32 bytes
// Total ClientHeaders: ~8,224 bytes
// Total ClientResponse: ~200 bytes
// Target total: ~9,500 B/op
//
// Trade-off: Maximum throughput, zero map allocations for 99%+ of responses
