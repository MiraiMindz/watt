---
name: performance-optimization
description: Advanced techniques for achieving zero-allocation, high-performance Go code. Use when optimizing hot paths, reducing allocations, improving throughput, or debugging performance issues.
allowed-tools: Read, Write, Edit, Bash
---

# Performance Optimization Skill

## When to Use This Skill

Invoke this skill when you need to:
- Reduce allocations in hot paths
- Optimize CPU-bound operations
- Improve memory efficiency
- Debug performance regressions
- Achieve zero-allocation code paths
- Profile and analyze performance

## Core Optimization Principles

### 1. Measure First, Optimize Second

**Never optimize without data.**

```bash
# Benchmark current performance
go test -bench=. -benchmem -benchtime=10s

# Profile CPU
go test -bench=BenchmarkHotPath -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Profile memory
go test -bench=BenchmarkHotPath -memprofile=mem.prof
go tool pprof mem.prof

# Profile allocations
go test -bench=BenchmarkHotPath -memprofile=mem.prof
go tool pprof -alloc_space mem.prof
```

### 2. Optimization Hierarchy

Priority order for optimizations:

1. **Algorithm** - O(n²) → O(n log n) matters more than micro-optimizations
2. **Data Structures** - Choose the right tool for the job
3. **Memory Allocations** - Reduce allocations, reuse memory
4. **CPU Cache** - Improve locality of reference
5. **Micro-optimizations** - Only after exhausting above

## Object Pooling Patterns

### Pattern 1: Basic sync.Pool

```go
var contextPool = sync.Pool{
    New: func() interface{} {
        return &Context{
            store: make(map[string]interface{}, 4),
        }
    },
}

func acquireContext() *Context {
    return contextPool.Get().(*Context)
}

func releaseContext(c *Context) {
    c.Reset()  // Critical: reset before pooling
    contextPool.Put(c)
}

// Usage
func handleRequest() {
    ctx := acquireContext()
    defer releaseContext(ctx)

    // Use ctx...
}
```

**Key Points:**
- Always reset objects before returning to pool
- Use `defer` to ensure release even on panic
- Pre-allocate internal slices/maps with reasonable capacity

### Pattern 2: Three-Tier Buffer Pool

```go
type SmartBufferPool struct {
    small  sync.Pool  // 512B
    medium sync.Pool  // 4KB
    large  sync.Pool  // 16KB
}

func NewSmartBufferPool() *SmartBufferPool {
    return &SmartBufferPool{
        small: sync.Pool{
            New: func() interface{} {
                buf := make([]byte, 0, 512)
                return &buf
            },
        },
        medium: sync.Pool{
            New: func() interface{} {
                buf := make([]byte, 0, 4096)
                return &buf
            },
        },
        large: sync.Pool{
            New: func() interface{} {
                buf := make([]byte, 0, 16384)
                return &buf
            },
        },
    }
}

func (p *SmartBufferPool) Get(size int) *[]byte {
    var bufPtr *[]byte
    switch {
    case size <= 512:
        bufPtr = p.small.Get().(*[]byte)
    case size <= 4096:
        bufPtr = p.medium.Get().(*[]byte)
    default:
        bufPtr = p.large.Get().(*[]byte)
    }

    // Reset length but keep capacity
    *bufPtr = (*bufPtr)[:0]
    return bufPtr
}

func (p *SmartBufferPool) Put(bufPtr *[]byte, maxRetainSize int) {
    // Don't pool oversized buffers
    if cap(*bufPtr) > maxRetainSize {
        return
    }

    switch {
    case cap(*bufPtr) <= 512:
        p.small.Put(bufPtr)
    case cap(*bufPtr) <= 4096:
        p.medium.Put(bufPtr)
    default:
        p.large.Put(bufPtr)
    }
}
```

**Memory Savings:**
```
Without tiered pooling:
  1000 concurrent requests × 16KB = 16MB

With tiered pooling:
  600 small × 512B  = 300KB
  300 medium × 4KB  = 1.2MB
  100 large × 16KB  = 1.6MB
  Total = 3.1MB (80% savings!)
```

