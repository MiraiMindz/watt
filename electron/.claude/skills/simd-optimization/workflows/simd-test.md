# Test SIMD Implementations

You are tasked with testing SIMD-accelerated implementations against scalar fallbacks to ensure correctness and performance.

## Your Task

1. **Run Cross-Platform Tests**
   ```bash
   # Run tests on all platforms
   go test ./simd/... -v

   # Run with different CPU features disabled (if possible)
   # This validates fallback paths
   ```

2. **Verify SIMD vs Scalar Equivalence**
   ```bash
   # Run fuzz tests to verify SIMD matches scalar
   go test -fuzz=FuzzSIMD ./simd/...
   ```

3. **Benchmark Performance**
   ```bash
   # Compare SIMD vs scalar performance
   go test -bench=. -benchmem ./simd/...
   ```

4. **Check CPU Feature Detection**
   ```bash
   # Verify CPU features are detected correctly
   go test -v -run=TestCPUFeatures ./simd/...
   ```

## Verification Checklist

For each SIMD operation:
- [ ] Scalar fallback exists
- [ ] SIMD matches scalar output (fuzz tested)
- [ ] CPU feature detection works
- [ ] Performance improvement is measured
- [ ] Both aligned and unaligned inputs tested
- [ ] Size variations tested (small, medium, large)
- [ ] Edge cases covered (empty, odd sizes, etc.)

## Output Format

```markdown
# SIMD Testing Report

## Summary
Tested [X] SIMD operations across [Y] platforms.

## CPU Features Detected
- AVX2: [Yes/No]
- AVX-512: [Yes/No]
- SSE4.2: [Yes/No]
- ARM NEON: [Yes/No]

## SIMD Operations Tested

### Operation: [Name]

**Correctness:**
- ✅ Scalar fallback exists
- ✅ SIMD matches scalar (fuzz tested)
- ✅ Edge cases pass

**Performance:**
```
BenchmarkOperation/scalar-8     1000000    1234 ns/op    0 B/op
BenchmarkOperation/simd-8       8000000     156 ns/op    0 B/op
```
- **Speedup:** 7.9x faster
- **Target:** >4x (✅ PASS)

**Issues:**
[Any issues found]

## Size Variation Results

| Size | Scalar (ns/op) | SIMD (ns/op) | Speedup |
|------|----------------|--------------|---------|
| 16   | 50             | 45           | 1.1x    |
| 32   | 100            | 25           | 4.0x    |
| 64   | 200            | 30           | 6.7x    |
| 1024 | 3000           | 250          | 12.0x   |

## Recommendations
- [Suggestions for improvement]

## Action Items
- [ ] [Fix any issues found]
- [ ] [Add missing tests]
- [ ] [Optimize further if below targets]
```

## Notes
- SIMD benefit increases with data size
- Small sizes (<32 bytes) may be slower due to setup overhead
- Test on multiple CPUs if possible (AMD, Intel, ARM)
- Verify assembly output if available: `go build -gcflags='-S'`
