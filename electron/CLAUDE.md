# Electron Project Guide

## Overview
Electron is Watt's foundational layer providing high-performance shared internals, utilities, and primitives. Like electrons power electrical systems, Electron provides the fundamental building blocks powering Watt's performance.

## Project Philosophy

### Core Principles
1. **Zero-Cost Abstractions** - Performance primitives with no overhead
2. **Memory Efficiency** - Advanced pooling and allocation strategies
3. **CPU Optimization** - SIMD, cache-aware algorithms, branch prediction
4. **Composability** - Modular utilities that combine efficiently
5. **Safety** - Memory-safe abstractions over unsafe operations

### Performance-First Mindset
Every line of code must be justified by performance metrics. We optimize for:
- **Allocation count** - Target: 0 allocations for hot paths
- **CPU efficiency** - SIMD acceleration where possible
- **Cache locality** - Data structures designed for CPU caches
- **Concurrency** - Lock-free when safe, minimal contention otherwise

## Project Structure

```
electron/
├── arena/          # Memory arena allocators
├── pool/           # Object pooling systems
├── simd/           # SIMD-accelerated operations
├── hash/           # High-performance hash functions
├── bits/           # Bit manipulation utilities
├── lockfree/       # Lock-free data structures
├── cache/          # Cache-aware structures
├── dense/          # Dense array storage
├── strings/        # Zero-allocation string ops
├── parse/          # Fast parsing utilities
├── io/             # Buffer management
├── encoding/       # Fast encoding (JSON, binary)
├── sync/           # Synchronization primitives
├── sched/          # Work scheduling
├── metrics/        # Performance metrics
├── trace/          # Lightweight tracing
└── test/           # Testing infrastructure
```

## Development Phases

**Current Phase:** Phase 1 - Memory Management (Week 1-2)

### Phase Progression
1. **Phase 1** - Memory Management (arena allocators, object pooling)
2. **Phase 2** - CPU Optimizations (SIMD, hash functions, bit ops)
3. **Phase 3** - Data Structures (lock-free, cache-aware)
4. **Phase 4** - String Utilities (zero-allocation operations)
5. **Phase 5** - IO Utilities (buffer management, encoding)
6. **Phase 6** - Concurrency Primitives (sync, scheduling)
7. **Phase 7** - Diagnostics & Profiling (metrics, tracing)
8. **Phase 8** - Integration & Testing (complete system)

## Code Standards

### Go Conventions

#### Package Organization
```go
// Each package has a clear, single responsibility
package arena // Memory arena allocation

// Public API at top of file
// Implementation details below
// Internal/private types at bottom
```

#### Naming Patterns
- **Types**: `PascalCase` for exported, `camelCase` for private
- **Functions**: Verb-based names (`AllocContext`, `GetBuffer`)
- **Variables**: Short names in small scopes, descriptive in larger scopes
- **Constants**: `PascalCase` for exported, `camelCase` for private

#### Documentation Requirements
Every exported symbol MUST have documentation:
```go
// Arena provides fast bump-pointer allocation with bulk deallocation.
// It allocates memory in large blocks and hands out chunks sequentially.
// When done, the entire arena can be freed at once.
//
// Arena is not thread-safe. Use one arena per goroutine or add external synchronization.
//
// Performance: Allocation is ~10ns, 50x faster than heap allocation.
type Arena struct {
    // ...
}

// NewArena creates an arena with the specified initial size.
// Size will be rounded up to the nearest power of 2 for alignment.
func NewArena(size int) *Arena {
    // ...
}
```

### Performance Standards

#### Benchmark Requirements
Every public API MUST have benchmarks:
```go
func BenchmarkArena_Alloc(b *testing.B) {
    arena := NewArena(1024 * 1024)
    defer arena.Free()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        arena.Alloc(64)
    }
}
```

#### Performance Targets
- **Arena allocation**: <10ns per allocation
- **Pool get/put**: <50ns
- **SIMD string compare**: >10GB/s
- **Lock-free queue**: >10M ops/sec
- **String interning**: O(1) lookup
- **Zero-allocation paths**: 0 allocs/op

#### Memory Requirements
- **Hot paths**: Zero allocations required
- **Warm paths**: <1 allocation per operation
- **Cold paths**: Minimize allocations, prefer pooling

### Unsafe Code Rules

#### When Unsafe is Acceptable
1. **Performance-critical paths** with measurable gains (>2x speedup)
2. **Zero-allocation requirements** that can't be met otherwise
3. **Interfacing with system APIs** or memory layouts
4. **SIMD operations** requiring specific alignments

#### Unsafe Code Requirements
1. **MUST** have comprehensive documentation explaining:
   - Why unsafe is necessary
   - What invariants must be maintained
   - What can go wrong if misused
2. **MUST** have unit tests covering edge cases
3. **MUST** have fuzz tests for parsing/conversion operations
4. **SHOULD** provide safe wrapper APIs when possible
5. **MUST** be reviewed by unsafe-code-reviewer agent

#### Example:
```go
// ToBytes converts a string to []byte without allocation.
// This is safe because the returned slice is read-only.
// Writing to the returned slice will cause undefined behavior.
//
// SAFETY: The returned slice MUST NOT be modified.
// The slice is valid only while the string is alive.
func ToBytes(s string) []byte {
    if s == "" {
        return nil
    }
    return unsafe.Slice(unsafe.StringData(s), len(s))
}
```

### Testing Requirements

#### Test Coverage
- **Minimum**: 80% line coverage
- **Target**: 90%+ line coverage
- **Critical paths**: 100% coverage

#### Test Types Required
1. **Unit tests** - All public APIs
2. **Benchmarks** - All performance-critical code
3. **Fuzz tests** - All parsing/validation code
4. **Integration tests** - Component interactions
5. **Leak tests** - Memory-allocating code

