# Getting Started with Shockwave Development

Welcome to Shockwave! This guide will help you get started with developing this high-performance HTTP library using Claude Code.

## Prerequisites

- Go 1.22 or later
- Claude Code CLI
- Basic understanding of HTTP protocols
- Familiarity with Go performance optimization (helpful but not required)

## Initial Setup

### 1. Clone and Explore

```bash
cd /path/to/shockwave

# Explore the structure
tree -L 2 .

# Read the project constitution
cat CLAUDE.md
```

### 2. Install Dependencies

```bash
# Install benchstat for performance comparison
go install golang.org/x/perf/cmd/benchstat@latest

# Install pprof (usually comes with Go)
go install -v golang.org/x/tools/cmd/pprof@latest
```

### 3. Run Initial Benchmarks

Establish a baseline for performance:

```bash
# In Claude Code, run:
/bench
```

This will:
- Run comprehensive benchmark suite
- Save results to `results/baseline.txt`
- Show current performance metrics

## Development Workflow

### Basic Workflow

1. **Understand the task**
   ```
   Ask Claude: "Explain how the HTTP/1.1 parser works"
   ```

2. **Make changes**
   - Claude will use skills automatically to guide optimization
   - Hooks will warn about performance anti-patterns

3. **Validate changes**
   ```
   /bench              # Run benchmarks
   /check-allocs       # Verify zero allocations
   /test-protocol      # Check RFC compliance
   ```

4. **Commit**
   - Hooks will validate benchmarks still pass before commit

### Advanced Workflow

#### Performance Optimization

1. **Profile first**
   ```
   /profile-cpu        # Find CPU bottlenecks
   /profile-mem        # Find memory allocations
   ```

2. **Analyze**
   ```
   "Use the performance-auditor agent to analyze pkg/shockwave/http11"
   ```

3. **Optimize**
   - Claude will invoke `go-performance-optimization` skill
   - Suggestions will be specific with code examples

4. **Verify**
   ```
   /bench
   /check-allocs
   ```

#### Protocol Implementation

1. **Understand the spec**
   ```
   "What does RFC 7540 Section 6.9 require for flow control?"
   ```

2. **Implement**
   - Claude will use `http-protocol-testing` skill
   - Provides test cases from RFC

3. **Validate**
   ```
   /test-protocol
   ```
   Or:
   ```
   "Use protocol-validator agent to check HTTP/2 compliance"
   ```

## Understanding the Claude Code Ecosystem

### Skills (Automatic)

Skills activate automatically based on your question:

| Your Question | Skill Activated |
|---------------|-----------------|
| "How can I reduce allocations?" | go-performance-optimization |
| "Does this comply with RFC 7230?" | http-protocol-testing |
| "Why is memory usage high?" | memory-profiling |
| "Analyze these benchmark results" | benchmark-analysis |

**You don't invoke skills manually** - Claude uses them when relevant.

### Commands (Manual)

Commands you run explicitly:

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `/bench` | Run benchmarks | After changes, before commit |
| `/profile-mem` | Memory analysis | High allocation rates |
| `/profile-cpu` | CPU analysis | Performance bottlenecks |
| `/check-allocs` | Verify zero allocs | After optimizing hot paths |
| `/compare-nethttp` | Compare with stdlib | Validate improvements |
| `/test-protocol` | RFC compliance | After protocol changes |

### Agents (Complex Tasks)

Agents for autonomous analysis:

| Agent | Purpose | When to Use |
|-------|---------|-------------|
| performance-auditor | Comprehensive performance audit | Before release, major refactoring |
| protocol-validator | Full RFC compliance check | Before release, protocol changes |
| benchmark-runner | Automated benchmark suite | CI/CD, regression testing |

**Example usage**:
```
"Use the performance-auditor agent to find optimization opportunities in the HTTP/2 implementation"
```

### Hooks (Automatic Validation)

Hooks run automatically in the background:

