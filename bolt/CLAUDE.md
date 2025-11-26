# Bolt Framework - Project Constitution

**Version:** 1.0.0
**Last Updated:** 2025-11-13
**Purpose:** Primary source of truth for Bolt framework development

---

## Project Overview

Bolt is a high-performance Go web framework achieving **1.7-3.7x faster** performance than standard library through aggressive optimization, Shockwave HTTP server integration, and a dual API design (Sugared + Unsugared + Generics).

**Performance Targets:**
- 1.5-3x faster than Echo, Gin, Fiber
- <1KB memory per request
- <3 allocations per request
- >80% test coverage
- Zero net/http dependencies

---

## Core Principles

### 1. Shockwave-Only Architecture

**CRITICAL RULE:** This project uses ONLY Shockwave HTTP server. Never use `net/http`.

```go
// ✅ CORRECT
import "github.com/yourusername/shockwave/pkg/shockwave/http11"
import "github.com/yourusername/shockwave/pkg/shockwave/server"

// ❌ FORBIDDEN
import "net/http"
import "net/http/httptest"
```

**Rationale:**
- Shockwave is 40-60% faster than net/http
- Zero-copy request parsing
- Built-in connection pooling
- Optimized for our use case

**Exception:**
- Test utilities may use net/http for test clients only
- Mark with `//nolint:nethttp` comment

### 2. Triple API Design

Bolt provides three complementary APIs:

#### **Sugared API** (Ergonomic, High Performance)
```go
app.Get("/users/:id", func(c *bolt.Context) error {
    return c.JSON(200, map[string]interface{}{
        "id": c.Param("id"),
        "name": "Alice",
    })
})
```
**Target:** 1,500-2,000 ns/op, 5-8 allocs/op

#### **Unsugared API** (Zero-Allocation)
```go
app.GetUnsugared("/users/:id", func(c *bolt.UnsugaredContext) error {
    return c.JSONFields(200,
        bolt.String("id", c.ParamString("id")),
        bolt.String("name", "Alice"),
    )
})
```
**Target:** <500 ns/op, 0-2 allocs/op

#### **Generics API** (Type-Safe, Metadata-Rich)
```go
app.Get[User]("/users/:id", func(c *bolt.Context) bolt.Data[User] {
    user, err := db.GetUser(c.Param("id"))
    if err != nil {
        return bolt.NotFound[User](err)
    }
    return bolt.OK(user).WithMeta("cached", true)
})
```
**Target:** 1,500-2,000 ns/op, type-safe, automatic error handling

### 3. Performance First

Every feature must be benchmarked. Performance regressions are blocking issues.

**Benchmark Requirements:**
- Every new feature MUST include benchmarks
- Run benchmarks: `go test -bench=. -benchmem`
- Compare against baseline before merging
- Document performance characteristics in code comments

**Optimization Techniques:**
- Object pooling (sync.Pool) for all request-scoped objects
- Zero-copy string/byte conversions (unsafe, but safe)
- Inline storage for small, fixed-size data
- Pre-allocated slices with known capacity
- Fast-path detection (skip work when possible)

### 4. Minimal Dependencies

**Required Dependencies:**
- `github.com/goccy/go-json` - Fast JSON (2-3x faster than stdlib)
- `github.com/json-iterator/go` - Alternative JSON library

**Optional Dependencies** (for addons):
- `gorm.io/gorm` - For CRUD builder
- `github.com/casbin/casbin/v2` - For authorization middleware
- `go.uber.org/zap` - For structured logging

**Forbidden:**
- No web framework dependencies (Gin, Echo, Fiber, etc.)
- No ORM in core framework
- No heavy dependencies

---

## Architecture

### Project Structure

```
bolt/
├── core/                    # Core framework
│   ├── app.go              # Main App struct
│   ├── context.go          # Request context
│   ├── router.go           # Hybrid routing system
│   ├── generics.go         # Generic API (Data[T])
│   └── types.go            # Core type definitions
│
├── pool/                    # Object pooling
│   ├── context_pool.go     # Context pooling
│   ├── buffer_pool.go      # Three-tier buffer pool
│   └── arena.go            # Arena allocators (build tag)
│
├── middleware/              # Built-in middleware
│   ├── cors.go
│   ├── logger.go
│   ├── recovery.go
│   └── jwt/
│
├── shockwave/              # Shockwave integration
│   ├── server.go           # Server wrapper
│   ├── adapter.go          # Type adapters
│   └── response.go         # Response helpers
│
├── benchmarks/             # Competitive benchmarks
│   ├── bolt_test.go
│   ├── gin_test.go
│   ├── echo_test.go
│   └── fiber_test.go
│
├── examples/               # Usage examples
│   ├── hello/
│   ├── rest-api/
│   └── generics/
│
└── .claude/               # Claude Code ecosystem
    ├── agents/
    ├── skills/
    ├── commands/
    └── settings/
```

