---
name: go-performance
description: Expert in high-performance Go development including zero-allocation patterns, benchmarking, profiling, and memory optimization. Use when implementing performance-critical code or optimizing existing implementations.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Go Performance Optimization Skill

## Purpose
This skill provides expertise in writing and optimizing high-performance Go code, with emphasis on zero-allocation patterns, efficient memory usage, and CPU optimization.

## When to Use This Skill
- Implementing performance-critical code paths
- Optimizing existing code for speed or memory
- Writing benchmarks and profiling code
- Reducing allocations in hot paths
- Analyzing and improving CPU efficiency

## Core Principles

### 1. Measure First, Optimize Second
Never optimize without data. Always:
1. Write correct implementation
2. Write comprehensive tests
3. Write benchmarks
4. Profile to find bottlenecks
5. Optimize based on evidence
6. Verify improvement with benchmarks

### 2. Zero-Allocation Patterns

#### String to []byte Conversion (Zero-Copy)
```go
import "unsafe"

// ToBytes converts string to []byte without allocation
// SAFETY: Returned slice must not be modified
func ToBytes(s string) []byte {
    if s == "" {
        return nil
    }
    return unsafe.Slice(unsafe.StringData(s), len(s))
}
```

#### Avoid String Concatenation
```go
// BAD: Multiple allocations
s := "Hello" + " " + "World"

// GOOD: Single allocation with strings.Builder
var b strings.Builder
b.Grow(11) // Pre-allocate if size known
b.WriteString("Hello")
b.WriteByte(' ')
b.WriteString("World")
s := b.String()

// BETTER: Use arena or pool for builder
```

#### Reuse Buffers
```go
// BAD: Allocates every call
func process() []byte {
    buf := make([]byte, 0, 1024)
    // use buf
    return buf
}

// GOOD: Pool buffers
var bufferPool = sync.Pool{
    New: func() interface{} {
        b := make([]byte, 0, 1024)
        return &b
    },
}

func process() []byte {
    buf := bufferPool.Get().(*[]byte)
    defer bufferPool.Put(buf)
    *buf = (*buf)[:0] // Reset length
    // use buf
    return *buf
}
```

#### Pre-allocate Slices
```go
// BAD: Multiple allocations as slice grows
var items []Item
for i := 0; i < 1000; i++ {
    items = append(items, Item{})
}

// GOOD: Single allocation
items := make([]Item, 0, 1000)
for i := 0; i < 1000; i++ {
    items = append(items, Item{})
}
```

### 3. Benchmark Writing

#### Standard Benchmark Template
```go
func BenchmarkOperation(b *testing.B) {
    // Setup
    data := setupTestData()

    // Reset timer after setup
    b.ResetTimer()

    // Report allocations
    b.ReportAllocs()

    // Run benchmark
    for i := 0; i < b.N; i++ {
        operation(data)
    }
}
```

#### Benchmark with Size Variations
```go
func BenchmarkOperationSizes(b *testing.B) {
    sizes := []int{16, 64, 256, 1024, 4096}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := make([]byte, size)
            b.SetBytes(int64(size))
            b.ResetTimer()
            b.ReportAllocs()

            for i := 0; i < b.N; i++ {
                operation(data)
            }
        })
    }
}
```

#### Prevent Compiler Optimizations
```go
var result int

func BenchmarkComputation(b *testing.B) {
    var r int
    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        r = expensiveComputation()
    }

    // Prevent dead code elimination
    result = r
}
```

### 4. Profiling Workflow

#### CPU Profiling
```bash
# Run benchmark with CPU profile
go test -bench=BenchmarkSlow -cpuprofile=cpu.prof

# Analyze interactively
go tool pprof cpu.prof
# Commands: top, list FunctionName, web

# Or use web UI
go tool pprof -http=:8080 cpu.prof
```

#### Memory Profiling
```bash
# Memory allocation profile
go test -bench=BenchmarkMemory -memprofile=mem.prof

# Analyze allocations
go tool pprof -alloc_space mem.prof

# Analyze objects in use
go tool pprof -inuse_space mem.prof
```

#### Allocation Tracking
```bash
# See allocations per operation
go test -bench=. -benchmem

# Example output:
# BenchmarkOperation-8   1000000   1234 ns/op   256 B/op   3 allocs/op
#                                   ^^^^^^^^^^   ^^^^^^^^   ^^^^^^^^^^^^
#                                   time/op      bytes/op   allocs/op
```

### 5. Common Optimization Patterns

