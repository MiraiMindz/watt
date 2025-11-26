# Compare Performance with net/http

Benchmark Shockwave against Go's standard library net/http to measure improvement.

## Your Task

1. Create a comparison benchmark if it doesn't exist in `benchmarks/comparison_test.go`:
   ```go
   func BenchmarkShockwave(b *testing.B) {
       // Shockwave server implementation
   }

   func BenchmarkNetHTTP(b *testing.B) {
       // net/http server implementation
   }
   ```

2. Run comparison benchmarks:
   ```bash
   go test -bench=Benchmark -benchmem ./benchmarks/
   ```

3. Analyze metrics:
   - Throughput (ops/sec): Shockwave vs net/http
   - Latency (ns/op): Shockwave vs net/http
   - Memory (B/op): Shockwave vs net/http
   - Allocations (allocs/op): Shockwave vs net/http

4. Calculate improvements:
   - % throughput improvement
   - % latency reduction
   - % memory reduction
   - Allocation reduction

5. Test scenarios:
   - Simple GET requests
   - POST with body
   - Large responses
   - Keep-alive connections
   - Concurrent connections

6. Provide summary table:
   | Metric | net/http | Shockwave | Improvement |
   |--------|----------|-----------|-------------|
   | ops/sec | X | Y | Z% |
   | ns/op | X | Y | Z% |
   | B/op | X | Y | Z% |
   | allocs/op | X | Y | Z |

7. Highlight Shockwave advantages:
   - Zero-allocation parsing
   - Better connection pooling
   - Socket optimizations
   - Protocol-specific features
