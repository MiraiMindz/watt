# Shockwave HTTP Library - Project Constitution

## Project Overview

Shockwave is a production-grade, high-performance HTTP library for Go designed to replace `net/http` with significant performance improvements through:
- Zero-allocation parsing
- Multiple memory management strategies (arena, green tea GC, standard pooling)
- Comprehensive protocol support (HTTP/1.1, HTTP/2, HTTP/3, WebSocket)
- Advanced socket-level optimizations

**Performance is the primary design constraint. Every change must be justified by benchmarks.**

---

## Architecture Principles

### 1. Zero-Allocation Philosophy
- **Critical path must be zero-allocation**: Request parsing, header processing, and response writing should not allocate
- Use inline arrays (max 32 headers) to avoid heap allocations
- Leverage `sync.Pool` for all reusable objects
- Profile every optimization with `-benchmem`

### 2. Memory Management Hierarchy
1. **Arena Allocation** (experimental, `arenas` build tag) - Zero GC pressure for request lifetime
2. **Green Tea GC** (`greenteagc` build tag) - Spatial/temporal locality optimization
3. **Standard Pooling** (default) - Object pooling with `sync.Pool`

Choose the appropriate mode based on workload characteristics.

### 3. Protocol Layering
```
Application Layer
    ↓
Protocol Layer (HTTP/1.1, HTTP/2, HTTP/3, WebSocket)
    ↓
Transport Layer (TCP, QUIC)
    ↓
Socket Optimizations (TCP_QUICKACK, TCP_FASTOPEN, etc.)
```

Each layer must be independently testable and benchmarkable.

### 4. Compatibility First
- Maintain `net/http` compatibility via adapter layer
- Support drop-in replacement scenarios
- Follow Go stdlib conventions for APIs

---

## Code Standards

### File Organization
```
pkg/shockwave/
├── http11/          # HTTP/1.1 implementation
├── http2/           # HTTP/2 implementation
├── http3/           # HTTP/3/QUIC implementation
├── websocket/       # WebSocket implementation
├── tls/             # TLS and ACME
├── client/          # HTTP client with pooling
├── server/          # Server implementations
├── memory/          # Memory management (arena, pools)
├── socket/          # Socket-level optimizations
└── internal/        # Shared internal utilities
```

### Naming Conventions
- **Files**: `snake_case.go` (e.g., `buffer_pool.go`)
- **Platform-specific**: `file_linux.go`, `file_darwin.go`
- **Build tags**: `file_arena.go` for arena-specific code
- **Types**: `PascalCase` for exported, `camelCase` for internal
- **Constants**: `PascalCase` for exported, `camelCase` for package-level

### Build Tags
- `customhttp` - Enable custom HTTP implementation
- `arenas` - Enable Go arena allocations (requires `GOEXPERIMENT=arenas`)
- `greenteagc` - Enable Green Tea GC optimizations
- Platform tags: `linux`, `darwin`, `windows` for socket tuning

### Documentation Requirements
- Every exported function must have a doc comment
- Performance-critical functions must include allocation behavior
- Example:
  ```go
  // ParseRequest parses an HTTP request from r.
  // It performs zero allocations for requests with ≤32 headers.
  // Returns non-nil error if the request is malformed.
  func ParseRequest(r io.Reader) (*Request, error)
  ```

---

## Performance Requirements

### Benchmarking Standards
1. **Always benchmark before and after changes**
   ```bash
   go test -bench=. -benchmem -count=10 > old.txt
   # Make changes
   go test -bench=. -benchmem -count=10 > new.txt
   benchstat old.txt new.txt
   ```

2. **Critical benchmarks must show**:
   - ops/sec improvement
   - Allocation reduction (B/op, allocs/op)
   - Memory usage (via profiling)

3. **Regression tolerance**: Max 5% performance regression in any benchmark

### Allocation Targets
- **Request parsing**: 0 allocs for ≤32 headers
- **Response writing**: 0 allocs for pre-compiled status/headers
- **Keep-alive handling**: 0 allocs per connection reuse
- **Header lookup**: O(1) with pre-compiled constants

### Profiling Requirements
- CPU profiling for hot paths
- Memory profiling for allocation analysis
- Escape analysis to prevent unintended heap allocations
  ```bash
  go build -gcflags="-m -m" 2>&1 | grep "escapes to heap"
  ```

