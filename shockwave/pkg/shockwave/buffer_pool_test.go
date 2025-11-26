package shockwave

import (
	"sync"
	"testing"
)

// TestBufferPoolSizes verifies correct buffer size selection
func TestBufferPoolSizes(t *testing.T) {
	pool := NewBufferPool()

	tests := []struct {
		name         string
		requestedSize int
		expectedSize  int
	}{
		{"Small 1KB", 1024, BufferSize2KB},
		{"Exact 2KB", BufferSize2KB, BufferSize2KB},
		{"Between 2KB-4KB", 3 * 1024, BufferSize4KB},
		{"Exact 4KB", BufferSize4KB, BufferSize4KB},
		{"Between 4KB-8KB", 6 * 1024, BufferSize8KB},
		{"Exact 8KB", BufferSize8KB, BufferSize8KB},
		{"Between 8KB-16KB", 12 * 1024, BufferSize16KB},
		{"Exact 16KB", BufferSize16KB, BufferSize16KB},
		{"Between 16KB-32KB", 24 * 1024, BufferSize32KB},
		{"Exact 32KB", BufferSize32KB, BufferSize32KB},
		{"Between 32KB-64KB", 48 * 1024, BufferSize64KB},
		{"Exact 64KB", BufferSize64KB, BufferSize64KB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := pool.Get(tt.requestedSize)
			defer pool.Put(buf)

			if cap(buf) < tt.requestedSize {
				t.Errorf("Buffer capacity %d < requested size %d", cap(buf), tt.requestedSize)
			}

			if cap(buf) != tt.expectedSize {
				t.Errorf("Expected buffer size %d, got %d", tt.expectedSize, cap(buf))
			}
		})
	}
}

// TestBufferPoolLargeSize verifies buffers larger than 64KB are allocated directly
func TestBufferPoolLargeSize(t *testing.T) {
	pool := NewBufferPool()

	// Request 128KB buffer (too large for pooling)
	buf := pool.Get(128 * 1024)
	if len(buf) != 128*1024 {
		t.Errorf("Expected buffer length 128KB, got %d", len(buf))
	}

	// This should not affect pool metrics significantly
	pool.Put(buf)
}

// TestBufferPoolReuse verifies buffer reuse
func TestBufferPoolReuse(t *testing.T) {
	pool := NewBufferPool()
	pool.ResetMetrics()

	// Get and put a 4KB buffer
	buf1 := pool.Get(4096)
	pool.Put(buf1)

	// Get another 4KB buffer - should reuse
	buf2 := pool.Get(4096)
	pool.Put(buf2)

	metrics := pool.GetMetrics()

	// We should have at least 1 hit (second Get reused first buffer)
	if metrics.Pool4KB.Hits < 1 {
		t.Errorf("Expected at least 1 hit, got %d", metrics.Pool4KB.Hits)
	}

	// Hit rate should be >0%
	if metrics.Pool4KB.HitRate == 0 {
		t.Error("Expected non-zero hit rate")
	}
}

// TestBufferPoolMetrics verifies metrics tracking
func TestBufferPoolMetrics(t *testing.T) {
	pool := NewBufferPool()
	pool.ResetMetrics()

	// Perform operations
	const iterations = 100

	for i := 0; i < iterations; i++ {
		buf := pool.Get(4096)
		pool.Put(buf)
	}

	metrics := pool.GetMetrics()

	// Verify total operations
	if metrics.Pool4KB.Gets != iterations {
		t.Errorf("Expected %d gets, got %d", iterations, metrics.Pool4KB.Gets)
	}

	if metrics.Pool4KB.Puts != iterations {
		t.Errorf("Expected %d puts, got %d", iterations, metrics.Pool4KB.Puts)
	}

	// Verify hits + misses = iterations
	totalOps := metrics.Pool4KB.Hits + metrics.Pool4KB.Misses
	if totalOps != iterations {
		t.Errorf("Expected %d total ops (hits+misses), got %d", iterations, totalOps)
	}

	// Hit rate should be high (>90% since we're reusing)
	if metrics.Pool4KB.HitRate < 90.0 {
		t.Errorf("Expected hit rate >90%%, got %.2f%%", metrics.Pool4KB.HitRate)
	}

	// Should have very few misses (ideally 1, but may vary)
	if metrics.Pool4KB.Misses > 10 {
		t.Errorf("Expected ≤10 misses, got %d", metrics.Pool4KB.Misses)
	}
}

