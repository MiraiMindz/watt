package shockwave

import (
	"bytes"
	"testing"
)

// TestSecureBufferPool_Zeroed verifies buffers are zeroed
func TestSecureBufferPool_Zeroed(t *testing.T) {
	pool := NewSecureBufferPool(false)

	// Get buffer and fill with data
	buf := pool.Get(4096)
	for i := range buf {
		buf[i] = 0xFF
	}

	// Return to pool
	pool.Put(buf)

	// Get another buffer - should be zeroed
	buf2 := pool.Get(4096)
	defer pool.Put(buf2)

	// Verify all zeros
	for i, b := range buf2 {
		if b != 0 {
			t.Errorf("Buffer not zeroed at index %d: expected 0, got %d", i, b)
			break
		}
	}
}

// TestSecureBufferPool_RandomFill verifies random fill works
func TestSecureBufferPool_RandomFill(t *testing.T) {
	pool := NewSecureBufferPool(true)

	// Get two buffers
	buf1 := pool.Get(4096)
	buf2 := pool.Get(4096)
	defer pool.Put(buf1)
	defer pool.Put(buf2)

	// They should be different (random fill)
	if bytes.Equal(buf1, buf2) {
		t.Error("Random-filled buffers should not be identical")
	}

	// Check that buffer is not all zeros
	allZeros := true
	for _, b := range buf1 {
		if b != 0 {
			allZeros = false
			break
		}
	}

	if allZeros {
		t.Error("Random-filled buffer should not be all zeros")
	}
}

// TestSecureBufferPool_DefenseInDepth verifies Put zeros buffer
func TestSecureBufferPool_DefenseInDepth(t *testing.T) {
	pool := NewSecureBufferPool(false)

	// Get buffer and fill with sensitive data
	buf := pool.Get(4096)
	sensitiveData := []byte("super secret password 12345")
	copy(buf, sensitiveData)

	// Verify data is there
	if !bytes.Equal(buf[:len(sensitiveData)], sensitiveData) {
		t.Error("Data not copied correctly")
	}

	// Return to pool (should zero)
	pool.Put(buf)

	// Buffer should now be zeroed (even though we returned it)
	for i, b := range buf[:len(sensitiveData)] {
		if b != 0 {
			t.Errorf("Buffer not zeroed after Put at index %d: got %d", i, b)
			break
		}
	}
}

// TestSecureBufferPool_SizeSelection verifies correct size selection
func TestSecureBufferPool_SizeSelection(t *testing.T) {
	pool := NewSecureBufferPool(false)

	tests := []struct {
		requestedSize int
		expectedSize  int
	}{
		{1024, BufferSize2KB},
		{BufferSize2KB, BufferSize2KB},
		{3 * 1024, BufferSize4KB},
		{BufferSize4KB, BufferSize4KB},
		{6 * 1024, BufferSize8KB},
		{BufferSize8KB, BufferSize8KB},
	}

	for _, tt := range tests {
		buf := pool.Get(tt.requestedSize)
		if cap(buf) != tt.expectedSize {
			t.Errorf("Size %d: expected capacity %d, got %d",
				tt.requestedSize, tt.expectedSize, cap(buf))
		}
		pool.Put(buf)
	}
}

// TestSecureBufferPool_GlobalFunctions verifies global API
func TestSecureBufferPool_GlobalFunctions(t *testing.T) {
	// Test zero-filled global pool
	buf := GetSecureBuffer(4096)
	for i := range buf {
		buf[i] = 0xFF
	}
	PutSecureBuffer(buf)

	buf2 := GetSecureBuffer(4096)
	defer PutSecureBuffer(buf2)

	// Should be zeroed
	for i, b := range buf2 {
		if b != 0 {
			t.Errorf("Global secure buffer not zeroed at index %d", i)
			break
		}
	}

	// Test random-filled global pool
	randomBuf := GetRandomBuffer(4096)
	defer PutRandomBuffer(randomBuf)

	// Should not be all zeros
	allZeros := true
	for _, b := range randomBuf {
		if b != 0 {
			allZeros = false
			break
		}
	}

	if allZeros {
		t.Error("Random buffer should not be all zeros")
	}
}

