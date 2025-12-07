# Capacitor DAL/DTO Project Constitution

## Project Overview

Capacitor is a high-performance Data Access Layer (DAL) and Data Transfer Object (DTO) library for the WATT Toolkit. It is designed with a **ZERO allocation philosophy** and focuses on **CPU and Memory Performance & Efficiency**.

### Project Goals
- Build the fastest, most performant DAL/DTO layer for Go
- Support multiple storage backends (in-memory, on-disk, network, specialized databases)
- Provide ORM-like API leveraging Go generics
- Enable composable multi-layer caching and data management
- Support encryption, validation, and comprehensive data type handling
- Maintain zero-allocation patterns wherever possible

## Architecture Principles

### 1. Zero Allocation Philosophy
- Minimize or eliminate heap allocations in hot paths
- Use object pooling extensively (sync.Pool, custom pools)
- Prefer stack allocation over heap allocation
- Reuse buffers and data structures
- Avoid unnecessary interface conversions
- Use unsafe operations judiciously when they provide significant performance gains

### 2. Performance First
- Benchmark everything - no optimization without measurement
- CPU efficiency is as important as memory efficiency
- Avoid reflection in hot paths
- Use generics to eliminate type assertions
- Inline critical functions
- Profile regularly (CPU, memory, allocations)

### 3. Composable Architecture
- Layer-based design: users configure which layers they need
- Each layer should be independent and swappable
- Support for L1 (low-term), L2 (mid-term), L3 (long-term) caching patterns
- Clean separation between cache, state management, and persistent storage

### 4. Type Safety
- Leverage Go generics for type-safe operations
- Support full PostgreSQL type system
- Handle specialized types: geographical, time-based, JSON/BLOB, vector embeddings
- Provide compile-time type checking wherever possible

### 5. Dependency Injection
- User initializes external dependencies (databases, cache libraries)
- Capacitor provides high-performance abstraction layer
- No hidden dependencies or magic initialization
- Clear configuration API

## Code Standards

### File Structure
```
capacitor/
├── CLAUDE.md                 # This file
├── README.md                 # User-facing documentation
├── .claude/                  # Claude Code ecosystem
│   ├── settings.json         # Project settings
│   ├── agents/              # Specialized agents
│   ├── skills/              # Agent skills
│   └── commands/            # Slash commands
├── pkg/
│   ├── capacitor/           # Main package
│   │   ├── dal.go           # Core DAL interface
│   │   ├── dto.go           # DTO definitions
│   │   ├── layer.go         # Layer abstraction
│   │   ├── config.go        # Configuration types
│   │   └── pool/            # Pooling utilities
│   ├── cache/               # Cache layer implementations
│   │   ├── memory/          # In-memory cache
│   │   ├── disk/            # On-disk cache
│   │   └── network/         # Network cache
│   ├── database/            # Database layer implementations
│   │   ├── sql/             # SQL databases
│   │   ├── nosql/           # NoSQL databases
│   │   └── vector/          # Vector databases
│   ├── types/               # Data type support
│   │   ├── postgres/        # PostgreSQL types
│   │   ├── geo/             # Geographical types
│   │   ├── time/            # Time types
│   │   └── json/            # JSON/BLOB types
│   └── validation/          # Validation framework
├── benchmarks/              # Benchmark suites
│   ├── comparative/         # Comparisons with other solutions
│   └── internal/            # Internal benchmarks
├── examples/                # Usage examples
└── tools/                   # Development tools
```

### Naming Conventions
- Use descriptive names: `UserDataAccessLayer`, not `UDAL`
- Interfaces end with `er`: `Cacher`, `Storer`, `Validator`
- Implementations are concrete: `MemoryCache`, `PostgresStore`
- Generic type parameters: `T` for single type, `K, V` for key-value
- Private fields start with lowercase, exported with uppercase
- Method names are verbs: `Get`, `Set`, `Delete`, `Validate`

### Testing Requirements
- Every public function MUST have tests
- Minimum 85% code coverage
- Use table-driven tests for multiple scenarios
- Benchmark all performance-critical paths
- Test with race detector: `go test -race`
- Test with various data sizes: small (1KB), medium (100KB), large (10MB)

### Benchmarking Standards
```go
// Every benchmark must follow this pattern:
func BenchmarkOperation(b *testing.B) {
    // Setup
    setup := createTestSetup()
    defer cleanup(setup)

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        // Operation being benchmarked
        _ = operation(setup)
    }
}

// Include comparative benchmarks
func BenchmarkCapacitorVsCompetitor(b *testing.B) {
    b.Run("Capacitor", benchmarkCapacitor)
    b.Run("CompetitorX", benchmarkCompetitorX)
}
```

### Documentation Requirements
- All exported types, functions, and methods MUST have godoc comments
- Comments should explain WHY, not WHAT
- Include usage examples for complex APIs
- Document performance characteristics
- Note any unsafe operations and their rationale
- Link to relevant benchmarks

### Code Quality Rules
1. **Never use `any` type** - Always use concrete types or generics
2. **Avoid reflection** - Use generics instead
3. **No panics in library code** - Return errors
4. **Context-aware** - Pass context.Context for cancellation
5. **No global state** - Everything through configuration
6. **Explicit errors** - Wrap errors with context using fmt.Errorf with %w
7. **Resource cleanup** - Always provide Close() or Cleanup() methods
8. **Thread-safe by default** - Document when not

### Performance Patterns

#### Pool Usage
```go
// Always use pools for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func operation() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    // Use buf
}
```

#### Inlining
```go
// Keep hot-path functions small for inlining
//go:inline
func fastOperation(x int) int {
    return x * 2
}
```

