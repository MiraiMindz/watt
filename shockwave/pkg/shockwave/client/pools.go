package client

import (
	"bufio"
	"io"
	"sync"

	"github.com/yourusername/shockwave/pkg/shockwave"
)

// Global pools for reusable objects
// All pools use sync.Pool for zero-allocation reuse
var (
	// Request pooling
	clientRequestPool = sync.Pool{
		New: func() interface{} {
			return &ClientRequest{}
		},
	}

	// Response pooling
	clientResponsePool = sync.Pool{
		New: func() interface{} {
			return &ClientResponse{}
		},
	}

	// Header pooling
	headerPool = sync.Pool{
		New: func() interface{} {
			return &ClientHeaders{}
		},
	}

	// bufio.Reader pooling (for response reading)
	bufioReaderPool = sync.Pool{
		New: func() interface{} {
			return bufio.NewReaderSize(nil, DefaultBufferSize)
		},
	}

	// bufio.Writer pooling (for request writing)
	bufioWriterPool = sync.Pool{
		New: func() interface{} {
			return bufio.NewWriterSize(nil, DefaultBufferSize)
		},
	}

	// Small buffer pool (4KB) - for general operations
	smallBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, DefaultBufferSize)
			return &buf
		},
	}

	// Large buffer pool (16KB) - for parsing
	largeBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, LargeBufferSize)
			return &buf
		},
	}
)

// ClientRequest pooling

// GetClientRequest retrieves a ClientRequest from the pool.
// The returned request has been reset and is ready for use.
//
// IMPORTANT: You MUST call PutClientRequest when done.
//
// Allocation behavior: 0 allocs/op (reuses pooled object)
func GetClientRequest() *ClientRequest {
	req := clientRequestPool.Get().(*ClientRequest)
	req.Reset()
	return req
}

// PutClientRequest returns a ClientRequest to the pool.
// The request is reset before being returned.
//
// After calling PutClientRequest, you MUST NOT use the request anymore.
//
// Allocation behavior: 0 allocs/op
func PutClientRequest(req *ClientRequest) {
	if req != nil {
		req.Reset()
		clientRequestPool.Put(req)
	}
}

// ClientResponse pooling

// GetClientResponse retrieves a ClientResponse from the pool.
// The returned response has been reset and is ready for use.
//
// IMPORTANT: You MUST call PutClientResponse when done.
//
// Allocation behavior: 0 allocs/op (reuses pooled object)
func GetClientResponse() *ClientResponse {
	resp := clientResponsePool.Get().(*ClientResponse)
	resp.Reset()
	return resp
}

// PutClientResponse returns a ClientResponse to the pool.
// The response is reset before being returned.
//
// After calling PutClientResponse, you MUST NOT use the response anymore.
//
// Allocation behavior: 0 allocs/op
func PutClientResponse(resp *ClientResponse) {
	if resp != nil {
		resp.Reset()
		clientResponsePool.Put(resp)
	}
}

// Header pooling

// GetHeaders retrieves a ClientHeaders from the pool.
// The returned headers have been reset and are ready for use.
//
// IMPORTANT: You MUST call PutHeaders when done.
//
// Allocation behavior: 0 allocs/op (reuses pooled object)
func GetHeaders() *ClientHeaders {
	h := headerPool.Get().(*ClientHeaders)
	h.Reset()
	return h
}

// PutHeaders returns ClientHeaders to the pool.
// The headers are reset before being returned.
//
// After calling PutHeaders, you MUST NOT use the headers anymore.
//
// Allocation behavior: 0 allocs/op
func PutHeaders(h *ClientHeaders) {
	if h != nil {
		h.Reset()
		headerPool.Put(h)
	}
}

// bufio pooling

// GetBufioReader retrieves a bufio.Reader from the pool.
// The reader is configured with the given io.Reader.
//
// IMPORTANT: You MUST call PutBufioReader when done.
//
// Allocation behavior: 0 allocs/op (reuses pooled reader)
func GetBufioReader(r io.Reader) *bufio.Reader {
	br := bufioReaderPool.Get().(*bufio.Reader)
	br.Reset(r)
	return br
}