---

## Testing Requirements

### Test Coverage
- Minimum 80% coverage for core protocol implementations
- 100% coverage for security-critical code (TLS, validation)
- Edge cases: malformed input, protocol violations, resource exhaustion

### Test Categories
1. **Unit tests**: `*_test.go` alongside implementation
2. **Integration tests**: `integration_test.go` for protocol compliance
3. **Benchmark tests**: `*_bench_test.go` for performance validation
4. **Fuzz tests**: `*_fuzz_test.go` for input validation

### Protocol Compliance Testing
- HTTP/1.1: Test against RFC 7230-7235
- HTTP/2: Test against RFC 7540
- HTTP/3: Test against RFC 9114
- WebSocket: Test against RFC 6455

### Test Execution
```bash
# All tests with all build tags
go test ./... -tags "customhttp,linux"

# With race detector
go test -race ./...

# With coverage
go test -cover ./... -coverprofile=coverage.out

# Benchmarks
go test -bench=. -benchmem ./...
```

---

## Security Standards

### TLS Requirements
- Minimum TLS 1.2 (1.3 preferred)
- No weak cipher suites
- Proper certificate validation
- Session resumption with secure caching

### Input Validation
- All external input must be validated
- Bounds checking for all array/slice access
- Prevent integer overflow in length calculations
- Timeout all network operations

### Memory Safety
- No unsafe pointers unless absolutely necessary (document why)
- All unsafe usage must have safety proof in comments
- Bounds checking cannot be disabled

---

## Socket Optimization Strategy

### Linux-Specific (`tuning_linux.go`)
- `TCP_QUICKACK` - Immediate ACKs, reduces latency
- `TCP_DEFER_ACCEPT` - Don't wake server until data arrives
- `TCP_FASTOPEN` - Reduce connection establishment latency
- Zero-copy sendfile for large responses

### Cross-Platform (`tuning.go`)
- `TCP_NODELAY` - Disable Nagle's algorithm
- Keep-alive configuration
- Buffer size tuning (SO_RCVBUF, SO_SNDBUF)

### When to Apply
- Apply conservatively with feature detection
- Provide fallback for unsupported platforms
- Document performance impact of each optimization

---

## Pre-Compiled Constants Strategy

### Dual Representation
Always provide both byte slice and string versions:
```go
var (
    statusOKBytes  = []byte("HTTP/1.1 200 OK\r\n")
    statusOKString = "HTTP/1.1 200 OK\r\n"
)
```

### Use Cases
- Status lines for common codes (200, 404, 500, etc.)
- Standard headers (Content-Type, Content-Length, etc.)
- Common MIME types
- JSON responses for APIs
- Method lookup tables

---

## Git Workflow

### Branch Strategy
- `main` - Production-ready code
- `develop` - Integration branch
- `feature/*` - Feature development
- `perf/*` - Performance optimization
- `fix/*` - Bug fixes

### Commit Messages
```
<type>(<scope>): <subject>

<body>

Performance Impact:
- Benchmark: <benchmark name>
- Before: <ops/sec, allocs/op>
- After: <ops/sec, allocs/op>
- Improvement: <percentage>
```

Types: `feat`, `fix`, `perf`, `refactor`, `test`, `docs`, `chore`

### Performance Validation Required
- All `perf/*` branches must include benchmark comparison
- All PRs to `main` must pass benchmark regression tests

---

## Common Pitfalls to Avoid

### ❌ Don't Do This
1. **String concatenation in hot paths** - Use `[]byte` and pre-allocated buffers
2. **`fmt.Sprintf` for fixed-format output** - Use pre-compiled constants
3. **`strings.Split`** - Write custom zero-allocation parser
4. **Interface boxing in tight loops** - Use concrete types
5. **Defer in hot paths** - Manual cleanup when performance-critical
6. **Map lookups for fixed sets** - Use switch or array lookup

### ✅ Do This Instead
1. Use `copy()` for byte slice operations
2. Pre-compile format strings as `[]byte`
3. State machine parsers with inline buffers
4. Struct composition over interfaces where possible
5. Explicit cleanup with clear lifetime management
6. Const-based method IDs with switch statements

