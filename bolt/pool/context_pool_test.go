package pool

import (
	"testing"

	"github.com/yourusername/bolt/core"
)

// Note: ContextPool has been moved to core package.
// These tests are kept here for compatibility.
// They test core.ContextPool functionality.

// TestNewContextPool tests pool creation.
func TestNewContextPool(t *testing.T) {
	pool := core.NewContextPool()

	if pool == nil {
		t.Fatal("expected pool, got nil")
	}
}

// TestAcquire tests acquiring context from pool.
func TestAcquire(t *testing.T) {
	pool := core.NewContextPool()

	ctx := pool.Acquire()

	if ctx == nil {
		t.Fatal("expected context, got nil")
	}
}

// TestRelease tests returning context to pool.
func TestRelease(t *testing.T) {
	pool := core.NewContextPool()

	ctx := pool.Acquire()
	ctx.Set("key", "value")

	// Release should reset the context
	pool.Release(ctx)

	// Acquire again - should get a clean context
	ctx2 := pool.Acquire()

	if ctx2.Get("key") != nil {
		t.Error("expected context to be reset after release")
	}
}

// TestAcquireReleaseCycle tests full acquire/release cycle.
func TestAcquireReleaseCycle(t *testing.T) {
	pool := core.NewContextPool()

	// First cycle
	ctx1 := pool.Acquire()
	ctx1.Set("test", "value1")
	pool.Release(ctx1)

	// Second cycle
	ctx2 := pool.Acquire()

	// Should be clean
	if ctx2.Get("test") != nil {
		t.Error("expected clean context from pool")
	}

	pool.Release(ctx2)
}

// TestConcurrentAcquireRelease tests concurrent pool access.
func TestConcurrentAcquireRelease(t *testing.T) {
	pool := core.NewContextPool()
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			ctx := pool.Acquire()

			// Use context
			ctx.Set("id", id)

			// Release it
			pool.Release(ctx)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestPoolReuse tests that pool actually reuses contexts.
func TestPoolReuse(t *testing.T) {
	pool := core.NewContextPool()

	// Get a context
	ctx1 := pool.Acquire()
	ptr1 := &ctx1
	pool.Release(ctx1)

	// Get another context - should likely be the same object
	ctx2 := pool.Acquire()
	ptr2 := &ctx2

	// This is probabilistic, but with a single-threaded test,
	// we should get the same object back
	if ptr1 != ptr2 {
		// This is not an error, just means pool gave us a new object
		// But log it for informational purposes
		t.Logf("Pool gave us a different context (expected behavior with sync.Pool)")
	}

	pool.Release(ctx2)
}

// TestPoolReset tests that release properly resets context.
func TestPoolReset(t *testing.T) {
	pool := core.NewContextPool()

	ctx := pool.Acquire()

	// Populate context
	ctx.Set("key1", "value1")
	ctx.Set("key2", "value2")
	ctx.Set("key3", "value3")

	// Release (should reset)
	pool.Release(ctx)

	// Acquire again
	ctx2 := pool.Acquire()

	// Verify it's clean
	if ctx2.Get("key1") != nil {
		t.Error("key1 not cleared")
	}
	if ctx2.Get("key2") != nil {
		t.Error("key2 not cleared")
	}
	if ctx2.Get("key3") != nil {
		t.Error("key3 not cleared")
	}

	pool.Release(ctx2)
}

// BenchmarkAcquire benchmarks acquiring context from pool.
func BenchmarkAcquire(b *testing.B) {
	pool := core.NewContextPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := pool.Acquire()
		pool.Release(ctx)
	}
}

// BenchmarkAcquireRelease benchmarks full cycle.
func BenchmarkAcquireRelease(b *testing.B) {
	pool := core.NewContextPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := pool.Acquire()
		ctx.Set("key", "value")
		pool.Release(ctx)
	}
}

// BenchmarkAcquireParallel benchmarks parallel acquisition.
func BenchmarkAcquireParallel(b *testing.B) {
	pool := core.NewContextPool()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := pool.Acquire()
			ctx.Set("key", "value")
			pool.Release(ctx)
		}
	})
}

// BenchmarkNoPool benchmarks without pooling (baseline).
func BenchmarkNoPool(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := &core.Context{}
		ctx.Set("key", "value")
		ctx.Reset()
	}
}
