---
name: unsafe-go
description: Expert in safe usage of Go's unsafe package for performance-critical operations. Use when implementing zero-copy operations, custom memory layouts, or interfacing with system APIs. Ensures safety invariants are maintained.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Unsafe Go Skill

## Purpose
This skill provides expertise in using Go's `unsafe` package correctly and safely for performance-critical operations while maintaining program correctness.

## When to Use This Skill
- Implementing zero-copy string/byte conversions
- Creating custom memory allocators
- Optimizing struct layouts and memory access
- Interfacing with C code or system APIs
- Implementing performance primitives where safe alternatives are too slow

## Core Philosophy

**Unsafe is not "anything goes"** - It's a tool for writing code that the compiler can't verify as safe, but which YOU know is safe through careful reasoning and testing.

## The Rules of Unsafe

### 1. Unsafe Must Be Justified
Every use of `unsafe` must have:
- Clear performance benefit (>2x speedup or zero-allocation requirement)
- Comprehensive documentation explaining why it's safe
- Tests that would catch violations of safety invariants
- A comment explaining what could go wrong

### 2. Safety Invariants Must Be Documented
```go
// ToBytes converts a string to []byte without allocation.
//
// SAFETY INVARIANTS:
// 1. The returned slice MUST NOT be modified (undefined behavior)
// 2. The slice is only valid while the string is alive
// 3. The string's backing array must not be moved by GC (it won't be)
//
// This is safe because:
// - Strings are immutable in Go
// - We only expose a read-only view
// - The slice shares the string's backing storage
// - Go's GC won't move the string data
func ToBytes(s string) []byte {
    if s == "" {
        return nil
    }
    return unsafe.Slice(unsafe.StringData(s), len(s))
}
```

### 3. Provide Safe Wrappers When Possible
```go
// Internal unsafe implementation
func unsafeToBytes(s string) []byte {
    if s == "" {
        return nil
    }
    return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Safe public API that copies
func SafeToBytes(s string) []byte {
    return []byte(s)
}

// Unsafe public API with clear naming
func UnsafeToBytes(s string) []byte {
    return unsafeToBytes(s)
}
```

## Common Unsafe Patterns

### 1. String ↔ []byte Conversions

#### String to []byte (Zero-Copy)
```go
func StringToBytes(s string) []byte {
    if s == "" {
        return nil
    }
    // SAFETY: Caller must not modify returned slice
    return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Usage:
s := "hello"
b := StringToBytes(s)
// DO NOT: b[0] = 'H' // undefined behavior!
// DO: Read from b
```

#### []byte to String (Zero-Copy)
```go
func BytesToString(b []byte) string {
    if len(b) == 0 {
        return ""
    }
    // SAFETY: Caller must not modify the slice after conversion
    // The slice's backing array becomes part of the string
    return unsafe.String(&b[0], len(b))
}

// Usage:
b := []byte("hello")
s := BytesToString(b)
// DO NOT: b[0] = 'H' after this point!
// The string 's' now shares the backing array
```

### 2. Pointer Arithmetic

#### Safe Pointer Arithmetic
```go
type Block struct {
    data []byte
    pos  int
}

func (b *Block) AllocAligned(size, align int) unsafe.Pointer {
    // Round up to alignment
    mask := align - 1
    pos := (b.pos + mask) &^ mask

    if pos+size > len(b.data) {
        return nil
    }

    // SAFETY:
    // - pos is within bounds of b.data
    // - pos is aligned to 'align' bytes
    // - size bytes from pos are within b.data
    ptr := unsafe.Pointer(&b.data[pos])
    b.pos = pos + size
    return ptr
}
```

#### Struct Field Offsets
```go
type Header struct {
    Magic   uint32
    Version uint32
    Length  uint64
}

func (h *Header) LengthPtr() *uint64 {
    // SAFETY: Length is a field of Header, so this offset is stable
    const lengthOffset = unsafe.Offsetof(h.Length)
    ptr := unsafe.Pointer(h)
    return (*uint64)(unsafe.Add(ptr, lengthOffset))
}
```