---

## Performance Debugging Workflow

1. **Identify the bottleneck**
   ```bash
   go test -bench=BenchmarkTarget -cpuprofile=cpu.prof
   go tool pprof -http=:8080 cpu.prof
   ```

2. **Check allocations**
   ```bash
   go test -bench=BenchmarkTarget -memprofile=mem.prof
   go tool pprof -alloc_space -http=:8080 mem.prof
   ```

3. **Verify escape analysis**
   ```bash
   go build -gcflags="-m -m" 2>&1 | grep -A5 "FunctionName"
   ```

4. **Compare with baseline**
   ```bash
   benchstat baseline.txt current.txt
   ```

---

## Decision Log Format

For significant architectural decisions, add entries to `docs/decisions/`:

```markdown
# ADR-NNN: Title

## Status
Accepted | Rejected | Superseded

## Context
What is the issue we're facing?

## Decision
What did we decide?

## Consequences
- Performance impact: <benchmark data>
- Memory impact: <allocation data>
- Maintenance impact: <complexity analysis>
```

---

## When in Doubt

1. **Measure, don't guess** - Run benchmarks
2. **Profile, don't assume** - Use pprof
3. **Test, don't hope** - Write tests first
4. **Document, don't trust memory** - Explain why, not what

**The benchmark is the source of truth.**

---

## Claude 4.5 Interaction Patterns

This section provides guidance for working effectively with Claude Sonnet 4.5 on Shockwave development.

### Explicit Instructions with Context

Claude 4.5 excels at precise instruction following when given explicit goals and context.

**Effective:**
```
Optimize ParseRequest in pkg/shockwave/http11/parser.go to achieve zero allocations.
This is critical because it's called for every HTTP request, and allocations here
create GC pressure that degrades p99 latency. Target: 0 allocs/op while maintaining
RFC 7230 compliance.
```

**Less Effective:**
```
Make the parser faster
```

### Default to Action for Implementation

<default_to_action>
When working on Shockwave optimizations or bug fixes, default to implementing
changes rather than only suggesting them. Use available tools to discover missing
details (read files, run benchmarks, check escape analysis) rather than making
assumptions.

The exception: Always verify before:
- Committing changes to git
- Modifying test files (could break compliance verification)
- Changing public APIs (could break compatibility)
</default_to_action>

### Investigate Before Answering

<investigate_before_answering>
Never speculate about code you haven't read. When analyzing performance issues:

1. Read the actual implementation files
2. Run benchmarks to measure current state
3. Use escape analysis to verify assumptions
4. Examine git history if behavior changed
5. Only then provide evidence-based recommendations

Make zero claims about code before investigating. The benchmark is the source of truth.
</investigate_before_answering>

### Parallel Tool Execution

<use_parallel_tool_calls>
Shockwave optimization benefits from parallel operations. When reading multiple files
or running independent operations, execute them in parallel:

Examples:
- Read parser.go, pool.go, and benchmark files simultaneously
- Run benchmarks and escape analysis in parallel
- Profile CPU and memory concurrently

Only run operations sequentially when one depends on the output of another.
Never use placeholders or guess parameters.
</use_parallel_tool_calls>

### Long-Horizon Optimization Sessions

For multi-session optimization work:

**State Tracking:**
- Use git commits for progress checkpoints
- Maintain `optimization_progress.json` with baseline benchmarks and targets
- Keep `progress_notes.txt` with current context and next steps
- Update after each logical milestone

**Session Continuity:**
```bash
# Starting fresh session
git log --oneline              # Review what's been done
cat progress_notes.txt         # Load context
go test -bench=. -benchmem     # Verify current state
# Continue from optimization plan
```

**Context Window Management:**
As you approach token budget limits:
1. Commit working changes with descriptive messages
2. Update progress_notes.txt with current status
3. Save state to structured files (JSON for data, text for context)
4. You can continue with a fresh context window by reviewing state files

### Zero-Allocation Validation Workflow

<zero_allocation_validation>
After optimizing any function in critical paths:
- pkg/shockwave/http11/parser.go
- pkg/shockwave/http11/response.go
- pkg/shockwave/server/handler.go
- Any function called per-request

