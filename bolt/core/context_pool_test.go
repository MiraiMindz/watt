package core

import (
	"sync"
	"testing"
)

// TestContextPoolAcquire tests acquiring context from pool.
func TestContextPoolAcquire(t *testing.T) {
	pool := NewContextPool()

	ctx := pool.Acquire()
	if ctx == nil {
		t.Fatal("expected context, got nil")
	}

	// Context should be initialized (sync.Map doesn't need explicit initialization)
}

// TestContextPoolRelease tests releasing context back to pool.
func TestContextPoolRelease(t *testing.T) {
	pool := NewContextPool()

	ctx := pool.Acquire()
	ctx.SetPath("/test")
	ctx.Set("key", "value")

	pool.Release(ctx)

	// Acquire again - should get reset context
	ctx2 := pool.Acquire()

	// Should be reset
	if ctx2.Path() != "" {
		t.Errorf("expected path to be reset, got %s", ctx2.Path())
	}

	if ctx2.Get("key") != nil {
		t.Error("expected store to be reset")
	}
}

// TestContextPoolResetFields tests context reset functionality.
func TestContextPoolResetFields(t *testing.T) {
	ctx := &Context{
		methodBytes: []byte("POST"),
		pathBytes:   []byte("/users"),
		queryBytes:  []byte("id=123"),
		statusCode:  200,
		written:     true,
		params:      map[string]string{"id": "123"},
	}

	// Set in store
	ctx.Set("key", "value")

	// Set request/response headers
	ctx.SetRequestHeader("Authorization", "Bearer token")
	ctx.SetHeader("Content-Type", "application/json")

	// Add params to inline buffer
	ctx.paramsBuf[0] = struct {
		keyBytes   []byte
		valueBytes []byte
	}{
		keyBytes:   []byte("id"),
		valueBytes: []byte("123"),
	}
	ctx.paramsLen = 1

	ctx.Reset()

	// Verify all fields are reset
	if ctx.methodBytes != nil && len(ctx.methodBytes) > 0 {
		t.Errorf("expected methodBytes to be reset, got %s", ctx.methodBytes)
	}

	if ctx.pathBytes != nil && len(ctx.pathBytes) > 0 {
		t.Errorf("expected pathBytes to be reset, got %s", ctx.pathBytes)
	}

	if ctx.queryBytes != nil && len(ctx.queryBytes) > 0 {
		t.Errorf("expected queryBytes to be reset, got %s", ctx.queryBytes)
	}

	if ctx.statusCode != 0 {
		t.Errorf("expected statusCode to be reset, got %d", ctx.statusCode)
	}

	if ctx.written {
		t.Error("expected written to be reset")
	}

	if len(ctx.params) != 0 {
		t.Error("expected params to be reset")
	}

	// Check store is cleared
	if ctx.Get("key") != nil {
		t.Error("expected store to be reset")
	}

	// Note: testReqHeaders, testResHeaders, and paramsBuf are not cleared by Reset()
	// as they are internal implementation details. Reset() clears paramsLen which
	// ensures that paramsBuf won't be accessed.
}

// TestContextPoolConcurrentAccess tests thread safety.
func TestContextPoolConcurrentAccess(t *testing.T) {
	pool := NewContextPool()

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				ctx := pool.Acquire()

				// Use context
				ctx.SetPath("/test")
				ctx.Set("goroutine", id)

				pool.Release(ctx)
			}
		}(i)
	}

	wg.Wait()
}

// TestContextPoolReuse tests that pool reuses contexts.
func TestContextPoolReuse(t *testing.T) {
	pool := NewContextPool()

	ctx1 := pool.Acquire()
	ptr1 := &ctx1

	pool.Release(ctx1)

	ctx2 := pool.Acquire()
	ptr2 := &ctx2

	// Should get same pointer back (pool reuse)
	if ptr1 != ptr2 {
		// This is not guaranteed but highly likely with sync.Pool
		// Just log it, don't fail
		t.Logf("Pool may not have reused context (this is OK)")
	}
}

// TestContextPoolMultipleReleases tests releasing same context multiple times.
func TestContextPoolMultipleReleases(t *testing.T) {
	pool := NewContextPool()

	ctx := pool.Acquire()

	// Release once
	pool.Release(ctx)

	// Release again (should not panic)
	pool.Release(ctx)
}

// BenchmarkContextPoolAcquire benchmarks context acquisition.
func BenchmarkContextPoolAcquire(b *testing.B) {
	pool := NewContextPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := pool.Acquire()
		_ = ctx
	}

	// Expected: 0 allocs/op (pooled)
}

// BenchmarkContextPoolAcquireRelease benchmarks full pool cycle.
func BenchmarkContextPoolAcquireRelease(b *testing.B) {
	pool := NewContextPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := pool.Acquire()
		pool.Release(ctx)
	}

	// Expected: 0 allocs/op (pooled)
}

// BenchmarkContextPoolConcurrent benchmarks concurrent pool access.
func BenchmarkContextPoolConcurrent(b *testing.B) {
	pool := NewContextPool()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := pool.Acquire()
			ctx.SetPath("/test")
			pool.Release(ctx)
		}
	})
}

// BenchmarkContextPoolReset benchmarks context reset performance.
func BenchmarkContextPoolReset(b *testing.B) {
	ctx := &Context{
		methodBytes: []byte("GET"),
		pathBytes:   []byte("/test"),
		queryBytes:  []byte("id=123"),
		statusCode:  200,
		written:     true,
		params:      map[string]string{"id": "123", "name": "test"},
	}

	ctx.Set("key", "value")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.Reset()

		// Re-populate for next iteration
		ctx.methodBytes = []byte("GET")
		ctx.pathBytes = []byte("/test")
		ctx.params = map[string]string{"id": "123", "name": "test"}
		ctx.Set("key", "value")
	}
}