#### Avoid Interface Boxing
```go
// BAD: Boxing int to interface{}
func sum(values []interface{}) int {
    total := 0
    for _, v := range values {
        total += v.(int) // Slow type assertion
    }
    return total
}

// GOOD: Generic or type-specific
func sum(values []int) int {
    total := 0
    for _, v := range values {
        total += v
    }
    return total
}
```

#### Struct Layout for Cache Efficiency
```go
// BAD: Poor cache locality (40 bytes with padding)
type BadStruct struct {
    a bool   // 1 byte + 7 padding
    b int64  // 8 bytes
    c bool   // 1 byte + 7 padding
    d int64  // 8 bytes
    e bool   // 1 byte + 7 padding
}

// GOOD: Grouped by size (24 bytes)
type GoodStruct struct {
    b int64  // 8 bytes
    d int64  // 8 bytes
    a bool   // 1 byte
    c bool   // 1 byte
    e bool   // 1 byte + 5 padding
}
```

#### Loop Optimizations
```go
// BAD: Recalculates len() each iteration
for i := 0; i < len(slice); i++ {
    process(slice[i])
}

// GOOD: Hoist invariant
n := len(slice)
for i := 0; i < n; i++ {
    process(slice[i])
}

// BETTER: Range is already optimized
for i := range slice {
    process(slice[i])
}
```

#### Defer Overhead
```go
// BAD: Defer in hot loop (defer has overhead)
for i := 0; i < 1000000; i++ {
    mu.Lock()
    defer mu.Unlock()
    // work
}

// GOOD: Manual unlock in hot paths
for i := 0; i < 1000000; i++ {
    mu.Lock()
    // work
    mu.Unlock()
}
```

### 6. Memory Management Patterns

#### Arena Allocation
```go
// For request-scoped allocations
type Arena struct {
    buf    []byte
    offset int
}

func (a *Arena) Alloc(size int) unsafe.Pointer {
    // Align to 8 bytes
    size = (size + 7) &^ 7

    if a.offset+size > len(a.buf) {
        // Grow or panic
    }

    ptr := unsafe.Pointer(&a.buf[a.offset])
    a.offset += size
    return ptr
}

func (a *Arena) Reset() {
    a.offset = 0
}
```

#### Object Pooling
```go
// Size-classed pools
type BufferPool struct {
    pools [10]sync.Pool // 64B to 32KB
}

func (p *BufferPool) Get(size int) []byte {
    idx := sizeClass(size)
    if buf := p.pools[idx].Get(); buf != nil {
        return buf.([]byte)[:0]
    }
    return make([]byte, 0, classSize(idx))
}

func (p *BufferPool) Put(buf []byte) {
    idx := sizeClass(cap(buf))
    p.pools[idx].Put(buf)
}
```

### 7. Escape Analysis

#### Check What Escapes
```bash
go build -gcflags='-m' file.go
# Look for "escapes to heap" messages
```

#### Prevent Escapes
```go
// BAD: Pointer escapes to heap
func createItem() *Item {
    item := Item{value: 42}
    return &item // item escapes
}

// GOOD: Return by value
func createItem() Item {
    return Item{value: 42}
}

// When pointer needed, let caller decide
func createItem(item *Item) {
    *item = Item{value: 42}
}
```

## Performance Checklist

Before considering optimization complete:
- [ ] Benchmarks show measurable improvement
- [ ] Allocation count reduced (visible in -benchmem)
- [ ] CPU profile shows reduced time in hot functions
- [ ] Memory profile shows reduced allocations
- [ ] Tests still pass
- [ ] Code remains readable and maintainable
- [ ] Performance gains documented

## Common Anti-Patterns to Avoid

1. **Premature Optimization** - Optimize only after profiling
2. **Micro-optimizations** - Focus on algorithmic improvements first
3. **Ignoring Readability** - Performance at the cost of unmaintainable code
4. **No Benchmarks** - Can't verify improvement without measurements
5. **Optimizing Cold Paths** - Focus on hot paths (>90% of time)
6. **Unsafe Without Reason** - Use unsafe only when measurably better

## Workflow Commands

This skill includes workflow commands:
- `/bench` - Run benchmarks and compare to targets
- `/profile` - Run CPU and memory profiling
- `/check-allocs` - Check for allocations in zero-alloc paths

See `.claude/skills/go-performance/workflows/` for command implementations.

## References

- [Go Performance Book](https://github.com/dgryski/go-perfbook)
- [Escape Analysis](https://www.ardanlabs.com/blog/2017/05/language-mechanics-on-escape-analysis.html)
- [Memory Profiling](https://go.dev/blog/pprof)
- [Writing High-Performance Go](https://dave.cheney.net/high-performance-go-workshop/gopherchina-2019.html)

---

**Remember**: The fastest code is code that doesn't run. Eliminate work before optimizing work.
