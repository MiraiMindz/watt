package http11

import (
	"bytes"
	"bufio"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Comparison benchmarks: Shockwave vs net/http
//
// These benchmarks compare the performance of Shockwave's HTTP/1.1 engine
// against Go's standard library net/http implementation.
//
// Run with: go test -bench=BenchmarkComparison -benchmem

// ===================================================================
// Request Parsing Benchmarks
// ===================================================================

// Shared test data
var (
	simpleGETRequest = "GET /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: Go-http-client/1.1\r\n" +
		"\r\n"

	postWithBodyRequest = "POST /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 27\r\n" +
		"\r\n" +
		`{"name":"Alice","age":30}`

	multipleHeadersRequest = "GET /api/data HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: Mozilla/5.0\r\n" +
		"Accept: application/json\r\n" +
		"Accept-Encoding: gzip, deflate\r\n" +
		"Accept-Language: en-US,en;q=0.9\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Connection: keep-alive\r\n" +
		"Cookie: session=abc123\r\n" +
		"Referer: https://example.com\r\n" +
		"Authorization: Bearer token123\r\n" +
		"\r\n"
)

// BenchmarkComparison_ParseSimpleGET_Shockwave benchmarks Shockwave request parsing
func BenchmarkComparison_ParseSimpleGET_Shockwave(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(simpleGETRequest)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		r := strings.NewReader(simpleGETRequest)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkComparison_ParseSimpleGET_NetHTTP benchmarks net/http request parsing
func BenchmarkComparison_ParseSimpleGET_NetHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(simpleGETRequest)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(strings.NewReader(simpleGETRequest))
		req, err := http.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}
		_ = req // Use request
	}
}

// BenchmarkComparison_ParsePOST_Shockwave benchmarks Shockwave POST parsing
func BenchmarkComparison_ParsePOST_Shockwave(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(postWithBodyRequest)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Get fresh parser each iteration to avoid body contamination
		parser := GetParser()
		r := strings.NewReader(postWithBodyRequest)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkComparison_ParsePOST_NetHTTP benchmarks net/http POST parsing
func BenchmarkComparison_ParsePOST_NetHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(postWithBodyRequest)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(strings.NewReader(postWithBodyRequest))
		req, err := http.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}
		_ = req
	}
}

// BenchmarkComparison_ParseMultipleHeaders_Shockwave benchmarks Shockwave with many headers
func BenchmarkComparison_ParseMultipleHeaders_Shockwave(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(multipleHeadersRequest)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		r := strings.NewReader(multipleHeadersRequest)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkComparison_ParseMultipleHeaders_NetHTTP benchmarks net/http with many headers
func BenchmarkComparison_ParseMultipleHeaders_NetHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(multipleHeadersRequest)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(strings.NewReader(multipleHeadersRequest))
		req, err := http.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}
		_ = req
	}
}

// ===================================================================
// Response Writing Benchmarks
// ===================================================================

// BenchmarkComparison_WriteSimpleResponse_Shockwave benchmarks Shockwave response writing
func BenchmarkComparison_WriteSimpleResponse_Shockwave(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		rw := GetResponseWriter(bufWriter)
		err := rw.WriteText(200, []byte("Hello, World!"))
		if err != nil {
			b.Fatal(err)
		}
		err = rw.Flush()
		if err != nil {
			b.Fatal(err)
		}
		PutResponseWriter(rw)
	}
	b.SetBytes(int64(buf.Len()))
}

// BenchmarkComparison_WriteSimpleResponse_NetHTTP benchmarks net/http response writing
func BenchmarkComparison_WriteSimpleResponse_NetHTTP(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		resp := &http.Response{
			StatusCode: 200,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("Hello, World!")),
		}
		resp.Header.Set("Content-Type", "text/plain")
		err := resp.Write(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(buf.Len()))
}

// BenchmarkComparison_WriteJSONResponse_Shockwave benchmarks Shockwave JSON response
func BenchmarkComparison_WriteJSONResponse_Shockwave(b *testing.B) {
	b.ReportAllocs()

	jsonData := []byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`)
	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		rw := GetResponseWriter(bufWriter)
		err := rw.WriteJSON(200, jsonData)
		if err != nil {
			b.Fatal(err)
		}
		err = rw.Flush()
		if err != nil {
			b.Fatal(err)
		}
		PutResponseWriter(rw)
	}
	b.SetBytes(int64(len(jsonData)))
}

// BenchmarkComparison_WriteJSONResponse_NetHTTP benchmarks net/http JSON response
func BenchmarkComparison_WriteJSONResponse_NetHTTP(b *testing.B) {
	b.ReportAllocs()

	jsonData := []byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`)
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		resp := &http.Response{
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(jsonData)),
			ContentLength: int64(len(jsonData)),
		}
		resp.Header.Set("Content-Type", "application/json")
		err := resp.Write(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(jsonData)))
}

