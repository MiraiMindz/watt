# Quick Start Guide

## What This Tool Does

This tool automatically:
1. Discovers and runs ALL benchmarks in BOLT and SHOCKWAVE projects
2. Compares frameworks within their respective categories:
   - **HTTP Servers**: SHOCKWAVE vs net/http vs fasthttp
   - **Web Frameworks**: BOLT vs Gin vs Echo vs Fiber
3. Analyzes performance across CPU, Memory, and Allocation metrics
4. Generates comprehensive rankings and statistical reports

## Quick Usage

### 1. Build the Tool

```bash
cd /home/mirai/Documents/Programming/Projects/watt/benchstat
go build -o benchstat
```

### 2. Run with Time Budget (Recommended)

```bash
cd /home/mirai/Documents/Programming/Projects/watt

# 30 minutes total (most common)
./benchstat/benchstat -total-time 30m

# 1 hour for comprehensive analysis
./benchstat/benchstat -total-time 1h

# 5 minutes for quick CI checks
./benchstat/benchstat -total-time 5m
```

The tool will:
- Split time evenly: 50% for BOLT project, 50% for SHOCKWAVE project
- Automatically optimize `-benchtime` and `-count` to fit within budget
- Run all benchmarks including competitors
- Generate detailed reports

### 3. View Results

```bash
# Results are in benchmark_results/
ls -lh benchmark_results/

# View the latest report
cat benchmark_results/benchmark_report_*.txt | less
```

## Example Output

### Console Output (Quick Summary)

```
Starting benchmark orchestrator...
Discovering benchmarks...
Found 156 benchmarks total (78 BOLT, 78 SHOCKWAVE)
Optimizing benchmark parameters for 30m0s total time...
Optimized: benchtime=1.8s, count=8

Running benchmarks...
Completed 1248 benchmark runs in 28m47s

================================================================================
 Quick Summary
================================================================================

HTTP SERVERS:
  RequestParsing            : shockwave (score: 567.89)
  ResponseWriting           : shockwave (score: 890.12)
  KeepAlive                 : fasthttp (score: 1234.56)
  SimpleGET                 : shockwave (score: 456.78)

WEB FRAMEWORKS:
  StaticRoute               : bolt (score: 1234.56)
  DynamicRoute              : fiber (score: 1456.78)
  MiddlewareChain           : bolt (score: 987.65)
  LargeJSON                 : gin (score: 2345.67)
  QueryParams               : echo (score: 1678.90)
```

### Report File Output

```
================================================================================
 BOLT & SHOCKWAVE Comprehensive Benchmark Report
================================================================================

Generated: 2025-11-15 14:26:33
System: linux/amd64 (CPUs: 8)
Total Execution Time: 28m47s

================================================================================
 HTTP SERVER BENCHMARKS
================================================================================

Scenario: RequestParsing
-------------------------------------------------------------------------------
Rank   Framework        Score        ns/op        B/op    allocs/op
-------------------------------------------------------------------------------
1      shockwave      567.89       550.23       15.00         0.05
2      fasthttp       678.90       650.45       20.00         0.08
3      nethttp        890.12       850.00       35.00         0.15

Best Performers:
  CPU (ns/op):     shockwave
  Memory (B/op):   shockwave
  Allocs:          shockwave

...

================================================================================
 WEB FRAMEWORK BENCHMARKS
================================================================================

Scenario: StaticRoute
-------------------------------------------------------------------------------
Rank   Framework        Score        ns/op        B/op    allocs/op
-------------------------------------------------------------------------------
1      bolt           1234.56      1184.23       42.00         0.12
2      fiber          1345.67      1298.45       45.00         0.15
3      gin            1456.78      1398.23       48.00         0.18
4      echo           1567.89      1487.34       52.00         0.22

Best Performers:
  CPU (ns/op):     bolt
  Memory (B/op):   bolt
  Allocs:          bolt

...
```

## How Categories Work

The tool automatically categorizes frameworks:

### HTTP Server Category
- **shockwave** - Your HTTP server
- **nethttp** - Go's standard library `net/http`
- **fasthttp** - valyala/fasthttp

These are compared in scenarios like:
- Request parsing
- Response writing
- Keep-alive handling
- WebSocket connections
- Header processing

