package memory

import (
	"runtime"
	"testing"
)

// BenchmarkStandardPool benchmarks standard sync.Pool allocation
func BenchmarkStandardPool_HTTPRequest(b *testing.B) {
	// Simulate standard HTTP request allocation pattern
	type StandardRequest struct {
		Method string
		Path   string
		Proto  string

		HeaderKeys   []string
		HeaderValues []string

		Body []byte

		ContentLength int64
		Close         bool
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &StandardRequest{
			Method: "GET",
			Path:   "/api/v1/users/12345?expand=profile",
			Proto:  "HTTP/1.1",
		}

		// Simulate headers
		req.HeaderKeys = make([]string, 0, 8)
		req.HeaderValues = make([]string, 0, 8)
		req.HeaderKeys = append(req.HeaderKeys, "Host", "Content-Type", "User-Agent", "Accept")
		req.HeaderValues = append(req.HeaderValues, "example.com", "application/json", "Go-http-client/1.1", "*/*")

		// Simulate small body
		req.Body = make([]byte, 512)

		// Use the request
		_ = req.Method
		_ = req.Path
		_ = len(req.Body)
	}
}

// BenchmarkGreenTeaGC_HTTPRequest benchmarks Green Tea GC allocation
func BenchmarkGreenTeaGC_HTTPRequest(b *testing.B) {
	pool := NewGreenTeaRequestPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := pool.GetRequest()

		// Set method, path, proto
		req.SetMethod([]byte("GET"))
		req.SetPath([]byte("/api/v1/users/12345?expand=profile"))
		req.SetProto([]byte("HTTP/1.1"))

		// Add headers
		req.AddHeader([]byte("Host"), []byte("example.com"))
		req.AddHeader([]byte("Content-Type"), []byte("application/json"))
		req.AddHeader([]byte("User-Agent"), []byte("Go-http-client/1.1"))
		req.AddHeader([]byte("Accept"), []byte("*/*"))

		// Set body
		body := make([]byte, 512)
		req.SetBody(body)

		// Use the request
		_ = req.Method
		_ = req.Path
		_ = len(req.Body)

		// Return to pool
		pool.PutRequest(req)
	}
}

// BenchmarkStandardPool_LargeRequest benchmarks standard allocation with large body
func BenchmarkStandardPool_LargeRequest(b *testing.B) {
	type StandardRequest struct {
		Method string
		Path   string
		Body   []byte
	}

	bodySize := 64 * 1024 // 64KB

	b.SetBytes(int64(bodySize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &StandardRequest{
			Method: "POST",
			Path:   "/api/v1/upload",
			Body:   make([]byte, bodySize),
		}

		_ = req
	}
}

// BenchmarkGreenTeaGC_LargeRequest benchmarks Green Tea GC with large body
func BenchmarkGreenTeaGC_LargeRequest(b *testing.B) {
	pool := NewGreenTeaRequestPool()
	bodySize := 64 * 1024 // 64KB

	b.SetBytes(int64(bodySize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := pool.GetRequest()
		req.SetMethod([]byte("POST"))
		req.SetPath([]byte("/api/v1/upload"))

		body := make([]byte, bodySize)
		req.SetBody(body)

		_ = req

		pool.PutRequest(req)
	}
}

// BenchmarkStandardPool_ManyHeaders benchmarks standard allocation with many headers
func BenchmarkStandardPool_ManyHeaders(b *testing.B) {
	type StandardRequest struct {
		HeaderKeys   []string
		HeaderValues []string
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &StandardRequest{
			HeaderKeys:   make([]string, 0, 32),
			HeaderValues: make([]string, 0, 32),
		}

		// Add 32 headers
		for j := 0; j < 32; j++ {
			req.HeaderKeys = append(req.HeaderKeys, "X-Custom-Header-"+string(rune('A'+j)))
			req.HeaderValues = append(req.HeaderValues, "value-"+string(rune('A'+j)))
		}

		_ = req
	}
}

// BenchmarkGreenTeaGC_ManyHeaders benchmarks Green Tea GC with many headers
func BenchmarkGreenTeaGC_ManyHeaders(b *testing.B) {
	pool := NewGreenTeaRequestPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := pool.GetRequest()

		// Add 32 headers
		for j := 0; j < 32; j++ {
			key := []byte("X-Custom-Header-A")
			key[len(key)-1] = byte('A' + j)
			value := []byte("value-A")
			value[len(value)-1] = byte('A' + j)
			req.AddHeader(key, value)
		}

		_ = req

		pool.PutRequest(req)
	}
}

// BenchmarkGCPressure_Standard measures GC pressure with standard allocation
func BenchmarkGCPressure_Standard(b *testing.B) {
	var ms1, ms2 runtime.MemStats

	// Measure GC stats before
	runtime.GC()
	runtime.ReadMemStats(&ms1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Allocate request
		data := make([]byte, 1024)
		_ = data

		// Allocate headers
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["User-Agent"] = "Test"
		_ = headers
	}

	b.StopTimer()

	// Measure GC stats after
	runtime.GC()
	runtime.ReadMemStats(&ms2)

	// Calculate GC overhead
	gcPauses := ms2.PauseTotalNs - ms1.PauseTotalNs
	numGC := ms2.NumGC - ms1.NumGC

	b.ReportMetric(float64(gcPauses)/float64(b.N), "ns/op-gc")
	b.ReportMetric(float64(numGC), "GCs")
}

// BenchmarkGCPressure_GreenTea measures GC pressure with Green Tea GC
func BenchmarkGCPressure_GreenTea(b *testing.B) {
	var ms1, ms2 runtime.MemStats
	pool := NewGreenTeaRequestPool()

	// Measure GC stats before
	runtime.GC()
	runtime.ReadMemStats(&ms1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := pool.GetRequest()
		req.SetMethod([]byte("GET"))
		req.SetPath([]byte("/test"))
		req.AddHeader([]byte("Content-Type"), []byte("application/json"))
		req.AddHeader([]byte("User-Agent"), []byte("Test"))

		pool.PutRequest(req)
	}

	b.StopTimer()

	// Measure GC stats after
	runtime.GC()
	runtime.ReadMemStats(&ms2)

	// Calculate GC overhead
	gcPauses := ms2.PauseTotalNs - ms1.PauseTotalNs
	numGC := ms2.NumGC - ms1.NumGC

	b.ReportMetric(float64(gcPauses)/float64(b.N), "ns/op-gc")
	b.ReportMetric(float64(numGC), "GCs")
}

// BenchmarkAllocationRate_Standard measures allocation rate for standard mode
func BenchmarkAllocationRate_Standard(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		method := "GET"
		path := "/api/v1/users"
		headers := make(map[string]string, 4)
		headers["Host"] = "example.com"
		body := make([]byte, 1024)

		_ = method
		_ = path
		_ = headers
		_ = body
	}
}