### Component Responsibilities

#### **App (core/app.go)**
- Route registration (Get, Post, Put, Delete, etc.)
- Middleware management
- Server lifecycle (Listen, Shutdown)
- Shockwave server integration

#### **Router (core/router.go)**
- Hybrid routing (static hash map + radix tree)
- O(1) static route lookup
- O(log n) dynamic route lookup
- Thread-safe concurrent access

#### **Context (core/context.go)**
- Request/response handling
- Parameter access (path, query, header)
- JSON/text/HTML response methods
- State storage for middleware

#### **Pool (pool/)**
- Context pooling with sync.Pool
- Three-tier buffer pooling (512B, 4KB, 16KB)
- Arena allocators (experimental, build tag)

#### **Shockwave Integration (shockwave/)**
- Wrap Shockwave HTTP server
- Adapt Shockwave types to Bolt types
- Zero-copy request mapping

---

## Coding Standards

### File Organization

**File Naming:**
- `feature.go` - Main implementation
- `feature_test.go` - Unit tests
- `feature_bench_test.go` - Benchmarks
- `feature_arena.go` - Arena version (build tag: `goexperiment.arenas`)
- `feature_arena_fallback.go` - Standard version (build tag: `!goexperiment.arenas`)

**Package Structure:**
- One package per directory
- Package name matches directory name
- Internal packages in `internal/`

### Code Style

**Formatting:**
```bash
gofmt -s -w .
```

**Naming Conventions:**
- Exported types: `PascalCase`
- Unexported types: `camelCase`
- Constants: `PascalCase` or `SCREAMING_SNAKE_CASE` (for enums)
- Interfaces: End with `-er` (Handler, Router, Pooler)

**Comments:**
```go
// Handler defines a request handler function.
// It receives a Context and returns an error.
type Handler func(*Context) error

// Get registers a GET route.
//
// Example:
//     app.Get("/users/:id", getUser)
//
// Performance: <50ns routing overhead
func (app *App) Get(path string, handler Handler) *ChainLink {
    // Implementation
}
```

**Error Handling:**
```go
// ✅ CORRECT: Return errors, don't panic
func (c *Context) JSON(status int, data interface{}) error {
    if err := encode(data); err != nil {
        return fmt.Errorf("json encode: %w", err)
    }
    return nil
}

// ❌ WRONG: Don't panic in library code
func (c *Context) JSON(status int, data interface{}) {
    if err := encode(data); err != nil {
        panic(err) // DON'T DO THIS
    }
}
```

### Performance Standards

**Zero-Allocation Paths:**
```go
// Hot paths MUST have zero allocations
// Verify with: go test -bench=BenchmarkHotPath -benchmem
// Expected: 0 B/op, 0 allocs/op

func BenchmarkContextAcquire(b *testing.B) {
    pool := NewContextPool()
    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        ctx := pool.Acquire()
        pool.Release(ctx)
    }
    // MUST report: 0 B/op, 0 allocs/op
}
```

**Object Pooling:**
```go
// ✅ CORRECT: Pool request-scoped objects
var contextPool = sync.Pool{
    New: func() interface{} {
        return &Context{
            store: make(map[string]interface{}, 4),
        }
    },
}

ctx := contextPool.Get().(*Context)
defer contextPool.Put(ctx)

// ❌ WRONG: Allocate on every request
ctx := &Context{
    store: make(map[string]interface{}),
}
```

**Unsafe Usage:**
```go
// ✅ SAFE: Read-only zero-copy conversion
func stringToBytes(s string) []byte {
    return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Usage: ONLY when bytes won't be modified
pathBytes := stringToBytes(path) // Read-only access OK

// ❌ UNSAFE: Modifying converted bytes
pathBytes[0] = '/'  // DON'T DO THIS - undefined behavior
```

---

## Testing Requirements

### Coverage Targets

- **Overall:** >80% line coverage
- **Core packages:** >90% line coverage
- **Hot paths:** 100% coverage

**Check coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Types

