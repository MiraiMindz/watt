---
name: simd-optimization
description: Expert in SIMD (Single Instruction Multiple Data) optimizations for Go, including AVX, AVX2, AVX-512, and ARM NEON. Use when implementing vectorized operations for string processing, hashing, or data transformation.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# SIMD Optimization Skill

## Purpose
This skill provides expertise in using SIMD instructions to accelerate data-parallel operations in Go, with automatic fallback to scalar implementations.

## When to Use This Skill
- Implementing batch string operations (compare, search, hash)
- Accelerating data transformations and encoding
- Optimizing mathematical operations on arrays
- Implementing high-throughput parsers
- Any operation processing multiple data elements identically

## Core Principles

### 1. Always Provide Scalar Fallback
SIMD code must work on all platforms:
```go
func BatchHash(keys []string) []uint64 {
    if hasAVX2 {
        return batchHashAVX2(keys)
    }
    return batchHashScalar(keys)
}
```

### 2. CPU Feature Detection
Detect features at initialization:
```go
import "golang.org/x/sys/cpu"

var (
    hasSSE42  = cpu.X86.HasSSE42
    hasAVX    = cpu.X86.HasAVX
    hasAVX2   = cpu.X86.HasAVX2
    hasAVX512 = cpu.X86.HasAVX512F && cpu.X86.HasAVX512BW
)

func init() {
    log.Printf("CPU features: SSE4.2=%v AVX=%v AVX2=%v AVX512=%v",
        hasSSE42, hasAVX, hasAVX2, hasAVX512)
}
```

### 3. Build Tags for Platform-Specific Code
```
simd/
├── batch_amd64.go       # AMD64-specific implementations
├── batch_amd64.s        # Assembly implementations
├── batch_arm64.go       # ARM64-specific implementations
├── batch_generic.go     # Fallback for all platforms
└── batch_test.go        # Tests run on all platforms
```

## Go SIMD Approaches

### Approach 1: Assembly (Most Control)
```
// batch_amd64.s
#include "textflag.h"

// func batchHashAVX2(keys []string) []uint64
TEXT ·batchHashAVX2(SB), NOSPLIT, $0-48
    MOVQ    keys_base+0(FP), SI    // SI = keys ptr
    MOVQ    keys_len+8(FP), CX     // CX = len(keys)
    MOVQ    ret_base+24(FP), DI    // DI = result ptr

    // Load constants into YMM registers
    VBROADCASTSS const_prime(SB), Y0

loop:
    // Process 4 strings at once
    // ... AVX2 instructions ...

    ADDQ    $32, SI
    SUBQ    $4, CX
    JA      loop

    RET
```

```go
// batch_amd64.go
//go:build amd64

func batchHashAVX2(keys []string) []uint64 // Implemented in assembly
```

### Approach 2: Intrinsics via x/arch (Recommended)
```go
//go:build amd64

import "golang.org/x/arch/x86/x86asm"

func compareAVX2(a, b []byte) bool {
    // Use x86asm for intrinsic-like operations
    // This is less common; mostly used for disassembly
}
```

### Approach 3: Compiler Auto-Vectorization
```go
// Write vectorizable code and let compiler handle it
func add(a, b []float64) []float64 {
    result := make([]float64, len(a))
    for i := range a {
        result[i] = a[i] + b[i] // May be vectorized by compiler
    }
    return result
}
```

### Approach 4: CGO with C Intrinsics
```go
/*
#cgo CFLAGS: -mavx2 -O3
#include <immintrin.h>

void batch_hash_avx2(char** keys, int count, uint64_t* output) {
    __m256i prime = _mm256_set1_epi64x(0x9e3779b97f4a7c15ULL);
    // ... AVX2 code ...
}
*/
import "C"

func batchHashAVX2(keys []string, output []uint64) {
    // Convert to C types and call
}
```

## Common SIMD Patterns

### 1. String Comparison (AVX2)

#### Scalar Version
```go
func equalScalar(a, b []byte) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}
```

#### AVX2 Version (Conceptual)
```go
//go:build amd64

func equalAVX2(a, b []byte) bool {
    if len(a) != len(b) {
        return false
    }

    n := len(a)
    i := 0

    // Process 32 bytes at a time with AVX2
    for i+32 <= n {
        // Load 32 bytes from a and b into YMM registers
        // Compare with VPCMPEQB
        // Create mask with VPMOVMSKB
        // If mask != 0xFFFFFFFF, bytes differ
        i += 32
    }

    // Handle remaining bytes with scalar code
    for i < n {
        if a[i] != b[i] {
            return false
        }
        i++
    }

    return true
}
```