### 3. Type Conversions

#### Slice Header Manipulation (Go 1.20+)
```go
// Modern way: Use unsafe.Slice and unsafe.SliceData
func resizeSlice(data []byte, newLen int) []byte {
    if newLen <= cap(data) {
        // SAFETY: newLen is within capacity
        return unsafe.Slice(unsafe.SliceData(data), newLen)
    }
    // Need to allocate
    return append(data[:0:0], data...)
}
```

#### Reinterpreting Bytes
```go
func BytesToUint64(b []byte) uint64 {
    if len(b) < 8 {
        panic("slice too short")
    }
    // SAFETY:
    // - We checked len(b) >= 8
    // - b[0] exists and has 8 bytes available
    // - Alignment might be wrong (use encoding/binary for portable code)
    return *(*uint64)(unsafe.Pointer(&b[0]))
}

// BETTER: Use encoding/binary for portability
import "encoding/binary"

func BytesToUint64Safe(b []byte) uint64 {
    return binary.LittleEndian.Uint64(b)
}
```

### 4. Custom Memory Layouts

#### Struct of Arrays (SOA)
```go
type Particles struct {
    count int
    x     []float64  // All X coordinates
    y     []float64  // All Y coordinates
    z     []float64  // All Z coordinates
}

// Better cache locality for SIMD operations
func (p *Particles) UpdateX(scale float64) {
    for i := range p.x {
        p.x[i] *= scale
    }
}
```

#### Cache-Aligned Structures
```go
const cacheLineSize = 64

type CacheAligned struct {
    value uint64
    _     [cacheLineSize - 8]byte // Padding
}

// Prevent false sharing
type NoPadding struct {
    _     [cacheLineSize]byte
    value uint64
    _     [cacheLineSize - 8]byte
}
```

## Unsafe Gotchas and How to Avoid Them

### 1. GC and Pointer Validity

#### BAD: Pointer to Stack Variable
```go
func getPointer() unsafe.Pointer {
    x := 42
    return unsafe.Pointer(&x) // WRONG: x is on stack, will be invalid
}
```

#### GOOD: Keep Variables Alive
```go
func getPointer() (unsafe.Pointer, *int) {
    x := 42
    ptr := &x // x escapes to heap
    return unsafe.Pointer(ptr), ptr // Keep ptr alive
}
```

### 2. Alignment Requirements

#### Check Alignment
```go
func isAligned(ptr unsafe.Pointer, align uintptr) bool {
    return uintptr(ptr)%align == 0
}

func allocAligned(size, align int) unsafe.Pointer {
    // Over-allocate to ensure alignment
    buf := make([]byte, size+align-1)
    addr := uintptr(unsafe.Pointer(&buf[0]))
    offset := (align - int(addr%uintptr(align))) % align
    return unsafe.Pointer(&buf[offset])
}
```

### 3. Pointer Arithmetic Bounds

#### Always Check Bounds
```go
func readAt(base unsafe.Pointer, offset, limit int) byte {
    if offset < 0 || offset >= limit {
        panic("offset out of bounds")
    }
    // SAFETY: Checked bounds above
    return *(*byte)(unsafe.Add(base, offset))
}
```

### 4. Concurrent Access

#### Unsafe + Concurrency = Extra Care
```go
type LockFreeQueue struct {
    head unsafe.Pointer // *node
    tail unsafe.Pointer // *node
}

func (q *LockFreeQueue) Enqueue(value int) {
    node := &node{value: value}
    newPtr := unsafe.Pointer(node)

    for {
        tail := atomic.LoadPointer(&q.tail)
        // SAFETY: All pointer access uses atomic operations
        if atomic.CompareAndSwapPointer(&q.tail, tail, newPtr) {
            break
        }
    }
}
```

## Testing Unsafe Code

### 1. Race Detector
```bash
# ALWAYS test unsafe code with race detector
go test -race ./...
```

