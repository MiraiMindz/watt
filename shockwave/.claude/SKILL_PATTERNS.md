# Skill Writing Patterns for Shockwave

This guide documents effective patterns for creating Claude Skills in the Shockwave project, optimized for Claude 4.5's instruction-following capabilities.

---

## Understanding Skills

**Skills are model-invoked capabilities that:**
- Inject comprehensive instruction sets when relevant
- Modify how Claude reasons about tasks
- Provide domain expertise and workflows
- Are discovered automatically based on description matching

**Skills vs Commands vs Agents:**
- **Skills**: Model-invoked expertise (Claude decides when to use)
- **Commands**: User-invoked actions (you explicitly call with `/command`)
- **Agents**: Autonomous workers (separate Claude instances)

**When to create a Skill:**
- Reusable expertise that applies to a class of problems
- Complex workflows that benefit from detailed guidance
- Domain knowledge that Claude should apply automatically
- Patterns that recur across multiple sessions

**When NOT to create a Skill:**
- One-off tasks (use Commands instead)
- Simple operations (just ask Claude directly)
- Tasks requiring isolation (use Agents instead)

---

## Skill Anatomy

### Directory Structure

```
.claude/skills/
└── go-performance-optimization/
    ├── SKILL.md              # Required: Main instructions
    ├── context/              # Optional: Reference materials
    │   └── go-perf-guide.md
    ├── workflows/            # Optional: Related commands
    │   └── optimize.md
    ├── templates/            # Optional: Code templates
    │   └── pool-pattern.go
    └── scripts/              # Optional: Helper scripts
        └── escape-check.sh
```

### SKILL.md Frontmatter

```markdown
---
name: go-performance-optimization
description: Optimize Go code for high-performance HTTP serving. Use when analyzing allocations, improving hot paths, or reducing GC pressure. Specializes in zero-allocation techniques, escape analysis, and memory pooling.
allowed-tools: Read, Grep, Glob, Bash, Edit
---
```

**Key fields:**
- `name`: Unique identifier (kebab-case)
- `description`: **When to invoke this skill** (critical for auto-discovery)
- `allowed-tools`: Tools Claude can use with this skill (optional restriction)

**Description best practices:**
- Start with **what the skill does**: "Optimize Go code..."
- Include **when to use it**: "Use when analyzing allocations..."
- Add **specific capabilities**: "Specializes in zero-allocation techniques..."
- Use keywords that match likely user requests: "allocations", "hot paths", "GC pressure"

### Skill Content Structure

```markdown
# [Skill Name]

## Core Principles
[Fundamental concepts and constraints]

## Workflow
[Step-by-step process]

### Step 1: [Action]
[Detailed instructions with examples]

### Step 2: [Action]
[Continue...]

## Techniques
[Specific patterns and anti-patterns]

### Technique 1: [Name]
❌ **Before:** [Anti-pattern]
✅ **After:** [Correct pattern]

## Output Format
[Template for results]

## References
[Links to context files, external docs]
```

---

## Pattern 1: Technical Expertise Skill

**Use case:** Domain-specific knowledge (performance, security, protocols)

**Example: Go Performance Optimization**

