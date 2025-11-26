# Quick Start - Statistical Benchmarks

## TL;DR - Just Run This

```bash
cd /home/mirai/Documents/Programming/Projects/watt/bolt/benchmarks
./run_statistical_benchmarks.sh 20 3s
```

This will:
- Run 20 iterations of all benchmarks
- Take ~15-20 minutes
- Generate a statistical summary
- Save results to `benchmark_results_YYYYMMDD_HHMMSS/`

## What You'll Get

### During Execution
You'll see progress like this:
```
========================================
Bolt Statistical Benchmark Runner
========================================

Configuration:
  Iterations: 20
  Benchtime:  3s
  Output Dir: ./benchmark_results_20251114_153045

Running iteration 1/20...
BenchmarkFull_Bolt_StaticRoute-8   ...
✓ Iteration 1 complete

Running iteration 2/20...
...
```

### After Completion
A summary file showing:
```
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

## Recommended Runs

### Quick Check (5-10 minutes)
```bash
./run_statistical_benchmarks.sh 10 3s
```

### Standard (15-20 minutes) - **RECOMMENDED**
```bash
./run_statistical_benchmarks.sh 20 3s
```

### High Confidence (30-40 minutes)
```bash
./run_statistical_benchmarks.sh 30 5s
```

## After Running

1. **View the summary:**
```bash
cat benchmark_results_*/summary.txt
```

2. **Share the results folder with me:**
Just let me know the folder name (e.g., `benchmark_results_20251114_153045`) and I'll analyze the detailed statistics.

## Tips for Best Results

### Before Running
- Close heavy applications (browsers, IDEs, etc.)
- Make sure laptop is plugged in (not on battery)
- Ensure system is not under heavy load

### If You See High Variance
If standard deviations are >30ns, consider:
- Running more iterations (30+)
- Checking system load (`top` or `htop`)
- Disabling CPU frequency scaling (for advanced users)

## That's It!

The script does everything automatically. Just run it and share the results folder when done.

See `STATISTICAL_BENCHMARKS_README.md` for detailed documentation.
