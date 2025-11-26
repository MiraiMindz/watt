# Bench Command

Run benchmarks and analyze performance.

## Usage

```
/bench [package]
```

## What This Does

1. Runs benchmarks for all or specific package
2. Compares against performance targets
3. Identifies regressions
4. Suggests optimizations

## Example

```bash
# All benchmarks
/bench

# Specific package
/bench pkg/lexer

# With memory profiling
/bench -mem
```

## Output

```markdown
## Benchmark Results

### Lexer
- Throughput: 1050 lines/ms ✅ (target: 1000)
- Allocations: 2 per op ✅
- Memory: 1024 B/op ✅

### Parser
- Throughput: 480 lines/ms ⚠️ (target: 500)
- Allocations: 5 per op ⚠️ (increase from 3)
- Memory: 2048 B/op ✅

### Recommendations
- Parser: Investigate allocation increase
- Consider pre-allocating slice in parseChildren()
```

## Profiling

For detailed analysis:
```bash
go test -cpuprofile=cpu.prof -bench=BenchmarkParser
go tool pprof cpu.prof
```
