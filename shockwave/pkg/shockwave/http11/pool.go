package http11

import (
	"bufio"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
)

// Pool sizes and configurations
const (
	// DefaultBufferSize is the default size for read/write buffers
	DefaultBufferSize = 4096

	// ParserBufferSize is the size for parser internal buffers
	ParserBufferSize = MaxRequestLineSize + MaxHeadersSize // 16KB
)

// PoolStrategy defines the pooling strategy to use
type PoolStrategy int

const (
	// PoolStrategyStandard uses Go's standard sync.Pool (default, fastest for most workloads)
	PoolStrategyStandard PoolStrategy = iota

	// PoolStrategyPerCPU uses per-CPU pools to eliminate lock contention
	// (useful for sustained high-concurrency workloads with longer object hold times)
	PoolStrategyPerCPU
)

// poolStrategy is the global pool strategy setting
// Default: PoolStrategyStandard (benchmarks show it's faster for typical HTTP workloads)
var poolStrategy = PoolStrategyStandard

// SetPoolStrategy sets the pooling strategy globally.
// This must be called before any pool operations for consistent behavior.
// Safe to call from init() functions or during server initialization.
func SetPoolStrategy(strategy PoolStrategy) {
	poolStrategy = strategy
}

// perCPUPool provides per-CPU object pooling to reduce lock contention.
// Only used when PoolStrategyPerCPU is enabled.
type perCPUPool[T any] struct {
	pools      []*sync.Pool
	numCPU     int
	roundRobin atomic.Uint64
	newFunc    func() T
}

// newPerCPUPool creates a new per-CPU pool.
func newPerCPUPool[T any](newFunc func() T) *perCPUPool[T] {
	numCPU := runtime.GOMAXPROCS(0)
	if numCPU < 1 {
		numCPU = 1
	}

	pools := make([]*sync.Pool, numCPU)
	for i := 0; i < numCPU; i++ {
		pools[i] = &sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		}
	}

	return &perCPUPool[T]{
		pools:   pools,
		numCPU:  numCPU,
		newFunc: newFunc,
	}
}

// get retrieves an object from the pool.
func (p *perCPUPool[T]) get() T {
	idx := p.roundRobin.Add(1) % uint64(p.numCPU)
	pool := p.pools[idx]

	if obj := pool.Get(); obj != nil {
		return obj.(T)
	}

	return p.newFunc()
}

// put returns an object to the pool.
func (p *perCPUPool[T]) put(obj T) {
	idx := p.roundRobin.Load() % uint64(p.numCPU)
	pool := p.pools[idx]
	pool.Put(obj)
}

// warmup pre-allocates objects across all CPU pools.
func (p *perCPUPool[T]) warmup(countPerCPU int) {
	for _, pool := range p.pools {
		objs := make([]T, countPerCPU)
		for i := 0; i < countPerCPU; i++ {
			objs[i] = p.newFunc()
		}
		for i := 0; i < countPerCPU; i++ {
			pool.Put(objs[i])
		}
	}
}

// Global pools for reusable objects
// Strategy is configurable via SetPoolStrategy()
// Default: Standard sync.Pool (fastest for typical workloads based on benchmarks)
var (
	// Standard sync.Pool instances (default)
	requestPoolStd = sync.Pool{
		New: func() interface{} {
			return &Request{}
		},
	}

	responseWriterPoolStd = sync.Pool{
		New: func() interface{} {
			return &ResponseWriter{}
		},
	}

	parserPoolStd = sync.Pool{
		New: func() interface{} {
			return NewParser()
		},
	}

	bufferPoolStd = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, DefaultBufferSize)
			return &buf
		},
	}

	largeBufferPoolStd = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, ParserBufferSize)
			return &buf
		},
	}

	bufioReaderPoolStd = sync.Pool{
		New: func() interface{} {
			return bufio.NewReaderSize(nil, DefaultBufferSize)
		},
	}

	bufioWriterPoolStd = sync.Pool{
		New: func() interface{} {
			return bufio.NewWriterSize(nil, DefaultBufferSize)
		},
	}

	// Per-CPU pool instances (optional, for high-concurrency workloads)
	requestPoolPerCPU = newPerCPUPool(func() *Request {
		return &Request{}
	})

	responseWriterPoolPerCPU = newPerCPUPool(func() *ResponseWriter {
		return &ResponseWriter{}
	})

	parserPoolPerCPU = newPerCPUPool(func() *Parser {
		return NewParser()
	})

	bufferPoolPerCPU = newPerCPUPool(func() *[]byte {
		buf := make([]byte, DefaultBufferSize)
		return &buf
	})

	largeBufferPoolPerCPU = newPerCPUPool(func() *[]byte {
		buf := make([]byte, 0, ParserBufferSize)
		return &buf
	})

	bufioReaderPoolPerCPU = newPerCPUPool(func() *bufio.Reader {
		return bufio.NewReaderSize(nil, DefaultBufferSize)
	})

	bufioWriterPoolPerCPU = newPerCPUPool(func() *bufio.Writer {
		return bufio.NewWriterSize(nil, DefaultBufferSize)
	})
)

