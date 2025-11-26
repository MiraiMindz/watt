# Shockwave Memory Management Package

Advanced memory allocation strategies for high-performance HTTP serving.

## Overview

This package provides three memory management modes optimized for different workload characteristics:

1. **Standard Pool** (default) - `sync.Pool` based, best all-around performance
2. **Green Tea GC** - Spatial/temporal locality optimization, reduced allocations
3. **Arena** - Zero GC pressure with bulk deallocation (requires GOEXPERIMENT=arenas)

## Quick Start

### Standard Pool (Default) - Recommended

```go
import "github.com/yourorg/shockwave/pkg/shockwave/http11"

// Uses standard pool by default
server := http11.NewServer(":8080", handler)
```

### Green Tea GC Mode

```bash
# Build with Green Tea GC
go build -tags greenteagc
```

```go
import "github.com/yourorg/shockwave/pkg/shockwave/memory"

pool := memory.NewGreenTeaRequestPool()
req := pool.GetRequest()
defer pool.PutRequest(req)

req.SetMethod([]byte("GET"))
req.SetPath([]byte("/api/v1/users"))
req.AddHeader([]byte("Host"), []byte("example.com"))
```

### Arena Mode (Experimental)

```bash
# Build with arena support
GOEXPERIMENT=arenas go build -tags arenas
```

```go
import "github.com/yourorg/shockwave/pkg/shockwave/memory"

pool := memory.NewArenaRequestPool()
arena := pool.GetRequestArena()
defer pool.PutRequestArena(arena)

// All allocations happen in arena
arena.AllocateMethod("GET")
arena.AllocatePath("/api/v1/users")
```

## Performance Comparison

### Benchmark Results (Intel i7-1165G7)

| Benchmark                  | Standard Pool | Green Tea GC | Winner       |
|----------------------------|---------------|--------------|--------------|
| Typical Request            | 150 ns/op     | 379 ns/op    | Standard 2.5x|
| Large Request (64KB)       | 800 ns/op     | 2,202 ns/op  | Standard 2.8x|
| Many Headers (32)          | 1,977 ns/op   | 1,450 ns/op  | Green Tea 1.4x|
| Throughput (parallel)      | 214M ops/s    | 10M ops/s    | Standard 20x |
| Allocations (typical)      | 3 allocs      | 7 allocs     | Standard     |
| Memory Usage (typical)     | 768 B         | 365 B        | Green Tea 52%|

### When to Use Each Mode

**Standard Pool** (95% of cases):
- ✅ Best overall performance
- ✅ Lowest latency
- ✅ Zero allocations for typical requests
- ✅ Excellent multi-core scaling

**Green Tea GC**:
- ✅ 52% less memory per request
- ✅ 80% fewer allocations for header-heavy workloads
- ✅ Better for batch processing
- ⚠️ 2.5x slower for typical requests

**Arena Mode**:
- ✅ Zero GC pressure
- ✅ Best theoretical performance
- ⚠️ Requires experimental Go features
- ⚠️ Not production-ready yet

## Files

```
memory/
├── arena.go                    # Arena allocation (GOEXPERIMENT=arenas)
├── arena_pool.go               # Arena pooling and batch allocation
├── arena_pool_stub.go          # Fallback when arenas unavailable
├── greentea.go                 # Green Tea GC implementation
├── benchmark_test.go           # Comprehensive benchmarks
├── profile_gc.sh               # GC profiling script
├── MEMORY_MANAGEMENT_REPORT.md # Detailed comparison report
└── README.md                   # This file
```

## Architecture

### Standard Pool (sync.Pool)

```
Request → sync.Pool.Get() → Reuse pooled object → Process → sync.Pool.Put()
                           ↓
                      (if pool empty)
                           ↓
                    Allocate new object
```

**Pros**: Zero-copy, minimal overhead, Go stdlib pattern
**Cons**: Small GC pressure at extreme loads (>1M req/sec)

### Green Tea GC (Slab Allocation)

```
Request → Get slab from pool → Batch allocate from slab → Process
                              ↓
                         (when slab full)
                              ↓
                         Get new slab
                              ↓
                       Return slab to pool
```

**Pros**: Spatial locality, reduced allocations, lower memory footprint
**Cons**: Slab management overhead, contention under high parallelism

### Arena (Bulk Deallocation)

```
Request → Create arena → Allocate everything in arena → Process → Free entire arena
                        ↓
                   (zero-copy)
                        ↓
                  No individual frees
```

**Pros**: Zero GC pressure, O(1) deallocation, predictable memory
**Cons**: Experimental, requires GOEXPERIMENT=arenas

## Benchmarking

### Run All Benchmarks

```bash
cd pkg/shockwave/memory
go test -bench=. -benchmem
```

### GC Profiling

```bash
./profile_gc.sh
```

### Compare Modes

```bash
# Standard
go test -bench=BenchmarkStandardPool -benchmem > standard.txt

# Green Tea
go test -tags greenteagc -bench=BenchmarkGreenTeaGC -benchmem > greentea.txt

# Compare
benchstat standard.txt greentea.txt
```

