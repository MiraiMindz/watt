package client

import (
	"bytes"
	"io"
	"testing"
)

// Benchmark OptimizedReader vs bufio.Reader
func BenchmarkOptimizedReaderReadLine(b *testing.B) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r := GetOptimizedReader(bytes.NewReader(data))

		// Read lines
		_, _ = r.ReadLine()
		_, _ = r.ReadLine()
		_, _ = r.ReadLine()

		PutOptimizedReader(r)
	}
}

// Benchmark URL cache hit rate
func BenchmarkURLCacheHit(b *testing.B) {
	cache := NewURLCache(100)
	testURL := "http://example.com:8080/api/users?page=1"

	// Pre-populate cache
	cache.ParseURL(testURL)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _, _, _, _ = cache.ParseURL(testURL)
	}
}

// Benchmark URL cache miss
func BenchmarkURLCacheMiss(b *testing.B) {
	cache := NewURLCache(100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Different URL each time (cache miss)
		url := "http://example.com/api/users?page=" + string(rune(i%10))
		_, _, _, _, _, _ = cache.ParseURL(url)
	}
}

// Benchmark InlineStringBuilder
func BenchmarkInlineStringBuilder(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var sb InlineStringBuilder
		sb.WriteString("example.com")
		sb.WriteByte(':')
		sb.WriteString("8080")
		_ = sb.String()
	}
}

// Benchmark BuildHostPort helper
func BenchmarkBuildHostPort(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = BuildHostPort("example.com", "8080")
	}
}

// Comparison: Standard string concatenation
func BenchmarkStandardConcat(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = "example.com" + ":" + "8080"
	}
}

// Benchmark complete optimized response parsing
func BenchmarkOptimizedResponseParsing(b *testing.B) {
	responseBytes := []byte("HTTP/1.1 200 OK\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n" +
		"{\"status\":\"ok\"}")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp := GetClientResponse()
		or := GetOptimizedReader(bytes.NewReader(responseBytes))

		_ = resp.ParseResponseOptimized(or)

		PutOptimizedReader(or)
		PutClientResponse(resp)
	}
}

// Benchmark pooling effectiveness
func BenchmarkPooling(b *testing.B) {
	b.Run("Request", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req := GetClientRequest()
			PutClientRequest(req)
		}
	})

	b.Run("Response", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp := GetClientResponse()
			PutClientResponse(resp)
		}
	})

	b.Run("Headers", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			h := GetHeaders()
			PutHeaders(h)
		}
	})

	b.Run("OptimizedReader", func(b *testing.B) {
		r := bytes.NewReader([]byte("test"))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			or := GetOptimizedReader(r)
			PutOptimizedReader(or)
		}
	})
}

// Benchmark zero-copy vs copy
func BenchmarkZeroCopyVsCopy(b *testing.B) {
	data := []byte("Content-Type: application/json")

	b.Run("ZeroCopy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Zero-copy: just slice reference
			_ = data[0:12]
		}
	})

	b.Run("Copy", func(b *testing.B) {
		b.ReportAllocs()
		buf := make([]byte, 12)
		for i := 0; i < b.N; i++ {
			// Copy to new buffer
			copy(buf, data[0:12])
		}
	})

	b.Run("String", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// String conversion (allocates)
			_ = string(data[0:12])
		}
	})
}

// Benchmark complete request lifecycle with all optimizations
func BenchmarkCompleteRequestLifecycle(b *testing.B) {
	server := newMockServer(b)
	defer server.Close()

	client := NewClient()
	client.Warmup(10)
	defer client.Close()

	// Warm up URL cache
	client.Get(server.URL())

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL())
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body())
		resp.Close()
	}
}

// Helper: Create mock server
func newMockServer(b *testing.B) *mockServer {
	return &mockServer{}
}

type mockServer struct{}

func (m *mockServer) Close() {}
func (m *mockServer) URL() string {
	return "http://127.0.0.1:8080/test"
}
