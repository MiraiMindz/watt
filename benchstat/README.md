# Benchmark Orchestrator for BOLT & SHOCKWAVE

A comprehensive Go benchmarking tool that automatically discovers, runs, analyzes, and compares benchmarks across BOLT (web framework) and SHOCKWAVE (HTTP server) projects.

## Features

- **Automatic Benchmark Discovery**: Scans both projects and discovers all benchmark functions
- **Intelligent Time Budget Optimization**: Automatically calculates optimal `-benchtime` and `-count` values based on total time budget
- **Comprehensive Statistical Analysis**: Calculates mean, standard deviation, min/max for all metrics
- **Competitive Rankings**: Compares BOLT vs SHOCKWAVE across multiple scenarios
- **Multi-Metric Analysis**: Analyzes CPU efficiency (ns/op), memory efficiency (B/op), and allocation efficiency (allocs/op)
- **Detailed Reporting**: Generates comprehensive reports with rankings and statistical insights

## Installation

```bash
cd benchstat
go build -o benchstat
```

## Usage

### Basic Usage

Run all benchmarks with default settings (1s benchtime, 5 count):

```bash
./benchstat
```

### Time Budget Mode (Recommended)

Automatically optimize benchmark parameters to complete within a specified time:

```bash
# Run all benchmarks in 30 minutes
./benchstat -total-time 30m

# Run all benchmarks in 1 hour
./benchstat -total-time 1h

# Run all benchmarks in 15 minutes
./benchstat -total-time 15m
```

The tool will automatically:
1. Discover all benchmarks in both projects
2. Calculate the optimal `-benchtime` and `-count` values
3. Split time evenly between BOLT and SHOCKWAVE (15m each for 30m total)
4. Account for overhead and ensure completion within budget

### Custom Configuration

```bash
# Custom benchtime and count
./benchstat -benchtime 3s -count 10

# Custom CPU list
./benchstat -cpu 1,2,4,8,16

# Custom project paths
./benchstat -bolt ../bolt -shockwave ../shockwave

# Custom output directory
./benchstat -output ./my_results

# Verbose output
./benchstat -v

# Skip automatic optimization
./benchstat -total-time 30m -no-optimize
```

### All Flags

```
-total-time <duration>    Total time budget (e.g., 30m, 1h, 2h30m)
-benchtime <duration>     Time per benchmark iteration (default: 1s)
-count <int>              Number of times to run each benchmark (default: 5)
-cpu <list>               Comma-separated CPU counts (default: 1,2,4,8)
-bolt <path>              Path to BOLT project (default: bolt)
-shockwave <path>         Path to SHOCKWAVE project (default: shockwave)
-output <path>            Output directory for results (default: benchmark_results)
-v                        Verbose output
-no-optimize              Skip automatic flag optimization
```

## How Time Budget Optimization Works

When you specify `-total-time`, the tool:

1. **Discovers** all benchmarks in both projects
2. **Calculates** the number of total benchmark executions:
   - `executions = benchmarks × CPU_configs × count`
3. **Optimizes** the combination of `benchtime` and `count` to maximize statistical confidence while staying within budget:
   - Higher `count` = better statistical confidence
   - Longer `benchtime` = more stable measurements
   - The algorithm finds the optimal balance
4. **Accounts** for overhead (10% overhead buffer + 5% execution buffer)
5. **Splits** time evenly: 50% for BOLT, 50% for SHOCKWAVE

### Example Calculation

For 30 minutes total time with 100 benchmarks per project and 4 CPU configs:

```
Time per project = 30m / 2 = 15m
Available time = 15m × 0.95 / 1.1 = 12m 57s

Total executions = 100 benchmarks × 4 CPUs × count
Time = executions × benchtime

Optimizer tries different count values (1-50) and finds:
count = 8, benchtime = 1.9s
→ Total: 100 × 4 × 8 × 1.9s = 6,080s ≈ 12m 57s
```

## Output

The tool generates a comprehensive report with:

### 1. Overall Rankings by Scenario

Example scenarios:
- StaticRoute
- DynamicRoute
- MiddlewareChain
- LargeJSON
- QueryParams
- RequestParsing
- KeepAlive

For each scenario, it ranks BOLT vs SHOCKWAVE by:
- Composite score (weighted: ns/op + 0.1×B/op + 100×allocs/op)
- Individual metrics: ns/op, B/op, allocs/op
- Best performers per metric

### 2. Detailed Statistics

Per-benchmark statistics including:
- Mean performance (ns/op)
- Standard deviation
- Memory usage (B/op)
- Allocation count (allocs/op)

### 3. Overall Summary

- Average performance across all benchmarks
- Direct comparison: "SHOCKWAVE is 1.45x faster than BOLT"
- Memory efficiency comparison
- Allocation efficiency comparison

## Example Usage Scenarios

### Quick Test (5 minutes)

```bash
./benchstat -total-time 5m
```

Good for: Quick validation, CI/CD pipelines

### Standard Benchmarking (30 minutes)

```bash
./benchstat -total-time 30m
```

Good for: Regular development benchmarking, feature comparison

### Comprehensive Analysis (1-2 hours)