### 2. Find Byte (AVX2)

```go
//go:build amd64

func indexByteAVX2(s []byte, c byte) int {
    n := len(s)
    if n == 0 {
        return -1
    }

    // Broadcast target byte to all positions in YMM
    // target := _mm256_set1_epi8(c)

    i := 0
    for i+32 <= n {
        // chunk := _mm256_loadu_si256(&s[i])
        // cmp := _mm256_cmpeq_epi8(chunk, target)
        // mask := _mm256_movemask_epi8(cmp)
        // if mask != 0 {
        //     return i + trailing_zeros(mask)
        // }
        i += 32
    }

    // Scalar fallback for remainder
    for i < n {
        if s[i] == c {
            return i
        }
        i++
    }

    return -1
}
```

### 3. Batch Hashing (AVX2)

```go
//go:build amd64

func batchHashAVX2(keys []string) []uint64 {
    results := make([]uint64, len(keys))

    // Process 4 strings at a time (4 x 64-bit hashes)
    i := 0
    for i+4 <= len(keys) {
        // Load 4 hash initial states into YMM
        // Process each string's bytes
        // Store results
        i += 4
    }

    // Process remaining strings with scalar
    for i < len(keys) {
        results[i] = scalarHash(keys[i])
        i++
    }

    return results
}
```

### 4. Count Occurrences (AVX2)

```go
//go:build amd64

func countByteAVX2(s []byte, c byte) int {
    count := 0
    i := 0

    // Process 32 bytes at a time
    for i+32 <= len(s) {
        // Load 32 bytes
        // Compare all to target byte
        // Count set bits in mask (popcnt)
        i += 32
    }

    // Scalar remainder
    for i < len(s) {
        if s[i] == c {
            count++
        }
        i++
    }

    return count
}
```

## Performance Considerations

### 1. Data Alignment
```go
const alignment = 32 // For AVX2 (256-bit)

func alignedAlloc(size int) []byte {
    // Allocate extra for alignment
    buf := make([]byte, size+alignment)

    // Calculate aligned offset
    addr := uintptr(unsafe.Pointer(&buf[0]))
    offset := (alignment - (addr % alignment)) % alignment

    return buf[offset : offset+size]
}
```

### 2. Loop Unrolling
```go
// Process multiple chunks per iteration
for i+128 <= len(data) {
    processAVX2(&data[i+0])   // First 32 bytes
    processAVX2(&data[i+32])  // Second 32 bytes
    processAVX2(&data[i+64])  // Third 32 bytes
    processAVX2(&data[i+96])  // Fourth 32 bytes
    i += 128
}
```

### 3. Branch Prediction
```go
// BAD: Unpredictable branch in hot loop
for i := range data {
    if data[i] == target { // Unpredictable
        count++
    }
}

// GOOD: Branchless with SIMD
count := countByteAVX2(data, target)
```

### 4. Cache Optimization
```go
const chunkSize = 64 * 1024 // L1 cache size

func processLarge(data []byte) {
    // Process in cache-friendly chunks
    for offset := 0; offset < len(data); offset += chunkSize {
        end := min(offset+chunkSize, len(data))
        processChunkAVX2(data[offset:end])
    }
}
```

## Testing SIMD Code

### 1. Cross-Platform Testing
```go
func TestBatchHash(t *testing.T) {
    keys := []string{"hello", "world", "test"}

    // Test SIMD version (if available)
    simdResult := BatchHash(keys)

    // Test against known-good scalar version
    scalarResult := batchHashScalar(keys)

    if !reflect.DeepEqual(simdResult, scalarResult) {
        t.Errorf("SIMD and scalar results differ")
    }
}
```

### 2. Fuzz Testing
```go
func FuzzSIMDComparison(f *testing.F) {
    f.Add([]byte("hello"), []byte("hello"))
    f.Add([]byte(""), []byte(""))

    f.Fuzz(func(t *testing.T, a, b []byte) {
        scalarResult := equalScalar(a, b)
        simdResult := Equal(a, b) // Dispatches to SIMD or scalar

        if scalarResult != simdResult {
            t.Errorf("scalar=%v simd=%v for a=%q b=%q",
                scalarResult, simdResult, a, b)
        }
    })
}
```