// PutBufioReader returns a bufio.Reader to the pool.
// The reader is reset before being returned.
//
// After calling PutBufioReader, you MUST NOT use the reader anymore.
//
// Allocation behavior: 0 allocs/op
func PutBufioReader(br *bufio.Reader) {
	if br != nil {
		br.Reset(nil)
		bufioReaderPool.Put(br)
	}
}

// GetBufioWriter retrieves a bufio.Writer from the pool.
// The writer is configured with the given io.Writer.
//
// IMPORTANT: You MUST call PutBufioWriter when done.
//
// Allocation behavior: 0 allocs/op (reuses pooled writer)
func GetBufioWriter(w io.Writer) *bufio.Writer {
	bw := bufioWriterPool.Get().(*bufio.Writer)
	bw.Reset(w)
	return bw
}

// PutBufioWriter returns a bufio.Writer to the pool.
// The writer is flushed and reset before being returned.
//
// After calling PutBufioWriter, you MUST NOT use the writer anymore.
//
// Allocation behavior: 0 allocs/op
func PutBufioWriter(bw *bufio.Writer) {
	if bw != nil {
		bw.Flush()
		bw.Reset(nil)
		bufioWriterPool.Put(bw)
	}
}

// Buffer pooling (delegates to global buffer pool from shockwave package)

// GetBuffer retrieves a buffer of the specified size.
// Uses the global buffer pool from shockwave package for size-based pooling.
//
// IMPORTANT: You MUST call PutBuffer when done.
//
// Allocation behavior: 0 allocs/op on pool hit
func GetBuffer(size int) []byte {
	return shockwave.GetBuffer(size)
}

// PutBuffer returns a buffer to the pool.
// Delegates to the global buffer pool.
//
// After calling PutBuffer, you MUST NOT use the buffer anymore.
//
// Allocation behavior: 0 allocs/op
func PutBuffer(buf []byte) {
	shockwave.PutBuffer(buf)
}

// Note: GetSmallBuffer and PutSmallBuffer are now in buffer_opt.go
// with improved size-based pooling

// GetLargeBuffer retrieves a large buffer (16KB) from the pool.
// Returns a buffer with length 0 but capacity LargeBufferSize.
//
// IMPORTANT: You MUST call PutLargeBuffer when done.
//
// Allocation behavior: 0 allocs/op (reuses pooled buffer)
func GetLargeBuffer() []byte {
	bufPtr := largeBufferPool.Get().(*[]byte)
	buf := *bufPtr
	return buf[:0] // Reset length but keep capacity
}

// PutLargeBuffer returns a large buffer to the pool.
//
// After calling PutLargeBuffer, you MUST NOT use the buffer anymore.
//
// Allocation behavior: 0 allocs/op
func PutLargeBuffer(buf []byte) {
	if buf == nil || cap(buf) < LargeBufferSize {
		return
	}
	buf = buf[:0]
	largeBufferPool.Put(&buf)
}

// WarmupPools pre-allocates objects in all pools.
// This is useful for avoiding allocations during the first requests.
// Call this during client initialization for optimal performance.
//
// The count parameter specifies how many objects to pre-allocate per pool.
// Recommended values: 10-100 for most workloads.
func WarmupPools(count int) {
	// Warmup request pool
	for i := 0; i < count; i++ {
		req := GetClientRequest()
		PutClientRequest(req)
	}

	// Warmup response pool
	for i := 0; i < count; i++ {
		resp := GetClientResponse()
		PutClientResponse(resp)
	}

	// Warmup header pool
	for i := 0; i < count; i++ {
		h := GetHeaders()
		PutHeaders(h)
	}

	// Warmup buffer pools
	for i := 0; i < count; i++ {
		smallBuf := GetSmallBuffer(DefaultBufferSize)
		PutSmallBuffer(smallBuf)

		largeBuf := GetLargeBuffer()
		PutLargeBuffer(largeBuf)
	}

	// Warmup bufio pools
	for i := 0; i < count; i++ {
		br := GetBufioReader(nil)
		PutBufioReader(br)

		bw := GetBufioWriter(nil)
		PutBufioWriter(bw)
	}

	// Warmup global buffer pool
	shockwave.WarmupBufferPool(count)
}
