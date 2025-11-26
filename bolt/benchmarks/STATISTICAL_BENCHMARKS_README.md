# Statistical Benchmark Runner - User Guide

## Overview

This script runs multiple benchmark iterations to collect statistically significant performance data, helping to distinguish real performance differences from measurement noise.

## Quick Start

### Basic Usage (10 iterations, 3s benchtime)
```bash
cd /home/mirai/Documents/Programming/Projects/watt/bolt/benchmarks
./run_statistical_benchmarks.sh
```

### Custom Iterations (20 iterations)
```bash
./run_statistical_benchmarks.sh 20
```

### Custom Benchtime (20 iterations, 5s benchtime)
```bash
./run_statistical_benchmarks.sh 20 5s
```

## What It Does

### 1. Runs Multiple Iterations
- Executes the full competitive benchmark suite N times (default: 10)
- Each iteration runs all 6 scenarios (Static Route, Dynamic Route, Middleware, Large JSON, Query Params, Concurrent)
- Each iteration benchmarks all 4 frameworks (Bolt, Gin, Echo, Fiber)

### 2. Collects Raw Results
- Saves each iteration's output to `benchmark_results_YYYYMMDD_HHMMSS/iteration_N.txt`
- Preserves all benchmark data for manual inspection

### 3. Calculates Statistics
For each benchmark, calculates:
- **Mean (average):** Central tendency
- **Median:** Middle value (less affected by outliers)
- **Standard Deviation:** Variability/consistency
- **Min/Max:** Performance range
- **Statistical Significance:** Determines if differences are real or noise

### 4. Generates Summary Report
Creates a comprehensive summary including:
- Per-scenario statistics table
- Winner identification
- Performance differences vs winner
- Statistical significance markers
- Overall win count

## Output Format

### Example Summary Output

```
================================================================================
Scenario: StaticRoute
================================================================================

Framework    Mean (ns/op)    StdDev      Min         Max         B/op      allocs/op
------------------------------------------------------------------------------------------
Bolt         345.32          12.45       322.10      368.50      16        1.0       🏆
Gin          348.67          9.82        336.40      361.20      40        2.0
Echo         351.23          11.33       345.10      370.80      48        1.0

Performance vs Winner:
  Gin: +3.4ns (+1.0%) ✓ within variance
  Echo: +5.9ns (+1.7%) ✓ within variance

================================================================================
OVERALL WINS SUMMARY
================================================================================

StaticRoute          Winner: Bolt (345.3 ns/op)
DynamicRoute         Winner: Bolt (673.2 ns/op)
MiddlewareChain      Winner: Bolt (378.5 ns/op)
LargeJSON            Winner: Bolt (228456.3 ns/op)
QueryParams          Winner: Bolt (841.7 ns/op)
Concurrent           Winner: Gin (171.3 ns/op)

================================================================================
WIN COUNT:
  Bolt: 5/6 scenarios (83.3%)
  Gin: 1/6 scenarios (16.7%)
================================================================================
```

## Statistical Significance

The script calculates statistical significance using standard deviation:

- **✓ within variance:** Difference is ≤2 standard deviations
  - These differences are likely due to measurement noise
  - In practice, performance is equivalent

- **⚠️ SIGNIFICANT:** Difference is >2 standard deviations
  - This is a real, meaningful performance difference
  - Winner is consistently faster across iterations

## Recommended Configurations

### Quick Validation (5-10 minutes)
```bash
./run_statistical_benchmarks.sh 10 3s
```
- 10 iterations, 3s per benchmark
- Good for confirming recent changes
- ~5-10 minutes runtime

### Standard Analysis (15-25 minutes)
```bash
./run_statistical_benchmarks.sh 20 3s
```
- 20 iterations, 3s per benchmark
- Recommended for reliable statistics
- ~15-25 minutes runtime

### High Confidence (30-50 minutes)
```bash
./run_statistical_benchmarks.sh 30 5s
```
- 30 iterations, 5s per benchmark
- Very high statistical confidence
- ~30-50 minutes runtime

### Publication Quality (1-2 hours)
```bash
./run_statistical_benchmarks.sh 50 5s
```
- 50 iterations, 5s per benchmark
- Suitable for research papers or blog posts
- ~1-2 hours runtime

