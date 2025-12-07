package memory

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Benchmark standard cache vs sharded cache

func BenchmarkStandardCache_Get_Parallel(b *testing.B) {
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
			cache.Get(ctx, fmt.Sprintf("key%d", i%1000))
			i++
		}
	})
}

func BenchmarkShardedCache_Get_Parallel(b *testing.B) {
	cache := NewSharded[string, int](ShardedConfig{
		Config: Config{
			MaxSize:       10000,
			DefaultTTL:    0,
			EvictionMode:  EvictionLRU,
			EnableMetrics: false,
		},
		ShardCount: 32,
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
			cache.Get(ctx, fmt.Sprintf("key%d", i%1000))
			i++
		}
	})
}

func BenchmarkStandardCache_Set_Parallel(b *testing.B) {
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

func BenchmarkShardedCache_Set_Parallel(b *testing.B) {
	cache := NewSharded[string, int](ShardedConfig{
		Config: Config{
			MaxSize:       100000,
			DefaultTTL:    0,
			EvictionMode:  EvictionLRU,
			EnableMetrics: false,
		},
		ShardCount: 32,
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

func BenchmarkStandardCache_MixedWorkload_Parallel(b *testing.B) {
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

func BenchmarkShardedCache_MixedWorkload_Parallel(b *testing.B) {
	cache := NewSharded[string, int](ShardedConfig{
		Config: Config{
			MaxSize:       10000,
			DefaultTTL:    0,
			EvictionMode:  EvictionLRU,
			EnableMetrics: false,
		},
		ShardCount: 32,
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

// Benchmark atomic metrics vs mutex metrics

func BenchmarkAtomicMetrics_RecordHit(b *testing.B) {
	m := NewAtomicMetrics()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.RecordHit()
	}
}

func BenchmarkAtomicMetrics_RecordHit_Parallel(b *testing.B) {
	m := NewAtomicMetrics()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.RecordHit()
		}
	})
}

func BenchmarkAtomicMetrics_Snapshot(b *testing.B) {
	m := NewAtomicMetrics()

	// Prime with some data
	for i := 0; i < 1000; i++ {
		m.RecordHit()
		m.RecordMiss()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.Snapshot()
	}
}

// Benchmark custom map vs standard map

func BenchmarkCustomMap_Get(b *testing.B) {
	m := newCustomMap[int](1000)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		m.set(fmt.Sprintf("key%d", i), i, zeroTime)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.get("key500")
	}
}

func BenchmarkStandardMap_Get(b *testing.B) {
	m := make(map[string]int, 1000)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		m[fmt.Sprintf("key%d", i)] = i
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m["key500"]
	}
}

func BenchmarkCustomMap_Set(b *testing.B) {
	m := newCustomMap[int](10000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.set(fmt.Sprintf("key%d", i%10000), i, zeroTime)
	}
}

func BenchmarkStandardMap_Set(b *testing.B) {
	m := make(map[string]int, 10000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m[fmt.Sprintf("key%d", i%10000)] = i
	}
}

func BenchmarkCustomMap_Delete(b *testing.B) {
	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	m := newCustomMap[int](b.N)
	for i := 0; i < b.N; i++ {
		m.set(keys[i], i, zeroTime)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.delete(keys[i])
	}
}

func BenchmarkStandardMap_Delete(b *testing.B) {
	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	m := make(map[string]int, b.N)
	for i := 0; i < b.N; i++ {
		m[keys[i]] = i
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		delete(m, keys[i])
	}
}

// Benchmark hash functions

func BenchmarkHashString_FNV1a(b *testing.B) {
	key := "test-key-with-reasonable-length-for-cache"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = hashString(key)
	}
}

// Scalability benchmarks with different shard counts

func BenchmarkShardedCache_Shards16_Parallel(b *testing.B) {
	benchmarkShardedCacheParallel(b, 16)
}

func BenchmarkShardedCache_Shards32_Parallel(b *testing.B) {
	benchmarkShardedCacheParallel(b, 32)
}

func BenchmarkShardedCache_Shards64_Parallel(b *testing.B) {
	benchmarkShardedCacheParallel(b, 64)
}

func BenchmarkShardedCache_Shards128_Parallel(b *testing.B) {
	benchmarkShardedCacheParallel(b, 128)
}

func benchmarkShardedCacheParallel(b *testing.B, shardCount int) {
	cache := NewSharded[string, int](ShardedConfig{
		Config: Config{
			MaxSize:       10000,
			DefaultTTL:    0,
			EvictionMode:  EvictionLRU,
			EnableMetrics: false,
		},
		ShardCount: shardCount,
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

var zeroTime = time.Time{}