### Pattern 3: Bounded Pool (Prevent Bloat)

```go
const maxPooledParamCount = 8

func (c *Context) Reset() {
    // Only pool maps with reasonable size
    if len(c.params) <= maxPooledParamCount {
        // Clear without reallocating
        for k := range c.params {
            delete(c.params, k)
        }
    } else {
        // Too big, create new small map
        c.params = make(map[string]string, 4)
    }
}
```

**Rationale:** Maps in Go never shrink. If a map grows to 1000 entries, it stays that size even after clearing. Limit pooled map size to prevent memory bloat.

## Zero-Allocation Techniques

### Technique 1: Inline Storage for Small Data

```go
type Context struct {
    // Map for overflow
    params    map[string]string

    // Inline array for common case (≤4 params)
    paramsBuf [4]struct{ key, val string }
    paramLen  int
}

func (c *Context) setParam(key, val string) {
    if c.paramLen < 4 {
        // Use inline storage (zero allocation)
        c.paramsBuf[c.paramLen] = struct{ key, val string }{key, val}
        c.paramLen++
    } else {
        // Overflow to map
        if c.params == nil {
            c.params = make(map[string]string, 8)
            // Copy inline params to map
            for i := 0; i < 4; i++ {
                c.params[c.paramsBuf[i].key] = c.paramsBuf[i].val
            }
        }
        c.params[key] = val
    }
}

func (c *Context) getParam(key string) string {
    // Check inline storage first
    for i := 0; i < c.paramLen && i < 4; i++ {
        if c.paramsBuf[i].key == key {
            return c.paramsBuf[i].val
        }
    }
    // Check map
    return c.params[key]
}
```

**Performance:**
- 0 allocations for ≤4 parameters (90% of routes)
- O(n) lookup for inline (but n ≤ 4, so ~10ns)
- Fallback to map for rare cases with >4 params

### Technique 2: Zero-Copy String ↔ Bytes

```go
// SAFE: Read-only conversion
func stringToBytes(s string) []byte {
    return unsafe.Slice(unsafe.StringData(s), len(s))
}

// SAFE: Read-only conversion
func bytesToString(b []byte) string {
    return unsafe.String(unsafe.SliceData(b), len(b))
}

// Usage (ONLY when data won't be modified)
func (c *Context) matchPath(pattern string) bool {
    pathBytes := stringToBytes(c.path)
    patternBytes := stringToBytes(pattern)

    // Read-only comparison (safe)
    return bytes.Equal(pathBytes, patternBytes)
}
```

**Safety Rules:**
- ✅ Use for read-only operations
- ✅ Use when source data won't be modified
- ❌ Never modify the result
- ❌ Never store the result beyond its source's lifetime

### Technique 3: Pre-Allocated Slices

```go
// ❌ BAD: Append without capacity (reallocations)
func collectItems() []Item {
    var items []Item
    for _, data := range source {
        items = append(items, process(data))
    }
    return items
}

// ✅ GOOD: Pre-allocate with known size
func collectItems() []Item {
    items := make([]Item, 0, len(source))
    for _, data := range source {
        items = append(items, process(data))
    }
    return items
}
```

**Performance:**
```
Bad:  3-5 allocations as slice grows (512B → 1KB → 2KB → 4KB)
Good: 1 allocation with exact size needed
```

### Technique 4: Avoid Interface{} in Hot Paths

```go
// ❌ BAD: Interface allocation
func (c *Context) Set(key string, value interface{}) {
    c.store[key] = value  // Allocates to box value
}

// ✅ GOOD: Concrete types
func (c *Context) SetString(key, value string) {
    c.stringStore[key] = value  // No allocation
}

func (c *Context) SetInt(key string, value int) {
    c.intStore[key] = value  // No allocation
}
```

**Alternative: Generics (Go 1.18+)**
```go
func Set[T any](c *Context, key string, value T) {
    // Type-specific storage
    // Compiler generates concrete implementations
}
```

### Technique 5: String Builder for Concatenation

