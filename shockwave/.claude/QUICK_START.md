# Quick Start Guide - Claude Code Ecosystem

Get started with the Shockwave `.claude/` ecosystem in 5 minutes.

---

## 🎯 Quick Reference

### For Optimization Work

```bash
# Explicit instruction with context
"Optimize ParseRequest in pkg/shockwave/http11/parser.go to achieve zero
allocations. This is critical because allocations degrade p99 latency.
Target: 0 allocs/op, maintain RFC 7230 compliance."

# Auto-invokes: go-performance-optimization skill
# Claude will: read code in parallel, run benchmarks, apply techniques, verify
```

### For Analysis

```bash
# Delegate to specialized agent
"Use the performance-auditor agent to analyze pkg/shockwave/http11/ for
allocation hotspots. Focus on parser.go. Return file:line references and
prioritized recommendations."

# Agent runs autonomously, returns comprehensive report
```

### For Quick Operations

```bash
/bench              # Run benchmark suite
/check-allocs       # Verify zero allocations
/profile-cpu        # CPU profiling
/profile-mem        # Memory profiling
```

---

## 📚 The Three Documents You Need

### 1. [PROMPTING_GUIDE.md](PROMPTING_GUIDE.md)
**Claude 4.5 best practices**

Key takeaways:
- Be explicit with context (what, why, success criteria)
- Use parallel tool execution
- Track state for long sessions (git + progress files)
- Investigate before answering (no speculation)

### 2. [AGENT_PATTERNS.md](AGENT_PATTERNS.md)
**Effective agent orchestration**

When to use agents:
- Complex analysis (comprehensive audits)
- Tasks benefiting from isolated context
- Parallel work (agent + main Claude)
- Specialized expertise (protocol validation)

### 3. [SKILL_PATTERNS.md](SKILL_PATTERNS.md)
**Writing discoverable skills**

Good skills have:
- Keyword-rich descriptions
- Context explaining rationale
- Concrete before/after examples
- Verification steps
- Clear workflows

---

## 🚀 Common Workflows

### Workflow 1: Optimize Function

```
1. You: "Optimize ParseRequest for zero allocations. Target: 0 allocs/op."

2. Claude (auto-invokes go-performance-optimization skill):
   - Reads parser.go, related files in parallel
   - Runs baseline benchmark
   - Runs escape analysis
   - Identifies allocations

3. Claude implements optimizations:
   - Replaces string concatenation with pre-compiled constants
   - Switches to inline arrays for headers
   - Uses sync.Pool for buffers

4. Claude verifies:
   - Runs new benchmark (0 allocs/op achieved)
   - Checks escape analysis (no new escapes)
   - Runs full suite (no regressions)

5. Claude reports:
   "Optimized ParseRequest (parser.go:145-178). Reduced from 3 allocs/op
   to 0 allocs/op, 16% faster. Full benchmark suite passes."
```

### Workflow 2: Comprehensive Audit

```
1. You: "Use performance-auditor agent to audit pkg/shockwave/http11/"

2. Agent (runs autonomously):
   - Reviews all .go files in package
   - Runs escape analysis on each
   - Executes benchmarks
   - Profiles CPU and memory
   - Generates detailed report

3. Agent returns:
   "Performance Audit Report
   - 15 files audited
   - 7 critical issues (zero-alloc violations)
   - 12 optimization opportunities
   - See full report with file:line references and fixes"

4. You (or Claude) implement fixes based on prioritized recommendations
```

### Workflow 3: Protocol Compliance

```
1. You: "Use protocol-validator agent to check HTTP/2 RFC 7540 compliance"

2. Agent:
   - Reads RFC requirements
   - Reviews HTTP/2 implementation
   - Runs compliance test suite
   - Checks edge cases (malformed input, limits)

3. Agent returns:
   "HTTP/2 Compliance Report
   - 45/47 RFC MUST requirements: ✓
   - 2 violations found (frame parsing, flow control)
   - 8 missing test cases
   - See compliance_report.md for details"
```

---

## 🎨 Best Practices

### ✅ Do This

**Explicit with context:**
```
Optimize X to achieve Y because Z. Target: concrete metrics.
```

**Parallel operations:**
```
Read parser.go, pool.go, and benchmarks simultaneously.
Run benchmarks and escape analysis in parallel.
```

**Evidence-based:**
```
Run benchmarks before optimization.
Verify with benchstat (p < 0.05).
Check escape analysis.
```

