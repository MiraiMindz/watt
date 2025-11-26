# Check Allocations in Hot Paths

Verify that critical hot paths maintain zero allocations.

## Your Task

1. Run benchmarks with allocation tracking:
   ```bash
   go test -bench=. -benchmem | grep -E "BenchmarkParseRequest|BenchmarkKeepAlive|BenchmarkWriteStatus"
   ```

2. Check escape analysis for hot path files:
   ```bash
   go build -gcflags="-m -m" ./pkg/shockwave/http11 2>&1 | grep "escapes to heap"
   go build -gcflags="-m -m" ./pkg/shockwave/http2 2>&1 | grep "escapes to heap"
   ```

3. For each hot path, verify:
   - `allocs/op` is 0
   - No unexpected heap escapes
   - Inline arrays used for bounded collections
   - String operations use []byte

4. Critical hot paths to check:
   - HTTP/1.1 request parsing
   - Response status line writing
   - Keep-alive connection handling
   - Header lookup and storage
   - Method dispatch

5. Report any violations with:
   - Function name and location
   - Current allocs/op
   - Root cause of allocation
   - Specific fix recommendation

6. If all hot paths are zero-allocation, provide confirmation and allocation summary.

Invoke `go-performance-optimization` skill for detailed escape analysis.