```markdown
---
name: go-performance-optimization
description: Optimize Go code for zero allocations and minimal GC pressure. Use when improving hot paths, analyzing escape analysis, or reducing memory overhead.
allowed-tools: Read, Grep, Glob, Bash, Edit
---

# Go Performance Optimization Skill

You are an expert in Go performance optimization with deep knowledge of memory
allocation patterns, escape analysis, and zero-allocation programming.

## Core Principles

### 1. Measure First, Optimize Second
Never optimize without benchmark evidence. The workflow is:
1. Establish baseline: `go test -bench=. -benchmem > baseline.txt`
2. Identify bottleneck: CPU/memory profiling
3. Optimize targeted code
4. Verify improvement: `benchstat baseline.txt current.txt`

Context: Shockwave targets zero allocations in request handling hot paths.
Every allocation adds GC pressure that degrades p99 latency.

### 2. Zero-Allocation Target
Critical paths must achieve 0 allocs/op:
- Request parsing (pkg/shockwave/http11/parser.go)
- Response writing (pkg/shockwave/http11/response.go)
- Keep-alive handling

Use inline arrays, sync.Pool, and pre-compiled constants.

### 3. Escape Analysis is Your Friend
```bash
go build -gcflags="-m -m" ./... 2>&1 | grep "escapes to heap"
```

Unexpected escapes indicate optimization opportunities.

## Workflow

### Step 1: Identify Hot Paths
Read the code to find performance-critical functions. In Shockwave:
- Request parsing: pkg/shockwave/http11/parser.go
- Connection handling: pkg/shockwave/server/handler.go
- Protocol implementations: pkg/shockwave/http*/

Look for functions called per-request or in tight loops.

### Step 2: Benchmark Current State
```bash
go test -bench=BenchmarkTargetFunction -benchmem -count=10 > baseline.txt
```

Note current metrics:
- ns/op (latency)
- B/op (bytes allocated)
- allocs/op (allocation count)

Target: 0 allocs/op for hot paths.

### Step 3: Profile for Bottlenecks
```bash
# CPU profile
go test -bench=BenchmarkTarget -cpuprofile=cpu.prof
go tool pprof -top cpu.prof

# Memory profile
go test -bench=BenchmarkTarget -memprofile=mem.prof
go tool pprof -alloc_space -top mem.prof
```

Focus on functions taking >5% CPU or allocating significant memory.

### Step 4: Run Escape Analysis
```bash
go build -gcflags="-m -m" ./pkg/shockwave/http11 2>&1 | tee escape.txt
grep "escapes to heap" escape.txt
```

Every escape in a hot path is a potential optimization.

### Step 5: Apply Zero-Allocation Techniques

[Continue with specific techniques...]

## Techniques

### Technique 1: Replace String Operations

Context: String concatenation and fmt.Sprintf allocate. In hot paths, use
pre-compiled byte slice constants or buffer pools.

❌ **Before: String concatenation**
```go
statusLine := "HTTP/1.1 " + statusCode + " " + statusText + "\r\n"
// Problem: 3+ allocations, temporary strings created
```

✅ **After: Pre-compiled constant**
```go
var statusLineOK = []byte("HTTP/1.1 200 OK\r\n")
// Zero allocations for common status codes
```

✅ **After: Buffer pool (dynamic cases)**
```go
buf := bufferPool.Get().(*bytes.Buffer)
buf.WriteString("HTTP/1.1 ")
buf.WriteString(statusCode)
buf.WriteString("\r\n")
// Reuses buffer, single allocation amortized across requests
```

**When to use:**
- Pre-compiled: Fixed strings (status lines, headers)
- Buffer pool: Dynamic content with known max size
- Never use: String concatenation in hot paths

[Continue with more techniques...]

## Verification Checklist

After optimization, verify:
- [ ] Benchmark shows improvement (benchstat p < 0.05)
- [ ] Zero allocations achieved for hot path (or justified deviation)
- [ ] No regressions in other benchmarks
- [ ] Escape analysis clean (no new escapes)
- [ ] Code remains readable and maintainable

## Output Format

When completing optimization, report:

```markdown
## Optimization Report

**Function**: ParseRequest (pkg/shockwave/http11/parser.go:145)

**Baseline**:
- 1250 ns/op
- 256 B/op
- 3 allocs/op

**Changes**:
1. Replaced string concatenation with pre-compiled constant (saved 2 allocs)
2. Used inline array for headers instead of map (saved 1 alloc)

**Results**:
- 1050 ns/op (16% faster)
- 0 B/op (100% reduction)
- 0 allocs/op (TARGET ACHIEVED ✓)

