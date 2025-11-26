# /test Command

Run comprehensive test suite with coverage and race detection.

## Usage

```
/test [package]
```

## What This Command Does

1. Runs unit tests
2. Checks test coverage
3. Runs race detector
4. Validates zero-allocation claims
5. Generates test report

## Options

```
/test              # Test all packages
/test core         # Test core package only
/test --coverage   # Generate HTML coverage report
/test --race       # Run with race detector (always included)
/test --verbose    # Verbose output
```

## Commands Executed

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# With race detector
go test -race ./...

# Allocation tests
go test -run=TestAlloc ./...
```

## Success Criteria

- [ ] All tests pass
- [ ] Coverage >80%
- [ ] No race conditions detected
- [ ] Zero-allocation paths verified (where claimed)

## Example Output

```
Package: github.com/yourusername/bolt/core
=== RUN   TestContextParam
--- PASS: TestContextParam (0.00s)
=== RUN   TestRouterLookup
--- PASS: TestRouterLookup (0.00s)
PASS
coverage: 87.5% of statements

Package: github.com/yourusername/bolt/pool
=== RUN   TestContextPool
--- PASS: TestContextPool (0.00s)
=== RUN   TestZeroAllocation
--- PASS: TestZeroAllocation (0.00s)
PASS
coverage: 92.3% of statements

Race detector: PASS (no data races detected)

Summary:
  Total packages: 5
  Tests passed: 42/42
  Coverage: 87.5%
  Race conditions: 0
```

## When Coverage is Below 80%

The command will:
1. Identify files below coverage target
2. Suggest missing test scenarios
3. Create test templates for uncovered code

## Agent Used

This command invokes the **tester** agent which:
- Has Read, Write, Edit, Bash tools
- Creates comprehensive test suites
- Verifies zero-allocation claims
- Detects race conditions
- Ensures >80% coverage

---

*See MASTER_PROMPTS.md for testing guidelines*