## Understanding the Results

### Mean vs Median
- **Mean:** Average of all measurements
  - More affected by outliers
  - Good for understanding typical performance

- **Median:** Middle value when sorted
  - Less affected by outliers
  - Better for understanding "most likely" performance

### Standard Deviation
- **Low StdDev (<10ns):** Very consistent performance
  - Framework is stable and predictable

- **Medium StdDev (10-30ns):** Normal variance
  - Typical for most benchmarks

- **High StdDev (>30ns):** High variability
  - May indicate GC pauses, CPU throttling, or measurement noise
  - Consider running more iterations or investigating system load

### Min/Max Range
- **Narrow range (<20ns):** Highly consistent
- **Wide range (>50ns):** Check for system interference
  - Background processes
  - CPU frequency scaling
  - Thermal throttling

## Interpreting Winners

### Clear Winner
- Mean difference >5%
- Statistical significance: ⚠️ SIGNIFICANT
- Consistent across all iterations
- **Conclusion:** Framework A is definitively faster than B

### Effective Tie
- Mean difference <3%
- Statistical significance: ✓ within variance
- Winners vary across iterations
- **Conclusion:** Frameworks are equivalent in practice

### Edge Case
- Mean difference 3-5%
- Statistical significance: borderline
- **Recommendation:** Run more iterations to clarify

## Troubleshooting

### Python3 Not Found
If you see "Python3 not found":
```bash
# Install Python 3
sudo apt-get install python3  # Debian/Ubuntu
sudo yum install python3       # RedHat/CentOS
brew install python3           # macOS
```

### Benchmarks Timeout
If benchmarks timeout (300s limit):
- Reduce benchtime: `./run_statistical_benchmarks.sh 10 2s`
- Or edit script and increase timeout value

### High Variance
If standard deviations are >30ns:
1. Close background applications
2. Disable CPU frequency scaling: `sudo cpupower frequency-set -g performance`
3. Run on dedicated benchmark machine
4. Increase number of iterations to average out noise

## Output Files

All files are saved to `benchmark_results_YYYYMMDD_HHMMSS/`:

- `iteration_1.txt` through `iteration_N.txt`: Raw benchmark outputs
- `parse_results.py`: Statistical analysis script (auto-generated)
- `summary.txt`: Final statistical summary report

## Best Practices

1. **Consistent Environment**
   - Close unnecessary applications
   - Disable CPU frequency scaling
   - Run when system is idle

2. **Adequate Sample Size**
   - Minimum 10 iterations
   - Recommended 20+ iterations
   - 30+ for high confidence

3. **Appropriate Benchtime**
   - 3s is good for most cases
   - 5s for very stable results
   - 10s for micro-benchmarks with high variance

4. **Validation**
   - Check standard deviations (<20ns is good)
   - Review min/max ranges
   - Look for outliers in raw data

## Example Workflow

```bash
# 1. Navigate to benchmarks directory
cd /home/mirai/Documents/Programming/Projects/watt/bolt/benchmarks

# 2. Run statistical benchmarks (20 iterations)
./run_statistical_benchmarks.sh 20 3s

# 3. Wait for completion (~15-20 minutes)
# Script will display progress and final summary

# 4. Review results
cat benchmark_results_*/summary.txt

# 5. (Optional) Analyze specific iteration
cat benchmark_results_*/iteration_5.txt

# 6. Share results directory for analysis
# Send the entire benchmark_results_* folder
```

## Next Steps After Running

Once you have the results:

1. **Review summary.txt** - Check overall winners and significance markers
2. **Verify consistency** - Look for low standard deviations
3. **Share results directory** - Provide the full `benchmark_results_*` folder for detailed analysis
4. **Update documentation** - Use the statistical results to update FINAL_PERFORMANCE_REPORT.md

## Notes

- Each iteration includes a 2-second delay to let the system stabilize
- Total runtime ≈ (iterations × 60s) + (iterations × 2s delay)
- Results are deterministic given same system conditions
- Variance is expected - that's why we run multiple iterations!

## Questions?

If you encounter issues or have questions about interpreting results, provide:
1. The `summary.txt` file
2. System information (CPU, RAM, OS)
3. Any error messages from the script