// TestBufferPoolConcurrent verifies thread safety
func TestBufferPoolConcurrent(t *testing.T) {
	pool := NewBufferPool()
	pool.ResetMetrics()

	const (
		goroutines = 100
		iterations = 1000
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				buf := pool.Get(4096)
				// Simulate some work
				buf[0] = byte(j)
				pool.Put(buf)
			}
		}()
	}

	wg.Wait()

	metrics := pool.GetMetrics()

	// Verify total operations
	expectedTotal := uint64(goroutines * iterations)
	if metrics.Pool4KB.Gets != expectedTotal {
		t.Errorf("Expected %d gets, got %d", expectedTotal, metrics.Pool4KB.Gets)
	}

	if metrics.Pool4KB.Puts != expectedTotal {
		t.Errorf("Expected %d puts, got %d", expectedTotal, metrics.Pool4KB.Puts)
	}

	// Hit rate should be high (>90%)
	if metrics.Pool4KB.HitRate < 90.0 {
		t.Errorf("Expected hit rate >90%%, got %.2f%%", metrics.Pool4KB.HitRate)
	}
}

// TestBufferPoolReset verifies PutWithReset zeros the buffer
func TestBufferPoolReset(t *testing.T) {
	pool := NewBufferPool()

	// Get buffer and fill with data
	buf := pool.Get(4096)
	for i := range buf {
		buf[i] = 0xFF
	}

	// Put with reset
	pool.PutWithReset(buf)

	// Get another buffer (should be the same one, now zeroed)
	buf2 := pool.Get(4096)
	defer pool.Put(buf2)

	// Verify it's zeroed
	for i, b := range buf2 {
		if b != 0 {
			t.Errorf("Buffer not zeroed at index %d: expected 0, got %d", i, b)
			break
		}
	}
}

// TestBufferPoolWarmup verifies warmup pre-allocates buffers
func TestBufferPoolWarmup(t *testing.T) {
	pool := NewBufferPool()
	pool.ResetMetrics()

	// Warmup with 10 buffers per pool
	pool.Warmup(10)

	// All pools should have 10 gets and 10 puts
	metrics := pool.GetMetrics()

	pools := []SizedPoolMetrics{
		metrics.Pool2KB, metrics.Pool4KB, metrics.Pool8KB,
		metrics.Pool16KB, metrics.Pool32KB, metrics.Pool64KB,
	}

	for i, pm := range pools {
		if pm.Gets != 10 {
			t.Errorf("Pool %d: expected 10 gets, got %d", i, pm.Gets)
		}
		if pm.Puts != 10 {
			t.Errorf("Pool %d: expected 10 puts, got %d", i, pm.Puts)
		}
	}
}

// TestBufferPoolGlobalFunctions verifies global convenience functions
func TestBufferPoolGlobalFunctions(t *testing.T) {
	ResetBufferPoolMetrics()

	// Use global functions
	buf := GetBuffer(4096)
	PutBuffer(buf)

	metrics := GetBufferPoolMetrics()

	if metrics.Pool4KB.Gets != 1 {
		t.Errorf("Expected 1 get, got %d", metrics.Pool4KB.Gets)
	}

	if metrics.Pool4KB.Puts != 1 {
		t.Errorf("Expected 1 put, got %d", metrics.Pool4KB.Puts)
	}
}