// ===================================================================
// Full Cycle Benchmarks (Parse + Handle + Write)
// ===================================================================

// BenchmarkComparison_FullCycleSimpleGET_Shockwave benchmarks full request/response cycle
func BenchmarkComparison_FullCycleSimpleGET_Shockwave(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(simpleGETRequest)))

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		// Parse request
		parser := GetParser()
		r := strings.NewReader(simpleGETRequest)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

		// Write response
		rw := GetResponseWriter(bufWriter)
		responseBody := []byte(`{"users":[{"id":1,"name":"Alice"}]}`)
		err = rw.WriteJSON(200, responseBody)
		if err != nil {
			b.Fatal(err)
		}
		err = rw.Flush()
		if err != nil {
			b.Fatal(err)
		}

		PutResponseWriter(rw)
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkComparison_FullCycleSimpleGET_NetHTTP benchmarks net/http full cycle
func BenchmarkComparison_FullCycleSimpleGET_NetHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(simpleGETRequest)))

	var buf bytes.Buffer
	responseBody := []byte(`{"users":[{"id":1,"name":"Alice"}]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		// Parse request
		r := bufio.NewReader(strings.NewReader(simpleGETRequest))
		req, err := http.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}

		// Write response
		resp := &http.Response{
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(responseBody)),
			ContentLength: int64(len(responseBody)),
		}
		resp.Header.Set("Content-Type", "application/json")
		err = resp.Write(&buf)
		if err != nil {
			b.Fatal(err)
		}

		_ = req
	}
}

// ===================================================================
// Throughput Benchmarks (Large Responses)
// ===================================================================

// BenchmarkComparison_Throughput1KB_Shockwave benchmarks 1KB response throughput
func BenchmarkComparison_Throughput1KB_Shockwave(b *testing.B) {
	b.ReportAllocs()

	response1KB := bytes.Repeat([]byte("x"), 1024)
	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		rw := GetResponseWriter(bufWriter)
		err := rw.WriteText(200, response1KB)
		if err != nil {
			b.Fatal(err)
		}
		err = rw.Flush()
		if err != nil {
			b.Fatal(err)
		}
		PutResponseWriter(rw)
	}
	b.SetBytes(int64(len(response1KB)))
}

// BenchmarkComparison_Throughput1KB_NetHTTP benchmarks net/http 1KB response
func BenchmarkComparison_Throughput1KB_NetHTTP(b *testing.B) {
	b.ReportAllocs()

	response1KB := bytes.Repeat([]byte("x"), 1024)
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		resp := &http.Response{
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(response1KB)),
			ContentLength: int64(len(response1KB)),
		}
		resp.Header.Set("Content-Type", "text/plain")
		err := resp.Write(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(response1KB)))
}

// BenchmarkComparison_Throughput10KB_Shockwave benchmarks 10KB response throughput
func BenchmarkComparison_Throughput10KB_Shockwave(b *testing.B) {
	b.ReportAllocs()

	response10KB := bytes.Repeat([]byte("x"), 10*1024)
	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		rw := GetResponseWriter(bufWriter)
		err := rw.WriteText(200, response10KB)
		if err != nil {
			b.Fatal(err)
		}
		err = rw.Flush()
		if err != nil {
			b.Fatal(err)
		}
		PutResponseWriter(rw)
	}
	b.SetBytes(int64(len(response10KB)))
}

// BenchmarkComparison_Throughput10KB_NetHTTP benchmarks net/http 10KB response
func BenchmarkComparison_Throughput10KB_NetHTTP(b *testing.B) {
	b.ReportAllocs()

	response10KB := bytes.Repeat([]byte("x"), 10*1024)
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		resp := &http.Response{
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(response10KB)),
			ContentLength: int64(len(response10KB)),
		}
		resp.Header.Set("Content-Type", "text/plain")
		err := resp.Write(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(response10KB)))
}
