# Run Comprehensive Benchmark Suite

Execute the full benchmark suite for Shockwave and analyze results.

## Your Task

1. Run benchmarks with proper iterations for statistical significance:
   ```bash
   go test -bench=. -benchmem -count=10 -benchtime=3s ./...
   ```

2. If baseline exists at `benchmarks/baseline.txt`, compare:
   ```bash
   benchstat benchmarks/baseline.txt /tmp/current_bench.txt
   ```

3. Check for regressions:
   - Latency: Max 5% regression acceptable
   - Allocations: Zero allocations must stay zero
   - Memory: Max 10% regression acceptable

4. Provide summary with:
   - Key performance metrics (ns/op, B/op, allocs/op)
   - Comparison with baseline (if exists)
   - Any regressions detected
   - Recommendations

5. If this is a new baseline, offer to save results to `benchmarks/baseline.txt`
