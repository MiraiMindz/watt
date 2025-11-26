package http11

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Benchmark per-CPU pool vs standard sync.Pool under high concurrency

// Standard sync.Pool for comparison
var standardRequestPool = sync.Pool{
	New: func() interface{} {
		return &Request{}
	},
}

// Benchmark standard sync.Pool (baseline)
func BenchmarkPool_Standard_LowConcurrency(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := standardRequestPool.Get().(*Request)
			req.Reset()
			standardRequestPool.Put(req)
		}
	})
}

// Benchmark per-CPU pool (optimized)
func BenchmarkPool_PerCPU_LowConcurrency(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := GetRequest()
			PutRequest(req)
		}
	})
}

// Benchmark standard sync.Pool with high contention
func BenchmarkPool_Standard_HighContention(b *testing.B) {
	var counter atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := standardRequestPool.Get().(*Request)
			req.Reset()

			// Simulate some work
			counter.Add(1)

			standardRequestPool.Put(req)
		}
	})
}

// Benchmark per-CPU pool with high contention
func BenchmarkPool_PerCPU_HighContention(b *testing.B) {
	var counter atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := GetRequest()

			// Simulate some work
			counter.Add(1)

			PutRequest(req)
		}
	})
}

// Benchmark mixed pool usage (Get+Put for multiple types)
func BenchmarkPool_Standard_MixedUsage(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Get and put request
			req := standardRequestPool.Get().(*Request)
			standardRequestPool.Put(req)

			// Get and put buffer
			buf := GetBuffer()
			PutBuffer(buf)

			// Get and put large buffer
			largeBuf := GetLargeBuffer()
			PutLargeBuffer(largeBuf)
		}
	})
}

// Benchmark per-CPU pool mixed usage
func BenchmarkPool_PerCPU_MixedUsage(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Get and put request
			req := GetRequest()
			PutRequest(req)

			// Get and put buffer
			buf := GetBuffer()
			PutBuffer(buf)

			// Get and put large buffer
			largeBuf := GetLargeBuffer()
			PutLargeBuffer(largeBuf)
		}
	})
}

// Realistic HTTP request handling simulation
func BenchmarkPool_Standard_RealisticHTTP(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate HTTP request handling with pooled objects
			req := standardRequestPool.Get().(*Request)
			req.Reset()

			// Simulate header parsing
			req.Header.Set([]byte("Content-Type"), []byte("application/json"))
			req.Header.Set([]byte("Host"), []byte("localhost"))

			// Return to pool
			standardRequestPool.Put(req)
		}
	})
}

// Realistic HTTP request handling with per-CPU pools
func BenchmarkPool_PerCPU_RealisticHTTP(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate HTTP request handling with per-CPU pooled objects
			req := GetRequest()

			// Simulate header parsing
			req.Header.Set([]byte("Content-Type"), []byte("application/json"))
			req.Header.Set([]byte("Host"), []byte("localhost"))

			// Return to pool
			PutRequest(req)
		}
	})
}

// Benchmark warmup effectiveness
func BenchmarkPool_PerCPU_WithWarmup(b *testing.B) {
	// Warmup pools with 100 objects per CPU
	WarmupPools(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := GetRequest()
			PutRequest(req)
		}
	})
}

// Benchmark without warmup (cold start)
func BenchmarkPool_PerCPU_NoWarmup(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := GetRequest()
			PutRequest(req)
		}
	})
}
