# /benchmark Command

Run competitive benchmarks against Echo, Gin, and Fiber.

## Usage

```
/benchmark [scenario]
```

## What This Command Does

1. Runs Bolt benchmarks
2. Runs competitor benchmarks (Gin, Echo, Fiber)
3. Compares results
4. Generates performance report
5. Validates performance targets

## Scenarios

```
/benchmark            # Run all scenarios
/benchmark static     # Static route benchmark only
/benchmark dynamic    # Dynamic route benchmark only
/benchmark middleware # Middleware chain benchmark
/benchmark json       # Large JSON encoding
/benchmark concurrent # Concurrent throughput
/benchmark all        # Comprehensive suite
```

## Commands Executed

```bash
# Run benchmarks with 10-second runs
go test -bench=. -benchmem -benchtime=10s ./benchmarks

# Generate profiles
go test -bench=BenchmarkBolt -cpuprofile=cpu.prof ./benchmarks
go test -bench=BenchmarkBolt -memprofile=mem.prof ./benchmarks

# Statistical analysis
go install golang.org/x/perf/cmd/benchstat@latest
benchstat results.txt
```

## Benchmark Scenarios

### 1. Static Route
- Simple JSON response
- Measures framework overhead
- Target: <2000ns/op, <1000 B/op, <5 allocs/op

### 2. Dynamic Route
- Path parameter extraction
- Measures routing performance
- Target: <3000ns/op, <1000 B/op, <8 allocs/op

### 3. Middleware Chain
- 5 middleware stack
- Measures composition overhead
- Target: <1000ns/op, <600 B/op, <3 allocs/op

### 4. Large JSON
- 10KB JSON response
- Measures encoding performance
- Target: <2000ns/op, <11000 B/op, <2 allocs/op

### 5. Concurrent Throughput
- Parallel requests
- Measures scalability
- Target: >1M req/s

## Example Output

```
# Bolt Framework - Competitive Benchmark Report

Date: 2025-11-13
Go Version: go1.21.5
CPU: AMD Ryzen 9 5900X
RAM: 32GB

## Static Route Performance

| Framework | ns/op | B/op | allocs/op | Winner |
|-----------|-------|------|-----------|--------|
| Bolt      | 1,184 | 512  | 3         | ✓      |
| Gin       | 2,089 | 800  | 5         |        |
| Echo      | 2,387 | 920  | 6         |        |
| Fiber     | 1,756 | 680  | 4         |        |

Bolt is 43% faster than Gin, 50% faster than Echo

## Summary

Bolt wins: 6/6 scenarios (100%)
Average improvement: 62% vs Gin, 67% vs Echo

Performance targets: MET ✓
```

## Performance Regression Detection

If performance degrades:
```
WARNING: Performance regression detected!

Benchmark: BenchmarkStaticRoute
Previous: 1,200 ns/op
Current:  2,400 ns/op
Regression: 100% slower

Suggested actions:
1. Run: go test -bench=BenchmarkStaticRoute -cpuprofile=cpu.prof
2. Analyze: go tool pprof cpu.prof
3. Check recent commits for changes
```

## Fairness Validation

The command ensures:
- [ ] Same Go version for all frameworks
- [ ] Production configurations used
- [ ] Identical test scenarios
- [ ] Multiple iterations (10s benchtime)
- [ ] Statistical significance

## Agent Used

This command invokes the **benchmarker** agent which:
- Has Read, Write, Bash tools
- Creates fair, realistic benchmarks
- Compares against competitors
- Detects performance regressions
- Generates comprehensive reports

---

*See MASTER_PROMPTS.md for benchmarking guidelines*
