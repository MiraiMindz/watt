package client

import "unsafe"

// bytesToStringUnsafe converts a byte slice to a string without allocation.
// WARNING: The returned string references the original byte slice.
// Do NOT modify the byte slice after calling this function.
// Only use when the byte slice lifetime is guaranteed to outlive the string.
//
// Allocation behavior: 0 allocs/op
func bytesToStringUnsafe(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// stringToBytesUnsafe converts a string to a byte slice without allocation.
// WARNING: The returned slice references the original string's backing array.
// Do NOT modify the returned slice.
//
// Allocation behavior: 0 allocs/op
func stringToBytesUnsafe(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
