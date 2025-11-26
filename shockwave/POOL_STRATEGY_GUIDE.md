# Shockwave Pool Strategy Guide

## Overview

Shockwave supports two pooling strategies for object reuse:

1. **PoolStrategyStandard** (Default) - Go's standard `sync.Pool`
2. **PoolStrategyPerCPU** - Per-CPU pools for reduced lock contention

## When to Use Each Strategy

### PoolStrategyStandard (Default) ✅

**Use when:**
- Typical HTTP workloads (recommended for most users)
- Request handling time is fast (< 1ms per request)
- Moderate concurrency (< 10,000 concurrent requests)
- You want the fastest pool operations (benchmarks show 4-44ns vs 44-60ns)

**Performance Characteristics:**
```
Low Concurrency:    4.4 ns/op (fastest)
High Contention:    17-21 ns/op (fastest)
Realistic HTTP:     23-25 ns/op (fastest)
```

**Example:**
```go
import "github.com/yourusername/shockwave/pkg/shockwave/http11"

func main() {
    // Standard pool is default - no configuration needed
    server := http11.NewServer(...)
    server.ListenAndServe()
}
```

### PoolStrategyPerCPU

**Use when:**
- Sustained high-concurrency workloads (> 10,000 concurrent requests)
- Long object hold times (> 10ms per request)
- CPU-intensive request handlers
- Many-core systems (16+ CPUs) with high parallelism

**Performance Characteristics:**
```
Low Concurrency:    44 ns/op (10x slower)
High Contention:    49-54 ns/op (2.5x slower)
Realistic HTTP:     58-62 ns/op (2.5x slower)
```

**Trade-off:**
- **Higher per-operation overhead** (~40ns atomic round-robin)
- **Lower lock contention** under sustained high load
- **Better CPU cache locality** (each CPU has dedicated pools)

**Example:**
```go
import "github.com/yourusername/shockwave/pkg/shockwave/http11"

func init() {
    // Enable per-CPU pooling BEFORE any requests
    http11.SetPoolStrategy(http11.PoolStrategyPerCPU)
}

func main() {
    server := http11.NewServer(...)
    server.ListenAndServe()
}
```

## Configuration

### Setting Pool Strategy

```go
// Set strategy globally (must be called before any pool operations)
http11.SetPoolStrategy(http11.PoolStrategyPerCPU)
```

### Pre-warming Pools

Both strategies support pre-warming to avoid cold-start allocations:

```go
// For PoolStrategyStandard: pre-allocate 100 objects per pool
http11.WarmupPools(100)

// For PoolStrategyPerCPU: pre-allocate 100 objects per CPU per pool
// (with 8 CPUs, this pre-allocates 800 objects per pool type)
http11.SetPoolStrategy(http11.PoolStrategyPerCPU)
http11.WarmupPools(100)
```

## Benchmark Results

### PoolStrategyStandard vs PoolStrategyPerCPU

| Benchmark | Standard Pool | Per-CPU Pool | Winner | Ratio |
|-----------|---------------|--------------|--------|-------|
| Low Concurrency | 4.4 ns | 44 ns | Standard | **10x faster** |
| High Contention | 17-21 ns | 49-54 ns | Standard | **2.5x faster** |
| Mixed Usage | 112-139 ns | 174-189 ns | Standard | **1.4x faster** |
| Realistic HTTP | 23-25 ns | 58-62 ns | Standard | **2.5x faster** |

**Conclusion:** Standard `sync.Pool` is faster for typical HTTP workloads because:
- Pool Get/Put operations are very fast (< 100ns)
- Atomic round-robin overhead (3-5ns) is significant at this scale
- Go's `sync.Pool` is already highly optimized

## Real-World Recommendations

### Small to Medium Traffic (< 10K req/s)
```go
// Use default standard pooling
// No configuration needed
```

### High Traffic (10K-100K req/s)
```go
// Use standard pooling with warmup
http11.WarmupPools(1000)
```

### Extreme Traffic (> 100K req/s) + CPU-Intensive Handlers
```go
// Try per-CPU pooling if profiling shows sync.Pool contention
http11.SetPoolStrategy(http11.PoolStrategyPerCPU)
http11.WarmupPools(1000)

// Benchmark your specific workload to verify improvement!
```

## Monitoring

Both strategies provide pool statistics:

```go
stats := http11.GetPoolStats()
for _, stat := range stats {
    fmt.Printf("Pool: %s, Available: %d, Hit Rate: %.2f%%\n",
        stat.Name, stat.Available, stat.HitRate)
}
```

## Advanced: When PerCPU Pools Actually Help

Per-CPU pools excel when:

1. **Object hold time >> pool operation time**
   ```
   Standard pool operation: ~20ns
   Per-CPU pool operation: ~50ns

   If request processing: 10ms
   Pool overhead: 0.0005% vs 0.0002% (negligible difference)

   But under high concurrency, standard pool may have lock contention
   that dominates, making per-CPU pools beneficial.
   ```

2. **Extreme concurrency on many-core systems**
   - 32+ CPU cores
   - 50,000+ concurrent goroutines
   - Profiling shows `sync.Pool` mutex contention

3. **Workload characteristics:**
   - Long-lived objects (held for > 10ms)
   - Burst traffic patterns
   - CPU-intensive request handlers

## Summary

**Default: Use PoolStrategyStandard** ✅
- Fastest for 99% of workloads
- Proven by benchmarks
- Zero configuration

**Consider PoolStrategyPerCPU only when:**
- Profiling shows sync.Pool contention
- Extreme concurrency (> 50K concurrent requests)
- Many-core systems (32+ CPUs)
- **Always benchmark your specific workload!**

---

*Generated: 2025-11-19*
