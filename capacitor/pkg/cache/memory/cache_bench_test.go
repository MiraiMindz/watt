package memory

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkCache_Get(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       10000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false, // Disable for pure Get performance
	})
	defer cache.Close()

	ctx := context.Background()

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "key500")
	}
}

func BenchmarkCache_Get_Parallel(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       10000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Get(ctx, "key500")
		}
	})
}

func BenchmarkCache_Set(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       100000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i%10000), i)
	}
}

func BenchmarkCache_Set_Parallel(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       100000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Set(ctx, fmt.Sprintf("key%d", i%10000), i)
			i++
		}
	})
}

func BenchmarkCache_SetWithTTL(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       100000,
		DefaultTTL:    5 * time.Minute,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i%10000), i, WithTTL(10*time.Minute))
	}
}

func BenchmarkCache_GetMiss(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       10000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "nonexistent")
	}
}

func BenchmarkCache_Delete(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       100000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	// Pre-generate keys to avoid fmt.Sprintf overhead in benchmark
	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
		cache.Set(ctx, keys[i], i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Delete(ctx, keys[i])
	}
}

func BenchmarkCache_Exists(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       10000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Exists(ctx, "key500")
	}
}

func BenchmarkCache_Update(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       10000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "key500", i)
	}
}

func BenchmarkCache_LRUEviction(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       1000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Each set will trigger eviction
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, fmt.Sprintf("new%d", i), i)
	}
}

func BenchmarkCache_MixedWorkload(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       10000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		switch i % 10 {
		case 0, 1, 2, 3, 4, 5, 6, 7: // 80% reads
			cache.Get(ctx, fmt.Sprintf("key%d", i%1000))
		case 8: // 10% writes
			cache.Set(ctx, fmt.Sprintf("key%d", i%1000), i)
		case 9: // 10% deletes
			cache.Delete(ctx, fmt.Sprintf("key%d", i%1000))
		}
	}
}

func BenchmarkCache_MixedWorkload_Parallel(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       10000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 10 {
			case 0, 1, 2, 3, 4, 5, 6, 7: // 80% reads
				cache.Get(ctx, fmt.Sprintf("key%d", i%1000))
			case 8: // 10% writes
				cache.Set(ctx, fmt.Sprintf("key%d", i%1000), i)
			case 9: // 10% deletes
				cache.Delete(ctx, fmt.Sprintf("key%d", i%1000))
			}
			i++
		}
	})
}

func BenchmarkCache_WithMetrics(b *testing.B) {
	cache := New[string, int](Config{
		MaxSize:       10000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true, // Enable metrics
	})
	defer cache.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "key500")
	}
}

// Benchmarks for specific sizes
func BenchmarkCache_SmallCache_Get(b *testing.B) {
	benchmarkCacheGet(b, 100)
}

func BenchmarkCache_MediumCache_Get(b *testing.B) {
	benchmarkCacheGet(b, 10000)
}

func BenchmarkCache_LargeCache_Get(b *testing.B) {
	benchmarkCacheGet(b, 1000000)
}

func benchmarkCacheGet(b *testing.B, size int) {
	cache := New[string, int](Config{
		MaxSize:       size,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false,
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache
	for i := 0; i < size; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Get(ctx, fmt.Sprintf("key%d", i%size))
	}
}

// Benchmark pool efficiency
func BenchmarkEntryPool_GetPut(b *testing.B) {
	pool := newEntryPool[string, int]()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		e := pool.get()
		e.key = "test"
		e.value = 123
		pool.put(e)
	}
}

// Benchmark LRU operations
func BenchmarkLRU_PushFront(b *testing.B) {
	lru := newLRUList[string]()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lru.pushFront(fmt.Sprintf("key%d", i))
	}
}

func BenchmarkLRU_MoveToFront(b *testing.B) {
	lru := newLRUList[string]()

	// Create nodes
	nodes := make([]*lruNode[string], 1000)
	for i := 0; i < 1000; i++ {
		nodes[i] = lru.pushFront(fmt.Sprintf("key%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lru.moveToFront(nodes[i%1000])
	}
}

func BenchmarkLRU_Remove(b *testing.B) {
	// Create fresh LRU for each iteration
	nodes := make([]*lruNode[string], b.N)
	lru := newLRUList[string]()
	for i := 0; i < b.N; i++ {
		nodes[i] = lru.pushFront(fmt.Sprintf("key%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lru.remove(nodes[i])
	}
}
