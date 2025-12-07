---
name: performance-analyzer
description: Analyze and optimize Go code performance for the Capacitor DAL. Creates benchmarks, runs profiling, identifies allocation hotspots, and suggests optimizations. Use when investigating performance issues, comparing implementations, or validating zero-allocation claims.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Performance Analyzer Skill

This skill helps you analyze, benchmark, and optimize the performance of Capacitor DAL implementations following the zero-allocation philosophy.

## Quick Start

### Running Benchmarks
```bash
# Run all benchmarks
go test -bench=. -benchmem -benchtime=5s ./...

# Run specific benchmark
go test -bench=BenchmarkCacheGet -benchmem ./pkg/cache/memory

# Compare two implementations
go test -bench=. -benchmem ./... | tee old.txt
# Make changes
go test -bench=. -benchmem ./... | tee new.txt
benchstat old.txt new.txt
```

### CPU Profiling
```bash
# Generate CPU profile
go test -bench=BenchmarkCacheGet -cpuprofile=cpu.prof ./pkg/cache/memory

# Analyze profile
go tool pprof -http=:8080 cpu.prof
# Or text mode
go tool pprof -top cpu.prof
```

### Memory Profiling
```bash
# Generate memory profile
go test -bench=BenchmarkCacheGet -memprofile=mem.prof ./pkg/cache/memory

# Analyze allocations
go tool pprof -alloc_space -http=:8080 mem.prof

# Find allocation hotspots
go tool pprof -top -alloc_objects mem.prof
```

## Benchmark Creation Patterns

### 1. Basic Benchmark Template
```go
func BenchmarkOperation(b *testing.B) {
    // Setup
    cache := NewCache[string, []byte](10 * 1024 * 1024)
    defer cache.Close()

    ctx := context.Background()
    data := make([]byte, 1024)

    // Warm up (optional)
    for i := 0; i < 100; i++ {
        cache.Set(ctx, fmt.Sprintf("key%d", i), data)
    }

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        cache.Set(ctx, "key", data)
    }
}
```

### 2. Variable Size Benchmark
```go
func BenchmarkCacheVaryingSize(b *testing.B) {
    sizes := []int{
        100,        // 100 bytes
        1024,       // 1 KB
        10 * 1024,  // 10 KB
        100 * 1024, // 100 KB
        1024 * 1024, // 1 MB
    }

    for _, size := range sizes {
        b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
            cache := NewCache[string, []byte](100 * 1024 * 1024)
            defer cache.Close()

            ctx := context.Background()
            data := make([]byte, size)

            b.ResetTimer()
            b.ReportAllocs()
            b.SetBytes(int64(size))

            for i := 0; i < b.N; i++ {
                cache.Set(ctx, "key", data)
            }
        })
    }
}
```

### 3. Concurrency Benchmark
```go
func BenchmarkCacheConcurrent(b *testing.B) {
    cache := NewCache[string, []byte](100 * 1024 * 1024)
    defer cache.Close()

    ctx := context.Background()
    data := make([]byte, 1024)

    // Pre-populate
    for i := 0; i < 1000; i++ {
        cache.Set(ctx, fmt.Sprintf("key%d", i), data)
    }

    b.ResetTimer()
    b.ReportAllocs()

    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            key := fmt.Sprintf("key%d", i%1000)
            cache.Get(ctx, key)
            i++
        }
    })
}
```

### 4. Comparative Benchmark
```go
func BenchmarkCacheComparison(b *testing.B) {
    ctx := context.Background()
    data := make([]byte, 1024)

    b.Run("Capacitor", func(b *testing.B) {
        cache := capacitor.NewCache[string, []byte](10 * 1024 * 1024)
        defer cache.Close()

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            cache.Set(ctx, "key", data)
        }
    })

    b.Run("RawMap", func(b *testing.B) {
        cache := make(map[string][]byte)

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            cache["key"] = data
        }
    })

    b.Run("SyncMap", func(b *testing.B) {
        var cache sync.Map

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            cache.Store("key", data)
        }
    })
}
```

## Performance Analysis Workflow

### Step 1: Establish Baseline
```bash
# Run baseline benchmarks
go test -bench=. -benchmem -count=5 ./... | tee baseline.txt

# Analyze results
benchstat baseline.txt
```