// TestSecureBufferPool_Metrics verifies metrics tracking
func TestSecureBufferPool_Metrics(t *testing.T) {
	pool := NewSecureBufferPool(false)

	const iterations = 100

	for i := 0; i < iterations; i++ {
		buf := pool.Get(4096)
		pool.Put(buf)
	}

	metrics := pool.GetMetrics()

	if metrics.Pool4KB.Gets != iterations {
		t.Errorf("Expected %d gets, got %d", iterations, metrics.Pool4KB.Gets)
	}

	if metrics.Pool4KB.Puts != iterations {
		t.Errorf("Expected %d puts, got %d", iterations, metrics.Pool4KB.Puts)
	}

	// Hit rate should be high after warmup
	if metrics.Pool4KB.HitRate < 90.0 {
		t.Errorf("Expected hit rate >90%%, got %.2f%%", metrics.Pool4KB.HitRate)
	}
}

// Benchmarks

// BenchmarkSecureBufferPool_Zeroed measures zeroed buffer performance
func BenchmarkSecureBufferPool_Zeroed(b *testing.B) {
	pool := NewSecureBufferPool(false)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.Get(BufferSize4KB)
		pool.Put(buf)
	}
}

// BenchmarkSecureBufferPool_Random measures random-filled buffer performance
func BenchmarkSecureBufferPool_Random(b *testing.B) {
	pool := NewSecureBufferPool(true)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.Get(BufferSize4KB)
		pool.Put(buf)
	}
}

// BenchmarkSecureVsRegular compares secure vs regular pool performance
func BenchmarkSecureVsRegular(b *testing.B) {
	securePool := NewSecureBufferPool(false)
	regularPool := NewBufferPool()

	b.Run("Regular", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := regularPool.Get(BufferSize4KB)
			regularPool.Put(buf)
		}
	})

	b.Run("Secure", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := securePool.Get(BufferSize4KB)
			securePool.Put(buf)
		}
	})

	b.Run("SecureRandom", func(b *testing.B) {
		randomPool := NewSecureBufferPool(true)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := randomPool.Get(BufferSize4KB)
			randomPool.Put(buf)
		}
	})
}

// BenchmarkSecureBufferPool_AllSizes measures performance across all sizes
func BenchmarkSecureBufferPool_AllSizes(b *testing.B) {
	pool := NewSecureBufferPool(false)

	sizes := []struct {
		name string
		size int
	}{
		{"2KB", BufferSize2KB},
		{"4KB", BufferSize4KB},
		{"8KB", BufferSize8KB},
		{"16KB", BufferSize16KB},
		{"32KB", BufferSize32KB},
		{"64KB", BufferSize64KB},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(s.size))

			for i := 0; i < b.N; i++ {
				buf := pool.Get(s.size)
				pool.Put(buf)
			}
		})
	}
}

// BenchmarkSecureBufferPool_PasswordUseCase simulates password processing
func BenchmarkSecureBufferPool_PasswordUseCase(b *testing.B) {
	password := []byte("SuperSecretPassword123!")

	b.Run("WithSecurePool", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := GetSecureBuffer(len(password))
			copy(buf, password)
			// Simulate hashing/processing
			_ = buf[0]
			PutSecureBuffer(buf)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, len(password))
			copy(buf, password)
			// Simulate hashing/processing
			_ = buf[0]
			// Zero manually
			for j := range buf {
				buf[j] = 0
			}
		}
	})
}

// BenchmarkSecureBufferPool_Parallel measures concurrent performance
func BenchmarkSecureBufferPool_Parallel(b *testing.B) {
	pool := NewSecureBufferPool(false)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(BufferSize4KB)
			pool.Put(buf)
		}
	})
}

// TestSecureBufferPool_NoLeaks verifies buffers are properly zeroed and don't leak
func TestSecureBufferPool_NoLeaks(t *testing.T) {
	pool := NewSecureBufferPool(false)

	// Fill buffer with sensitive data
	sensitiveData := []byte("TOP SECRET CLASSIFIED INFORMATION")

	buf := pool.Get(4096)
	copy(buf, sensitiveData)

	// Return to pool
	pool.Put(buf)

	// Get same buffer again (or another one)
	buf2 := pool.Get(4096)
	defer pool.Put(buf2)

	// Verify sensitive data is not present
	if bytes.Contains(buf2, sensitiveData) {
		t.Error("Sensitive data leaked in buffer! Data not properly zeroed.")
	}
}