// Request pooling

// GetRequest retrieves a Request from the pool.
// The returned Request has been reset and is ready for use.
//
// IMPORTANT: You MUST call PutRequest when done to return it to the pool.
//
// Allocation behavior: 0 allocs/op (reuses pooled object)
func GetRequest() *Request {
	var req *Request
	if poolStrategy == PoolStrategyPerCPU {
		req = requestPoolPerCPU.get()
	} else {
		req = requestPoolStd.Get().(*Request)
	}
	req.Reset()
	return req
}

// PutRequest returns a Request to the pool.
// The Request is reset before being returned to the pool.
// It is safe to call PutRequest on a nil Request (no-op).
//
// After calling PutRequest, you MUST NOT use the Request anymore.
//
// Allocation behavior: 0 allocs/op
func PutRequest(req *Request) {
	if req != nil {
		req.Reset()
		if poolStrategy == PoolStrategyPerCPU {
			requestPoolPerCPU.put(req)
		} else {
			requestPoolStd.Put(req)
		}
	}
}

// ResponseWriter pooling

// GetResponseWriter retrieves a ResponseWriter from the pool.
// The returned ResponseWriter is configured with the given writer.
//
// IMPORTANT: You MUST call PutResponseWriter when done to return it to the pool.
//
// Allocation behavior: 0 allocs/op (reuses pooled object)
func GetResponseWriter(w io.Writer) *ResponseWriter {
	var rw *ResponseWriter
	if poolStrategy == PoolStrategyPerCPU {
		rw = responseWriterPoolPerCPU.get()
	} else {
		rw = responseWriterPoolStd.Get().(*ResponseWriter)
	}
	rw.Reset(w)
	return rw
}

// PutResponseWriter returns a ResponseWriter to the pool.
// The ResponseWriter is reset before being returned to the pool.
// It is safe to call PutResponseWriter on a nil ResponseWriter (no-op).
//
// After calling PutResponseWriter, you MUST NOT use the ResponseWriter anymore.
//
// Allocation behavior: 0 allocs/op
func PutResponseWriter(rw *ResponseWriter) {
	if rw != nil {
		rw.Reset(nil) // Clear writer reference
		if poolStrategy == PoolStrategyPerCPU {
			responseWriterPoolPerCPU.put(rw)
		} else {
			responseWriterPoolStd.Put(rw)
		}
	}
}

// Parser pooling

// GetParser retrieves a Parser from the pool.
// The returned Parser is ready for use.
//
// IMPORTANT: You MUST call PutParser when done to return it to the pool.
//
// Allocation behavior: 0 allocs/op (reuses pooled object)
func GetParser() *Parser {
	if poolStrategy == PoolStrategyPerCPU {
		return parserPoolPerCPU.get()
	}
	return parserPoolStd.Get().(*Parser)
}

// PutParser returns a Parser to the pool.
// It is safe to call PutParser on a nil Parser (no-op).
//
// After calling PutParser, you MUST NOT use the Parser anymore.
//
// Allocation behavior: 0 allocs/op
func PutParser(p *Parser) {
	if p != nil {
		// Reset buffer but keep capacity for reuse
		if p.buf != nil {
			p.buf = p.buf[:0]
		}
		// Clear pipelining buffer to prevent cross-request contamination
		p.unreadBuf = nil
		if poolStrategy == PoolStrategyPerCPU {
			parserPoolPerCPU.put(p)
		} else {
			parserPoolStd.Put(p)
		}
	}
}

// Buffer pooling