### Web Framework Category
- **bolt** - Your web framework
- **gin** - Gin web framework
- **echo** - Echo web framework
- **fiber** - Fiber web framework

These are compared in scenarios like:
- Static route handling
- Dynamic route with parameters
- Middleware chain execution
- Large JSON encoding
- Query parameter parsing

## Understanding Results

### Composite Score

```
Score = ns/op + (0.1 × B/op) + (100 × allocs/op)
```

**Lower is better**

### Metrics

- **ns/op**: Nanoseconds per operation (CPU efficiency)
- **B/op**: Bytes allocated per operation (memory efficiency)
- **allocs/op**: Number of allocations per operation (GC pressure)

### Example Comparison

```
HTTP SERVERS - RequestParsing:
1. shockwave:  550 ns/op,  15 B/op, 0.05 allocs/op → Score: 567.89 ✓ WINNER
2. fasthttp:   650 ns/op,  20 B/op, 0.08 allocs/op → Score: 678.90
3. nethttp:    850 ns/op,  35 B/op, 0.15 allocs/op → Score: 890.12

Conclusion: SHOCKWAVE is 18% faster than fasthttp, 54% faster than net/http
```

## Common Commands

### Default (Manual Settings)

```bash
# Uses default: benchtime=1s, count=5
./benchstat/benchstat
```

### Time Budget (Automatic Optimization)

```bash
# Quick test (5 minutes)
./benchstat/benchstat -total-time 5m

# Standard (30 minutes)
./benchstat/benchstat -total-time 30m

# Comprehensive (1-2 hours)
./benchstat/benchstat -total-time 1h
```

### Custom Settings

```bash
# High confidence results
./benchstat/benchstat -benchtime 5s -count 20

# Different CPU configurations
./benchstat/benchstat -cpu 1,2,4,8,16

# Verbose output
./benchstat/benchstat -total-time 30m -v

# Custom project paths
./benchstat/benchstat -bolt ../bolt -shockwave ../shockwave
```

## Tips for Best Results

1. **Close other applications** before running
2. **Use 30m-1h time budget** for reliable results
3. **Run multiple times** and check consistency
4. **Check system load** with `top` before running
5. **Consider `nice` priority**:
   ```bash
   sudo nice -n -20 ./benchstat/benchstat -total-time 30m
   ```

## What Gets Benchmarked

### SHOCKWAVE Benchmarks
- HTTP/1.1 parsing and response writing
- HTTP/2 frame handling
- HTTP/3/QUIC packet processing
- WebSocket connections
- Keep-alive handling
- Buffer pool performance
- Comparison vs net/http and fasthttp

### BOLT Benchmarks
- Router performance (static and dynamic routes)
- Middleware chain execution
- JSON encoding/decoding
- Query parameter parsing
- Context pooling
- Comparison vs Gin, Echo, and Fiber

## Interpreting Rankings

### Scenario Rankings

Each scenario shows head-to-head competition:

```
HTTP SERVERS - KeepAlive:
Rank 1: fasthttp    (best for this scenario)
Rank 2: shockwave   (close second)
Rank 3: nethttp     (standard library baseline)
```

### Overall Performance

Look at how many #1 rankings each has:

```
HTTP SERVERS - Summary:
shockwave: 5 first-place finishes
fasthttp:  2 first-place finishes
nethttp:   0 first-place finishes

WEB FRAMEWORKS - Summary:
bolt:  4 first-place finishes
gin:   1 first-place finish
echo:  1 first-place finish
fiber: 2 first-place finishes
```

## Troubleshooting

### "No benchmarks found"

```bash
# Check paths
./benchstat/benchstat -v -bolt ../bolt -shockwave ../shockwave
```

### "Takes too long"

```bash
# Reduce time budget
./benchstat/benchstat -total-time 10m

# Or reduce CPU configs
./benchstat/benchstat -cpu 1,4
```

### "Inconsistent results"

```bash
# Increase samples
./benchstat/benchstat -benchtime 5s -count 20
```

## Next Steps

- See `README.md` for full documentation
- See `USAGE_EXAMPLES.md` for detailed examples
- Run with `-h` for all flags

---

**Start benchmarking now:**

```bash
cd /home/mirai/Documents/Programming/Projects/watt
./benchstat/benchstat -total-time 30m
```