#### Unsafe Optimization
```go
// Document all unsafe usage
// SAFETY: This is safe because we control both the source and destination
// and ensure proper alignment and size.
func unsafeStringToBytes(s string) []byte {
    return unsafe.Slice(unsafe.StringData(s), len(s))
}
```

## Error Handling

### Error Types
Define custom error types for each layer:
```go
type CacheError struct {
    Op    string // Operation that failed
    Layer string // Cache layer
    Err   error  // Underlying error
}

func (e *CacheError) Error() string {
    return fmt.Sprintf("cache error in %s during %s: %v", e.Layer, e.Op, e.Err)
}

func (e *CacheError) Unwrap() error {
    return e.Err
}
```

### Error Handling Pattern
```go
// Always provide context in errors
if err := operation(); err != nil {
    return fmt.Errorf("failed to perform operation on %s: %w", resource, err)
}
```

## API Design Principles

### Generic DAL Interface
```go
type DAL[K comparable, V any] interface {
    Get(ctx context.Context, key K) (V, error)
    Set(ctx context.Context, key K, value V, opts ...SetOption) error
    Delete(ctx context.Context, key K) error
    Exists(ctx context.Context, key K) (bool, error)
}
```

### Layered Configuration
```go
type LayerConfig struct {
    Name     string        // Layer name (L1, L2, L3, persistent)
    Type     LayerType     // Cache, StateManagement, Database
    Backend  Backend       // User-provided backend
    TTL      time.Duration // Time to live
    MaxSize  int64         // Maximum size in bytes
    Strategy EvictionStrategy
}

type Config struct {
    Layers []LayerConfig
    // Encryption, validation, etc.
}
```

### Builder Pattern
```go
// Use builder pattern for complex configuration
config := capacitor.NewConfig().
    WithL1Cache(memoryBackend, 1*time.Minute, 100*MB).
    WithL2Cache(redisBackend, 1*time.Hour, 1*GB).
    WithPersistence(postgresBackend).
    WithEncryption(aes256).
    WithValidation(validator).
    Build()
```

## Dependencies and External Libraries

### Allowed Dependencies
- Standard library (preferred)
- Well-maintained, performance-focused libraries
- No abandoned or experimental dependencies
- Document rationale for each external dependency

### Integration Pattern
```go
// Users provide initialized backends
type Backend interface {
    Get(ctx context.Context, key []byte) ([]byte, error)
    Set(ctx context.Context, key, value []byte) error
    Delete(ctx context.Context, key []byte) error
    Close() error
}

// User code
redis := initializeRedis()
postgres := initializePostgres()

dal := capacitor.New(
    capacitor.WithCache("L1", NewRedisBackend(redis)),
    capacitor.WithDatabase("persistent", NewPostgresBackend(postgres)),
)
```

## Security Considerations

### Encryption
- Support at-rest and in-transit encryption
- Configurable per layer
- Use standard crypto libraries
- No custom crypto algorithms

### Validation
- Input validation at boundaries
- Schema validation for structured data
- Sanitization for injection prevention
- Configurable validation rules

## Monitoring and Observability

### Metrics
Expose metrics for:
- Operation latency (Get, Set, Delete)
- Cache hit/miss rates per layer
- Allocation counts
- Error rates
- Throughput

### Tracing
- Support OpenTelemetry
- Trace context propagation
- Layer-by-layer visibility

## Versioning and Compatibility

- Semantic versioning (semver)
- Maintain backward compatibility within major versions
- Document breaking changes in CHANGELOG
- Deprecation warnings for 2 minor versions before removal

## Development Workflow

### Before Implementation
1. Review this CLAUDE.md file
2. Check existing patterns in the codebase
3. Write tests first (TDD)
4. Write benchmarks for performance-critical code
5. Document API design decisions

### During Implementation
1. Follow zero-allocation principles
2. Add inline comments for complex logic
3. Use meaningful variable names
4. Keep functions small and focused
5. Run tests frequently

### Before Committing
1. Run all tests: `go test ./...`
2. Run benchmarks: `go test -bench=. -benchmem ./...`
3. Check coverage: `go test -cover ./...`
4. Run race detector: `go test -race ./...`
5. Format code: `gofmt -w .`
6. Run linter: `golangci-lint run`
7. Update documentation

### Commit Message Format
```
type(scope): brief description

Detailed explanation of the change and why it was necessary.

Benchmark results (if applicable):
- Before: X ns/op, Y allocs/op
- After: X ns/op, Y allocs/op

Fixes #issue_number
```

Types: feat, fix, perf, refactor, test, docs, chore

## Comparison Standards

When comparing with other solutions:
- Compare apples to apples (same operations, same data sizes)
- Include setup/teardown in benchmarks
- Test with realistic workloads
- Document hardware specifications
- Include multiple runs for statistical significance
- Compare against: GORM, sqlc, ent, custom solutions

## Questions to Ask

Before implementing any feature, ask:
1. Can this be done without allocations?
2. Can we use a pool for this?
3. Can we use generics to avoid reflection?
4. Is this the simplest solution?
5. How will this perform at scale?
6. What are the failure modes?
7. How will users debug this?
8. Does this follow the project's patterns?

## Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Performance Guide](https://github.com/dgryski/go-perfbook)
- [High Performance Go](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)
- [WATT Toolkit - Bolt Framework](../bolt/)
- [WATT Toolkit - Shockwave HTTP Engine](../shockwave/)

---

When in doubt, prioritize:
1. Correctness
2. Performance
3. Simplicity
4. Maintainability

Never sacrifice correctness for performance, but always consider performance implications.
