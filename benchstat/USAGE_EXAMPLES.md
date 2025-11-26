# Usage Examples

## Quick Start

### 1. Build the Tool

```bash
cd benchstat
go build -o benchstat
```

### 2. Run with Time Budget (Recommended)

```bash
# Run all benchmarks in 30 minutes (15min per project)
./benchstat -total-time 30m

# Run all benchmarks in 1 hour (30min per project)
./benchstat -total-time 1h
```

### 3. Check the Results

```bash
# Results are saved in benchmark_results/
ls -lh benchmark_results/

# View the latest report
cat benchmark_results/benchmark_report_*.txt | less
```

## Example Output

### Console Output

```
Starting benchmark orchestrator...
Discovering benchmarks...
Found 156 benchmarks total (78 BOLT, 78 SHOCKWAVE)
Optimizing benchmark parameters for 30m0s total time...
Optimized: benchtime=1.8s, count=8

Running benchmarks...
Running 78 BOLT benchmarks...
  Running 12 benchmarks in bolt/benchmarks...
  Completed in 2m15s
  Running 15 benchmarks in bolt/core...
  Completed in 3m42s
  ...

Running 78 SHOCKWAVE benchmarks...
  Running 20 benchmarks in shockwave/pkg/shockwave/http11...
  Completed in 4m18s
  ...

Completed 1248 benchmark runs in 28m47s
Analyzing results...
Report generated: benchmark_results/benchmark_report_20251115_142633.txt

================================================================================
 Quick Summary
================================================================================

StaticRoute                  : shockwave (score: 1234.56)
DynamicRoute                 : shockwave (score: 1456.78)
MiddlewareChain             : bolt (score: 987.65)
LargeJSON                    : bolt (score: 2345.67)
RequestParsing              : shockwave (score: 567.89)
KeepAlive                   : shockwave (score: 890.12)
```

### Report Output

```
================================================================================
 BOLT & SHOCKWAVE Comprehensive Benchmark Report
================================================================================

Generated: 2025-11-15 14:26:33
System: linux/amd64 (CPUs: 8)
Total Execution Time: 28m47s

Configuration:
  Benchtime: 1.8s
  Count: 8
  CPU List: 1,2,4,8
  Time Budget: 30m (optimized)

--------------------------------------------------------------------------------
 Overall Rankings by Scenario
--------------------------------------------------------------------------------

Scenario: StaticRoute
-------------------------------------------------------------------------------
Rank   Project          Score        ns/op        B/op    allocs/op
-------------------------------------------------------------------------------
1      shockwave      1234.56      1184.23       42.00         0.12
2      bolt           1456.78      1398.45       48.50         0.25

Best Performers:
  CPU (ns/op):     shockwave
  Memory (B/op):   shockwave
  Allocs:          shockwave


Scenario: DynamicRoute
-------------------------------------------------------------------------------
Rank   Project          Score        ns/op        B/op    allocs/op
-------------------------------------------------------------------------------
1      shockwave      1456.78      1402.34       45.00         0.15
2      bolt           1678.90      1587.23       52.00         0.30

Best Performers:
  CPU (ns/op):     shockwave
  Memory (B/op):   shockwave
  Allocs:          shockwave

...

--------------------------------------------------------------------------------
 Summary
--------------------------------------------------------------------------------

Overall Averages:

Project              Avg ns/op       Avg B/op   Avg allocs/op
-------------------------------------------------------------------------------
BOLT                   1523.45          48.75            0.28
SHOCKWAVE              1298.67          43.20            0.14

Performance Comparison:
  SHOCKWAVE is 1.17x faster than BOLT on average
  SHOCKWAVE uses 1.13x less memory than BOLT on average
```

## Common Use Cases

### 1. Pre-Release Benchmarking

```bash
# Comprehensive benchmarking before a release
./benchstat -total-time 2h -cpu 1,2,4,8,16 -v

# Results show detailed performance across all CPU configs
```