```go
// ❌ BAD: String concatenation (N allocations)
func buildPath(parts []string) string {
    result := ""
    for _, part := range parts {
        result += "/" + part  // Allocation on each iteration
    }
    return result
}

// ✅ GOOD: strings.Builder (1 allocation)
func buildPath(parts []string) string {
    var b strings.Builder
    b.Grow(estimateSize(parts))  // Pre-allocate
    for _, part := range parts {
        b.WriteString("/")
        b.WriteString(part)
    }
    return b.String()
}
```

## CPU Optimization Techniques

### Technique 1: Fast Path Detection

```go
func (c *Context) handleRequest() error {
    // Fast path: No query string
    if c.query == "" {
        return c.handleSimple()  // Skip query parsing
    }

    // Slow path: Parse query string
    return c.handleWithQuery()
}
```

### Technique 2: Loop Unrolling (Compiler Hint)

```go
// Compiler may unroll small, fixed-size loops
func copySmallSlice(dst, src []byte) {
    // For len(src) <= 8, compiler unrolls loop
    for i := 0; i < len(src) && i < 8; i++ {
        dst[i] = src[i]
    }
}
```

### Technique 3: Avoid Defer in Hot Paths

```go
// ❌ BAD: defer adds overhead (~50ns)
func (c *Context) processHotPath() error {
    mu.Lock()
    defer mu.Unlock()

    // Hot path code
}

// ✅ GOOD: Explicit unlock in hot path
func (c *Context) processHotPath() error {
    mu.Lock()

    // Hot path code
    result := compute()

    mu.Unlock()
    return result
}
```

**Note:** Use defer for clarity unless profiling shows it's a bottleneck.

### Technique 4: Inline Functions

```go
// Hint to compiler to inline
//go:inline
func add(a, b int) int {
    return a + b
}

// Small functions (<80 AST nodes) inline automatically
func fastCheck(x int) bool {
    return x > 0 && x < 100
}
```

## Memory Optimization Techniques

### Technique 1: Struct Field Ordering

```go
// ❌ BAD: Poor alignment (24 bytes)
type BadStruct struct {
    a bool   // 1 byte + 7 padding
    b int64  // 8 bytes
    c bool   // 1 byte + 7 padding
}

// ✅ GOOD: Optimal alignment (16 bytes)
type GoodStruct struct {
    b int64  // 8 bytes
    a bool   // 1 byte
    c bool   // 1 byte + 6 padding
}
```

**Tool to check:**
```bash
go install github.com/dominikh/go-tools/cmd/structlayout@latest
structlayout -json package Type | structlayout-optimize
```

### Technique 2: Reduce Pointer Chasing

```go
// ❌ BAD: Many pointers (cache misses)
type Node struct {
    Next  *Node
    Prev  *Node
    Value *Data
}

// ✅ GOOD: Inline data (cache-friendly)
type Node struct {
    Next  *Node
    Prev  *Node
    Value Data  // Inline, not pointer
}
```

### Technique 3: Use Arrays for Small, Fixed Collections

```go
// ❌ BAD: Slice allocation
type Context struct {
    headers []Header  // Heap allocation
}

// ✅ GOOD: Array (stack allocated)
type Context struct {
    headers [16]Header  // Stack allocated
    headerCount int
}
```

## Benchmarking Best Practices

### Accurate Benchmarks

```go
func BenchmarkFeature(b *testing.B) {
    // Setup (outside timer)
    setup := createExpensiveSetup()

    // Report allocations
    b.ReportAllocs()

    // Reset timer (exclude setup)
    b.ResetTimer()

    // Benchmark loop
    for i := 0; i < b.N; i++ {
        // Don't let compiler optimize away
        result := feature(setup)
        _ = result
    }
}
```

### Prevent Compiler Optimizations

```go
var sink interface{}

func BenchmarkFeature(b *testing.B) {
    for i := 0; i < b.N; i++ {
        result := expensiveComputation()

        // Prevent dead code elimination
        sink = result
    }
}
```

### Parallel Benchmarks

```go
func BenchmarkFeatureParallel(b *testing.B) {
    setup := createSetup()

    b.ReportAllocs()
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            feature(setup)
        }
    })
}
```

## Profiling Workflow

### 1. CPU Profiling