// TestBufferPoolWrongSize verifies discards for wrong-sized buffers
func TestBufferPoolWrongSize(t *testing.T) {
	pool := NewBufferPool()
	pool.ResetMetrics()

	// Create a buffer smaller than the smallest pool size (2KB)
	// This should be discarded by all pools
	tinyBuf := make([]byte, 1024) // 1KB - too small for any pool

	// Try to put it - should be discarded
	pool.Put(tinyBuf)

	metrics := pool.GetMetrics()

	// Should not be accepted by any pool (no puts recorded for properly sized buffers)
	totalPuts := metrics.Pool2KB.Puts + metrics.Pool4KB.Puts + metrics.Pool8KB.Puts +
		metrics.Pool16KB.Puts + metrics.Pool32KB.Puts + metrics.Pool64KB.Puts

	if totalPuts != 0 {
		t.Errorf("Expected 0 puts (buffer too small), got %d", totalPuts)
	}
}

// Benchmarks

// BenchmarkBufferPool_Get measures Get performance
func BenchmarkBufferPool_Get(b *testing.B) {
	pool := NewBufferPool()
	pool.Warmup(100) // Pre-warm to measure steady-state

	sizes := []int{
		BufferSize2KB,
		BufferSize4KB,
		BufferSize8KB,
		BufferSize16KB,
		BufferSize32KB,
		BufferSize64KB,
	}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				buf := pool.Get(size)
				pool.Put(buf)
			}
		})
	}
}

// BenchmarkBufferPool_GetNoPool measures allocation without pooling
func BenchmarkBufferPool_GetNoPool(b *testing.B) {
	sizes := []int{
		BufferSize2KB,
		BufferSize4KB,
		BufferSize8KB,
		BufferSize16KB,
		BufferSize32KB,
		BufferSize64KB,
	}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				buf := make([]byte, size)
				_ = buf
			}
		})
	}
}

// BenchmarkBufferPool_Parallel measures parallel Get/Put performance
func BenchmarkBufferPool_Parallel(b *testing.B) {
	pool := NewBufferPool()
	pool.Warmup(1000)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(BufferSize4KB)
			pool.Put(buf)
		}
	})
}

// BenchmarkBufferPool_HitRate measures hit rate under load
func BenchmarkBufferPool_HitRate(b *testing.B) {
	pool := NewBufferPool()
	pool.ResetMetrics()
	pool.Warmup(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get(BufferSize4KB)
		pool.Put(buf)
	}

	b.StopTimer()

	metrics := pool.GetMetrics()
	b.ReportMetric(metrics.Pool4KB.HitRate, "%hit")
	b.ReportMetric(float64(metrics.Pool4KB.Misses), "misses")
}

// BenchmarkBufferPool_MixedSizes measures performance with mixed buffer sizes
func BenchmarkBufferPool_MixedSizes(b *testing.B) {
	pool := NewBufferPool()
	pool.Warmup(100)

	sizes := []int{
		BufferSize2KB,
		BufferSize4KB,
		BufferSize8KB,
		BufferSize16KB,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		buf := pool.Get(size)
		pool.Put(buf)
	}
}

// BenchmarkBufferPool_Reset measures PutWithReset performance
func BenchmarkBufferPool_Reset(b *testing.B) {
	pool := NewBufferPool()
	pool.Warmup(100)

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get(BufferSize4KB)
		pool.PutWithReset(buf)
	}
}

// BenchmarkBufferPool_Global measures global function performance
func BenchmarkBufferPool_Global(b *testing.B) {
	WarmupBufferPool(100)

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := GetBuffer(BufferSize4KB)
		PutBuffer(buf)
	}
}

// Helper functions

func formatSize(size int) string {
	switch size {
	case BufferSize2KB:
		return "2KB"
	case BufferSize4KB:
		return "4KB"
	case BufferSize8KB:
		return "8KB"
	case BufferSize16KB:
		return "16KB"
	case BufferSize32KB:
		return "32KB"
	case BufferSize64KB:
		return "64KB"
	default:
		return "Unknown"
	}
}
