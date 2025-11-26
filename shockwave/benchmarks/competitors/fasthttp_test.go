package competitors

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// BenchmarkFastHTTPSimpleGET benchmarks a simple GET request
func BenchmarkFastHTTPSimpleGET(b *testing.B) {
	handler := func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString("OK")
	}

	server := &fasthttp.Server{
		Handler: handler,
	}
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go server.Serve(ln)

	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(2) // "OK" response

	var req fasthttp.Request
	var resp fasthttp.Response
	req.SetRequestURI("http://localhost/")

	for i := 0; i < b.N; i++ {
		if err := client.Do(&req, &resp); err != nil {
			b.Fatal(err)
		}
		resp.Reset()
	}
}

// BenchmarkFastHTTPPOST1KB benchmarks POST request with 1KB body
func BenchmarkFastHTTPPOST1KB(b *testing.B) {
	handler := func(ctx *fasthttp.RequestCtx) {
		_ = ctx.PostBody() // Read body
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString("OK")
	}

	server := &fasthttp.Server{
		Handler: handler,
	}
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go server.Serve(ln)

	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}

	body := generateBody(1024) // 1KB

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))

	var req fasthttp.Request
	var resp fasthttp.Response
	req.SetRequestURI("http://localhost/")
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/octet-stream")
	req.Header.SetHost("localhost")

	for i := 0; i < b.N; i++ {
		req.SetBody(body)
		if err := client.Do(&req, &resp); err != nil {
			b.Fatal(err)
		}
		resp.Reset()
		req.Reset()
		// Re-set headers after reset
		req.SetRequestURI("http://localhost/")
		req.Header.SetMethod("POST")
		req.Header.SetContentType("application/octet-stream")
		req.Header.SetHost("localhost")
	}
}

// BenchmarkFastHTTPKeepAlive benchmarks keep-alive connection reuse
func BenchmarkFastHTTPKeepAlive(b *testing.B) {
	handler := func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Connection", "keep-alive")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString("OK")
	}

	server := &fasthttp.Server{
		Handler: handler,
	}
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go server.Serve(ln)

	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
		MaxConnsPerHost:     100,
		MaxIdleConnDuration: 90 * time.Second,
	}

	// Warm up connection
	var req fasthttp.Request
	var resp fasthttp.Response
	req.SetRequestURI("http://localhost/")
	client.Do(&req, &resp)
	resp.Reset()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(2) // "OK" response

	for i := 0; i < b.N; i++ {
		if err := client.Do(&req, &resp); err != nil {
			b.Fatal(err)
		}
		resp.Reset()
	}
}

// BenchmarkFastHTTPConcurrent benchmarks concurrent requests (100 parallel)
func BenchmarkFastHTTPConcurrent(b *testing.B) {
	handler := func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString("OK")
	}

	server := &fasthttp.Server{
		Handler:     handler,
		Concurrency: 10000,
	}
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go server.Serve(ln)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(2) // "OK" response

	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		client := &fasthttp.Client{
			Dial: func(addr string) (net.Conn, error) {
				return ln.Dial()
			},
			MaxConnsPerHost: 100,
		}

		var req fasthttp.Request
		var resp fasthttp.Response
		req.SetRequestURI("http://localhost/")

		for pb.Next() {
			if err := client.Do(&req, &resp); err != nil {
				b.Error(err)
				return
			}
			resp.Reset()
		}
	})
}

// BenchmarkFastHTTPLargeResponse benchmarks 1MB response
func BenchmarkFastHTTPLargeResponse(b *testing.B) {
	largeData := generateBody(1024 * 1024) // 1MB
	handler := func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Write(largeData)
	}

	server := &fasthttp.Server{
		Handler: handler,
	}
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go server.Serve(ln)

	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
		MaxResponseBodySize: 2 * 1024 * 1024, // 2MB max
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(largeData)))

	var req fasthttp.Request
	var resp fasthttp.Response
	req.SetRequestURI("http://localhost/")

	for i := 0; i < b.N; i++ {
		if err := client.Do(&req, &resp); err != nil {
			b.Fatal(err)
		}
		_ = resp.Body() // Ensure body is read
		resp.Reset()
	}
}

// BenchmarkFastHTTPHeaderHeavy benchmarks request with 50 headers
func BenchmarkFastHTTPHeaderHeavy(b *testing.B) {
	handler := func(ctx *fasthttp.RequestCtx) {
		// Echo back header count
		headerCount := 0
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			headerCount++
		})
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString(fmt.Sprintf("Headers: %d", headerCount))
	}

	server := &fasthttp.Server{
		Handler: handler,
	}
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go server.Serve(ln)

	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	var req fasthttp.Request
	var resp fasthttp.Response
	req.SetRequestURI("http://localhost/")
	req.Header.SetHost("localhost")

	for i := 0; i < b.N; i++ {
		// Add 50 custom headers
		for j := 0; j < 50; j++ {
			req.Header.Set(fmt.Sprintf("X-Custom-Header-%d", j), fmt.Sprintf("value-%d", j))
		}

		if err := client.Do(&req, &resp); err != nil {
			b.Fatal(err)
		}
		resp.Reset()
		req.Reset()
		// Re-set headers after reset
		req.SetRequestURI("http://localhost/")
		req.Header.SetHost("localhost")
	}
}

// BenchmarkFastHTTPRequestParsing benchmarks raw HTTP request parsing
func BenchmarkFastHTTPRequestParsing(b *testing.B) {
	reqBytes := []byte("GET /path HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: benchmark\r\n" +
		"Accept: */*\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n")

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(reqBytes)))

	var req fasthttp.Request
	for i := 0; i < b.N; i++ {
		req.Reset()
		br := bufio.NewReader(bytes.NewReader(reqBytes))
		if err := req.Read(br); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFastHTTPResponseWriting benchmarks response writing
func BenchmarkFastHTTPResponseWriting(b *testing.B) {
	b.ReportAllocs()

	var resp fasthttp.Response
	var buf bytes.Buffer

	for i := 0; i < b.N; i++ {
		resp.Reset()
		buf.Reset()

		resp.SetStatusCode(fasthttp.StatusOK)
		resp.SetBody([]byte("Hello, World!"))
		resp.WriteTo(&buf)
	}
}