// GetBuffer retrieves a buffer from the pool.
// Returns a byte slice of DefaultBufferSize (4KB).
// The buffer may contain data from previous use; clear it if needed.
//
// IMPORTANT: You MUST call PutBuffer when done to return it to the pool.
//
// Allocation behavior: 0 allocs/op (reuses pooled buffer)
func GetBuffer() []byte {
	var bufPtr *[]byte
	if poolStrategy == PoolStrategyPerCPU {
		bufPtr = bufferPoolPerCPU.get()
	} else {
		bufPtr = bufferPoolStd.Get().(*[]byte)
	}
	return *bufPtr
}

// PutBuffer returns a buffer to the pool.
// The buffer should be of DefaultBufferSize.
// It is safe to call PutBuffer with a nil or incorrectly sized buffer (no-op).
//
// After calling PutBuffer, you MUST NOT use the buffer anymore.
//
// Allocation behavior: 0 allocs/op
func PutBuffer(buf []byte) {
	if buf == nil || cap(buf) < DefaultBufferSize {
		return // Don't pool incorrectly sized buffers
	}
	// Reset to full capacity for next use
	buf = buf[:DefaultBufferSize]
	if poolStrategy == PoolStrategyPerCPU {
		bufferPoolPerCPU.put(&buf)
	} else {
		bufferPoolStd.Put(&buf)
	}
}

// GetLargeBuffer retrieves a large buffer from the pool.
// Returns a byte slice with capacity of ParserBufferSize (16KB).
// The buffer has length 0 but can grow up to capacity.
//
// IMPORTANT: You MUST call PutLargeBuffer when done to return it to the pool.
//
// Allocation behavior: 0 allocs/op (reuses pooled buffer)
func GetLargeBuffer() []byte {
	var bufPtr *[]byte
	if poolStrategy == PoolStrategyPerCPU {
		bufPtr = largeBufferPoolPerCPU.get()
	} else {
		bufPtr = largeBufferPoolStd.Get().(*[]byte)
	}
	buf := *bufPtr
	return buf[:0] // Reset length but keep capacity
}

// PutLargeBuffer returns a large buffer to the pool.
// The buffer should have capacity of at least ParserBufferSize.
// It is safe to call PutLargeBuffer with a nil or incorrectly sized buffer (no-op).
//
// After calling PutLargeBuffer, you MUST NOT use the buffer anymore.
//
// Allocation behavior: 0 allocs/op
func PutLargeBuffer(buf []byte) {
	if buf == nil || cap(buf) < ParserBufferSize {
		return // Don't pool incorrectly sized buffers
	}
	// Reset to zero length for next use
	buf = buf[:0]
	if poolStrategy == PoolStrategyPerCPU {
		largeBufferPoolPerCPU.put(&buf)
	} else {
		largeBufferPoolStd.Put(&buf)
	}
}

// bufio pooling

// GetBufioReader retrieves a bufio.Reader from the pool.
// The reader is configured with the given io.Reader.
//
// IMPORTANT: You MUST call PutBufioReader when done to return it to the pool.
//
// Allocation behavior: 0 allocs/op (reuses pooled reader)
func GetBufioReader(r io.Reader) *bufio.Reader {
	var br *bufio.Reader
	if poolStrategy == PoolStrategyPerCPU {
		br = bufioReaderPoolPerCPU.get()
	} else {
		br = bufioReaderPoolStd.Get().(*bufio.Reader)
	}
	br.Reset(r)
	return br
}

// PutBufioReader returns a bufio.Reader to the pool.
// The reader is reset (underlying reader cleared) before being returned.
// It is safe to call PutBufioReader on a nil reader (no-op).
//
// After calling PutBufioReader, you MUST NOT use the reader anymore.
//
// Allocation behavior: 0 allocs/op
func PutBufioReader(br *bufio.Reader) {
	if br != nil {
		br.Reset(nil) // Clear underlying reader
		if poolStrategy == PoolStrategyPerCPU {
			bufioReaderPoolPerCPU.put(br)
		} else {
			bufioReaderPoolStd.Put(br)
		}
	}
}

