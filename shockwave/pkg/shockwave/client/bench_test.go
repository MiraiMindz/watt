package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valyala/fasthttp"
)

// Benchmark optimized client - Simple GET
func BenchmarkClientGET(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient()
	client.Warmup(10) // Pre-warm pools
	defer client.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body())
		resp.Close()
	}
}

// Benchmark original client - Simple GET (for comparison)
func BenchmarkOriginalClientGET(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.ReadAll(resp.Body())
		resp.Close()
	}
}

// Benchmark net/http - Simple GET (for comparison)
func BenchmarkNetHTTP_GET(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := &http.Client{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
}

// Benchmark fasthttp - Simple GET (for comparison)
func BenchmarkFasthttp_GET(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := &fasthttp.Client{}

	b.ReportAllocs()
	b.ResetTimer()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	for i := 0; i < b.N; i++ {
		req.SetRequestURI(server.URL)
		req.Header.SetMethod("GET")

		if err := client.Do(req, resp); err != nil {
			b.Fatal(err)
		}

		resp.Reset()
	}
}

// Benchmark optimized client - Concurrent GET requests
func BenchmarkClientConcurrent(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient()
	client.Warmup(100)
	defer client.Close()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body())
			resp.Close()
		}
	})
}

// Benchmark optimized client - With custom headers
func BenchmarkClientWithHeaders(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient()
	client.Warmup(10)
	defer client.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := GetClientRequest()

		if err := client.parseURL(req, server.URL); err != nil {
			b.Fatal(err)
		}

		req.SetMethod("GET")
		req.SetHeader("X-Custom-1", "value1")
		req.SetHeader("X-Custom-2", "value2")
		req.SetHeader("X-Custom-3", "value3")

		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body())
		resp.Close()
	}
}

// Benchmark optimized client - Connection reuse
func BenchmarkClientConnectionReuse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient()
	client.Warmup(10)
	defer client.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body())
		resp.Close()
	}
}

// Benchmark header inline storage
func BenchmarkHeaderInlineStorage(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h := GetHeaders()

		// Add headers (should be 0 allocs for ≤32 headers)
		h.Add([]byte("Content-Type"), []byte("application/json"))
		h.Add([]byte("Content-Length"), []byte("1234"))
		h.Add([]byte("Authorization"), []byte("Bearer token123"))
		h.Add([]byte("X-Custom"), []byte("value"))

		// Get header (0 allocs)
		_ = h.Get([]byte("Content-Type"))

		PutHeaders(h)
	}
}

// Benchmark request building
func BenchmarkRequestBuilding(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := GetClientRequest()

		req.SetMethod("GET")
		req.SetURL("http", "example.com", "80", "/api/users", "page=1")
		req.SetHeader("Content-Type", "application/json")
		req.SetHeader("Authorization", "Bearer token")

		// Build request (should be 0 allocs with pre-allocated buffer)
		_ = req.BuildRequest()

		PutClientRequest(req)
	}
}

// Benchmark response parsing
func BenchmarkResponseParsing(b *testing.B) {
	responseBytes := []byte("HTTP/1.1 200 OK\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n" +
		"{\"status\":\"ok\"}")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp := GetClientResponse()

		br := GetBufioReader(newBytesReader(responseBytes))
		_ = resp.ParseResponse(br)

		PutBufioReader(br)
		PutClientResponse(resp)
	}
}

// Helper: bytesReader implements io.Reader for benchmarking
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data, pos: 0}
}

func (br *bytesReader) Read(p []byte) (int, error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}

	n := copy(p, br.data[br.pos:])
	br.pos += n

	if br.pos >= len(br.data) {
		return n, io.EOF
	}

	return n, nil
}

// Benchmark pool warmup
func BenchmarkPoolWarmup(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		WarmupPools(10)
	}
}

// Benchmark method ID lookup
func BenchmarkMethodIDLookup(b *testing.B) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		method := methods[i%len(methods)]
		_ = methodToID(method)
	}
}

// Benchmark byte equality (case-insensitive)
func BenchmarkBytesEqualCaseInsensitive(b *testing.B) {
	a := []byte("Content-Type")
	b_val := []byte("content-type")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = bytesEqualCaseInsensitive(a, b_val)
	}
}

// Benchmark parseInt fast path
func BenchmarkParseIntFast(b *testing.B) {
	statusCodes := [][]byte{
		[]byte("200"),
		[]byte("404"),
		[]byte("500"),
		[]byte("301"),
		[]byte("204"),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		code := statusCodes[i%len(statusCodes)]
		_, _ = parseIntFast(code)
	}
}