**Verification**:
✓ benchstat shows p < 0.01 (statistically significant)
✓ Zero allocations confirmed
✓ No regressions in BenchmarkAll
✓ Escape analysis clean
```

## References
- Detailed guide: context/go-perf-guide.md
- Go escape analysis: https://go.dev/doc/faq#stack_or_heap
- Profiling: https://go.dev/blog/pprof
```

**Why this works:**
- **Clear principles** with context explaining why they matter
- **Explicit workflow** Claude can follow step-by-step
- **Concrete examples** showing before/after patterns
- **Verification steps** ensure correctness
- **Output format** for consistency

---

## Pattern 2: Process/Workflow Skill

**Use case:** Multi-step procedures (benchmarking, testing, deployment)

**Example: Benchmark Analysis**

```markdown
---
name: benchmark-analysis
description: Analyze Go benchmark results, detect regressions, compare performance. Use when interpreting benchmark output or validating optimizations.
allowed-tools: Read, Bash, Grep, Write
---

# Benchmark Analysis Skill

You specialize in analyzing Go benchmark results to detect performance changes,
validate optimizations, and identify regressions.

## Core Principles

### Statistical Significance Matters
Use `benchstat` for statistical comparison. Don't trust single runs.
A difference is real only if benchstat shows p < 0.05.

Context: Performance varies due to GC, CPU scheduling, thermal throttling.
Statistical analysis filters noise from signal.

### Regression Tolerance
Per Shockwave CLAUDE.md:
- Maximum 5% regression allowed in any benchmark
- Zero tolerance for allocation increases in zero-alloc targets
- Any regression requires investigation and justification

## Workflow

### Step 1: Run Benchmarks Properly
```bash
# Always use -count=10 or higher for stability
go test -bench=. -benchmem -count=10 > current.txt

# For comparison, you need baseline:
go test -bench=. -benchmem -count=10 > baseline.txt
```

**Never trust single runs**. Variance can be 10-20% without real changes.

### Step 2: Statistical Comparison
```bash
benchstat baseline.txt current.txt > comparison.txt
```

Read comparison.txt. Look for:
- `~` : No significant change (good for stability)
- `+X%`: Improvement (positive for latency reduction)
- `-X%`: Regression (investigate if >5%)
- `p=0.XXX`: Confidence level (need p < 0.05)

### Step 3: Interpret Results

Example benchstat output:
```
name              old time/op    new time/op    delta
ParseRequest-8      1.25µs ± 2%    1.05µs ± 1%  -16.00%  (p=0.000 n=10+10)

name              old alloc/op   new alloc/op   delta
ParseRequest-8       256B ± 0%        0B        -100.00%  (p=0.000 n=10+10)

name              old allocs/op  new allocs/op  delta
ParseRequest-8       3.00 ± 0%      0.00        -100.00%  (p=0.000 n=10+10)
```

**Analysis**:
- Time: 16% faster (statistically significant, p=0.000)
- Allocations: 100% reduction (ZERO-ALLOC TARGET ACHIEVED ✓)
- Confidence: Very high (p=0.000, n=10 runs)
- Variance: Low (±2%, ±1% - consistent results)

### Step 4: Identify Regressions
```bash
grep -E '\-[0-9]+\.[0-9]+%' comparison.txt | grep -v alloc
```

Any line showing performance decrease needs investigation:
1. Which function regressed?
2. Is it in a hot path? (Check CLAUDE.md hot paths list)
3. Is it >5%? (Violates regression tolerance)
4. What changed? (git diff)

### Step 5: Validate Zero-Alloc Targets
```bash
grep 'allocs/op' current.txt | grep -E 'ParseRequest|WriteResponse|ServeHTTP'
```

These functions MUST show `0 allocs/op`. Any non-zero is a critical failure.

### Step 6: Generate Report
```markdown
# Benchmark Analysis Report

## Summary
- Benchmarks analyzed: X
- Significant improvements: Y
- Regressions: Z
- Status: PASS | FAIL

## Critical Findings

### Zero-Alloc Validation
✓ ParseRequest: 0 allocs/op
✓ WriteResponse: 0 allocs/op
✗ ServeHTTP: 1 allocs/op (REGRESSION - was 0)

