package core

import (
	"unsafe"
)

// Zero-copy string/byte conversions using unsafe package.
//
// SAFETY WARNINGS:
// - These functions bypass Go's type safety and memory safety
// - Use ONLY when you understand the lifetime and mutability guarantees
// - Improper use can cause data corruption, race conditions, or crashes
//
// Performance benefit:
//   - Standard conversion: string([]byte) allocates memory + copies data
//   - Unsafe conversion: 0 allocations, 0 copies (just pointer cast)

// bytesToString converts a byte slice to a string with ZERO allocations.
//
// SAFETY REQUIREMENTS:
//   1. The returned string must be READ-ONLY (never modified)
//   2. The returned string must not outlive the source byte slice
//   3. The source byte slice must not be modified while string is in use
//
// Example SAFE usage:
//
//	pathBytes := []byte("/users/123")
//	path := bytesToString(pathBytes)  // Zero-copy conversion
//	fmt.Println(path)                  // Read-only use - SAFE
//	// pathBytes is still alive and unmodified
//
// Example UNSAFE usage:
//
//	pathBytes := []byte("/users/123")
//	path := bytesToString(pathBytes)
//	pathBytes[0] = 'X'                 // DANGER! Modifies string (undefined behavior)
//	return path                         // DANGER! pathBytes may be GC'd
//
// Performance: 0 ns/op, 0 B/op, 0 allocs/op (vs ~20ns, 1 alloc for string())
//
//go:inline
func bytesToString(b []byte) string {
	// Use unsafe.SliceData to get pointer to first element
	// Then construct string header with same pointer and length
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// stringToBytes converts a string to a byte slice with ZERO allocations.
//
// ⚠️ EXTREME DANGER WARNING ⚠️
//
// SAFETY REQUIREMENTS:
//   1. The returned []byte MUST NEVER BE MODIFIED (strings are immutable!)
//   2. The returned []byte must not outlive the source string
//   3. Use ONLY for read-only operations
//
// Example SAFE usage (READ-ONLY):
//
//	path := "/users/123"
//	pathBytes := stringToBytes(path)
//	if pathBytes[0] == '/' {           // Read-only comparison - SAFE
//	    // ...
//	}
//
// Example UNSAFE usage (MUTATION):
//
//	path := "/users/123"
//	pathBytes := stringToBytes(path)
//	pathBytes[0] = 'X'                 // DANGER! Modifies immutable string!
//	                                   // Causes undefined behavior, may crash
//
// WHY THIS IS DANGEROUS:
//   - Go strings are immutable and may be stored in read-only memory
//   - Writing to a string's backing array may cause segmentation fault
//   - Go compiler may optimize assuming string immutability
//   - Multiple strings may share the same backing array
//
// USE CASES:
//   - Passing string to function expecting []byte for reading
//   - Zero-copy comparison operations
//   - Temporary byte slice that won't be modified
//
// Performance: 0 ns/op, 0 B/op, 0 allocs/op (vs ~20ns, 1 alloc for []byte())
//
//go:inline
func stringToBytes(s string) []byte {
	// Use unsafe.StringData to get pointer to first byte
	// Then construct slice header with same pointer and length
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Safety notes for developers:
//
// 1. READ-ONLY GUARANTEE:
//    - bytesToString: Returned string must not be modified (Go enforces this)
//    - stringToBytes: Returned []byte must not be modified (YOU must enforce this!)
//
// 2. LIFETIME GUARANTEE:
//    - Source must outlive result
//    - Don't return converted values from functions unless source is also returned
//    - Prefer local variables with clear scope
//
// 3. WHEN TO USE:
//    - ✅ Router map key lookup (string needed, []byte available, read-only)
//    - ✅ Parameter comparison ([]byte needed, string available, read-only)
//    - ✅ Temporary conversions within same function scope
//    - ❌ Long-lived values that outlive source
//    - ❌ Concurrent access (race conditions with modifications)
//    - ❌ Returning from functions (unless source also returned)
//
// 4. TESTING:
//    - Run with -race detector to catch concurrent access issues
//    - Test edge cases (empty strings, nil slices)
//    - Fuzz test to find safety violations