### 2. Memory Sanitizer (with CGO)
```bash
# Detects uninitialized reads
go test -msan ./...
```

### 3. Fuzzing
```go
func FuzzUnsafeConversion(f *testing.F) {
    f.Add("hello")
    f.Add("")
    f.Add("a")

    f.Fuzz(func(t *testing.T, s string) {
        // Test unsafe string conversion
        b := UnsafeStringToBytes(s)
        s2 := UnsafeBytesToString(b)

        if s != s2 {
            t.Errorf("roundtrip failed: %q != %q", s, s2)
        }
    })
}
```

### 4. Property-Based Testing
```go
func TestStringBytesRoundtrip(t *testing.T) {
    f := func(s string) bool {
        b := StringToBytes(s)
        s2 := BytesToString(b)
        return s == s2
    }

    if err := quick.Check(f, nil); err != nil {
        t.Error(err)
    }
}
```

## Documentation Template

Use this template for all unsafe code:

```go
// FunctionName does X by using unsafe operations.
//
// SAFETY INVARIANTS:
// 1. [First invariant that must hold]
// 2. [Second invariant]
// 3. [etc.]
//
// SAFETY REASONING:
// This is safe because:
// - [Reason 1]
// - [Reason 2]
//
// PERFORMANCE:
// This is X% faster than [safe alternative] because [reason].
// Benchmark: BenchmarkName
//
// CAUTIONS:
// - [What can go wrong if misused]
// - [What caller must ensure]
func FunctionName(params) result {
    // Implementation
}
```

## Code Review Checklist for Unsafe Code

Before approving unsafe code:
- [ ] Performance benefit is documented and measured (>2x or zero-alloc)
- [ ] Safety invariants are explicitly documented
- [ ] Safety reasoning is clear and convincing
- [ ] All possible failure modes are considered
- [ ] Tests cover edge cases and invariant violations
- [ ] Race detector passes (`go test -race`)
- [ ] Fuzz tests exist for parsing/conversion operations
- [ ] Safe wrapper API exists or unsafe is clearly named
- [ ] Code has been reviewed by unsafe-code-reviewer agent
- [ ] Memory model implications are understood
- [ ] Platform-specific behavior is documented (if any)

## Common Patterns from Go Standard Library

### strings.Builder.String()
```go
// From strings/builder.go
func (b *Builder) String() string {
    return unsafe.String(&b.buf[0], len(b.buf))
}
```

### sync.Pool implementation
```go
// Uses unsafe to avoid interface allocation
type poolLocal struct {
    private interface{}
    shared  []interface{}
}

func (p *Pool) pin() *poolLocal {
    // Uses unsafe to index into per-P array
}
```

## When NOT to Use Unsafe

Don't use `unsafe` if:
1. Safe code is fast enough (profile first!)
2. The performance gain is <20% (not worth the complexity)
3. You can't prove the invariants hold
4. The code will be hard to maintain
5. Portable alternatives exist (e.g., encoding/binary)
6. You're just avoiding type conversions for convenience

## Migration Path from Unsafe

If unsafe code needs to be removed:
1. Write safe implementation alongside unsafe one
2. Add benchmarks comparing both
3. Use build tags to switch between implementations
4. Gradually migrate if performance is acceptable

```go
//go:build unsafe
// file_unsafe.go

func convert(s string) []byte {
    return unsafeImplementation(s)
}
```

```go
//go:build !unsafe
// file_safe.go

func convert(s string) []byte {
    return []byte(s)
}
```

## Resources

- [Go Memory Model](https://go.dev/ref/mem)
- [unsafe Package Docs](https://pkg.go.dev/unsafe)
- [Go101 Unsafe Article](https://go101.org/article/unsafe.html)
- [Russ Cox on unsafe.Pointer](https://research.swtch.com/gorace)

---

**Remember**: With great power comes great responsibility. Unsafe is a scalpel, not a hammer.
