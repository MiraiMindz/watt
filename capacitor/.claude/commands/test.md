Run comprehensive tests on the project with coverage analysis.

Execute:
1. Run all tests with race detector:
   ```bash
   go test -race -v ./...
   ```

2. Run with coverage:
   ```bash
   go test -cover -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out -o coverage.html
   ```

3. Check coverage threshold (should be > 85%):
   ```bash
   go tool cover -func=coverage.out | grep total | awk '{print $3}'
   ```

4. Run specific test categories if requested:
   - Unit tests: `go test -short ./...`
   - Integration tests: `go test -run Integration ./...`
   - Benchmark tests: `go test -bench=. -run=^$ ./...`

5. Report:
   - Total tests run
   - Passed/Failed/Skipped
   - Coverage percentage per package
   - Coverage delta from last run (if available)
   - Race conditions detected (if any)
   - Failed tests with detailed output

6. If coverage < 85%:
   - Identify uncovered code
   - Suggest which functions need tests
   - Generate test templates for uncovered functions

7. If tests fail:
   - Show detailed failure output
   - Suggest potential fixes
   - Check if failures are in new code

8. Save results to test-results/ directory with timestamp
