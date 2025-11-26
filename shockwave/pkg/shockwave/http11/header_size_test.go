package http11

import (
	"testing"
	"unsafe"
)

// TestHeaderSize verifies the size of the Header struct
func TestHeaderSize(t *testing.T) {
	var h Header
	size := unsafe.Sizeof(h)

	// Expected size calculation (after Phase 3 optimization):
	// names:     32 * 64 bytes = 2048 bytes
	// values:    32 * 128 bytes = 4096 bytes (reduced from 8192)
	// nameLens:  32 * 1 byte = 32 bytes
	// valueLens: 32 * 1 byte = 32 bytes
	// count:     1 byte (but padded to 8 bytes for alignment)
	// overflow:  16 bytes (map pointer: 8 bytes + padding)
	// Total:     ~6224 bytes (reduced from ~10320 bytes, saving 4KB)

	t.Logf("Header struct size: %d bytes (%.2f KB)", size, float64(size)/1024)

	// Verify it's reasonable (should be around 6KB after Phase 3 optimization)
	if size < 6000 || size > 7000 {
		t.Errorf("Header size %d is unexpected, want ~6224 bytes", size)
	}

	// Verify inline arrays are part of the struct (not pointers)
	// This ensures stack allocation
	if unsafe.Sizeof(h.names) != 32*64 {
		t.Errorf("names array size = %d, want %d", unsafe.Sizeof(h.names), 32*64)
	}

	if unsafe.Sizeof(h.values) != 32*128 {
		t.Errorf("values array size = %d, want %d", unsafe.Sizeof(h.values), 32*128)
	}
}
