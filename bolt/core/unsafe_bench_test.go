package core

import (
	"testing"
)

// BenchmarkBytesToString_Standard benchmarks standard string conversion
func BenchmarkBytesToString_Standard(b *testing.B) {
	bytes := []byte("Hello, World! This is a test string for benchmarking.")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = string(bytes)
	}
}

// BenchmarkBytesToString_Unsafe benchmarks unsafe zero-copy conversion
func BenchmarkBytesToString_Unsafe(b *testing.B) {
	bytes := []byte("Hello, World! This is a test string for benchmarking.")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = bytesToString(bytes)
	}
}

// BenchmarkStringToBytes_Standard benchmarks standard []byte conversion
func BenchmarkStringToBytes_Standard(b *testing.B) {
	str := "Hello, World! This is a test string for benchmarking."

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = []byte(str)
	}
}

// BenchmarkStringToBytes_Unsafe benchmarks unsafe zero-copy conversion
func BenchmarkStringToBytes_Unsafe(b *testing.B) {
	str := "Hello, World! This is a test string for benchmarking."

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = stringToBytes(str)
	}
}

// BenchmarkParam_BeforeUnsafe simulates parameter access with allocations
func BenchmarkParam_BeforeUnsafe(b *testing.B) {
	ctx := &Context{}
	ctx.paramsBuf[0].keyBytes = []byte("id")
	ctx.paramsBuf[0].valueBytes = []byte("123456")
	ctx.paramsLen = 1

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate old implementation with allocations
		key := "id"
		keyBytes := []byte(key)  // Alloc 1

		for j := 0; j < ctx.paramsLen; j++ {
			if bytesEqual(ctx.paramsBuf[j].keyBytes, keyBytes) {
				_ = string(ctx.paramsBuf[j].valueBytes)  // Alloc 2
				break
			}
		}
	}
}

// BenchmarkParam_AfterUnsafe benchmarks parameter access with unsafe
func BenchmarkParam_AfterUnsafe(b *testing.B) {
	ctx := &Context{}
	ctx.paramsBuf[0].keyBytes = []byte("id")
	ctx.paramsBuf[0].valueBytes = []byte("123456")
	ctx.paramsLen = 1

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ctx.Param("id")
	}
}
