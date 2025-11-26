# CPU Profiling

Run CPU profiling to identify performance bottlenecks and hot paths.

## Your Task

1. Generate CPU profile during benchmarks:
   ```bash
   go test -bench=. -cpuprofile=cpu.prof
   ```

2. Analyze hot functions:
   ```bash
   go tool pprof -top cpu.prof
   ```

3. Identify bottlenecks:
   - Functions consuming >5% CPU time
   - Unexpected hot paths
   - Lock contention
   - System call overhead

4. For each hot function, provide:
   - % of total CPU time
   - File and line number
   - Call graph context
   - Optimization suggestions

5. Generate flame graph visualization:
   ```bash
   go tool pprof -http=:8080 cpu.prof
   ```

   Describe what the flame graph reveals about the call hierarchy.

6. Recommend optimizations:
   - Inlining opportunities
   - Reduce function call overhead
   - Algorithm improvements
   - Parallelization opportunities

Invoke the `go-performance-optimization` skill for detailed analysis.
