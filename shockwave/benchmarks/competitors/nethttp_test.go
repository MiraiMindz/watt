package competitors

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test data generators
func generateHeaders(count int) http.Header {
	headers := make(http.Header)
	for i := 0; i < count; i++ {
		headers.Set(fmt.Sprintf("X-Custom-Header-%d", i), fmt.Sprintf("value-%d", i))
	}
	return headers
}

func generateBody(size int) []byte {
	body := make([]byte, size)
	for i := range body {
		body[i] = byte('A' + (i % 26))
	}
	return body
}

// BenchmarkNetHTTPSimpleGET benchmarks a simple GET request
func BenchmarkNetHTTPSimpleGET(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			DisableCompression:  true,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(2) // "OK" response

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkNetHTTPPOST1KB benchmarks POST request with 1KB body
func BenchmarkNetHTTPPOST1KB(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			DisableCompression:  true,
		},
	}

	body := generateBody(1024) // 1KB
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))

	for i := 0; i < b.N; i++ {
		resp, err := client.Post(server.URL, "application/octet-stream", bytes.NewReader(body))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkNetHTTPKeepAlive benchmarks keep-alive connection reuse
func BenchmarkNetHTTPKeepAlive(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   100,
			DisableCompression:    true,
			DisableKeepAlives:     false,
			IdleConnTimeout:       90 * time.Second,
			MaxConnsPerHost:       100,
		},
	}

	// Warm up connection
	resp, err := client.Get(server.URL)
	if err != nil {
		b.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(2) // "OK" response

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkNetHTTPConcurrent benchmarks concurrent requests (100 parallel)
func BenchmarkNetHTTPConcurrent(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   100,
			MaxConnsPerHost:       100,
			DisableCompression:    true,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(2) // "OK" response

	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Error(err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkNetHTTPLargeResponse benchmarks 1MB response
func BenchmarkNetHTTPLargeResponse(b *testing.B) {
	largeData := generateBody(1024 * 1024) // 1MB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			DisableCompression:  true,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(largeData)))

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkNetHTTPHeaderHeavy benchmarks request with 50 headers
func BenchmarkNetHTTPHeaderHeavy(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back header count
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Headers: %d", len(r.Header))))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			DisableCompression:  true,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			b.Fatal(err)
		}

		// Add 50 custom headers
		for j := 0; j < 50; j++ {
			req.Header.Set(fmt.Sprintf("X-Custom-Header-%d", j), fmt.Sprintf("value-%d", j))
		}

		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkNetHTTPRequestParsing benchmarks raw HTTP request parsing
func BenchmarkNetHTTPRequestParsing(b *testing.B) {
	reqStr := "GET /path HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: benchmark\r\n" +
		"Accept: */*\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n"

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(reqStr)))

	for i := 0; i < b.N; i++ {
		req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(reqStr)))
		if err != nil {
			b.Fatal(err)
		}
		_ = req
	}
}

// BenchmarkNetHTTPResponseWriting benchmarks response writing
func BenchmarkNetHTTPResponseWriting(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		rec.Write([]byte("Hello, World!"))
		_ = rec.Result()
	}
}