### 2. Feature Comparison

```bash
# Benchmark current version
./benchstat -total-time 30m -output baseline

# After implementing a feature
./benchstat -total-time 30m -output after_feature

# Compare reports manually
diff baseline/benchmark_report_*.txt after_feature/benchmark_report_*.txt
```

### 3. Quick CI Check

```bash
# Fast benchmarking in CI (5 minutes)
./benchstat -total-time 5m -cpu 1,4

# Exit code 0 = success
echo $?  # 0
```

### 4. Memory Profiling Focus

```bash
# Run with specific focus on memory
./benchstat -benchtime 5s -count 20 -cpu 1

# Longer benchtime and higher count = better memory statistics
```

### 5. CPU Scalability Testing

```bash
# Test scalability across many CPU configs
./benchstat -total-time 1h -cpu 1,2,4,8,12,16,24,32

# See how each project scales with more CPUs
```

## Understanding Time Budget

### How the Optimizer Works

Given `-total-time 30m`:

1. **Split**: 15m for BOLT, 15m for SHOCKWAVE
2. **Discover**: Find all benchmarks (e.g., 100 per project)
3. **Calculate**:
   - CPU configs: 4 (from `-cpu 1,2,4,8`)
   - Available time: 15m × 0.95 / 1.1 = ~12m57s
4. **Optimize**:
   - Try count=1: benchtime = 12m57s / (100 × 4 × 1) = 1.94s
   - Try count=5: benchtime = 12m57s / (100 × 4 × 5) = 0.39s
   - Try count=8: benchtime = 12m57s / (100 × 4 × 8) = 0.24s
   - ...
   - Best: count=5, benchtime=0.39s (good balance)
5. **Execute**: Run all benchmarks with optimized parameters

### Manual Control

If you prefer manual control:

```bash
# Skip optimization
./benchstat -total-time 30m -no-optimize -benchtime 2s -count 10

# Or don't use -total-time at all
./benchstat -benchtime 3s -count 15 -cpu 1,2,4,8
```

## Tips for Better Results

### 1. System Preparation

```bash
# Close unnecessary applications
# Disable CPU frequency scaling (if possible)
# Run on a dedicated benchmarking machine

# Set process priority
sudo nice -n -20 ./benchstat -total-time 30m
```

### 2. Longer Time Budgets

```bash
# More time = better statistics
./benchstat -total-time 2h  # Excellent

# vs
./benchstat -total-time 5m  # Quick but less reliable
```

### 3. Multiple Runs

```bash
# Run multiple times and compare
./benchstat -total-time 30m -output run1
./benchstat -total-time 30m -output run2
./benchstat -total-time 30m -output run3

# Check consistency across runs
diff run1/benchmark_report_*.txt run2/benchmark_report_*.txt
```

### 4. CPU Configuration

```bash
# For development (fast)
./benchstat -cpu 1,4

# For production comparison (comprehensive)
./benchstat -cpu 1,2,4,8,16,32

# For single-threaded focus
./benchstat -cpu 1 -benchtime 10s -count 20
```

## Interpreting Results

### Composite Score

```
Score = ns/op + (0.1 × B/op) + (100 × allocs/op)
```

**Example**:
- SHOCKWAVE: 1200 ns/op, 40 B/op, 0.1 allocs/op
  - Score = 1200 + (0.1 × 40) + (100 × 0.1) = 1200 + 4 + 10 = 1214
- BOLT: 1300 ns/op, 35 B/op, 0.2 allocs/op
  - Score = 1300 + (0.1 × 35) + (100 × 0.2) = 1300 + 3.5 + 20 = 1323.5

**Winner**: SHOCKWAVE (lower score)

### Statistical Significance

Look at StdDev (standard deviation):

```
Benchmark                        Mean (ns)    StdDev
-------------------------------------------------
BenchmarkBolt_StaticRoute         1500.00     25.00   ← Very consistent
BenchmarkShockwave_StaticRoute    1200.00    150.00   ← More variable
```