### Performance Changes
**Improvements**:
- ParseRequest: 16% faster (p=0.000)
- WriteHeaders: 8% faster (p=0.003)

**Regressions**:
- HandleConnection: 7% slower (p=0.012) - VIOLATES 5% TOLERANCE

## Investigation Required
1. ServeHTTP allocation: Run escape analysis to find source
2. HandleConnection regression: Profile to identify bottleneck

## Full Results
[Attach benchstat comparison]
```

## Common Patterns

### Pattern: Noise vs Signal
If benchstat shows `~` (no change), ignore variance. Don't optimize noise.

### Pattern: Allocation Reduction Priority
Reducing allocs/op is often more valuable than raw speed improvements.
- 1 alloc/op → 0 allocs/op: High priority (eliminates GC pressure)
- 50ns/op → 40ns/op: Lower priority (10ns rarely matters)

### Pattern: Throughput vs Latency
For Shockwave:
- Throughput (ops/sec): Important for benchmarking
- Latency (ns/op): Critical for user experience
- Allocations (allocs/op): Critical for GC impact

Optimize in order: allocs, latency, throughput.

## Output Format

Always include:
1. **Summary**: Pass/fail, number of changes
2. **Zero-alloc validation**: Explicit check of critical functions
3. **Regressions**: Any performance decreases with severity
4. **Statistical confidence**: p-values and variance
5. **Next steps**: Investigations needed

## References
- benchstat: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
- Shockwave targets: ../CLAUDE.md (Allocation Targets section)
```

**Why this works:**
- **Explicit process** for consistent analysis
- **Statistical rigor** prevents false positives
- **Context** explaining Shockwave-specific priorities
- **Clear pass/fail criteria** for zero-alloc targets

---

## Pattern 3: Template/Generator Skill

**Use case:** Code generation following patterns

**Example: Pool Pattern Generator**

```markdown
---
name: pool-pattern-generator
description: Generate correct sync.Pool implementations for object reuse. Use when creating new pooled types or fixing pool leaks.
allowed-tools: Read, Edit, Write
---

# Pool Pattern Generator Skill

You specialize in creating correct, efficient sync.Pool implementations for
Shockwave's object pooling strategy.

## Core Principles

### sync.Pool Correctness
Common mistakes that cause leaks or panics:
1. Forgetting to Reset() before Put()
2. Not handling nil from Get()
3. Incorrect type assertions
4. Retaining pointers to pooled objects after Put()

Context: Pools reduce allocations by reusing objects. Incorrect pooling can
cause memory leaks (growing pool) or corruption (reusing dirty objects).

## Pool Pattern Template

```go
var [name]Pool = sync.Pool{
    New: func() interface{} {
        return &[Type]{
            // Initialize with appropriate capacity
            field: make([]byte, 0, [capacity]),
        }
    },
}

func Get[Name]() *[Type] {
    obj := [name]Pool.Get().(*[Type])
    // Reset fields to zero values if New doesn't guarantee clean state
    return obj
}

func Put[Name](obj *[Type]) {
    // Critical: Reset before returning to pool
    obj.Reset()
    [name]Pool.Put(obj)
}