### Step 2: Identify Hotspots
```bash
# CPU profiling
go test -bench=BenchmarkSlowest -cpuprofile=cpu.prof -benchtime=10s
go tool pprof -top cpu.prof

# Memory profiling
go test -bench=BenchmarkSlowest -memprofile=mem.prof -benchtime=10s
go tool pprof -top -alloc_objects mem.prof
```

### Step 3: Analyze Assembly
```bash
# View generated assembly for hot functions
go build -gcflags=-S ./... 2>&1 | grep -A 20 "TEXT.*YourHotFunction"

# Or use pprof
go tool pprof -disasm=YourHotFunction cpu.prof
```

### Step 4: Optimize and Compare
```bash
# Make optimization
# Run new benchmarks
go test -bench=. -benchmem -count=5 ./... | tee optimized.txt

# Compare
benchstat baseline.txt optimized.txt
```

## Common Performance Issues and Fixes

### Issue 1: Unexpected Allocations

**Detection:**
```bash
go test -bench=BenchmarkYourFunc -benchmem
# Shows: 150 B/op  5 allocs/op  (Expected: 0 allocs/op)
```

**Analysis:**
```bash
# Get memory profile
go test -bench=BenchmarkYourFunc -memprofile=mem.prof

# Find allocation sources
go tool pprof -alloc_objects -top mem.prof
go tool pprof -list=YourFunc mem.prof
```

**Common Causes:**
1. **Interface boxing**
   ```go
   // Bad: Boxing
   func store(v interface{}) { /* ... */ }

   // Good: Generics
   func store[T any](v T) { /* ... */ }
   ```

2. **String concatenation**
   ```go
   // Bad: Allocates
   key := "prefix" + id + "suffix"

   // Good: Use builder
   var b strings.Builder
   b.WriteString("prefix")
   b.WriteString(id)
   b.WriteString("suffix")
   key := b.String()
   ```

3. **Slice growth**
   ```go
   // Bad: May reallocate
   var items []Item
   for _, x := range data {
       items = append(items, x)
   }

   // Good: Preallocate
   items := make([]Item, 0, len(data))
   for _, x := range data {
       items = append(items, x)
   }
   ```

4. **Not using pools**
   ```go
   // Bad: Allocates every time
   func process() {
       buf := new(bytes.Buffer)
       // use buf
   }

   // Good: Use pool
   var bufPool = sync.Pool{
       New: func() interface{} { return new(bytes.Buffer) },
   }

   func process() {
       buf := bufPool.Get().(*bytes.Buffer)
       defer func() {
           buf.Reset()
           bufPool.Put(buf)
       }()
       // use buf
   }
   ```

### Issue 2: Lock Contention

**Detection:**
```bash
# Run with mutex profiling
go test -bench=BenchmarkConcurrent -mutexprofile=mutex.prof
go tool pprof -top mutex.prof
```

**Fixes:**
1. **Use RWMutex for read-heavy workloads**
2. **Shard locks** - Multiple locks for different key ranges
3. **Use atomic operations** when possible
4. **Per-CPU data structures** for hot paths

### Issue 3: False Sharing

**Detection:**
```go
// Bad: Fields accessed by different goroutines on same cache line
type Counter struct {
    reads  uint64 // CPU 1 increments
    writes uint64 // CPU 2 increments - false sharing!
}
```

**Fix:**
```go
// Good: Add padding
type Counter struct {
    reads  uint64
    _pad   [7]uint64 // Padding to avoid false sharing
    writes uint64
}
```

### Issue 4: Inefficient Serialization

**Detection:** High CPU usage in encode/decode functions

**Fixes:**
```go
// Consider alternatives based on use case:
// 1. JSON: encoding/json (slow but universal)
// 2. GOB: encoding/gob (Go-specific, medium speed)
// 3. Protobuf: google.golang.org/protobuf (fast, requires schema)
// 4. MessagePack: github.com/vmihailenco/msgpack (fast, flexible)
// 5. Cap'n Proto: zombiezen.com/go/capnproto2 (fastest, zero-copy)
```

## Benchmarking Best Practices

### 1. Stabilize the Environment
```bash
# Disable CPU frequency scaling
sudo cpupower frequency-set -g performance

# Disable turbo boost
echo 1 | sudo tee /sys/devices/system/cpu/intel_pstate/no_turbo

# Pin to specific CPUs
taskset -c 0-3 go test -bench=.
```

### 2. Run Multiple Times
```bash
# Get statistically significant results
go test -bench=. -count=10 | tee results.txt
benchstat results.txt
```