- Lower StdDev = more consistent performance
- High StdDev might indicate:
  - System noise
  - GC interference
  - Thermal throttling
  - Need more samples (increase `-count`)

### Memory Efficiency

```
B/op = Bytes allocated per operation
allocs/op = Number of allocations per operation
```

**Ideal**:
- B/op: < 100 bytes (excellent), < 1KB (good)
- allocs/op: 0 (zero-allocation), < 5 (good)

**Example**:
```
BenchmarkShockwave_ZeroAlloc     1000 ns/op    0 B/op    0 allocs/op  ← Perfect!
BenchmarkBolt_WithAlloc          1100 ns/op   64 B/op    2 allocs/op  ← Good
```

## Troubleshooting

### Issue: Benchmarks Taking Too Long

**Solution 1**: Reduce time budget
```bash
./benchstat -total-time 10m
```

**Solution 2**: Reduce CPU configs
```bash
./benchstat -cpu 1,4
```

**Solution 3**: Use manual control
```bash
./benchstat -benchtime 500ms -count 3
```

### Issue: Inconsistent Results

**Solution 1**: Increase samples
```bash
./benchstat -count 20
```

**Solution 2**: Increase benchtime
```bash
./benchstat -benchtime 5s
```

**Solution 3**: Check system load
```bash
top  # Close other applications
```

### Issue: Out of Memory

**Solution**: Reduce concurrency or benchmark fewer packages at once

The tool runs benchmarks package by package, so this should be rare. If it happens:
1. Close other applications
2. Run with smaller time budget
3. Monitor with `htop`

### Issue: Benchmarks Not Found

**Solution**: Check paths
```bash
./benchstat -v -bolt ../bolt -shockwave ../shockwave
```

Look for output:
```
Found X benchmarks total (Y BOLT, Z SHOCKWAVE)
```

If counts are 0, verify:
1. Paths are correct
2. Projects have `*_test.go` files with `func Benchmark*` functions

## Advanced Usage

### Custom Benchmark Selection

Currently the tool runs ALL benchmarks. To run specific benchmarks:

**Option 1**: Modify the code to filter by package name

**Option 2**: Run `go test` directly with your custom filters:
```bash
cd bolt/benchmarks
go test -bench=StaticRoute -benchmem -benchtime=3s -count=10
```

### Integration with benchstat Tool

The Go team provides a `benchstat` tool for statistical comparison:

```bash
# Install
go install golang.org/x/perf/cmd/benchstat@latest

# Use with our tool's output
# (Would need to export raw benchmark output - future enhancement)
```

### CI/CD Integration

```bash
#!/bin/bash
# ci-benchmark.sh

set -e

# Run quick benchmark
./benchstat -total-time 5m -output ci_results

# Parse results (simple check)
if grep -q "SHOCKWAVE is.*faster" ci_results/benchmark_report_*.txt; then
    echo "✓ Benchmarks passed"
    exit 0
else
    echo "✗ Performance regression detected"
    exit 1
fi
```

## FAQ

**Q: How long should I run benchmarks?**
A: 30 minutes for development, 1-2 hours for releases

**Q: What CPU configs should I use?**
A: Use powers of 2 up to your core count: `-cpu 1,2,4,8` on an 8-core machine

**Q: Why are results inconsistent?**
A: Increase `-count` and `-benchtime`, close other apps, check for thermal throttling

**Q: Can I benchmark just one project?**
A: Yes, modify the code to skip one project, or run `go test` directly

**Q: How do I compare with other frameworks (Gin, Echo, Fiber)?**
A: Their benchmarks are in `bolt/benchmarks/` - the tool includes them automatically

**Q: What's a good composite score?**
A: Lower is better. Absolute values don't matter - compare relative rankings

**Q: Should I use -no-optimize?**
A: Only if you want manual control over benchtime/count parameters

---

**Happy benchmarking! For issues, see README.md**
