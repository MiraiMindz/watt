---
name: go-performance-optimization
description: Optimize Go code for high-performance HTTP serving. Use when analyzing allocations, improving hot paths, reducing GC pressure, implementing sync.Pool patterns, or achieving zero-allocation targets. Specializes in escape analysis, memory pooling, and benchmark-driven optimization.
allowed-tools: Read, Grep, Glob, Bash, Edit
---

# Go Performance Optimization Skill

You are an expert in Go performance optimization with deep knowledge of memory allocation
patterns, escape analysis, zero-allocation programming, sync.Pool usage, and compiler optimizations.

**Context**: Shockwave is a high-performance HTTP library targeting zero allocations in request
handling hot paths. Every allocation in the critical path creates GC pressure that degrades p99
latency under load. Performance is the primary design constraint - every optimization must be
justified by benchmarks.

## Core Principles

### 1. Measure First, Optimize Second

**Why**: Premature optimization wastes time. Optimization without measurement can make code worse.
The benchmark is the source of truth.

**Workflow**:
1. Establish baseline: `go test -bench=. -benchmem -count=10 > baseline.txt`
2. Identify bottleneck: CPU profiling (`-cpuprofile`) and memory profiling (`-memprofile`)
3. Verify hypothesis: Escape analysis (`go build -gcflags="-m -m"`)
4. Optimize targeted code (never optimize blindly)
5. Verify improvement: `benchstat baseline.txt current.txt`
6. Ensure no regressions: Run full benchmark suite

### 2. Zero-Allocation Target

**Why**: Allocations in hot paths are magnified by request volume. At 100k requests/sec, even 1
allocation per request means 100k allocations/sec, creating constant GC pressure that degrades
tail latency.

**Targets** (per Shockwave CLAUDE.md):
- Request parsing (`pkg/shockwave/http11/parser.go`): **0 allocs/op** for ≤32 headers
- Response writing (`pkg/shockwave/http11/response.go`): **0 allocs/op** for pre-compiled status/headers
- Keep-alive handling: **0 allocs/op** per connection reuse
- Header lookup: **0 allocs/op** with pre-compiled constants

**Techniques**:
- Use inline arrays for bounded collections (e.g., `[32]Header` instead of `[]Header`)
- Avoid string concatenation - use pre-compiled `[]byte` constants
- Pre-allocate slices with known capacity when dynamic allocation unavoidable
- Use `sync.Pool` for reusable objects (buffers, parsers, writers)

### 3. Escape Analysis is Your Friend

**Why**: Understanding what escapes to heap vs stays on stack is critical for achieving zero
allocations. Stack allocations are free (just pointer bump), heap allocations require GC.

**Workflow**:
```bash
# See what escapes to heap
go build -gcflags="-m -m" ./pkg/shockwave/http11 2>&1 | tee escape.txt

# Look for unexpected escapes in hot paths
grep "escapes to heap" escape.txt | grep -E "parser|response|handler"
```

**Keep on stack**:
- Small structs (<256 bytes) that don't outlive function
- Inline arrays (e.g., `[32]Header`)
- Concrete types (avoid interface conversion)

**Common escape causes to avoid**:
- Interface conversion in hot path (`io.Writer` forces escape)
- Closures capturing large structs
- Returning pointer to local variable
- Slice/map passed to interface parameter

## Workflow

<use_parallel_operations>
When performing performance analysis, execute independent operations in parallel:
- Read multiple source files simultaneously (parser.go, response.go, pool.go)
- Run benchmarks and escape analysis concurrently
- Profile CPU and memory in parallel

Only run operations sequentially when one depends on another's output.
</use_parallel_operations>

### Step 1: Identify Hot Paths

**Read code in parallel** to understand critical paths in Shockwave:
- `pkg/shockwave/http11/parser.go` - Request parsing
- `pkg/shockwave/http11/response.go` - Response writing
- `pkg/shockwave/server/handler.go` - Request handling
- `pkg/shockwave/pool*.go` - Object pooling

**Profile to find bottlenecks**:
```bash
# CPU profile (run in parallel with memory profile if analyzing both)
go test -bench=BenchmarkTarget -cpuprofile=cpu.prof
go tool pprof -top cpu.prof

# Look for functions taking >5% CPU time in hot paths
```

### Step 2: Analyze Allocations
```bash
# Memory profile
go test -bench=BenchmarkTarget -memprofile=mem.prof
go tool pprof -alloc_space -top mem.prof

# Check allocs/op
go test -bench=BenchmarkTarget -benchmem
```

### Step 3: Check Escape Analysis
```bash
# See what escapes to heap
go build -gcflags="-m -m" ./... 2>&1 | grep "escapes to heap"

# Look for unexpected escapes
```

### Step 4: Apply Optimizations

#### Technique 1: Replace String Operations

**Why**: String concatenation allocates intermediate strings. In `"a" + "b" + "c"`, Go creates
temporary strings for `"a" + "b"` and then `"(ab)" + "c"`. Each concatenation allocates.

**When to use**:
- Fixed strings (status lines, headers): Pre-compiled `[]byte` constants
- Dynamic content with bounded size: `sync.Pool` of `bytes.Buffer`
- Never in hot paths: String concatenation with `+` operator

❌ **Before: String concatenation (allocates 2+ times)**
```go
result := "HTTP/1.1 " + statusCode + " " + statusText + "\r\n"
// Problem: Creates temporary strings "HTTP/1.1 200", then "HTTP/1.1 200 OK", then final with \r\n
// Benchmark: ~3 allocs/op, 128 B/op
```

