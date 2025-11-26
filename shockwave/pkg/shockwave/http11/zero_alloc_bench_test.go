package http11

import (
	"bytes"
	"testing"
)

// Zero-Allocation Benchmarks
//
// These benchmarks are designed to measure the true allocation behavior
// of the parser without interference from test infrastructure like strings.NewReader.
//
// Goal: Achieve 0 allocs/op for parsing requests with ≤32 headers

var (
	simpleGETBytes = []byte("GET /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: Go-http-client/1.1\r\n" +
		"\r\n")

	multiHeaderBytes = []byte("GET /api/data HTTP/1.1\r\n" +
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
		"\r\n")

	postWithBodyBytes = []byte("POST /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 27\r\n" +
		"\r\n" +
		`{"name":"Alice","age":30}`)
)

// BenchmarkZeroAlloc_ParseSimpleGET benchmarks parsing a simple GET request
// using bytes.Reader to avoid strings.NewReader allocation.
func BenchmarkZeroAlloc_ParseSimpleGET(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(simpleGETBytes)))

	// Pre-allocate readers to reuse
	readers := make([]*bytes.Reader, b.N)
	for i := 0; i < b.N; i++ {
		readers[i] = bytes.NewReader(simpleGETBytes)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		req, err := parser.Parse(readers[i])
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkZeroAlloc_ParseSimpleGET_NoPrealloc measures parsing without
// pre-allocating readers (to see baseline allocation).
func BenchmarkZeroAlloc_ParseSimpleGET_Baseline(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(simpleGETBytes)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		r := bytes.NewReader(simpleGETBytes)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkZeroAlloc_ParseMultipleHeaders benchmarks parsing with 10 headers.
func BenchmarkZeroAlloc_ParseMultipleHeaders(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(multiHeaderBytes)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		r := bytes.NewReader(multiHeaderBytes)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkZeroAlloc_ParsePOST benchmarks parsing POST with body.
func BenchmarkZeroAlloc_ParsePOST(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(postWithBodyBytes)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetParser()
		r := bytes.NewReader(postWithBodyBytes)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkZeroAlloc_ParseReuse benchmarks reusing the same parser
// to see if pooling eliminates allocations.
func BenchmarkZeroAlloc_ParseReuse(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(simpleGETBytes)))

	parser := GetParser()
	defer PutParser(parser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(simpleGETBytes)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		PutRequest(req)
	}
}

// BenchmarkZeroAlloc_FullCycle benchmarks full request/response cycle.
func BenchmarkZeroAlloc_FullCycle(b *testing.B) {
	b.ReportAllocs()

	jsonData := []byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Parse request
		parser := GetParser()
		r := bytes.NewReader(simpleGETBytes)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

		// Write response
		var buf bytes.Buffer
		rw := GetResponseWriter(&buf)
		err = rw.WriteJSON(200, jsonData)
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

// BenchmarkZeroAlloc_ParserOnly benchmarks just the parser operations
// without getting from pool, to isolate parser-specific allocations.
func BenchmarkZeroAlloc_ParserOnly(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(simpleGETBytes)))

	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(simpleGETBytes)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
		// Don't return to pool, just let req be collected
		_ = req
		// Reset parser state manually
		parser.buf = parser.buf[:0]
		parser.unreadBuf = nil
	}
}
