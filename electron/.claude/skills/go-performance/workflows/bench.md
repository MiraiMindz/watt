# Run Benchmarks and Compare to Targets

You are tasked with running Go benchmarks and comparing results to performance targets defined in the CLAUDE.md file.

## Your Task

1. **Run Benchmarks**
   ```bash
   go test -bench=. -benchmem -run=^$ ./...
   ```

2. **Analyze Results**
   - Compare results to targets in CLAUDE.md:
     - Arena allocation: <10ns/op, 0 allocs/op
     - Pool get/put: <50ns/op, 0 allocs/op
     - SIMD string compare: >10GB/s
     - Lock-free queue: >10M ops/sec

3. **Report Findings**
   Create a summary showing:
   - ✅ Benchmarks that meet targets
   - ❌ Benchmarks that miss targets
   - Performance comparison
   - Recommendations for improvements

## Output Format

```markdown
# Benchmark Report

## Summary
[X/Y benchmarks meet performance targets]

## Results by Component

### Arena Allocation
```
BenchmarkArena_Alloc-8    100000000    8.5 ns/op    0 B/op    0 allocs/op
```
- **Target:** <10ns/op, 0 allocs/op
- **Status:** ✅ PASS
- **Performance:** 15% under target latency

### [Other components...]

## Benchmarks Failing Targets
[List any that don't meet targets with recommendations]

## Recommendations
[Suggestions for optimization if needed]
```

## Notes
- Run benchmarks multiple times for stability: `-count=10`
- Use `benchstat` to compare if you have baseline results
- Save results for regression tracking
