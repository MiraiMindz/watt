# Memory Profiling

Run memory profiling analysis to identify allocation hotspots and potential memory leaks.

## Your Task

1. Run memory profiling during benchmarks:
   ```bash
   go test -bench=. -memprofile=mem.prof -memprofilerate=1
   ```

2. Analyze allocation space (total allocations):
   ```bash
   go tool pprof -alloc_space -top mem.prof
   ```

3. Check live heap (current memory usage):
   ```bash
   go tool pprof -inuse_space -top mem.prof
   ```

4. Identify top allocation sources and provide:
   - Top 10 functions by allocation rate
   - Total allocation rate (MB/s)
   - Hot paths that need optimization
   - Specific file:line references for each issue

5. If allocations found in zero-allocation hot paths (parser, keep-alive handler), flag as critical issues

6. Suggest specific optimizations:
   - Use sync.Pool for object
   - Pre-allocate slices
   - Replace string operations with []byte
   - Fix escape to heap issues

Invoke the `memory-profiling` skill to assist with analysis.