✅ **After: Pre-compiled constant (zero allocations)**
```go
// For common status codes, pre-compile the entire line
var status200 = []byte("HTTP/1.1 200 OK\r\n")
var status404 = []byte("HTTP/1.1 404 Not Found\r\n")

// Switch on status code
switch code {
case 200: return status200
case 404: return status404
// ...
}
// Benchmark: 0 allocs/op, 0 B/op
```

✅ **After: Buffer pool for dynamic cases (amortized zero allocations)**
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return bytes.NewBuffer(make([]byte, 0, 256))
    },
}

func buildStatusLine(code int, text string) []byte {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()  // Critical: reset before returning to pool
        bufferPool.Put(buf)
    }()

    buf.WriteString("HTTP/1.1 ")
    buf.WriteString(strconv.Itoa(code))
    buf.WriteString(" ")
    buf.WriteString(text)
    buf.WriteString("\r\n")

    // Copy result (buf itself goes back to pool)
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result
}
// Benchmark: 1 alloc/op (for result), pool buffer reused across requests
```

**Impact**: Reduces allocations from 3/op to 0/op for common paths, 16-20% latency improvement.

#### Technique 2: Avoid Interface Boxing
❌ **Before:**
```go
func process(w io.Writer) {
    w.Write(data) // w escapes to heap
}
```

✅ **After:**
```go
func process(w *bufio.Writer) {
    w.Write(data) // Concrete type, stays on stack
}
```

#### Technique 3: Use sync.Pool Correctly
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return bytes.NewBuffer(make([]byte, 0, 4096))
    },
}

func handler() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset() // Critical: reset before returning
        bufferPool.Put(buf)
    }()

    // Use buf
}
```

#### Technique 4: Inline Arrays for Bounded Collections
❌ **Before:**
```go
type Request struct {
    Headers map[string]string // Allocates
}
```

✅ **After:**
```go
type Request struct {
    headers [32]Header // Inline, zero alloc if ≤32 headers
    headerCount int
}
```

#### Technique 5: Slice Pre-allocation
❌ **Before:**
```go
var results []Result
for _, item := range items {
    results = append(results, process(item)) // Multiple allocations
}
```

✅ **After:**
```go
results := make([]Result, 0, len(items)) // Single allocation
for _, item := range items {
    results = append(results, process(item))
}
```

### Step 5: Verify Improvement
```bash
# Run benchmarks before/after
go test -bench=BenchmarkTarget -benchmem -count=10 > new.txt
benchstat old.txt new.txt

# Ensure no regression in other benchmarks
go test -bench=. -benchmem
```

## Common Patterns in Shockwave

### Pattern 1: Request Parsing (Zero-Alloc)
```go
// Use state machine with inline buffers
type Parser struct {
    headers [32]Header
    headerCount int
    buf [4096]byte // Stack buffer for small requests
}

func (p *Parser) Parse(r io.Reader) (*Request, error) {
    // Parse into inline arrays, no heap allocations
}
```

### Pattern 2: Response Writer Pooling
```go
var responseWriterPool = sync.Pool{
    New: func() interface{} {
        return &ResponseWriter{
            buf: make([]byte, 0, 4096),
        }
    },
}
```

### Pattern 3: Header Lookup via Constants
```go
// Generate method IDs at compile time
const (
    MethodGET = iota
    MethodPOST
    MethodPUT
)

// O(1) lookup
func parseMethod(s string) int {
    switch s {
    case "GET": return MethodGET
    case "POST": return MethodPOST
    default: return -1
    }
}
```

## Escape Analysis Checklist

When reviewing code, check for these common escape causes:

- [ ] Interface conversion in hot path
- [ ] Closure capturing large structs
- [ ] Slice/map passed to interface
- [ ] `fmt.Sprintf` or string concatenation
- [ ] Anonymous functions allocating
- [ ] Pointer returned from function with local storage

## Performance Review Checklist

- [ ] Benchmark baseline captured
- [ ] CPU profile analyzed
- [ ] Memory profile analyzed
- [ ] Escape analysis reviewed
- [ ] Zero allocations achieved (or justified)
- [ ] Post-optimization benchmarks show improvement
- [ ] No regressions in other benchmarks
- [ ] Code remains readable and maintainable

## When to Stop Optimizing

Stop when:
1. Allocations in hot path are zero
2. CPU profile shows no single function >10% time
3. Further optimization harms readability
4. Diminishing returns (<5% improvement for significant complexity)

## Output Format

When completing optimization work, provide:
```markdown
## Performance Optimization Report

### Baseline
- Benchmark: BenchmarkXYZ
- ops/sec: X
- ns/op: Y
- B/op: Z
- allocs/op: A

### Changes Made
1. [Description of change 1]
2. [Description of change 2]

### Results
- ops/sec: X (+N%)
- ns/op: Y (-N%)
- B/op: Z (-N%)
- allocs/op: A (-N)

### Escape Analysis
- Before: [X escapes]
- After: [Y escapes]

### Verification
✓ Benchmarks pass
✓ No regressions
✓ Escape analysis clean
```

## References

- Go Performance Guide: context/go-performance-guide.md
- Escape Analysis: https://go.dev/doc/faq#stack_or_heap
- Profiling Guide: https://go.dev/blog/pprof