// On the type itself
func (obj *[Type]) Reset() {
    // Reset all fields to allow reuse
    obj.field = obj.field[:0]  // Keep capacity, reset length
    obj.other = nil
    // Don't reallocate, just reset
}
```

## Workflow

### Step 1: Identify Poolable Type
Good candidates for pooling:
- Large structs (>256 bytes)
- Frequently allocated (per-request)
- Contains slice/map that can be reused
- Has predictable size/capacity

Example: Request parser buffers, response writers, header arrays.

### Step 2: Implement Reset Method
```go
func (r *Request) Reset() {
    // Reset slices (keep capacity)
    r.headers = r.headers[:0]

    // Reset maps (clear but keep allocation)
    for k := range r.params {
        delete(r.params, k)
    }

    // Nil out pointers
    r.Body = nil
    r.conn = nil

    // Reset primitive fields
    r.Method = 0
    r.parsed = false
}
```

**Critical**: Reset must return object to pristine state without allocating.

### Step 3: Create Pool
```go
var requestPool = sync.Pool{
    New: func() interface{} {
        return &Request{
            headers: make([]Header, 0, 32),  // Pre-allocate common case
            params:  make(map[string]string, 8),
        }
    },
}
```

**Capacity guidelines** (from Shockwave patterns):
- Headers: 32 (covers 99% of requests)
- Small buffers: 4KB
- Large buffers: 64KB
- Maps: 8-16 initial size

### Step 4: Implement Get/Put
```go
func GetRequest() *Request {
    return requestPool.Get().(*Request)
}

func PutRequest(r *Request) {
    r.Reset()  // CRITICAL: Must reset before Put
    requestPool.Put(r)
}
```

### Step 5: Use with Defer Pattern
```go
func handleRequest() {
    req := GetRequest()
    defer PutRequest(req)  // Ensure return to pool

    // Use req
    // Even if panic occurs, defer ensures cleanup
}
```

**Never**:
```go
req := GetRequest()
PutRequest(req)  // ❌ Immediate Put - req is now invalid!
// Using req here is a bug
```

## Common Anti-Patterns

### Anti-Pattern 1: Forgetting Reset

❌ **Wrong**:
```go
func PutBuffer(b *Buffer) {
    bufferPool.Put(b)  // Dirty buffer returned to pool!
}
```

✅ **Correct**:
```go
func PutBuffer(b *Buffer) {
    b.Reset()  // Clean before reuse
    bufferPool.Put(b)
}
```

**Impact**: Dirty objects cause data corruption or leaks.

### Anti-Pattern 2: Retaining References

❌ **Wrong**:
```go
var saved *Request  // Global
func handler() {
    req := GetRequest()
    defer PutRequest(req)

    saved = req  // ❌ Reference outlives pool lifetime!
}
// Later: saved.Method - USE AFTER FREE BUG
```

✅ **Correct**:
```go
func handler() {
    req := GetRequest()
    defer PutRequest(req)

    // Copy data out, don't keep references
    method := req.Method
    processMethod(method)
}
```

### Anti-Pattern 3: Allocating in Reset

❌ **Wrong**:
```go
func (r *Request) Reset() {
    r.headers = make([]Header, 0, 32)  // ❌ Allocates new slice every time!
}
```

✅ **Correct**:
```go
func (r *Request) Reset() {
    r.headers = r.headers[:0]  // ✓ Reuses existing capacity
}
```

## Verification

After implementing pool:

```bash
# Check zero allocations with pooling
go test -bench=BenchmarkWithPool -benchmem
# Should show 0 allocs/op or 1 alloc/op (amortized)

# Verify Reset doesn't allocate
go test -bench=BenchmarkReset -benchmem
# Should show 0 allocs/op

# Escape analysis - pool object should not escape
go build -gcflags="-m -m" 2>&1 | grep PoolGet
# Should NOT show "escapes to heap"
```

## Output Format

When generating pool code, provide:
1. Pool declaration
2. Type Reset() method
3. GetType() and PutType() functions
4. Example usage with defer
5. Benchmark to verify zero allocs
6. Comment explaining capacity choices

## References
- sync.Pool docs: https://pkg.go.dev/sync#Pool
- Shockwave pooling patterns: ../CLAUDE.md (Pre-Compiled Constants section)
```

**Why this works:**
- **Complete pattern** covering all aspects
- **Anti-patterns** prevent common mistakes
- **Verification steps** ensure correctness
- **Context** explains capacity choices

---

## Skill Writing Best Practices

### 1. Make Descriptions Discoverable

**Poor description:**
```yaml
description: Helps with code
```

