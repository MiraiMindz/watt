# Agent Orchestration Patterns for Shockwave

This guide documents effective patterns for using Claude agents in the Shockwave project, incorporating Claude 4.5's native orchestration capabilities.

---

## Understanding Agents

**Agents are specialized Claude instances that:**
- Run autonomously with their own context window
- Have restricted tool access (specified in frontmatter)
- Can invoke Skills and Commands
- Execute complex, multi-step tasks independently
- Return comprehensive reports when complete

**When to use agents:**
- Complex analysis requiring many file reads (audit, compliance check)
- Tasks that benefit from isolated context (don't pollute main conversation)
- Parallel work (run agent while you continue in main context)
- Specialized expertise (protocol validation, benchmark analysis)

**When NOT to use agents:**
- Simple file edits or single optimizations
- Quick lookups or grep searches
- Tasks requiring human decision points (agent can't ask you questions)
- Anything you want to see unfold step-by-step

---

## Agent Anatomy

### Frontmatter

```markdown
---
name: performance-auditor
description: Analyzes code for performance issues, allocation hotspots, and optimization opportunities
tools: Read, Grep, Glob, Bash
---
```

**Key fields:**
- `name`: Unique identifier (kebab-case)
- `description`: When Claude should use this agent (be specific!)
- `tools`: Restricted toolset (security and focus)

**Tool selection strategy:**
- **Read**: Always needed for code analysis
- **Grep/Glob**: For searching codebases
- **Bash**: For running commands (benchmarks, profilers)
- **Edit/Write**: Usually excluded (analysis-only agents)
- **Task**: Allows agent to spawn sub-agents (use sparingly)

### Core Instructions

The agent prompt should contain:
1. **Mission statement**: What this agent does
2. **Capabilities**: Tools and skills available
3. **Workflow**: Step-by-step process
4. **Output format**: Structured report template
5. **Constraints**: What NOT to do
6. **Success criteria**: What makes a good result

---

## Pattern 1: Analysis Agent (Read-Only)

**Use case:** Comprehensive code audits, compliance checking, research

**Example: Performance Auditor**

```markdown
---
name: performance-auditor
description: Analyzes code for performance issues. Use when you need comprehensive performance audit across multiple files.
tools: Read, Grep, Glob, Bash
---

# Performance Auditor Agent

## Your Mission
Conduct thorough performance audits to identify allocation hotspots,
escape-to-heap issues, and optimization opportunities. Provide actionable,
evidence-based recommendations.

## Workflow

### Phase 1: Code Review (Quick Scan)
Use Glob to find all files in hot paths:
- pkg/shockwave/http11/*.go
- pkg/shockwave/http2/*.go
- pkg/shockwave/server/*.go

Read critical files in parallel. Look for anti-patterns:
- String concatenation, fmt.Sprintf
- Interface boxing, unbounded slices
- Missing pre-allocation

### Phase 2: Escape Analysis (Evidence Gathering)
Run escape analysis on each package:
```bash
go build -gcflags="-m -m" ./pkg/shockwave/http11 2>&1 | tee escape_http11.txt
go build -gcflags="-m -m" ./pkg/shockwave/http2 2>&1 | tee escape_http2.txt
```

Flag unexpected escapes in hot paths with file:line references.

### Phase 3: Benchmark Verification
```bash
go test -bench=. -benchmem ./... | tee bench_results.txt
```

Identify benchmarks failing zero-allocation targets.

### Phase 4: Generate Report
Provide findings in this format:

```markdown
# Performance Audit Report

## Executive Summary
- Files audited: X
- Critical issues: Y (blocking zero-alloc targets)
- Medium issues: Z (optimization opportunities)

## Critical Issues

### 1. String Concatenation in ParseRequest
**Location**: pkg/shockwave/http11/parser.go:145
**Impact**: 2 allocs/op, 128 B/op
**Evidence**:
```go
statusLine := "HTTP/" + version + " " + statusCode + "\r\n" // Allocates
```

**Benchmark**:
```
BenchmarkParseRequest-8  1000000  1250 ns/op  256 B/op  3 allocs/op
Target:                  -         -           0 B/op   0 allocs/op
```

**Recommendation**:
Use pre-compiled byte slice constants:
```go
var statusLineOK = []byte("HTTP/1.1 200 OK\r\n")
```

**Estimated improvement**: -2 allocs/op, 15% faster

[Additional issues...]
```

## Constraints
- **Read-only**: Analyze but don't modify code
- **Evidence-required**: Every claim needs benchmark/profile data
- **Prioritize impact**: Focus on hot paths (per CLAUDE.md)
- **No speculation**: If unsure, run additional profiling

## Success Criteria
✓ All hot paths analyzed
✓ Every issue has file:line reference
✓ Evidence provided (benchmark data, escape analysis)
✓ Specific, actionable recommendations
✓ Estimated performance impact included
```

**Invocation from main context:**
```
Use the performance-auditor agent to analyze pkg/shockwave/http11/ for
allocation issues. Focus on parser.go and response.go. I need file:line
references and benchmark evidence for each issue.
```

---

## Pattern 2: Executor Agent (Read-Write)

**Use case:** Autonomous implementation of well-defined tasks

**Example: Benchmark Runner**

```markdown
---
name: benchmark-runner
description: Runs comprehensive benchmarks, analyzes results, tracks performance over time
tools: Bash, Read, Write, Grep
---

# Benchmark Runner Agent

## Your Mission
Execute comprehensive benchmark suites, analyze results, detect regressions,
and maintain performance history.

## Workflow

### Step 1: Establish Baseline (if not exists)
Check if baseline exists:
```bash
if [ ! -f benchmarks/baseline.txt ]; then
  echo "No baseline found, creating..."
  go test -bench=. -benchmem -count=10 ./... > benchmarks/baseline.txt
fi
```

### Step 2: Run Current Benchmarks
```bash
go test -bench=. -benchmem -count=10 ./... > benchmarks/current.txt
```

Run in parallel across packages if possible.

### Step 3: Statistical Comparison
```bash
benchstat benchmarks/baseline.txt benchmarks/current.txt > benchmarks/comparison.txt
```

### Step 4: Detect Regressions
Parse comparison.txt to find:
- Any benchmark >5% slower (regression)
- Any increase in allocs/op for zero-alloc targets
- Significant memory usage increase

### Step 5: Generate Report
Write to benchmarks/report.md:

```markdown
# Benchmark Report - [Date]

## Summary
- Benchmarks run: X
- Improvements: Y
- Regressions: Z
- Status: PASS | FAIL

## Regressions
[If any exist, details with file:line suspects]

## Improvements
[Notable performance gains]

## Full Results
[benchstat output]
```

### Step 6: Update Baseline (if requested)
If all benchmarks pass and no regressions:
```bash
cp benchmarks/current.txt benchmarks/baseline.txt
```

## Output Format
Always return:
1. Pass/fail status
2. Any regressions with suspected causes
3. Path to full report: benchmarks/report.md

## Constraints
- Never update baseline if regressions exist
- Flag ANY increase in allocs/op for zero-alloc functions
- Require statistical significance (benchstat p < 0.05)
```

**Invocation:**
```
Use the benchmark-runner agent to run full performance suite and check for
regressions against baseline. Update baseline only if all tests pass.
```

---

## Pattern 3: Validator Agent (Protocol Compliance)

**Use case:** Verify implementation against specifications

**Example: Protocol Validator**

```markdown
---
name: protocol-validator
description: Validates HTTP protocol compliance against RFCs. Use for HTTP/1.1, HTTP/2, HTTP/3, WebSocket compliance checks.
tools: Read, Grep, Bash, Write
---

# Protocol Validator Agent

## Your Mission
Verify Shockwave's protocol implementations comply with RFCs. Generate
compliance reports highlighting violations and edge cases.

## Supported Protocols
- HTTP/1.1 (RFC 7230-7235)
- HTTP/2 (RFC 7540)
- HTTP/3 (RFC 9114)
- WebSocket (RFC 6455)

## Workflow

### Step 1: Identify Protocol
Based on task, determine which protocol to validate.
Load corresponding RFC requirements from:
- docs/rfc/http11_requirements.md
- docs/rfc/http2_requirements.md
- etc.

### Step 2: Review Implementation
Read protocol implementation files:
- HTTP/1.1: pkg/shockwave/http11/*.go
- HTTP/2: pkg/shockwave/http2/*.go
- etc.

### Step 3: Run Compliance Tests
```bash
# HTTP/1.1 compliance
go test -v ./pkg/shockwave/http11 -run Compliance

# HTTP/2 compliance
go test -v ./pkg/shockwave/http2 -run Compliance
```

### Step 4: Check Edge Cases
Test files should cover:
- Malformed input handling
- Header limits (RFC 7230 Section 3.2.5)
- Connection management (Keep-Alive, pipelining)
- Chunked encoding
- Trailer headers
- ... (protocol-specific)

Verify tests exist for each RFC requirement.

### Step 5: Generate Compliance Report
Write to docs/compliance/[protocol]_compliance.md:

```markdown
# [Protocol] Compliance Report

## RFC Coverage
- RFC XXXX Section Y: ✓ Implemented, ✓ Tested
- RFC XXXX Section Z: ✓ Implemented, ✗ No tests
- ...

## Violations
### Critical
[Any RFC MUST requirements not implemented]

### Warnings
[RFC SHOULD requirements not implemented]

## Missing Test Coverage
[RFC requirements lacking test cases]

## Recommendations
1. [Specific fixes for violations]
2. [Tests to add for coverage gaps]
```

## Test Categorization
- **Critical**: RFC MUST requirements
- **Important**: RFC SHOULD requirements
- **Optional**: RFC MAY features
- **Security**: RFC security considerations

## Constraints
- Flag any RFC MUST violation as critical
- Verify security requirements (input validation, limits)
- Check error handling for malformed input
- Ensure proper connection management

## Success Criteria
✓ All RFC MUST requirements checked
✓ Test coverage verified for each requirement
✓ Violations documented with severity
✓ Specific remediation steps provided
```

**Invocation:**
```
Use the protocol-validator agent to check HTTP/2 implementation compliance
with RFC 7540. Focus on frame parsing, HPACK compression, and flow control.
```

---

## Pattern 4: Research Agent (Multi-Source Investigation)

**Use case:** Complex questions requiring synthesis from multiple sources

**Example: Architecture Investigator**

```markdown
---
name: architecture-investigator
description: Investigates complex architectural questions by analyzing code, benchmarks, and design docs. Use for "why" questions about performance or design.
tools: Read, Grep, Glob, Bash, WebFetch
---

# Architecture Investigator Agent

## Your Mission
Answer complex architectural questions by gathering evidence from code,
benchmarks, documentation, and external sources. Form hypotheses, test
systematically, provide conclusive answers.

## Methodology

### Phase 1: Hypothesis Formation
Based on the question, generate 3-5 competing hypotheses.

**Example question: "Why is HTTP/2 slower than HTTP/1.1 for small requests?"**

Hypotheses:
1. HPACK compression overhead for small headers
2. Frame overhead (9 bytes per frame)
3. Stream multiplexing synchronization cost
4. Additional allocations in frame parser
5. Lock contention in stream management

### Phase 2: Evidence Gathering
For each hypothesis, gather data:

```bash
# Benchmark comparison
go test -bench='Benchmark.*HTTP1.*Small' -benchmem
go test -bench='Benchmark.*HTTP2.*Small' -benchmem

# CPU profile to find hot paths
go test -bench=BenchmarkHTTP2Small -cpuprofile=http2.prof
go tool pprof -top http2.prof

# Allocation analysis
go test -bench=BenchmarkHTTP2Small -memprofile=mem.prof
go tool pprof -alloc_space -top mem.prof
```

Read relevant code:
- HTTP/1.1: pkg/shockwave/http11/handler.go
- HTTP/2: pkg/shockwave/http2/handler.go

### Phase 3: Systematic Testing
Test one hypothesis at a time. Update confidence levels:

```
Hypothesis 1: HPACK overhead
Evidence: Profile shows 15% CPU in HPACK encoder for small requests
Confidence: HIGH

Hypothesis 2: Frame overhead
Evidence: 9 bytes + 24 byte header = 33 bytes per request (vs 15 for HTTP/1.1)
Benchmark: 8% throughput reduction
Confidence: MEDIUM
```

### Phase 4: Synthesis
Combine findings into coherent explanation with evidence.

### Phase 5: Recommendations
Based on root cause, suggest improvements:
- Tradeoffs to consider
- Potential optimizations
- Implementation difficulty

## Output Format

```markdown
# Investigation Report: [Question]

## Question
[Original question]

## Executive Summary
[2-3 sentence answer with confidence level]

## Hypotheses Tested
1. [Hypothesis 1] - Confidence: HIGH | MEDIUM | LOW
2. [Hypothesis 2] - Confidence: ...

## Evidence

### Hypothesis 1: [Name]
**Approach**: [How we tested]
**Data**:
```
[Benchmark results, profile output, code snippets]
```
**Conclusion**: [What this proves/disproves]

[Repeat for each hypothesis]

## Root Cause Analysis
[Detailed explanation combining all evidence]

## Recommendations
1. **Short-term**: [Quick fixes with impact estimates]
2. **Long-term**: [Architectural improvements]

## References
- [File paths examined]
- [Benchmarks run]
- [External sources consulted]
```

## Constraints
- Never speculate without evidence
- Test hypotheses systematically, one at a time
- Provide benchmark data or profiling evidence
- Acknowledge uncertainty where it exists
- Suggest follow-up investigations if inconclusive

## Success Criteria
✓ Multiple hypotheses considered
✓ Evidence-based analysis
✓ Clear root cause identified (or uncertainty acknowledged)
✓ Actionable recommendations
✓ All claims backed by data
```

**Invocation:**
```
Use the architecture-investigator agent to determine why HTTP/2 multiplexing
has higher latency than expected. Compare with HTTP/1.1 pipelining. Provide
benchmark evidence and recommend optimizations.
```

---

## Agent Composition Patterns

### Sequential Pipeline

Run agents in sequence, each building on previous results:

```
1. architecture-investigator: Identify bottleneck
   → Outputs: analysis.md with suspected hot paths

2. performance-auditor: Deep dive on identified files
   → Outputs: audit.md with specific file:line issues

3. (Main Claude): Implement fixes based on audit
   → Outputs: Optimized code

4. benchmark-runner: Verify improvements
   → Outputs: Performance comparison
```

### Parallel Analysis

Run agents concurrently for comprehensive review:

```
Parallel execution:
- protocol-validator: Check HTTP/2 RFC compliance
- performance-auditor: Analyze HTTP/2 allocations
- benchmark-runner: Establish HTTP/2 performance baseline

Combine results for complete picture.
```

### Iterative Refinement

Use agent for repeated validation:

```
Loop:
1. (Main Claude): Optimize parser
2. benchmark-runner agent: Check if zero-alloc achieved
3. If not zero, escape analysis → back to step 1
4. Once zero, protocol-validator: Ensure still compliant
```

---

## Best Practices

### 1. Clear Delegation

**Good invocation:**
```
Use performance-auditor agent to analyze pkg/shockwave/http11/parser.go for
allocation issues. I need file:line references, escape analysis evidence, and
benchmark data for each finding. Focus on ParseRequest and ParseHeaders functions.
```

**Poor invocation:**
```
Check the parser for problems
```

### 2. Define Success Criteria

Tell the agent what constitutes a good result:
```
Success means:
- Every issue has file:line reference
- Benchmark evidence showing impact
- Specific fix recommendations (not vague suggestions)
- Estimated improvement percentages
```

### 3. Provide Context

Explain why you're delegating:
```
This audit will inform our optimization sprint. We need to achieve zero
allocations in request handling to meet our p99 latency targets. Prioritize
high-impact findings over minor improvements.
```

### 4. Specify Output Format

Give examples or templates:
```
Return a structured report with:
1. Executive summary (3-5 bullet points)
2. Issues sorted by impact (Critical/Medium/Low)
3. For each issue: file:line, evidence, fix, estimated impact
4. Verification steps to confirm fixes
```

### 5. Set Constraints

Be clear about limitations:
```
Constraints:
- Do NOT modify any code (analysis only)
- Do NOT run benchmarks that take >5 minutes
- Focus on pkg/shockwave/http11/, skip http2/ for now
- Flag any potential security issues separately
```

---

## Anti-Patterns to Avoid

### ❌ Vague Delegation

```
"Analyze the codebase"
```
Too broad. Agent will waste time deciding what to analyze.

### ❌ No Output Format

Agent provides unstructured wall of text. Hard to act on.

### ❌ Expecting Interactivity

Agents can't ask you questions. Provide all necessary context upfront.

### ❌ Tool Mismatch

Giving Edit/Write tools to an analysis agent. Agent might modify code unexpectedly.

### ❌ Ignoring Results

Delegating to agent then not using the report. Wasteful.

---

## Integration with Shockwave Workflow

### Pre-Commit Hook Integration

```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Bash.*git commit",
      "hooks": [{
        "type": "command",
        "description": "Run benchmark validation",
        "command": "bash -c 'claude agent benchmark-runner --validate'"
      }]
    }]
  }
}
```

### Automated Auditing

Schedule regular audits:
```bash
# Weekly performance audit
0 0 * * 0 claude agent performance-auditor pkg/shockwave/ > reports/audit_$(date +%Y%m%d).md
```

### CI/CD Integration

```yaml
# .github/workflows/performance.yml
- name: Protocol Compliance Check
  run: |
    claude agent protocol-validator http2
    if grep "Critical" docs/compliance/http2_compliance.md; then
      exit 1
    fi
```

---

## Measuring Agent Effectiveness

Good agents:
- Complete tasks without requiring follow-up questions
- Provide actionable, specific recommendations
- Include evidence (benchmarks, profiles) for claims
- Format output consistently for easy parsing
- Stay within scope (don't go on tangents)

Poor agents:
- Return vague suggestions ("consider optimizing")
- Make claims without evidence
- Go off-task or analyze irrelevant code
- Produce unstructured output
- Require multiple invocations to get useful results

**Iterate on agent prompts** based on actual results. Shockwave agents are living documents.

---

## Quick Reference

### Agent Creation Checklist

```markdown
[ ] Frontmatter with name, description, tools
[ ] Mission statement (what does this agent do?)
[ ] Workflow with phases/steps
[ ] Output format template
[ ] Constraints (what NOT to do)
[ ] Success criteria (what makes a good result?)
[ ] Examples of good output
[ ] Integration with Shockwave patterns
```

### Invocation Template

```
Use the [agent-name] agent to [specific task].

Context: [Why you need this, what it informs]

Focus on: [Specific files/areas]

Success criteria:
- [Criterion 1]
- [Criterion 2]

Return:
- [Expected output format]
```

---

## Conclusion

Effective agent orchestration in Shockwave means:
- Delegating well-defined analysis or execution tasks
- Providing clear success criteria and constraints
- Leveraging agents' isolated context for deep dives
- Using parallel agents for comprehensive reviews
- Iterating on agent prompts based on results

**Agents amplify your effectiveness. Use them strategically.** 🚀