// BenchmarkAllocationRate_GreenTea measures allocation rate for Green Tea mode
func BenchmarkAllocationRate_GreenTea(b *testing.B) {
	pool := NewGreenTeaRequestPool()

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := pool.GetRequest()
		req.SetMethod([]byte("GET"))
		req.SetPath([]byte("/api/v1/users"))
		req.AddHeader([]byte("Host"), []byte("example.com"))
		req.SetBody(make([]byte, 1024))

		pool.PutRequest(req)
	}
}

// BenchmarkCacheLocality_Standard measures cache performance with standard allocation
func BenchmarkCacheLocality_Standard(b *testing.B) {
	type Request struct {
		Method string
		Path   string
		Body   []byte
	}

	requests := make([]*Request, 100)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Allocate 100 requests (scattered in memory)
		for j := 0; j < 100; j++ {
			requests[j] = &Request{
				Method: "GET",
				Path:   "/test",
				Body:   make([]byte, 128),
			}
		}

		// Access all requests (measure cache misses)
		sum := 0
		for j := 0; j < 100; j++ {
			sum += len(requests[j].Method)
			sum += len(requests[j].Path)
			sum += len(requests[j].Body)
		}
		_ = sum
	}
}

// BenchmarkCacheLocality_GreenTea measures cache performance with Green Tea GC
func BenchmarkCacheLocality_GreenTea(b *testing.B) {
	pool := NewGreenTeaRequestPool()
	requests := make([]*GreenTeaHTTPRequest, 100)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Allocate 100 requests from same slab (adjacent in memory)
		for j := 0; j < 100; j++ {
			req := pool.GetRequest()
			req.SetMethod([]byte("GET"))
			req.SetPath([]byte("/test"))
			req.SetBody(make([]byte, 128))
			requests[j] = req
		}

		// Access all requests (should have better cache locality)
		sum := 0
		for j := 0; j < 100; j++ {
			sum += len(requests[j].Method)
			sum += len(requests[j].Path)
			sum += len(requests[j].Body)
		}
		_ = sum

		// Return to pool
		for j := 0; j < 100; j++ {
			pool.PutRequest(requests[j])
		}
	}
}

// BenchmarkThroughput_Standard measures throughput with standard allocation
func BenchmarkThroughput_Standard(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			method := "GET"
			path := "/api/v1/test"
			headers := make(map[string]string, 4)
			headers["Host"] = "example.com"
			body := make([]byte, 512)

			_ = method
			_ = path
			_ = headers
			_ = body
		}
	})
}

// BenchmarkThroughput_GreenTea measures throughput with Green Tea GC
func BenchmarkThroughput_GreenTea(b *testing.B) {
	pool := NewGreenTeaRequestPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := pool.GetRequest()
			req.SetMethod([]byte("GET"))
			req.SetPath([]byte("/api/v1/test"))
			req.AddHeader([]byte("Host"), []byte("example.com"))
			req.SetBody(make([]byte, 512))

			pool.PutRequest(req)
		}
	})
}