**Good description:**
```yaml
description: Optimize Go code for zero allocations and minimal GC pressure. Use when improving hot paths, analyzing escape analysis, reducing memory overhead, or implementing sync.Pool patterns. Specializes in performance-critical HTTP server code.
```

**Why**: Description must match likely user queries. Keywords: "allocations", "GC pressure", "hot paths", "sync.Pool", "performance".

### 2. Provide Context for Constraints

**Poor:**
```markdown
NEVER use string concatenation
```

**Good:**
```markdown
NEVER use string concatenation in hot paths because Shockwave targets zero
allocations for request handling. Every allocation adds GC pressure that
degrades p99 latency under load. Use pre-compiled []byte constants or
sync.Pool buffers instead.
```

**Why**: Claude 4.5 performs better with explicit reasoning. Explaining "why" helps it generalize the rule correctly.

### 3. Use Concrete Examples

**Poor:**
```markdown
Optimize memory usage
```

**Good:**
```markdown
### Optimize Memory Usage

❌ **Before: Map for headers**
```go
type Request struct {
    Headers map[string]string  // Allocates, variable size
}
```

✅ **After: Inline array**
```go
type Request struct {
    headers [32]Header  // Zero-alloc if ≤32 headers (99% of requests)
    headerCount int
}
```

**Rationale**: Inline arrays avoid heap allocation. Benchmarking shows 99%
of real-world requests have ≤32 headers. Edge cases with more headers can
fall back to map with explicit allocation.
```

**Why**: Concrete before/after with reasoning teaches patterns, not just rules.

### 4. Include Verification Steps

Every technique should have verification:
```markdown
## Verification
After applying this optimization:
1. Run: `go test -bench=BenchmarkTarget -benchmem`
2. Verify: `allocs/op` decreased (or is 0)
3. Run: `benchstat baseline.txt current.txt`
4. Confirm: Improvement is statistically significant (p < 0.05)
5. Check: No regressions in other benchmarks
```

**Why**: Ensures Claude validates changes, prevents regressions.

### 5. Reference External Context

```markdown
## References
- Detailed patterns: context/go-perf-guide.md
- Shockwave hot paths: ../CLAUDE.md (Performance Requirements)
- Go escape analysis: https://go.dev/doc/faq#stack_or_heap
```

**Why**: Skills should be concise. Deep dives go in context/.

### 6. Use XML Tags for Complex Logic

```markdown
<zero_allocation_workflow>
For functions in hot paths (per ../CLAUDE.md):
1. Establish baseline: go test -bench=BenchmarkFunc -benchmem > baseline.txt
2. Check current allocs/op: Must be 0 for critical path functions
3. If not zero:
   a. Run escape analysis: go build -gcflags="-m -m"
   b. Identify what escapes to heap
   c. Apply appropriate technique (inline arrays, pooling, pre-compiled constants)
   d. Re-benchmark
4. Verify with benchstat: benchstat baseline.txt current.txt
5. Ensure no regressions in full suite: go test -bench=. -benchmem
</zero_allocation_workflow>
```

**Why**: Complex workflows benefit from structured tags. Claude 4.5 respects XML structure.

---

## Skill Activation Patterns

### Pattern: Keyword Matching

User says → Skill activates:
- "optimize allocations" → go-performance-optimization
- "check protocol compliance" → http-protocol-testing
- "analyze memory usage" → memory-profiling
- "compare benchmarks" → benchmark-analysis

### Pattern: Tool/Domain Matching

User provides context → Skill activates:
- Reading parser.go + asks about perf → go-performance-optimization
- Mentions RFC 7540 → http-protocol-testing
- Shows benchmark output → benchmark-analysis

### Pattern: Explicit Invocation

User directly requests (though not required):
- "Use the performance optimization skill to analyze this"
- "Apply Go performance patterns to reduce allocations"

---

## Integration with Commands and Agents

### Skills → Commands

Skills can reference related commands:

```markdown
## Quick Actions
For common operations, use these commands:
- `/bench` - Run full benchmark suite
- `/check-allocs` - Verify zero allocations
- `/profile-mem` - Generate memory profile

This skill provides the analysis methodology; commands automate execution.
```