#### 1. Unit Tests
```go
func TestContextParam(t *testing.T) {
    ctx := &Context{params: map[string]string{"id": "123"}}
    if got := ctx.Param("id"); got != "123" {
        t.Errorf("expected 123, got %s", got)
    }
}
```

#### 2. Benchmark Tests
```go
func BenchmarkRouterLookup(b *testing.B) {
    router := NewRouter()
    router.Add(MethodGet, "/users/:id", handler)

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        router.Lookup(MethodGet, "/users/123")
    }
}
```

#### 3. Race Detection Tests
```go
func TestConcurrentRequests(t *testing.T) {
    // Run with: go test -race
    app := New()
    app.Get("/test", handler)

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Simulate concurrent requests
        }()
    }
    wg.Wait()
}
```

#### 4. Integration Tests
```go
func TestShockwaveIntegration(t *testing.T) {
    app := New()
    app.Get("/ping", func(c *Context) error {
        return c.JSON(200, map[string]string{"message": "pong"})
    })

    // Use Shockwave test utilities
    // Verify full request/response cycle
}
```

### Test Execution

**Run all tests:**
```bash
go test ./...
go test -race ./...
go test -cover ./...
```

**Run benchmarks:**
```bash
go test -bench=. -benchmem ./...
go test -bench=BenchmarkRouter -benchtime=10s
```

**Profile performance:**
```bash
go test -bench=. -cpuprofile=cpu.prof
go test -bench=. -memprofile=mem.prof
go tool pprof cpu.prof
```

---

## Build System

### Build Tags

**Arena Allocators:**
```bash
GOEXPERIMENT=arenas go build
```

**Custom HTTP (when available):**
```bash
go build -tags=customhttp
```

**All features:**
```bash
GOEXPERIMENT=arenas go build -tags=customhttp
```

### Build Validation

**Pre-build checks:**
```bash
# Format code
gofmt -s -w .

# Vet code
go vet ./...

# Run tests
go test ./...

# Check coverage
go test -cover ./... | grep -v "100.0%"

# Build
go build ./...
```

---

## Git Workflow

### Branch Naming

- `main` - Stable releases
- `develop` - Development branch
- `feature/feature-name` - New features
- `fix/bug-description` - Bug fixes
- `perf/optimization-name` - Performance improvements
- `docs/documentation-update` - Documentation

### Commit Messages

**Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `perf` - Performance improvement
- `refactor` - Code refactoring
- `test` - Test additions/improvements
- `docs` - Documentation
- `chore` - Build, dependencies, etc.

**Example:**
```
perf(router): optimize static route lookup

Implement hash map for static routes achieving O(1) lookup.
Reduces routing overhead from 200ns to 45ns (77% improvement).

Benchmarks:
- BenchmarkStaticRoute: 1200ns -> 450ns (62% faster)
- Zero allocation maintained (0 allocs/op)

Closes #123
```

### Pre-Commit Checklist

- [ ] Code formatted (`gofmt -s -w .`)
- [ ] Tests pass (`go test ./...`)
- [ ] Race detector clean (`go test -race ./...`)
- [ ] Coverage >80% (`go test -cover ./...`)
- [ ] Benchmarks run (`go test -bench=. -benchmem`)
- [ ] No net/http imports (`grep -r "net/http" --include="*.go"`)
- [ ] Documentation updated

---

## Performance Philosophy

### Optimization Principles

1. **Measure First:** Never optimize without benchmarks
2. **Profile Everything:** Use pprof to find bottlenecks
3. **Pool Aggressively:** Reuse allocations via sync.Pool
4. **Zero-Copy When Safe:** Use unsafe for read-only conversions
5. **Fast Path First:** Optimize common cases, handle edge cases separately
6. **Inline Hot Functions:** Help compiler optimize hot paths
7. **Avoid Interface{} in Hot Paths:** Use concrete types or generics

### Common Optimizations

**Before:**
```go
func (c *Context) Param(key string) string {
    if c.params == nil {
        c.params = make(map[string]string)
    }
    return c.params[key]
}
// Allocation: 1 map allocation if nil
```

**After:**
```go
func (c *Context) Param(key string) string {
    // Check inline storage first (no allocation)
    for i := 0; i < len(c.paramsBuf); i++ {
        if c.paramsBuf[i].key == key {
            return c.paramsBuf[i].val
        }
    }
    // Fall back to map
    return c.params[key]
}
// Allocation: 0 for ≤4 params
```

### Benchmark Targets

