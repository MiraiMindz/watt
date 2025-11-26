---
name: go-testing
description: Expert in Go testing including unit tests, benchmarks, fuzz tests, table-driven tests, and test utilities. Use when writing or improving tests, creating benchmarks, or setting up testing infrastructure.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Go Testing Skill

## Purpose
This skill provides expertise in writing comprehensive, maintainable tests for Go code with emphasis on performance validation and correctness verification.

## When to Use This Skill
- Writing unit tests for new code
- Creating benchmarks for performance validation
- Setting up fuzz tests for parsers/validators
- Detecting memory leaks
- Building test utilities and helpers

## Core Principles

### 1. Test Pyramid
```
      /\
     /  \    E2E Tests (few)
    /----\
   /      \  Integration Tests (some)
  /--------\
 /          \ Unit Tests (many)
/------------\
```

Focus on unit tests, with targeted integration and E2E tests.

### 2. Table-Driven Tests
The Go way of testing multiple cases:
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"positive", 2, 3, 5},
        {"negative", -1, -1, -2},
        {"zero", 0, 5, 5},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := add(tt.a, tt.b); got != tt.want {
                t.Errorf("add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
            }
        })
    }
}
```

### 3. Test Organization
```
package/
├── arena.go           # Implementation
├── arena_test.go      # Unit tests
├── bench_test.go      # Benchmarks (separate file)
├── fuzz_test.go       # Fuzz tests
├── example_test.go    # Examples (documentation)
└── testutil/          # Test utilities
    └── helpers.go