Mandatory verification:
1. Run: `go test -bench=BenchmarkFunctionName -benchmem`
2. Verify: `allocs/op = 0` (not 1, not 2, exactly 0)
3. Run: `go build -gcflags="-m -m" ./pkg/... 2>&1 | grep FunctionName`
4. Confirm: No unexpected "escapes to heap" for function parameters
5. Run: `benchstat baseline.txt current.txt`
6. Verify: No regressions in other benchmarks (max 5% tolerance)

Zero allocations in hot paths is non-negotiable. Every allocation adds GC pressure.
</zero_allocation_validation>

### Principled Solutions Over Test Hacking

<principled_solutions>
When fixing failing tests, implement the correct general algorithm per the RFC
specification, not a solution that just makes the specific test pass.

Bad:
```go
if input == "test-case-1" {
    return hardcodedExpectedOutput
}
```

Good:
```go
// Implement RFC 7230 Section 3.2.5 header parsing
// Handles all valid inputs, not just test cases
```

If a test is incorrect, explain why rather than working around it. The test suite
ensures protocol compliance - never compromise correctness for convenience.
</principled_solutions>

### Minimize File Creation

<minimize_file_creation>
Shockwave's repository should contain only production code, tests, and documentation.

Avoid creating temporary files for iteration:
- Prefer reading/editing existing files over creating new ones
- Use bash commands with pipes for temporary data processing
- Store analysis notes in .claude/ directory, not project root
- Clean up any temporary files created during analysis

If you create temporary files:
1. Use descriptive names (.claude/temp_analysis.txt)
2. Clean up when task completes
3. Document why the file was necessary
</minimize_file_creation>

### Communication Style

After completing optimization work involving tool use, provide a brief summary:

**Example:**
```
Optimized ParseRequest in pkg/shockwave/http11/parser.go:145-178.
Eliminated string concatenation (saved 2 allocs) and switched to inline
header array (saved 1 alloc). Results: 0 allocs/op (target achieved),
16% throughput improvement. Running full benchmark suite to verify no
regressions.
```

Include:
- What was changed (file:line)
- What technique was applied
- Benchmark results (before → after)
- Next verification step

### Using the .claude/ Ecosystem

The `.claude/` directory provides specialized tools:

**Skills** (Auto-invoked by Claude):
- `go-performance-optimization` - Zero-allocation techniques and workflows
- `benchmark-analysis` - Statistical benchmark interpretation
- `http-protocol-testing` - RFC compliance validation
- `memory-profiling` - Allocation analysis and escape analysis

**Commands** (User-invoked with `/command`):
- `/bench` - Run comprehensive benchmark suite
- `/check-allocs` - Verify zero allocations in critical paths
- `/profile-cpu` - Generate and analyze CPU profiles
- `/profile-mem` - Generate and analyze memory profiles
- `/compare-nethttp` - Benchmark against standard library

**Agents** (For complex autonomous tasks):
- `performance-auditor` - Comprehensive performance audit across packages
- `protocol-validator` - RFC compliance checking with detailed reports
- `benchmark-runner` - Automated benchmark execution and regression detection

**Best Practices:**
- Let skills activate automatically based on task
- Use commands for quick, specific operations
- Delegate to agents for comprehensive analysis requiring isolated context
- See `.claude/PROMPTING_GUIDE.md` for detailed guidance

### Model Identification

<model_identification>
The assistant is Claude, created by Anthropic.
Current model: Claude Sonnet 4.5 (claude-sonnet-4-5-20250929)

When generating code comments or documentation, use "AI-assisted" or
"machine-generated" attribution. Never claim human authorship for
generated content.
</model_identification>

### Extended Thinking for Complex Decisions

<extended_thinking>
After running benchmarks, profiling, or escape analysis, reflect on the results
before proceeding:

- Are improvements real or statistical noise? (Check benchstat p-value)
- Did we create regressions elsewhere? (Review full benchmark suite)
- Does escape analysis confirm zero-alloc? (Check for unexpected escapes)
- Does this align with architectural principles? (Zero-alloc, RFC compliance)
- Are there edge cases not covered by benchmarks? (Malformed input, limits)

Use thinking to plan next steps based on data, not assumptions.
</extended_thinking>

---