| Operation | Target ns/op | Target allocs/op |
|-----------|--------------|------------------|
| Static route lookup | <50 | 0 |
| Dynamic route lookup | <200 | 0 |
| Context acquire/release | <30 | 0 |
| JSON encoding (small) | <500 | 0-1 |
| Parameter extraction | <20 | 0 |
| Middleware execution | <100 | 0 |

---

## Documentation Standards

### Code Documentation

**Package-level:**
```go
// Package bolt provides a high-performance web framework built on Shockwave.
//
// Bolt achieves 1.7-3.7x better performance than standard library through:
//   - Shockwave HTTP server integration
//   - Dual API design (Sugared + Unsugared)
//   - Aggressive object pooling
//   - Zero-copy optimizations
//
// Example usage:
//     app := bolt.New()
//     app.Get("/hello", func(c *bolt.Context) error {
//         return c.JSON(200, map[string]string{"message": "Hello, World!"})
//     })
//     app.Listen(":8080")
package bolt
```

**Function-level:**
```go
// Get registers a GET route with the given path and handler.
//
// The path may include parameters using :param syntax or wildcards using *param.
// Parameters are extracted and made available via Context.Param(key).
//
// Example:
//     app.Get("/users/:id", func(c *bolt.Context) error {
//         userID := c.Param("id")
//         return c.JSON(200, getUserByID(userID))
//     })
//
// Performance: <50ns routing overhead for static routes,
// <200ns for dynamic routes.
func (app *App) Get(path string, handler Handler) *ChainLink {
    return app.addRoute(MethodGet, path, handler)
}
```

### README Structure

Every package MUST have a README.md with:
1. Package overview
2. Installation instructions
3. Quick start example
4. API reference
5. Performance characteristics
6. Examples

---

## Security Guidelines

### Input Validation

```go
// ✅ CORRECT: Validate and sanitize input
func (c *Context) BindJSON(v interface{}) error {
    if c.Request.ContentLength() > MaxRequestSize {
        return ErrRequestTooLarge
    }

    decoder := json.NewDecoder(c.Request.Body())
    decoder.DisallowUnknownFields() // Strict parsing
    return decoder.Decode(v)
}
```

### Error Handling

```go
// ✅ CORRECT: Don't expose internal errors
func (c *Context) handleError(err error) error {
    // Log full error internally
    log.Error("internal error", "error", err)

    // Return sanitized error to client
    return c.JSON(500, map[string]string{
        "error": "Internal server error",
    })
}

// ❌ WRONG: Exposing internal details
return c.JSON(500, map[string]string{
    "error": err.Error(), // May leak sensitive info
})
```

### Dependency Management

```bash
# Audit dependencies
go list -m all
go mod tidy
go mod verify

# Check for vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

---

## Claude Code Integration

This project uses Claude Code ecosystem for development automation.

**Agents:** `.claude/agents/`
- `architect.md` - System design and planning
- `implementer.md` - Code implementation
- `tester.md` - Test writing and validation
- `benchmarker.md` - Performance benchmarking

**Skills:** `.claude/skills/`
- `shockwave-integration/` - Shockwave integration patterns
- `performance-optimization/` - Optimization techniques
- `generics-api/` - Generic API design

**Commands:** `.claude/commands/`
- `/plan` - Create implementation plan
- `/implement` - Implement from plan
- `/test` - Run test suite
- `/benchmark` - Run benchmarks
- `/optimize` - Optimize performance

**Hooks:** `.claude/settings/hooks.json`
- PreToolUse - Validate before writes
- PostToolUse - Run tests after changes
- SubagentStop - Report completion

**See:** `MASTER_PROMPTS.md` for detailed prompts and workflows.

---

## Continuous Improvement

### Regular Audits

**Weekly:**
- Run full benchmark suite
- Check test coverage
- Review performance profiles

**Monthly:**
- Dependency updates
- Security audit
- Competitive benchmarking vs Echo/Gin/Fiber

**Per Release:**
- Full regression testing
- Performance validation
- Documentation review

---

## Questions and Support

**For development questions:**
1. Check MASTER_PROMPTS.md
2. Review BOLT_FRAMEWORK_BLUEPRINT.md
3. Check implementation_plan.md

**For Shockwave integration:**
1. Review ../shockwave/USAGE_GUIDE.md
2. Check .claude/skills/shockwave-integration/

---

**This is the constitution. All code MUST comply with these rules.**

**Last Updated:** 2025-11-13
**Version:** 1.0.0