#### Test Organization
```go
// table-driven tests preferred
func TestArena_Alloc(t *testing.T) {
    tests := []struct {
        name string
        size int
        want int
    }{
        {"small", 64, 64},
        {"aligned", 65, 72}, // aligned to 8 bytes
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### SIMD Code Standards

#### CPU Feature Detection
Always detect CPU features at runtime:
```go
import "golang.org/x/sys/cpu"

var (
    hasAVX2   = cpu.X86.HasAVX2
    hasAVX512 = cpu.X86.HasAVX512F
)

func optimizedFunction(data []byte) {
    if hasAVX512 {
        return avx512Implementation(data)
    }
    if hasAVX2 {
        return avx2Implementation(data)
    }
    return scalarFallback(data)
}
```

#### Platform-Specific Code
Use build tags appropriately:
```go
// file_amd64.go
//go:build amd64

// file_arm64.go
//go:build arm64

// file_generic.go
//go:build !amd64 && !arm64
```

## Git Workflow

### Commit Messages
Follow conventional commits:
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `perf`, `refactor`, `test`, `docs`, `chore`

Examples:
- `feat(arena): add aligned allocation support`
- `perf(simd): use AVX-512 for batch hashing`
- `fix(pool): prevent memory leak in buffer return`

### Branch Strategy
- `main` - Stable, tested code
- `phase/N` - Phase-specific development branches
- `feat/description` - Feature branches
- `perf/description` - Performance optimization branches

## Architecture Decisions

### Memory Strategy: Arena-of-Pools
Based on benchmark results, we use a hybrid approach:
- **Arenas** for request-scoped allocations
- **Pools** for reusable objects
- **Arena-of-Pools** for optimal performance (36% faster)

### Concurrency Model
- **Lock-free** for high-contention data structures
- **Sharded locks** for medium contention
- **Regular mutexes** for low contention
- **RCU** for read-heavy workloads

### Error Handling
- **Panics** for programmer errors (invalid API usage)
- **Errors** for runtime failures (allocation, I/O)
- **No errors** for infallible operations (when safe)

## Dependencies

### Required
- `golang.org/x/sys/cpu` - CPU feature detection
- `golang.org/x/arch` - SIMD intrinsics (if needed)

### Testing
- Standard library `testing` package
- Standard library `testing/quick` for property testing
- No external test frameworks

### Forbidden
- No reflection in hot paths
- No CGO unless absolutely necessary
- No external dependencies for core functionality

## Integration Points

### With Other Watt Components
- **Conduit** - Uses arena allocators for request contexts
- **Jolt** - Uses object pools for template rendering
- **Shockwave** - Uses lock-free queues for event handling
- **Capacitor** - Uses SIMD for data transformation

### API Stability
- **Internal packages** (`internal/`) - No stability guarantees
- **Experimental packages** (`experimental/`) - May change
- **Core packages** - Semver compatibility

## Performance Profiling

### Before Optimization
1. Write correct implementation first
2. Write comprehensive tests
3. Write benchmarks
4. Profile to find bottlenecks
5. Optimize based on data

### Profiling Tools
```bash
# CPU profiling
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof
go tool pprof mem.prof

# Allocation tracking
go test -bench=. -benchmem
```

### Optimization Checklist
- [ ] Benchmark shows performance issue
- [ ] Profile identifies root cause
- [ ] Optimization implemented
- [ ] Benchmarks show improvement
- [ ] Tests still pass
- [ ] No memory leaks introduced
- [ ] Documentation updated

## Code Review Checklist

Before submitting code:
- [ ] All tests pass
- [ ] Benchmarks meet performance targets
- [ ] No memory leaks (checked with `-memprofile`)
- [ ] Documentation is complete
- [ ] Unsafe code is justified and reviewed
- [ ] SIMD code has scalar fallback
- [ ] Examples are provided for complex APIs
- [ ] Architecture decisions are documented

## Common Patterns

### Arena Usage
```go
// Request-scoped allocation
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    arena := arena.NewArena(64 * 1024) // 64KB
    defer arena.Free()

    ctx := arena.New[Context](arena)
    // use context for request lifetime
}
```

### Pool Usage
```go
// Reusable buffers
var bufferPool = pool.NewBufferPool()

func ProcessData(data []byte) {
    buf := bufferPool.Get(len(data))
    defer bufferPool.Put(buf)

    // use buffer
}
```

### Lock-Free Pattern
```go
// MPMC queue
queue := lockfree.NewQueue[*Task](1024)

// Producer
queue.Enqueue(task)

// Consumer
if task := queue.Dequeue(); task != nil {
    process(task)
}
```

## Troubleshooting

### Performance Issues
1. Run benchmarks: `go test -bench=. -benchmem`
2. Profile: `go test -bench=BenchmarkSlow -cpuprofile=cpu.prof`
3. Analyze: `go tool pprof -http=:8080 cpu.prof`

### Memory Leaks
1. Run with profiling: `go test -bench=. -memprofile=mem.prof`
2. Look for growth: `go tool pprof -alloc_space mem.prof`
3. Check arenas are freed, pools return objects

### Race Conditions
1. Run with race detector: `go test -race`
2. Check atomic operations are used correctly
3. Verify memory ordering in lock-free code

## Resources

- [Go Performance Tips](https://github.com/dgryski/go-perfbook)
- [SIMD in Go](https://golang.org/x/arch)
- [Lock-Free Patterns](https://preshing.com/20120612/an-introduction-to-lock-free-programming/)
- [Memory Barriers](https://www.kernel.org/doc/Documentation/memory-barriers.txt)

---

**Remember**: Premature optimization is the root of all evil, but Electron is ALL ABOUT optimization. Every decision is driven by benchmarks, not intuition.