- **Before Write/Edit**: Warns about performance anti-patterns
- **After editing hot paths**: Suggests running benchmarks
- **Before commit**: Validates benchmarks pass
- **On session start**: Loads performance context

You don't interact with hooks directly - they just work.

## Common Tasks

### Task: Optimize Request Parsing

```
1. Ask: "Profile the HTTP/1.1 request parser for allocations"
2. Claude runs: /profile-mem
3. Claude identifies allocations
4. Ask: "Optimize the parser to eliminate allocations"
5. Claude implements changes using go-performance-optimization skill
6. Verify: /check-allocs
7. Validate: /bench
```

### Task: Add New HTTP/2 Feature

```
1. Ask: "What does RFC 7540 require for server push?"
2. Claude explains using http-protocol-testing skill
3. Ask: "Implement server push support"
4. Claude implements with protocol guidance
5. Verify: /test-protocol
6. Benchmark: /bench
```

### Task: Fix Memory Leak

```
1. Report: "Memory usage keeps growing"
2. Claude runs: /profile-mem
3. Ask: "Use memory-profiling skill to find the leak"
4. Claude analyzes heap growth
5. Fix the issue
6. Verify: Run server and monitor
```

### Task: Pre-Release Audit

```
1. "Use performance-auditor agent for full audit"
2. Agent provides comprehensive report
3. "Use protocol-validator agent to check all protocols"
4. Agent validates RFC compliance
5. Fix identified issues
6. /bench to verify improvements
```

## Performance Targets

Keep these targets in mind:

| Metric | Target | Check With |
|--------|--------|------------|
| Request parsing | 0 allocs/op | `/check-allocs` |
| Keep-alive | 0 allocs/op | `/check-allocs` |
| Status write | 0 allocs/op | `/check-allocs` |
| Header lookup | <10 ns/op | `/bench` |
| HTTP/1.1 throughput | >500k req/s | `/compare-nethttp` |
| HTTP/2 throughput | >300k req/s | `/bench` |
| p99 latency | <100μs | `/bench` |

## Best Practices

### 1. Measure Before Optimizing

```bash
# Wrong
"Optimize this function"

# Right
/profile-cpu
"Optimize the top CPU consumer identified in the profile"
```

### 2. Verify Zero Allocations

After any hot path change:

```bash
/check-allocs
```

### 3. Use Agents for Big Tasks

```bash
# Wrong: Ask many small questions
"Check this file"
"Check that file"
"Check another file"

# Right: Use an agent
"Use performance-auditor agent to check all hot paths"
```

### 4. Trust the Benchmarks

```bash
# Before claiming improvement
/bench
# Shows statistical comparison with baseline
```

## Troubleshooting

### "No baseline found"

Run `/bench` first to create baseline:
```bash
/bench
```

### "Benchmark regression detected"

Check what changed:
```bash
git diff main -- pkg/shockwave/
/bench
```

### "Allocations in hot path"

Profile and fix:
```bash
/profile-mem
/check-allocs
```

### "Protocol test failing"

Check RFC requirements:
```bash
/test-protocol
"What does RFC [number] Section [X] require?"
```

## Next Steps

1. **Read** `CLAUDE.md` - Project constitution
2. **Run** `/bench` - Establish baseline
3. **Explore** `implementation_analysis.md` - Understand current implementation
4. **Start** with a simple task: "Explain the request parsing flow"

## Getting Help

- **General questions**: Just ask Claude
- **Performance questions**: Claude will use `go-performance-optimization` skill
- **Protocol questions**: Claude will use `http-protocol-testing` skill
- **Complex analysis**: Use agents (e.g., "Use performance-auditor agent")

## Resources

- **Main README**: `README.md`
- **Project Rules**: `CLAUDE.md`
- **Claude Ecosystem**: `.claude/README.md`
- **Performance Guide**: `.claude/skills/go-performance-optimization/context/go-performance-guide.md`

---

**Remember**: The benchmark is the source of truth. Measure, optimize, verify! 📊