### 3. Benchmark All Sizes
```go
func BenchmarkEqual(b *testing.B) {
    sizes := []int{8, 16, 32, 64, 128, 256, 512, 1024, 4096}

    for _, size := range sizes {
        data1 := make([]byte, size)
        data2 := make([]byte, size)

        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            b.SetBytes(int64(size))
            b.ResetTimer()

            for i := 0; i < b.N; i++ {
                Equal(data1, data2)
            }
        })
    }
}
```

## Build Tags and Organization

### File Structure
```
simd/
├── simd.go              // Public API + feature detection
├── scalar.go            // Scalar fallback (all platforms)
├── simd_amd64.go        // AMD64 Go wrappers
├── simd_amd64.s         // AMD64 assembly
├── simd_arm64.go        // ARM64 Go wrappers
├── simd_arm64.s         // ARM64 assembly
├── simd_test.go         // Tests (all platforms)
└── bench_test.go        // Benchmarks
```

### Build Tag Example
```go
//go:build amd64
// +build amd64

package simd

import "golang.org/x/sys/cpu"

func init() {
    if cpu.X86.HasAVX2 {
        equal = equalAVX2
        indexByte = indexByteAVX2
    } else {
        equal = equalScalar
        indexByte = indexByteScalar
    }
}
```

## ARM NEON Support

### Feature Detection
```go
import "golang.org/x/sys/cpu"

var hasNEON = cpu.ARM64.HasASIMD

func batchHash(keys []string) []uint64 {
    if hasNEON {
        return batchHashNEON(keys)
    }
    return batchHashScalar(keys)
}
```

### NEON Assembly (ARM64)
```
//go:build arm64

// func equalNEON(a, b []byte) bool
TEXT ·equalNEON(SB), NOSPLIT, $0-25
    // NEON instructions
    // LD1, CMP, etc.
    RET
```

## Performance Targets

### Expected Speedups (vs Scalar)
- **String comparison**: 4-8x with AVX2, 8-16x with AVX-512
- **Byte search**: 8-16x with AVX2
- **Batch hashing**: 2-4x with AVX2
- **Counting**: 8-16x with AVX2 + POPCNT

### Benchmark Output Example
```
BenchmarkEqual/scalar/size=1024-8      1000000    1234 ns/op    829 MB/s
BenchmarkEqual/avx2/size=1024-8        8000000     156 ns/op   6564 MB/s  (8.1x)
BenchmarkEqual/avx512/size=1024-8     16000000      89 ns/op  11505 MB/s (13.8x)
```

## Common Pitfalls

### 1. Not Handling Alignment
- AVX2 loads/stores can be aligned or unaligned
- Use `VMOVDQU` (unaligned) unless you guarantee alignment
- Aligned loads (`VMOVDQA`) are faster but require alignment

### 2. Forgetting Scalar Fallback
- Always provide scalar version for:
  - Non-x86 platforms
  - Old CPUs without SIMD support
  - Testing and validation

### 3. Incorrect Remainder Handling
- SIMD processes chunks (32 bytes for AVX2)
- Must handle remaining bytes with scalar code
- Fence-post errors are common

### 4. Premature Optimization
- Only use SIMD if profiling shows benefit
- Small data sizes may be slower with SIMD (setup overhead)
- Benchmark with realistic data sizes

## Workflow Commands

This skill includes commands:
- `/simd-test` - Test SIMD implementations against scalar
- `/simd-bench` - Benchmark SIMD vs scalar performance

See `.claude/skills/simd-optimization/workflows/` for implementations.

## Resources

- [Intel Intrinsics Guide](https://software.intel.com/sites/landingpage/IntrinsicsGuide/)
- [AVX2 Reference](https://www.intel.com/content/www/us/en/docs/intrinsics-guide/index.html)
- [ARM NEON Intrinsics](https://developer.arm.com/architectures/instruction-sets/intrinsics/)
- [Go Assembly Guide](https://go.dev/doc/asm)
- [Agner Fog's Optimization Manuals](https://www.agner.org/optimize/)

---

**Remember**: SIMD is powerful but complex. Always validate correctness and measure performance gains.
