# /optimize Command

Optimize code for performance using profiling and analysis.

## Usage

```
/optimize <target>
```

## What This Command Does

1. Runs baseline benchmarks
2. Profiles with pprof (CPU and memory)
3. Identifies hot spots and allocation sources
4. Applies optimizations
5. Re-benchmarks to verify improvements
6. Updates code with optimization notes

## Targets

```
/optimize <function>  # Optimize specific function
/optimize <package>   # Optimize package
/optimize hotpath     # Optimize hot paths automatically
/optimize allocations # Focus on reducing allocations
```

## Optimization Process

### 1. Baseline Measurement
```bash
go test -bench=Benchmark<Target> -benchmem -benchtime=10s
# Capture baseline: Xns/op, Y B/op, Z allocs/op
```

### 2. CPU Profiling
```bash
go test -bench=Benchmark<Target> -cpuprofile=cpu.prof
go tool pprof cpu.prof
> top10
> list <hotFunction>
```

### 3. Memory Profiling
```bash
go test -bench=Benchmark<Target> -memprofile=mem.prof
go tool pprof -alloc_space mem.prof
> top10
> list <allocFunction>
```

### 4. Apply Optimizations

Common optimizations applied:
- Add sync.Pool for repeated allocations
- Use zero-copy string conversions (where safe)
- Pre-allocate slices with known capacity
- Add inline storage for small, fixed data
- Reduce interface{} usage in hot paths
- Optimize struct field ordering

### 5. Verification
```bash
go test -bench=Benchmark<Target> -benchmem -benchtime=10s
# Compare: improvement percentage
```

## Example

```
/optimize router
```

### Before:
```
BenchmarkRouterLookup-8   5000000   320 ns/op   128 B/op   3 allocs/op
```

### Optimizations Applied:
1. Added sync.Pool for parameter maps
2. Implemented inline storage for ≤4 params
3. Zero-copy string views for path matching

### After:
```
BenchmarkRouterLookup-8   8000000   185 ns/op   0 B/op   0 allocs/op
```

### Improvement:
- Latency: 42% faster (320ns → 185ns)
- Memory: 100% reduction (128B → 0B)
- Allocations: 100% reduction (3 → 0)

## Optimization Techniques Applied

### 1. Object Pooling
```go
var paramMapPool = sync.Pool{
    New: func() interface{} {
        return make(map[string]string, 4)
    },
}
```

### 2. Zero-Copy Conversions
```go
// Safe for read-only
func stringToBytes(s string) []byte {
    return unsafe.Slice(unsafe.StringData(s), len(s))
}
```

### 3. Inline Storage
```go
type Context struct {
    paramsBuf [4]struct{ key, val string }
    params    map[string]string
}
```

### 4. Pre-Allocation
```go
items := make([]Item, 0, len(source))
```

### 5. Struct Optimization
```go
// Run: structlayout-optimize
type Optimized struct {
    // 64-bit fields first
    ptr *Data
    num int64

    // 32-bit fields
    count int32

    // Small fields last
    flag bool
}
```

## Validation

After optimization:
- [ ] Benchmarks show improvement (>20%)
- [ ] Tests still pass
- [ ] No race conditions introduced
- [ ] Code maintainability preserved
- [ ] Optimization documented in comments

## Documentation

The command adds comments like:
```go
// Optimized: Uses sync.Pool to reduce allocations from 3 to 0 per request.
// Performance: 42% faster, 100% less memory.
// Benchmark: BenchmarkRouterLookup-8  185ns/op  0 B/op  0 allocs/op
func (r *Router) Lookup(path string) Handler {
    // Implementation...
}
```

## Safety Checks

Before applying unsafe optimizations:
- [ ] Verify read-only usage
- [ ] Add safety comments
- [ ] Test thoroughly
- [ ] Run race detector
- [ ] Document assumptions

## Performance Targets

- Minimum 20% improvement in latency OR
- Zero-allocation achieved OR
- Significant memory reduction (>50%)

## Agent Used

This command can invoke multiple agents:
- **tester** - Runs baseline benchmarks
- **implementer** - Applies optimizations
- **benchmarker** - Verifies improvements

Uses **performance-optimization** skill for optimization techniques.

---

*See MASTER_PROMPTS.md for optimization guidelines*
