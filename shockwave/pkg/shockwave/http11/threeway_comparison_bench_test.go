package http11

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

// Three-Way Comparison Benchmarks: Shockwave vs fasthttp vs net/http
//
// These benchmarks compare the performance of three HTTP implementations:
// 1. Shockwave - Our custom high-performance HTTP/1.1 engine
// 2. fasthttp - valyala/fasthttp (popular high-performance HTTP library)
// 3. net/http - Go standard library
//
// Run with: go test -bench=BenchmarkThreeWay -benchmem -benchtime=3s

// ===================================================================
// Shared Test Data
// ===================================================================

var (
	threeWaySimpleGET = "GET /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: Go-http-client/1.1\r\n" +
		"\r\n"

	threeWayPOST = "POST /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 27\r\n" +
		"\r\n" +
		`{"name":"Alice","age":30}`

	threeWayMultipleHeaders = "GET /api/data HTTP/1.1\r\n" +
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

	threeWayJSONData = []byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`)
	threeWay1KB      = bytes.Repeat([]byte("x"), 1024)
	threeWay10KB     = bytes.Repeat([]byte("x"), 10*1024)
)

// ===================================================================
// Request Parsing Benchmarks
// ===================================================================

// Simple GET Request Parsing

func BenchmarkThreeWay_ParseSimpleGET_Shockwave(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWaySimpleGET)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		r := strings.NewReader(threeWaySimpleGET)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

func BenchmarkThreeWay_ParseSimpleGET_FastHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWaySimpleGET)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req fasthttp.Request
		err := req.Read(bufio.NewReader(strings.NewReader(threeWaySimpleGET)))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThreeWay_ParseSimpleGET_NetHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWaySimpleGET)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(strings.NewReader(threeWaySimpleGET))
		req, err := http.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}
		_ = req
	}
}

// POST with Body Parsing

func BenchmarkThreeWay_ParsePOST_Shockwave(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWayPOST)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		r := strings.NewReader(threeWayPOST)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

func BenchmarkThreeWay_ParsePOST_FastHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWayPOST)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req fasthttp.Request
		err := req.Read(bufio.NewReader(strings.NewReader(threeWayPOST)))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThreeWay_ParsePOST_NetHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWayPOST)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(strings.NewReader(threeWayPOST))
		req, err := http.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}
		_ = req
	}
}

// Multiple Headers Parsing

func BenchmarkThreeWay_ParseMultipleHeaders_Shockwave(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWayMultipleHeaders)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		r := strings.NewReader(threeWayMultipleHeaders)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

func BenchmarkThreeWay_ParseMultipleHeaders_FastHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWayMultipleHeaders)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req fasthttp.Request
		err := req.Read(bufio.NewReader(strings.NewReader(threeWayMultipleHeaders)))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThreeWay_ParseMultipleHeaders_NetHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWayMultipleHeaders)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(strings.NewReader(threeWayMultipleHeaders))
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

// Simple Text Response

func BenchmarkThreeWay_WriteSimpleText_Shockwave(b *testing.B) {
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

func BenchmarkThreeWay_WriteSimpleText_FastHTTP(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		var resp fasthttp.Response
		resp.SetStatusCode(200)
		resp.Header.SetContentType("text/plain")
		resp.SetBodyString("Hello, World!")
		_, err := resp.WriteTo(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(buf.Len()))
}

func BenchmarkThreeWay_WriteSimpleText_NetHTTP(b *testing.B) {
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

// JSON Response

func BenchmarkThreeWay_WriteJSON_Shockwave(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		rw := GetResponseWriter(bufWriter)
		err := rw.WriteJSON(200, threeWayJSONData)
		if err != nil {
			b.Fatal(err)
		}
		err = rw.Flush()
		if err != nil {
			b.Fatal(err)
		}
		PutResponseWriter(rw)
	}
	b.SetBytes(int64(len(threeWayJSONData)))
}

func BenchmarkThreeWay_WriteJSON_FastHTTP(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		var resp fasthttp.Response
		resp.SetStatusCode(200)
		resp.Header.SetContentType("application/json")
		resp.SetBody(threeWayJSONData)
		_, err := resp.WriteTo(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(threeWayJSONData)))
}

func BenchmarkThreeWay_WriteJSON_NetHTTP(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		resp := &http.Response{
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(threeWayJSONData)),
			ContentLength: int64(len(threeWayJSONData)),
		}
		resp.Header.Set("Content-Type", "application/json")
		err := resp.Write(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(threeWayJSONData)))
}

// ===================================================================
// Full Cycle Benchmarks (Parse + Handle + Write)
// ===================================================================

func BenchmarkThreeWay_FullCycle_Shockwave(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWaySimpleGET)))

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		// Parse request
		parser := GetParser()
		r := strings.NewReader(threeWaySimpleGET)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

		// Write response
		rw := GetResponseWriter(bufWriter)
		err = rw.WriteJSON(200, threeWayJSONData)
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

func BenchmarkThreeWay_FullCycle_FastHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWaySimpleGET)))

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		// Parse request
		var req fasthttp.Request
		err := req.Read(bufio.NewReader(strings.NewReader(threeWaySimpleGET)))
		if err != nil {
			b.Fatal(err)
		}

		// Write response
		var resp fasthttp.Response
		resp.SetStatusCode(200)
		resp.Header.SetContentType("application/json")
		resp.SetBody(threeWayJSONData)
		_, err = resp.WriteTo(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThreeWay_FullCycle_NetHTTP(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(threeWaySimpleGET)))

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		// Parse request
		r := bufio.NewReader(strings.NewReader(threeWaySimpleGET))
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
			Body:          io.NopCloser(bytes.NewReader(threeWayJSONData)),
			ContentLength: int64(len(threeWayJSONData)),
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

// 1KB Response Throughput

func BenchmarkThreeWay_Throughput1KB_Shockwave(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		rw := GetResponseWriter(bufWriter)
		err := rw.WriteText(200, threeWay1KB)
		if err != nil {
			b.Fatal(err)
		}
		err = rw.Flush()
		if err != nil {
			b.Fatal(err)
		}
		PutResponseWriter(rw)
	}
	b.SetBytes(int64(len(threeWay1KB)))
}

func BenchmarkThreeWay_Throughput1KB_FastHTTP(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		var resp fasthttp.Response
		resp.SetStatusCode(200)
		resp.Header.SetContentType("text/plain")
		resp.SetBody(threeWay1KB)
		_, err := resp.WriteTo(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(threeWay1KB)))
}

func BenchmarkThreeWay_Throughput1KB_NetHTTP(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		resp := &http.Response{
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(threeWay1KB)),
			ContentLength: int64(len(threeWay1KB)),
		}
		resp.Header.Set("Content-Type", "text/plain")
		err := resp.Write(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(threeWay1KB)))
}

// 10KB Response Throughput

func BenchmarkThreeWay_Throughput10KB_Shockwave(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		bufWriter.Reset(&buf)

		rw := GetResponseWriter(bufWriter)
		err := rw.WriteText(200, threeWay10KB)
		if err != nil {
			b.Fatal(err)
		}
		err = rw.Flush()
		if err != nil {
			b.Fatal(err)
		}
		PutResponseWriter(rw)
	}
	b.SetBytes(int64(len(threeWay10KB)))
}

func BenchmarkThreeWay_Throughput10KB_FastHTTP(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		var resp fasthttp.Response
		resp.SetStatusCode(200)
		resp.Header.SetContentType("text/plain")
		resp.SetBody(threeWay10KB)
		_, err := resp.WriteTo(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(threeWay10KB)))
}

func BenchmarkThreeWay_Throughput10KB_NetHTTP(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		resp := &http.Response{
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(threeWay10KB)),
			ContentLength: int64(len(threeWay10KB)),
		}
		resp.Header.Set("Content-Type", "text/plain")
		err := resp.Write(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(threeWay10KB)))
}

// ===================================================================
// Header Operations Benchmarks
// ===================================================================

// Header Lookup Performance

func BenchmarkThreeWay_HeaderLookup_Shockwave(b *testing.B) {
	b.ReportAllocs()

	parser := GetParser()
	r := strings.NewReader(threeWayMultipleHeaders)
	req, err := parser.Parse(r)
	if err != nil {
		b.Fatal(err)
	}
	defer PutRequest(req)
	defer PutParser(parser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Header.Get([]byte("User-Agent"))
		_ = req.Header.Get([]byte("Accept"))
		_ = req.Header.Get([]byte("Authorization"))
	}
}

func BenchmarkThreeWay_HeaderLookup_FastHTTP(b *testing.B) {
	b.ReportAllocs()

	var req fasthttp.Request
	err := req.Read(bufio.NewReader(strings.NewReader(threeWayMultipleHeaders)))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Header.Peek("User-Agent")
		_ = req.Header.Peek("Accept")
		_ = req.Header.Peek("Authorization")
	}
}

func BenchmarkThreeWay_HeaderLookup_NetHTTP(b *testing.B) {
	b.ReportAllocs()

	r := bufio.NewReader(strings.NewReader(threeWayMultipleHeaders))
	req, err := http.ReadRequest(r)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Header.Get("User-Agent")
		_ = req.Header.Get("Accept")
		_ = req.Header.Get("Authorization")
	}
}