```bash
./benchstat -total-time 1h -cpu 1,2,4,8,16
```

Good for: Release benchmarking, detailed performance analysis, blog posts

### Custom Scenario

```bash
./benchstat -benchtime 5s -count 20 -cpu 1,4,8 -v
```

Good for: Manual control, specific testing requirements

## Understanding the Output

### Composite Score

The tool calculates a composite score for ranking:

```
Score = ns/op + (0.1 × B/op) + (100 × allocs/op)
```

This weights:
- **CPU performance** (ns/op) as primary
- **Memory efficiency** (B/op) as secondary (10% weight)
- **Allocation efficiency** (allocs/op) as important (high weight since allocations are expensive)

Lower score = better performance

### Statistical Metrics

- **Mean (ns/op)**: Average time per operation
- **StdDev**: Standard deviation (lower = more consistent)
- **B/op**: Bytes allocated per operation (lower = more memory efficient)
- **allocs/op**: Number of allocations per operation (lower = less GC pressure)

## Tips for Best Results

1. **Close other applications** to reduce system noise
2. **Run on a stable system** - avoid thermal throttling
3. **Use longer time budgets** for more reliable results (30m-1h)
4. **Run multiple times** and compare results for consistency
5. **Check system load** with `top` or `htop` before running
6. **Consider using `nice`** for process priority:
   ```bash
   sudo nice -n -20 ./benchstat -total-time 30m
   ```

## Interpreting Rankings

### Scenario Rankings

Each scenario compares BOLT vs SHOCKWAVE in specific use cases:

- **Rank 1** = Best overall performance (lowest score)
- Check individual metrics to see where each excels
- Look for **Best Performers** to see who wins each metric

### Overall Summary

The overall summary shows average performance across ALL benchmarks:

- If "SHOCKWAVE is 1.5x faster" → SHOCKWAVE has 50% better CPU performance
- If "BOLT uses 2x less memory" → BOLT allocates half the memory

## Troubleshooting

### Benchmarks Not Found

```bash
./benchstat -v
# Check output for "Found X benchmarks"
# Verify paths with -bolt and -shockwave flags
```

### Benchmarks Take Too Long

```bash
# Reduce time budget
./benchstat -total-time 10m

# Or reduce CPU configs
./benchstat -cpu 1,4
```

### Inconsistent Results

- Increase count: `-count 20`
- Increase benchtime: `-benchtime 5s`
- Close background applications
- Run on a dedicated machine

## Advanced Usage

### Comparing Specific Packages

```bash
# Discover what's available
./benchstat -v 2>&1 | grep "Running benchmarks in"

# Then manually filter by editing the code or using separate runs
```

### Continuous Benchmarking

```bash
# Create a benchmark baseline
./benchstat -total-time 30m -output baseline_v1.0

# After changes, compare
./benchstat -total-time 30m -output after_changes

# Compare reports manually or use benchstat tool
```

### Integration with CI/CD

```bash
# Quick CI check (5 minutes)
./benchstat -total-time 5m || exit 1

# Store results
cp benchmark_results/benchmark_report_*.txt artifacts/
```

## Project Structure

```
benchstat/
├── main.go           # Main orchestrator
├── go.mod            # Go module definition
├── README.md         # This file
└── benchmark_results/ # Generated reports (created automatically)
    └── benchmark_report_YYYYMMDD_HHMMSS.txt
```

## How It Works

1. **Discovery Phase**
   - Walks file tree in both projects
   - Finds all `*_test.go` files
   - Extracts `func Benchmark*` functions using regex
   - Groups by package and project

2. **Optimization Phase** (if -total-time is set)
   - Calculates total executions needed
   - Tries count values from 1-50
   - For each count, calculates optimal benchtime
   - Scores each combination: `score = count × log(benchtime)`
   - Selects best score within time budget

3. **Execution Phase**
   - Groups benchmarks by package
   - Runs `go test -bench=. -benchmem` per package
   - Parses output in real-time
   - Collects all results

4. **Analysis Phase**
   - Groups results by benchmark name
   - Calculates mean, stddev, min, max
   - Creates statistical summaries

5. **Ranking Phase**
   - Extracts scenarios from benchmark names
   - Groups by scenario
   - Calculates composite scores
   - Ranks projects per scenario

6. **Reporting Phase**
   - Generates comprehensive text report
   - Prints console summary
   - Saves to timestamped file

## Performance Characteristics

- **Discovery**: O(n) where n = number of files
- **Optimization**: O(50) constant time (tries 50 count values)
- **Execution**: O(b × c × p) where b=benchmarks, c=count, p=CPU configs
- **Analysis**: O(r) where r = number of results
- **Ranking**: O(s × log s) where s = number of scenarios

## Requirements

- Go 1.21 or later
- BOLT and SHOCKWAVE projects in parent directory (or custom paths)
- Sufficient disk space for results (typically <10MB)

## Contributing

To add new features:

1. Fork and modify `main.go`
2. Test with both projects
3. Update README
4. Submit PR

## License

Same as parent project (BOLT/SHOCKWAVE)

---

**Happy Benchmarking! 🚀**
