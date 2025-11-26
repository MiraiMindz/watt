# Profile CPU and Memory Usage

You are tasked with profiling Go code to identify performance bottlenecks and memory issues.

## Your Task

1. **CPU Profiling**
   ```bash
   # Run benchmarks with CPU profiling
   go test -bench=. -cpuprofile=cpu.prof ./...

   # Analyze top functions
   go tool pprof -top cpu.prof

   # Identify hot functions (>10% of time)
   go tool pprof -list=. cpu.prof
   ```

2. **Memory Profiling**
   ```bash
   # Run benchmarks with memory profiling
   go test -bench=. -memprofile=mem.prof ./...

   # Analyze allocation sources
   go tool pprof -alloc_space mem.prof -top

   # Find allocation hot spots
   go tool pprof -alloc_space mem.prof -list=.
   ```

3. **Escape Analysis**
   ```bash
   # Check what escapes to heap
   go build -gcflags='-m -m' ./... 2>&1 | grep "escapes to heap"
   ```

## Your Analysis

Provide a report with:

1. **CPU Hot Spots**
   - Functions consuming >10% CPU
   - Optimization opportunities
   - Expected impact

2. **Memory Hot Spots**
   - Top allocation sources
   - Escape analysis findings
   - Zero-allocation opportunities

3. **Recommendations**
   - Prioritized optimization suggestions
   - Expected performance gains
   - Implementation complexity

## Output Format

```markdown
# Performance Profile Report

## CPU Profile

### Top Functions (>10% CPU time)
1. **[Function Name]** - 15.2% CPU
   - Location: [file:line]
   - Optimization: [suggestion]
   - Expected gain: [estimate]

### Recommendations
- [Prioritized list of optimizations]

## Memory Profile

### Top Allocation Sources
1. **[Function Name]** - 2.5MB allocated
   - Location: [file:line]
   - Reason: [why allocating]
   - Fix: [how to eliminate]

### Escape Analysis Findings
- [List of unexpected heap escapes]

## Action Plan
1. [Highest priority optimization]
2. [Next priority]
3. [etc.]
```

## Notes
- Focus on functions >5% of total time/allocations
- Web UI available: `go tool pprof -http=:8080 cpu.prof`
- Compare before/after profiles to validate improvements
