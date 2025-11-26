//go:build amd64 && !noasm
// +build amd64,!noasm

package websocket

import (
	"golang.org/x/sys/cpu"
)

var hasAVX2 = cpu.X86.HasAVX2

// maskBytesAVX2 applies XOR masking using AVX2 SIMD instructions.
// Implemented in mask_amd64.s
func maskBytesAVX2(data []byte, maskKey [4]byte)

// maskBytesFast is the optimized masking function that uses AVX2 if available.
func maskBytesFast(data []byte, maskKey [4]byte) {
	if hasAVX2 && len(data) >= 32 {
		maskBytesAVX2(data, maskKey)
	} else {
		maskBytesScalar(data, maskKey)
	}
}

// maskBytesScalar is the scalar (non-SIMD) implementation.
// This is the original implementation that processes 8 bytes at a time.
func maskBytesScalar(data []byte, maskKey [4]byte) {
	// Fast path: process 8 bytes at a time using uint64
	if len(data) >= 8 {
		// Expand mask key to 8 bytes
		mask64 := uint64(maskKey[0]) |
			uint64(maskKey[1])<<8 |
			uint64(maskKey[2])<<16 |
			uint64(maskKey[3])<<24 |
			uint64(maskKey[0])<<32 |
			uint64(maskKey[1])<<40 |
			uint64(maskKey[2])<<48 |
			uint64(maskKey[3])<<56

		// Process 8 bytes at a time
		i := 0
		for ; i+8 <= len(data); i += 8 {
			// XOR 8 bytes at once
			val := uint64(data[i]) |
				uint64(data[i+1])<<8 |
				uint64(data[i+2])<<16 |
				uint64(data[i+3])<<24 |
				uint64(data[i+4])<<32 |
				uint64(data[i+5])<<40 |
				uint64(data[i+6])<<48 |
				uint64(data[i+7])<<56
			val ^= mask64

			// Write back
			data[i] = byte(val)
			data[i+1] = byte(val >> 8)
			data[i+2] = byte(val >> 16)
			data[i+3] = byte(val >> 24)
			data[i+4] = byte(val >> 32)
			data[i+5] = byte(val >> 40)
			data[i+6] = byte(val >> 48)
			data[i+7] = byte(val >> 56)
		}

		// Process remaining bytes
		for ; i < len(data); i++ {
			data[i] ^= maskKey[i%4]
		}
	} else {
		// Slow path: process byte by byte for small data
		for i := 0; i < len(data); i++ {
			data[i] ^= maskKey[i%4]
		}
	}
}

// Replace the original maskBytes with the fast version
func init() {
	// Use the fast implementation as the default
	maskBytes = maskBytesFast
}
