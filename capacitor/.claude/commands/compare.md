Compare Capacitor implementation against competitors or alternative approaches.

Steps:
1. Ask what to compare:
   - Capacitor cache vs stdlib map/sync.Map
   - Capacitor vs GORM
   - Capacitor vs sqlc
   - Capacitor vs ent
   - Capacitor vs custom implementation
   - New implementation vs baseline

2. Set up comparison:
   - Ensure both implementations do exactly the same thing
   - Use same data sizes and types
   - Same number of operations
   - Same concurrency level
   - Same hardware/environment

3. Create comparison benchmarks:
   ```go
   func BenchmarkComparison(b *testing.B) {
       b.Run("Capacitor", func(b *testing.B) {
           // Capacitor implementation
       })

       b.Run("Competitor", func(b *testing.B) {
           // Competitor implementation
       })
   }
   ```

4. Run benchmarks with multiple iterations:
   ```bash
   go test -bench=BenchmarkComparison -benchmem -count=10 | tee comparison.txt
   benchstat comparison.txt
   ```

5. Compare metrics:
   - Throughput (ops/sec)
   - Latency (ns/op)
   - Memory usage (B/op)
   - Allocations (allocs/op)
   - CPU efficiency
   - Concurrency scaling

6. Create comparison table:
   ```
   | Metric           | Capacitor    | Competitor   | Improvement |
   |-----------------|--------------|--------------|-------------|
   | Throughput      | 1.2M ops/s   | 800K ops/s   | +50%        |
   | Latency         | 85 ns/op     | 125 ns/op    | -32%        |
   | Memory          | 0 B/op       | 48 B/op      | -100%       |
   | Allocations     | 0 allocs/op  | 2 allocs/op  | -100%       |
   ```

7. Generate graphs if requested:
   - Latency comparison
   - Throughput comparison
   - Memory usage over time
   - Scaling with concurrency

8. Analyze and explain:
   - Why Capacitor is faster (or slower)
   - What optimizations contribute most
   - Trade-offs made
   - When to use each approach

9. Save comparison report to benchmarks/comparisons/ with timestamp
