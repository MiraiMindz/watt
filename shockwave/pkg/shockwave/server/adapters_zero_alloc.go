//go:build !goexperiment.arenas && !greenteagc
// +build !goexperiment.arenas,!greenteagc

package server

import (
	"github.com/yourusername/shockwave/pkg/shockwave/http11"
)

// adapterPair holds pre-allocated adapters for zero-allocation request handling.
// This struct is embedded in the connection handler to avoid per-request allocations.
type adapterPair struct {
	reqAdapter requestAdapter
	rwAdapter  responseWriterAdapter
}

// Reset clears the adapter pair for reuse
func (ap *adapterPair) Reset() {
	ap.reqAdapter.req = nil
	ap.rwAdapter.rw = nil
}

// Setup initializes the adapter pair with request/response
func (ap *adapterPair) Setup(req *http11.Request, rw *http11.ResponseWriter) {
	// Setup adapters without header caching to avoid circular references
	ap.reqAdapter.req = req
	ap.rwAdapter.rw = rw
}

// GetRequestAdapter returns the configured request adapter
func (ap *adapterPair) GetRequestAdapter() *requestAdapter {
	return &ap.reqAdapter
}

// GetResponseWriterAdapter returns the configured response writer adapter
func (ap *adapterPair) GetResponseWriterAdapter() *responseWriterAdapter {
	return &ap.rwAdapter
}
