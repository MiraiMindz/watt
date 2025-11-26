# Test Command

Run the full test suite and report results.

## Usage

```
/test [package]
```

## What This Does

1. Runs all tests or tests for specific package
2. Reports coverage
3. Runs benchmarks
4. Identifies failures
5. Suggests fixes if tests fail

## Examples

```bash
# All tests
/test

# Specific package
/test pkg/lexer

# With coverage
/test -coverage

# Benchmarks only
/test -bench
```

## Output Format

```markdown
## Test Results

**Status:** ✅ PASSING / ❌ FAILING
**Coverage:** XX%
**Tests:** XX passed, YY failed
**Benchmarks:** All targets met

### Failed Tests
- TestLexerJSX: Expected token JSX_LT, got ILLEGAL

### Performance
- Lexer: 1050 lines/ms ✅
- Parser: 480 lines/ms ⚠️ (target: 500)

### Recommendations
- Fix JSX tokenization in lexer.go:234
- Optimize parser expression handling
```