### 3. Control for Noise
```bash
# Close other applications
# Use dedicated benchmark machine
# Run at consistent times (avoid peak system load)
```

### 4. Document Hardware
```go
// In benchmark file or README
// Hardware: AMD Ryzen 9 5950X, 64GB DDR4-3200
// OS: Ubuntu 22.04, Kernel 5.15
// Go: go1.21.5 linux/amd64
```

## Automated Performance Testing

Create a performance test script:

```bash
#!/bin/bash
# scripts/perf-test.sh

set -e

echo "Running performance tests..."

# Baseline
echo "=== Baseline ==="
go test -bench=. -benchmem -benchtime=5s ./... | tee baseline.txt

# CPU profile
echo "=== CPU Profile ==="
go test -bench=BenchmarkCache -cpuprofile=cpu.prof -benchtime=10s ./pkg/cache/memory
go tool pprof -top cpu.prof | head -20

# Memory profile
echo "=== Memory Profile ==="
go test -bench=BenchmarkCache -memprofile=mem.prof -benchtime=10s ./pkg/cache/memory
go tool pprof -top -alloc_objects mem.prof | head -20

# Allocation check
echo "=== Allocation Check ==="
go test -bench=BenchmarkCache -benchmem ./... | grep "0 B/op.*0 allocs/op" || {
    echo "WARNING: Found unexpected allocations!"
    exit 1
}

echo "Performance tests completed!"
```

## Performance Regression Detection

Use benchstat to detect regressions:

```bash
# In CI pipeline
go test -bench=. -benchmem -count=5 ./... | tee new.txt

# Compare with baseline (from git)
git show main:benchmarks/baseline.txt > old.txt
benchstat old.txt new.txt

# Check for regressions (>10% slower)
if benchstat -delta-test=none -geomean old.txt new.txt | grep -E '\+[0-9]{2}\.[0-9]+%'; then
    echo "Performance regression detected!"
    exit 1
fi
```

## Profiling Tools Reference

### CPU Profiling
```bash
# Text output
go tool pprof -text cpu.prof

# Top functions
go tool pprof -top cpu.prof

# Function details
go tool pprof -list=FunctionName cpu.prof

# Call graph
go tool pprof -pdf cpu.prof > cpu.pdf

# Web UI
go tool pprof -http=:8080 cpu.prof
```

### Memory Profiling
```bash
# Allocation space
go tool pprof -alloc_space mem.prof

# Allocation objects
go tool pprof -alloc_objects mem.prof

# In-use space
go tool pprof -inuse_space mem.prof

# In-use objects
go tool pprof -inuse_objects mem.prof
```

### Execution Tracing
```bash
# Generate trace
go test -bench=BenchmarkYourFunc -trace=trace.out

# Analyze trace
go tool trace trace.out
```

## Performance Targets for Capacitor

All implementations should meet these targets:

### Memory Cache
- Get: < 100 ns/op, 0 allocs/op
- Set: < 200 ns/op, ≤1 alloc/op
- Delete: < 150 ns/op, 0 allocs/op

### Disk Cache
- Get: < 100 μs/op (SSD), ≤2 allocs/op
- Set: < 200 μs/op (SSD), ≤2 allocs/op

### Database Backend
- Get: < 1 ms/op (local network), ≤5 allocs/op
- Set: < 2 ms/op (local network), ≤5 allocs/op

### Multi-Layer DAL
- Get (L1 hit): < 150 ns/op, 0 allocs/op
- Get (L2 hit): < 10 μs/op, ≤1 alloc/op
- Set (all layers): < 5 μs/op, ≤3 allocs/op

## Integration with CI

Add to `.github/workflows/performance.yml`:

```yaml
name: Performance Tests

on: [pull_request]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run benchmarks
        run: |
          go test -bench=. -benchmem -count=5 ./... | tee new.txt

      - name: Download baseline
        run: |
          curl -o old.txt https://example.com/baseline.txt

      - name: Compare
        run: |
          go install golang.org/x/perf/cmd/benchstat@latest
          benchstat old.txt new.txt
```

## Summary

**Always:**
- Benchmark before optimizing
- Profile to find real bottlenecks
- Compare with baseline
- Document hardware specs
- Use appropriate tools for each problem

**Never:**
- Optimize without measuring
- Trust intuition over data
- Sacrifice correctness for performance
- Ignore statistical significance