### ❌ Avoid This

**Vague requests:**
```
"Make it faster"  // Too vague, Claude guesses what to optimize
```

**Sequential when parallel possible:**
```
Read file1, then file2, then file3  // Slow, should be parallel
```

**Speculation without data:**
```
"This probably allocates..."  // Should read code and run benchmarks first
```

---

## 📊 Key Metrics

### Zero-Allocation Targets

These functions MUST have `0 allocs/op`:
- `ParseRequest` (pkg/shockwave/http11/parser.go)
- `WriteResponse` (pkg/shockwave/http11/response.go)
- `ServeHTTP` for keep-alive connections
- `HeaderLookup` (constant lookup)

### Regression Tolerance

- Max 5% performance regression in any benchmark
- Zero tolerance for allocation increases in zero-alloc targets
- Statistical significance required (benchstat p < 0.05)

---

## 🔧 Available Tools

### Skills (Auto-Invoked)
- `go-performance-optimization` - Zero-allocation techniques
- `benchmark-analysis` - Statistical benchmark interpretation
- `http-protocol-testing` - RFC compliance
- `memory-profiling` - Allocation analysis

### Commands (User-Invoked)
- `/bench` - Comprehensive benchmarks
- `/check-allocs` - Zero-allocation verification
- `/profile-cpu` - CPU profiling
- `/profile-mem` - Memory profiling
- `/compare-nethttp` - Compare with stdlib
- `/test-protocol` - Protocol compliance tests

### Agents (Complex Tasks)
- `performance-auditor` - Comprehensive performance audit
- `protocol-validator` - RFC compliance checking
- `benchmark-runner` - Automated benchmark execution

---

## 🎓 Learning Path

### Beginner
1. Read this Quick Start
2. Try: "Optimize this function for zero allocations"
3. Observe how skills activate automatically

### Intermediate
1. Read [PROMPTING_GUIDE.md](PROMPTING_GUIDE.md)
2. Try: "Use performance-auditor agent for full audit"
3. Learn to provide explicit context and success criteria

### Advanced
1. Read [AGENT_PATTERNS.md](AGENT_PATTERNS.md) and [SKILL_PATTERNS.md](SKILL_PATTERNS.md)
2. Create custom skills for your domain
3. Build agent pipelines for complex workflows

---

## 💡 Pro Tips

### Tip 1: Let Skills Activate Naturally

Don't say "use the performance skill to..."
Just ask: "How can I reduce allocations in this parser?"
Claude will invoke the right skill automatically.

### Tip 2: Use Agents for Deep Dives

Simple optimization: Main Claude with skills
Comprehensive audit: Delegate to agent
Agent gets fresh context, focuses deeply.

### Tip 3: Track State for Long Sessions

```bash
# Before starting multi-session work
go test -bench=. -benchmem > baseline.txt
git commit -m "Baseline before optimization"

# During work
echo "Optimized parser, next: response writer" > progress_notes.txt

# Fresh session
git log --oneline  # See what's done
cat progress_notes.txt  # Load context
```

### Tip 4: Verify Everything

After optimization:
- Benchmark (0 allocs/op?)
- Escape analysis (no new escapes?)
- Full suite (no regressions?)
- Statistical significance (benchstat p < 0.05?)

Never trust without data.

### Tip 5: Communicate Clearly

After tool use, Claude should summarize:
- What changed (file:line)
- What technique applied
- Results (before → after metrics)
- Next step

---

## 🆘 Troubleshooting

### Skills Not Activating

Check skill description has keywords matching your query:
```yaml
description: Optimize Go code for zero allocations, reduce GC pressure,
improve hot paths, sync.Pool patterns...
```

### Agents Not Completing

Ensure:
- Clear mission and success criteria
- Appropriate tools granted
- Output format specified

### Hooks Not Running

```bash
chmod +x .claude/hooks/*.sh
```

---

## 📖 Full Documentation

- **[README.md](README.md)** - Complete ecosystem overview
- **[PROMPTING_GUIDE.md](PROMPTING_GUIDE.md)** - Claude 4.5 best practices
- **[AGENT_PATTERNS.md](AGENT_PATTERNS.md)** - Agent orchestration
- **[SKILL_PATTERNS.md](SKILL_PATTERNS.md)** - Skill authoring
- **[../CLAUDE.md](../CLAUDE.md)** - Project constitution

---

**The benchmark is the source of truth.** 🎯