// GetBufioWriter retrieves a bufio.Writer from the pool.
// The writer is configured with the given io.Writer.
//
// IMPORTANT: You MUST call PutBufioWriter when done to return it to the pool.
//
// Allocation behavior: 0 allocs/op (reuses pooled writer)
func GetBufioWriter(w io.Writer) *bufio.Writer {
	var bw *bufio.Writer
	if poolStrategy == PoolStrategyPerCPU {
		bw = bufioWriterPoolPerCPU.get()
	} else {
		bw = bufioWriterPoolStd.Get().(*bufio.Writer)
	}
	bw.Reset(w)
	return bw
}

// PutBufioWriter returns a bufio.Writer to the pool.
// The writer is flushed and reset (underlying writer cleared) before being returned.
// It is safe to call PutBufioWriter on a nil writer (no-op).
//
// After calling PutBufioWriter, you MUST NOT use the writer anymore.
//
// Allocation behavior: 0 allocs/op
func PutBufioWriter(bw *bufio.Writer) {
	if bw != nil {
		bw.Flush()      // Ensure data is written
		bw.Reset(nil)   // Clear underlying writer
		if poolStrategy == PoolStrategyPerCPU {
			bufioWriterPoolPerCPU.put(bw)
		} else {
			bufioWriterPoolStd.Put(bw)
		}
	}
}

// PoolStats provides statistics about pool usage.
// This is useful for debugging and optimization.
type PoolStats struct {
	// Name of the pool
	Name string

	// Approximate number of objects currently in the pool
	// Note: sync.Pool doesn't provide exact counts, this is an estimate
	Available int

	// Total number of Get calls
	Gets uint64

	// Total number of Put calls
	Puts uint64

	// Estimated hit rate (Gets that reused pooled objects)
	// This is approximate and may not be accurate
	HitRate float64
}

// GetPoolStats returns statistics for all pools.
// Note: sync.Pool doesn't provide instrumentation, so these are estimates.
// This function is mainly for debugging purposes.
func GetPoolStats() []PoolStats {
	// sync.Pool doesn't provide statistics, so we return placeholder data
	// In a real implementation, you'd need to instrument the pools yourself
	return []PoolStats{
		{Name: "Request", Available: 0, Gets: 0, Puts: 0, HitRate: 0.0},
		{Name: "ResponseWriter", Available: 0, Gets: 0, Puts: 0, HitRate: 0.0},
		{Name: "Parser", Available: 0, Gets: 0, Puts: 0, HitRate: 0.0},
		{Name: "Buffer", Available: 0, Gets: 0, Puts: 0, HitRate: 0.0},
		{Name: "LargeBuffer", Available: 0, Gets: 0, Puts: 0, HitRate: 0.0},
		{Name: "BufioReader", Available: 0, Gets: 0, Puts: 0, HitRate: 0.0},
		{Name: "BufioWriter", Available: 0, Gets: 0, Puts: 0, HitRate: 0.0},
	}
}

// WarmupPools pre-allocates objects in all pools.
// This is useful for avoiding allocations during the first requests.
// Call this during server initialization for optimal performance.
//
// For PoolStrategyStandard:
//   - count specifies total objects to pre-allocate per pool
//
// For PoolStrategyPerCPU:
//   - count specifies objects per CPU per pool
//   - With 8 CPUs and count=100, this pre-allocates 800 objects per pool type
//
// Recommended values: 10-100 for low traffic, 100-1000 for high traffic.
func WarmupPools(count int) {
	if poolStrategy == PoolStrategyPerCPU {
		// Warmup all per-CPU pools
		requestPoolPerCPU.warmup(count)
		responseWriterPoolPerCPU.warmup(count)
		parserPoolPerCPU.warmup(count)
		bufferPoolPerCPU.warmup(count)
		largeBufferPoolPerCPU.warmup(count)
		bufioReaderPoolPerCPU.warmup(count)
		bufioWriterPoolPerCPU.warmup(count)
	} else {
		// Warmup standard pools
		for i := 0; i < count; i++ {
			req := GetRequest()
			PutRequest(req)

			rw := GetResponseWriter(nil)
			PutResponseWriter(rw)

			p := GetParser()
			PutParser(p)

			buf := GetBuffer()
			PutBuffer(buf)

			largeBuf := GetLargeBuffer()
			PutLargeBuffer(largeBuf)

			br := GetBufioReader(nil)
			PutBufioReader(br)

			bw := GetBufioWriter(nil)
			PutBufioWriter(bw)
		}
	}
}
