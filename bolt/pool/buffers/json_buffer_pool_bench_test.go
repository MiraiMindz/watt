package buffers

import (
	"bytes"
	"testing"

	json "github.com/goccy/go-json"
)

// BenchmarkJSONBufferPool_AcquireRelease benchmarks buffer pool acquire/release overhead
func BenchmarkJSONBufferPool_AcquireRelease(b *testing.B) {
	b.Run("SmallBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := AcquireSmallJSONBuffer()
			ReleaseJSONBuffer(buf)
		}
	})

	b.Run("MediumBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := AcquireMediumJSONBuffer()
			ReleaseJSONBuffer(buf)
		}
	})

	b.Run("LargeBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := AcquireLargeJSONBuffer()
			ReleaseJSONBuffer(buf)
		}
	})
}

// BenchmarkJSONBufferPool_SmallJSON benchmarks small JSON encoding
func BenchmarkJSONBufferPool_SmallJSON(b *testing.B) {
	type SmallPayload struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	payload := SmallPayload{
		Status:  "ok",
		Message: "Success",
	}

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := AcquireSmallJSONBuffer()
			encoder := json.NewEncoder(buf)
			_ = encoder.Encode(payload)
			ReleaseJSONBuffer(buf)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(make([]byte, 0, 512))
			encoder := json.NewEncoder(buf)
			_ = encoder.Encode(payload)
		}
	})

	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(payload)
		}
	})
}

// BenchmarkJSONBufferPool_MediumJSON benchmarks medium JSON encoding
func BenchmarkJSONBufferPool_MediumJSON(b *testing.B) {
	type User struct {
		ID       int      `json:"id"`
		Name     string   `json:"name"`
		Email    string   `json:"email"`
		Age      int      `json:"age"`
		City     string   `json:"city"`
		Country  string   `json:"country"`
		Tags     []string `json:"tags"`
		Active   bool     `json:"active"`
		Balance  float64  `json:"balance"`
		Verified bool     `json:"verified"`
	}

	user := User{
		ID:       12345,
		Name:     "John Doe",
		Email:    "john.doe@example.com",
		Age:      30,
		City:     "New York",
		Country:  "USA",
		Tags:     []string{"developer", "golang", "performance"},
		Active:   true,
		Balance:  1234.56,
		Verified: true,
	}

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := AcquireMediumJSONBuffer()
			encoder := json.NewEncoder(buf)
			_ = encoder.Encode(user)
			ReleaseJSONBuffer(buf)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(make([]byte, 0, 4096))
			encoder := json.NewEncoder(buf)
			_ = encoder.Encode(user)
		}
	})

	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(user)
		}
	})
}

// BenchmarkJSONBufferPool_LargeJSON benchmarks large JSON encoding
func BenchmarkJSONBufferPool_LargeJSON(b *testing.B) {
	type Item struct {
		ID          int     `json:"id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		InStock     bool    `json:"in_stock"`
	}

	// Create large array of items (~10KB)
	items := make([]Item, 100)
	for i := 0; i < 100; i++ {
		items[i] = Item{
			ID:          i + 1,
			Name:        "Product " + string(rune('A'+i%26)),
			Description: "This is a detailed description for product item number " + string(rune('0'+i%10)),
			Price:       99.99 + float64(i),
			InStock:     i%2 == 0,
		}
	}

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := AcquireLargeJSONBuffer()
			encoder := json.NewEncoder(buf)
			_ = encoder.Encode(items)
			ReleaseJSONBuffer(buf)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(make([]byte, 0, 16384))
			encoder := json.NewEncoder(buf)
			_ = encoder.Encode(items)
		}
	})

	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(items)
		}
	})
}

// BenchmarkJSONBufferPool_Concurrency benchmarks concurrent buffer pool usage
func BenchmarkJSONBufferPool_Concurrency(b *testing.B) {
	type Payload struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	payload := Payload{ID: 123, Name: "Test"}

	b.Run("Concurrent_4", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(4)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := AcquireMediumJSONBuffer()
				encoder := json.NewEncoder(buf)
				_ = encoder.Encode(payload)
				ReleaseJSONBuffer(buf)
			}
		})
	})

	b.Run("Concurrent_8", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(8)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := AcquireMediumJSONBuffer()
				encoder := json.NewEncoder(buf)
				_ = encoder.Encode(payload)
				ReleaseJSONBuffer(buf)
			}
		})
	})

	b.Run("Concurrent_16", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(16)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := AcquireMediumJSONBuffer()
				encoder := json.NewEncoder(buf)
				_ = encoder.Encode(payload)
				ReleaseJSONBuffer(buf)
			}
		})
	})
}
