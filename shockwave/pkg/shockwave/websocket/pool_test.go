package websocket

import (
	"testing"
)

func TestBufferPool(t *testing.T) {
	pool := &BufferPool{}

	tests := []struct {
		name string
		size int
		want int
	}{
		{"256B", 256, 256},
		{"1KB", 1024, 1024},
		{"4KB", 4096, 4096},
		{"16KB", 16384, 16384},
		{"small", 100, 100},
		{"medium", 500, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get buffer
			buf := pool.Get(tt.size)
			if len(buf) != tt.size {
				t.Errorf("Get(%d) returned buffer of length %d", tt.size, len(buf))
			}

			// Check capacity
			if cap(buf) < tt.size {
				t.Errorf("Get(%d) returned buffer with capacity %d", tt.size, cap(buf))
			}

			// Put buffer back
			pool.Put(buf)
		})
	}
}

func TestBufferPoolConcurrent(t *testing.T) {
	pool := &BufferPool{}
	done := make(chan bool)

	// Run 10 goroutines concurrently
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				buf := pool.Get(1024)
				if len(buf) != 1024 {
					t.Error("concurrent Get failed")
				}
				pool.Put(buf)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkBufferPoolGet(b *testing.B) {
	pool := &BufferPool{}
	sizes := []int{256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := pool.Get(size)
				pool.Put(buf)
			}
		})
	}
}

func BenchmarkBufferPoolGetPut(b *testing.B) {
	pool := &BufferPool{}

	b.Run("1KB", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := pool.Get(1024)
			// Simulate some work
			_ = buf[0]
			pool.Put(buf)
		}
	})
}

func BenchmarkBufferPoolVsAlloc(b *testing.B) {
	pool := &BufferPool{}

	b.Run("Pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := pool.Get(1024)
			pool.Put(buf)
		}
	})

	b.Run("Alloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = make([]byte, 1024)
		}
	})
}