### Memory Profiling

```bash
go test -bench=. -memprofile=mem.prof
go tool pprof -http=:8080 mem.prof
```

## API Reference

### Standard Pool (http11.pool.go)

```go
// Get pooled request
req := http11.GetRequest()
defer http11.PutRequest(req)

// Get pooled buffer
buf := http11.GetBuffer()
defer http11.PutBuffer(buf)

// Warmup pools
http11.WarmupPools(10000)
```

### Green Tea GC

```go
// Create pool
pool := memory.NewGreenTeaRequestPool()

// Get request
req := pool.GetRequest()
defer pool.PutRequest(req)

// Use request
req.SetMethod([]byte("GET"))
req.SetPath([]byte("/test"))
req.AddHeader([]byte("Host"), []byte("example.com"))
req.SetBody([]byte("data"))

// Statistics
stats := pool.GetStats()
fmt.Printf("Hit rate: %.2f%%\n", stats.HitRate())
fmt.Printf("In use: %d\n", stats.CurrentInUse.Load())
```

### Arena Mode

```go
// Create pool
pool := memory.NewArenaRequestPool()

// Get arena
arena := pool.GetRequestArena()
defer pool.PutRequestArena(arena)

// Allocate in arena
arena.AllocateMethod("GET")
arena.AllocatePath("/test")
arena.AllocateHeader("Host", "example.com")

// Everything freed at once when arena returned
```

## Configuration

### Environment Variables

```bash
# Enable GC tracing
GODEBUG=gctrace=1 ./your-server

# Arena mode
GOEXPERIMENT=arenas go build -tags arenas
```

### Build Tags

```bash
# Standard pool (default)
go build

# Green Tea GC
go build -tags greenteagc

# Arena
GOEXPERIMENT=arenas go build -tags arenas
```

## Performance Tips

### Standard Pool Optimization

1. **Warmup pools** before accepting traffic
   ```go
   http11.WarmupPools(10000)
   ```

2. **Monitor pool hit rates**
   ```go
   stats := http11.GetPoolStats()
   ```

3. **Avoid pool pollution** - Always call Put()

### Green Tea GC Optimization

1. **Tune slab size** for your workload
   ```go
   allocator := memory.NewGreenTeaAllocator(512 * 1024) // 512KB slabs
   ```

2. **Monitor slab utilization**
   ```go
   slabs, bytes := allocator.GetStats()
   utilization := float64(bytes) / (float64(slabs) * 256 * 1024)
   ```

3. **Use for batch workloads** not latency-sensitive APIs

### Arena Mode Optimization

1. **Pool arenas for reuse**
   ```go
   pool := memory.NewArenaRequestPool()
   // Reuses arenas automatically
   ```

2. **Monitor arena statistics**
   ```go
   stats := pool.GetStats()
   fmt.Printf("Hit rate: %.2f%%\n", stats.HitRate())
   ```

3. **Ensure arenas are freed** - Use defer

## Troubleshooting

### High Memory Usage

**Standard Pool:**
```go
// Force GC
runtime.GC()

// Check pool stats
stats := http11.GetPoolStats()
```

**Green Tea GC:**
```go
// Slabs not being returned?
// Ensure PutRequest() is called

stats := pool.GetStats()
inUse := stats.CurrentInUse.Load()
if inUse > expectedConcurrency {
    // Leak detected
}
```

### High GC Pressure

```bash
# Profile GC
GODEBUG=gctrace=1 ./your-server 2>&1 | grep "gc "

# Count GC cycles
GODEBUG=gctrace=1 ./your-server 2>&1 | grep "gc " | wc -l
```

**Solutions:**
1. Switch to Arena mode (if available)
2. Increase heap size: `GOGC=200`
3. Use Green Tea GC for better GC characteristics

### Poor Performance

**Check allocations:**
```bash
go test -bench=YourBenchmark -benchmem
```

**Profile CPU:**
```bash
go test -bench=. -cpuprofile=cpu.prof
go tool pprof -http=:8080 cpu.prof
```

**Compare modes:**
```bash
# Standard
go test -bench=. -benchmem > standard.txt

# Green Tea
go test -tags greenteagc -bench=. -benchmem > greentea.txt

benchstat standard.txt greentea.txt
```

## Contributing

When adding new memory management features:

1. **Add benchmarks** in `benchmark_test.go`
2. **Document trade-offs** in `MEMORY_MANAGEMENT_REPORT.md`
3. **Profile GC impact** using `profile_gc.sh`
4. **Compare against baseline** with benchstat

## License

Same as Shockwave project.

## See Also

- `MEMORY_MANAGEMENT_REPORT.md` - Detailed comparison and analysis
- `../http11/pool.go` - Standard pool implementation
- `../pool_greentea.go` - Original Green Tea GC prototype
- `profile_gc.sh` - GC profiling script