### Skills → Agents

Skills can recommend agent delegation:

```markdown
## When to Delegate
For comprehensive codebase audits, use the performance-auditor agent:
```
Use performance-auditor agent to analyze all of pkg/shockwave/ for
allocation hotspots. This skill focuses on targeted optimization of
specific functions.
```
```

### Commands → Skills

Commands can invoke skills implicitly:

```markdown
# /optimize command

Optimize the current file for performance.

*This command works best when the go-performance-optimization skill is
active. The skill will guide the optimization workflow automatically.*
```

---

## Measuring Skill Effectiveness

Good skills:
- Activate when relevant (description matches use case)
- Provide clear, actionable guidance
- Include concrete examples and anti-patterns
- Have verification steps built in
- Reference external context appropriately
- Lead to correct results on first try

Poor skills:
- Too vague to activate automatically ("helps with code")
- Give abstract advice without examples
- Missing verification steps (Claude doesn't check work)
- Duplicate information from other skills
- Too long (>500 lines becomes hard to apply)

**Iterate on skills** based on actual usage. Track which skills activate and whether they help.

---

## Skill Lifecycle

### 1. Create
Write SKILL.md with clear description, workflow, examples.

### 2. Test
Try queries that should activate it:
```
"How do I reduce allocations in this parser?"
→ Should activate go-performance-optimization
```

### 3. Refine
If skill doesn't activate or gives poor advice:
- Improve description keywords
- Add more concrete examples
- Clarify workflow steps
- Add missing verification

### 4. Maintain
As patterns evolve:
- Update techniques with new best practices
- Add new anti-patterns discovered in code reviews
- Incorporate lessons from optimization sessions

### 5. Split or Merge
- Too broad? Split into multiple focused skills
- Too much overlap? Merge into single comprehensive skill

---

## Anti-Patterns to Avoid

### ❌ Skill Too Vague

```markdown
---
description: Helps with performance
---

Make it faster.
```

**Problem**: Won't activate reliably, no actionable guidance.

### ❌ Skill Too Specific

```markdown
---
description: Optimizes the ParseRequest function in parser.go line 145
---
```

**Problem**: Use case too narrow. Make it a command instead.

### ❌ Skill Duplicates CLAUDE.md

**Problem**: Redundancy. CLAUDE.md is always loaded. Skills should add specialized workflows, not repeat constitution.

### ❌ Skill Too Long (>1000 lines)

**Problem**: Bloated skills dilute focus. Split into multiple skills or move details to context/.

### ❌ No Verification

```markdown
## Technique: Use sync.Pool
[Example code]
```

**Problem**: No way to verify it worked. Always include verification steps.

---

## Quick Reference

### Skill Creation Checklist

```markdown
[ ] Clear, keyword-rich description
[ ] Frontmatter with name, description, allowed-tools
[ ] Core principles with context/rationale
[ ] Step-by-step workflow
[ ] Concrete examples (before/after)
[ ] Anti-patterns to avoid
[ ] Verification steps
[ ] Output format template
[ ] References to context files
[ ] Integration with commands/agents
```

### Skill Improvement Checklist

```markdown
[ ] Description matches actual user queries
[ ] Workflow is actionable (not vague)
[ ] Examples are specific to Shockwave patterns
[ ] Verification steps ensure correctness
[ ] Length is reasonable (<500 lines for core, context/ for details)
[ ] No duplication with CLAUDE.md or other skills
[ ] Techniques have rationale explaining why
```

---

## Conclusion

Effective Skills in Shockwave:
- **Discoverable**: Description matches user queries
- **Actionable**: Clear workflows with verification
- **Contextual**: Explain why, not just what
- **Concrete**: Before/after examples with code
- **Integrated**: Reference commands, agents, context files
- **Maintainable**: Focused scope, reasonable length

**Skills encode expertise. Make them worthy of the name.** 🎯