```

## Unit Testing Patterns

### 1. Basic Test Structure
```go
func TestFeature(t *testing.T) {
    // Arrange
    input := setupInput()
    expected := expectedOutput()

    // Act
    result := functionUnderTest(input)

    // Assert
    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

### 2. Subtests for Organization
```go
func TestArena(t *testing.T) {
    t.Run("allocation", func(t *testing.T) {
        arena := NewArena(1024)
        defer arena.Free()

        ptr := arena.Alloc(64)
        if ptr == nil {
            t.Fatal("allocation failed")
        }
    })

    t.Run("reset", func(t *testing.T) {
        arena := NewArena(1024)
        defer arena.Free()

        arena.Alloc(64)
        arena.Reset()

        // Should be able to allocate again
        ptr := arena.Alloc(64)
        if ptr == nil {
            t.Error("allocation after reset failed")
        }
    })
}
```

### 3. Helper Functions
```go
func TestComplexFeature(t *testing.T) {
    t.Helper() // Marks this as a helper

    arena := mustCreateArena(t, 1024)
    defer arena.Free()

    // Test code...
}

func mustCreateArena(t *testing.T, size int) *Arena {
    t.Helper()

    arena := NewArena(size)
    if arena == nil {
        t.Fatal("failed to create arena")
    }
    return arena
}
```

### 4. Testing Errors
```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
        errType error
    }{
        {"valid", "good input", false, nil},
        {"empty", "", true, ErrEmpty},
        {"invalid", "bad", true, ErrInvalid},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)

            if tt.wantErr && err == nil {
                t.Error("expected error, got nil")
            }

            if !tt.wantErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }

            if tt.errType != nil && !errors.Is(err, tt.errType) {
                t.Errorf("wrong error type: got %v, want %v", err, tt.errType)
            }
        })
    }
}
```

## Benchmarking

### 1. Basic Benchmark
```go
func BenchmarkArenaAlloc(b *testing.B) {
    arena := NewArena(1024 * 1024) // 1MB
    defer arena.Free()

    b.ResetTimer()      // Reset timer after setup
    b.ReportAllocs()    // Report allocations

    for i := 0; i < b.N; i++ {
        arena.Alloc(64)
    }
}
```

### 2. Benchmark with Sizes
```go
func BenchmarkPoolGet(b *testing.B) {
    sizes := []int{64, 256, 1024, 4096, 16384}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            pool := NewBufferPool()

            b.SetBytes(int64(size))  // For throughput calculation
            b.ResetTimer()
            b.ReportAllocs()

            for i := 0; i < b.N; i++ {
                buf := pool.Get(size)
                pool.Put(buf)
            }
        })
    }
}
```

### 3. Parallel Benchmarks
```go
func BenchmarkConcurrentQueue(b *testing.B) {
    q := NewQueue[int](1024)

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            q.Enqueue(42)
            q.Dequeue()
        }
    })
}
```

### 4. Preventing Compiler Optimizations
```go
var (
    resultInt    int
    resultString string
    resultBool   bool
)

func BenchmarkComputation(b *testing.B) {
    var r int

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r = expensiveComputation() // Store result
    }

    resultInt = r // Prevent dead code elimination
}
```

### 5. Benchmark Comparison
```bash
# Run benchmarks and save results
go test -bench=. -benchmem -count=5 > old.txt

# Make changes...

# Run again and compare
go test -bench=. -benchmem -count=5 > new.txt
benchstat old.txt new.txt
```

## Fuzz Testing

### 1. Basic Fuzz Test
```go
func FuzzParseURL(f *testing.F) {
    // Seed corpus
    f.Add("https://example.com/path")
    f.Add("http://localhost:8080")
    f.Add("")

    f.Fuzz(func(t *testing.T, input string) {
        url, err := ParseURL(input)

        if err == nil && url == nil {
            t.Error("nil URL without error")
        }

        // Should never panic
        _ = url.String()
    })
}
```

### 2. Multi-Parameter Fuzzing
```go
func FuzzArenaAlloc(f *testing.F) {
    f.Add(1024, 64)
    f.Add(4096, 128)

    f.Fuzz(func(t *testing.T, arenaSize, allocSize int) {
        // Validate inputs
        if arenaSize < 0 || arenaSize > 1024*1024 {
            t.Skip()
        }
        if allocSize < 0 || allocSize > arenaSize {
            t.Skip()
        }

        arena := NewArena(arenaSize)
        defer arena.Free()

        ptr := arena.Alloc(allocSize)
        if allocSize > 0 && ptr == nil {
            t.Error("allocation failed for valid size")
        }
    })
}
```

### 3. Invariant Testing
```go
func FuzzStringBytesRoundtrip(f *testing.F) {
    f.Add("hello")
    f.Add("")
    f.Add("unicode: 你好")

    f.Fuzz(func(t *testing.T, s string) {
        // String -> []byte -> string should be identity
        b := StringToBytes(s)
        s2 := BytesToString(b)

        if s != s2 {
            t.Errorf("roundtrip failed: %q != %q", s, s2)
        }
    })
}
```

## Memory Leak Detection

### 1. Allocation Tracking
```go
func TestNoMemoryLeak(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping leak test in short mode")
    }

    var before runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&before)

    // Run operation many times
    for i := 0; i < 10000; i++ {
        arena := NewArena(1024)
        arena.Alloc(64)
        arena.Free()
    }

    runtime.GC()
    var after runtime.MemStats
    runtime.ReadMemStats(&after)

    leaked := after.Alloc - before.Alloc
    if leaked > 1024*1024 { // 1MB threshold
        t.Errorf("memory leak detected: %d bytes leaked", leaked)
    }
}
```

### 2. Goroutine Leak Detection
```go
func TestNoGoroutineLeak(t *testing.T) {
    before := runtime.NumGoroutine()

    // Start and stop worker
    w := NewWorker()
    w.Start()
    w.Stop()

    // Wait for cleanup
    time.Sleep(100 * time.Millisecond)

    after := runtime.NumGoroutine()
    if after > before {
        t.Errorf("goroutine leak: before=%d after=%d", before, after)
    }
}
```

## Test Utilities

### 1. Test Fixtures
```go
package testutil

type Fixture struct {
    TempDir string
    Cleanup func()
}

func Setup(t *testing.T) *Fixture {
    t.Helper()

    dir := t.TempDir() // Automatically cleaned up

    return &Fixture{
        TempDir: dir,
        Cleanup: func() {
            // Additional cleanup if needed
        },
    }
}
```

### 2. Golden Files
```go
func TestRender(t *testing.T) {
    result := Render(input)

    goldenPath := filepath.Join("testdata", "golden.txt")

    if *update {
        // Update golden file
        os.WriteFile(goldenPath, []byte(result), 0644)
    }

    expected, _ := os.ReadFile(goldenPath)

    if result != string(expected) {
        t.Errorf("output differs from golden file")
        t.Logf("got:\n%s", result)
        t.Logf("want:\n%s", expected)
    }
}

var update = flag.Bool("update", false, "update golden files")
```

### 3. Test Doubles (Mocks)
```go
type MockStorage struct {
    GetFunc func(key string) ([]byte, error)
    PutFunc func(key string, value []byte) error
}

func (m *MockStorage) Get(key string) ([]byte, error) {
    if m.GetFunc != nil {
        return m.GetFunc(key)
    }
    return nil, errors.New("not implemented")
}

func (m *MockStorage) Put(key string, value []byte) error {
    if m.PutFunc != nil {
        return m.PutFunc(key, value)
    }
    return errors.New("not implemented")
}

// Usage:
func TestWithMock(t *testing.T) {
    mock := &MockStorage{
        GetFunc: func(key string) ([]byte, error) {
            if key == "test" {
                return []byte("value"), nil
            }
            return nil, errors.New("not found")
        },
    }

    // Test code using mock...
}
```

## Coverage Analysis

### 1. Running Coverage
```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Coverage by package
go test -cover ./...
```

### 2. Coverage Threshold
```bash
#!/bin/bash
# check-coverage.sh

THRESHOLD=80

coverage=$(go test -cover ./... | grep -oP '\d+\.\d+(?=%)')
if (( $(echo "$coverage < $THRESHOLD" | bc -l) )); then
    echo "Coverage $coverage% is below threshold $THRESHOLD%"
    exit 1
fi
```

## Testing Best Practices

### 1. Test Names
```go
// Good: Descriptive test names
func TestArena_AllocReturnsNilWhenOutOfMemory(t *testing.T)
func TestPool_PutResetsBuffer(t *testing.T)
func TestQueue_EnqueueReturnsFalseWhenFull(t *testing.T)

// Bad: Vague names
func TestArena1(t *testing.T)
func TestPool(t *testing.T)
func TestQueue_Test1(t *testing.T)
```

### 2. Test Independence
```go
// Bad: Tests depend on each other
var globalArena *Arena

func TestArenaInit(t *testing.T) {
    globalArena = NewArena(1024) // Modifies global state
}

func TestArenaAlloc(t *testing.T) {
    globalArena.Alloc(64) // Depends on previous test
}

// Good: Each test is independent
func TestArenaAlloc(t *testing.T) {
    arena := NewArena(1024)
    defer arena.Free()

    ptr := arena.Alloc(64)
    if ptr == nil {
        t.Fatal("allocation failed")
    }
}
```

### 3. Fast Tests
```go
func TestQuick(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping slow test in short mode")
    }

    // Slow test code...
}

// Run only fast tests:
// go test -short
```

### 4. Parallel Tests
```go
func TestParallel(t *testing.T) {
    t.Parallel() // Mark as parallel

    // This test can run concurrently with other parallel tests
}
```

## Example Tests

### 1. Example Functions (Documentation)
```go
func ExampleArena() {
    arena := NewArena(1024)
    defer arena.Free()

    // Allocate memory
    ptr := arena.Alloc(64)
    if ptr != nil {
        fmt.Println("Allocation succeeded")
    }

    // Reset and reuse
    arena.Reset()

    // Output:
    // Allocation succeeded
}
```

### 2. Testable Examples
```go
func ExampleBufferPool_Get() {
    pool := NewBufferPool()

    buf := pool.Get(1024)
    buf.WriteString("hello")
    fmt.Println(buf.Len())

    pool.Put(buf)

    // Output:
    // 5
}
```

## Benchmark Targets for Electron

### Expected Performance
```go
// arena/bench_test.go
func BenchmarkArena_Alloc(b *testing.B) {
    arena := NewArena(1024 * 1024)
    defer arena.Free()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        arena.Alloc(64)
    }

    // Target: <10ns/op, 0 allocs/op
}

// pool/bench_test.go
func BenchmarkPool_GetPut(b *testing.B) {
    pool := NewBufferPool()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        buf := pool.Get(1024)
        pool.Put(buf)
    }

    // Target: <50ns/op, 0 allocs/op
}

// lockfree/bench_test.go
func BenchmarkQueue_EnqueueDequeue(b *testing.B) {
    q := NewQueue[int](1024)

    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            q.Enqueue(42)
            q.Dequeue()
        }
    })

    // Target: >10M ops/sec
}
```

## Continuous Testing

### 1. Watch Mode (Development)
```bash
# Install go-watch or similar
go install github.com/mitranim/gow@latest

# Run tests on file change
gow test ./...
```

### 2. CI Configuration
```yaml
# .github/workflows/test.yml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out
```

## Workflow Commands

This skill includes commands:
- `/test` - Run all tests with coverage
- `/bench` - Run benchmarks
- `/fuzz` - Run fuzz tests

See `.claude/skills/go-testing/workflows/` for implementations.

## Resources

- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
- [Table-Driven Tests](https://dave.cheney.net/2013/06/09/writing-table-driven-tests-in-go)
- [Advanced Testing](https://about.sourcegraph.com/go/advanced-testing-in-go)
- [Fuzzing Guide](https://go.dev/doc/fuzz/)

---

**Remember**: Tests are documentation that never lies. Write tests that explain intent.