```bash
# Generate profile
go test -bench=BenchmarkHotPath -cpuprofile=cpu.prof

# Analyze (interactive)
go tool pprof cpu.prof
> top10
> list functionName
> web  # Requires graphviz

# Analyze (web UI)
go tool pprof -http=:8080 cpu.prof
```

### 2. Memory Profiling

```bash
# Allocations
go test -bench=. -memprofile=mem.prof
go tool pprof -alloc_space mem.prof

# In-use memory
go tool pprof -inuse_space mem.prof

# Interactive
> top10
> list functionName
```

### 3. Allocation Trace

```go
// In test
func TestAllocations(t *testing.T) {
    allocs := testing.AllocsPerRun(1000, func() {
        // Code to test
        feature()
    })

    if allocs > 0 {
        t.Errorf("Expected 0 allocations, got %.2f", allocs)
    }
}
```

## Common Performance Anti-Patterns

### Anti-Pattern 1: Premature Optimization

```go
// ❌ DON'T: Optimize without measuring
func optimize() {
    // Complex, unreadable "optimization"
    // That might not even be on hot path
}

// ✅ DO: Profile first, then optimize
// 1. Run benchmarks
// 2. Profile with pprof
// 3. Identify bottleneck
// 4. Optimize bottleneck only
// 5. Verify improvement
```

### Anti-Pattern 2: Ignoring Algorithm Complexity

```go
// ❌ BAD: O(n²) no matter how optimized
for i := range items {
    for j := range items {
        compare(items[i], items[j])
    }
}

// ✅ GOOD: O(n log n) with simple code
sort.Slice(items, func(i, j int) bool {
    return items[i] < items[j]
})
```

### Anti-Pattern 3: Over-Optimization

```go
// ❌ TOO COMPLEX: Hard to maintain for 2ns gain
func add(a, b int) int {
    // 50 lines of assembly
}

// ✅ SIMPLE: Clear and fast enough
func add(a, b int) int {
    return a + b
}
```

## Performance Checklist

Before declaring code "optimized":

### Allocations
- [ ] Zero allocations in hot path (verified with `testing.AllocsPerRun`)
- [ ] sync.Pool used for request-scoped objects
- [ ] Buffers pooled and reused
- [ ] Pre-allocated slices with known capacity
- [ ] Inline storage for small, fixed data

### CPU
- [ ] Fast paths for common cases
- [ ] Defer avoided in hot paths (if benchmarks show impact)
- [ ] Small functions eligible for inlining
- [ ] Algorithm complexity optimal (no O(n²) where O(n log n) possible)

### Memory
- [ ] Struct fields ordered for alignment
- [ ] Minimal pointer chasing
- [ ] No unnecessary pointers
- [ ] String ↔ bytes conversions zero-copy (when safe)

### Verification
- [ ] Benchmarks show improvement
- [ ] pprof profile analyzed
- [ ] No performance regression
- [ ] Code still maintainable

## Example: Optimizing JSON Encoding

### Before (Baseline)
```go
func (c *Context) JSON(status int, data interface{}) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    c.Response.Header().Set("Content-Type", "application/json")
    c.Response.WriteHeader(status)
    c.Response.Write(jsonData)
    return nil
}

// Benchmark: 2500ns/op, 1024 B/op, 3 allocs/op
```

### After (Optimized)
```go
var jsonBufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func (c *Context) JSON(status int, data interface{}) error {
    // Get buffer from pool
    buf := jsonBufferPool.Get().(*bytes.Buffer)
    defer jsonBufferPool.Put(buf)
    buf.Reset()

    // Encode directly to buffer
    if err := json.NewEncoder(buf).Encode(data); err != nil {
        return err
    }

    // Write using zero-copy when possible
    c.Response.Header().Set("Content-Type", "application/json")
    c.Response.WriteHeader(status)
    _, err := c.Response.Write(buf.Bytes())
    return err
}

// Benchmark: 1200ns/op, 256 B/op, 1 allocs/op
// Improvement: 52% faster, 75% less memory, 67% fewer allocations
```

---

**This skill makes you an expert in Go performance optimization. Use it to achieve Bolt's ambitious performance targets.**
