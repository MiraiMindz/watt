package http11

import (
	"bytes"
	"testing"
)

// Response Writer Allocation Benchmarks
//
// These benchmarks verify zero-allocation behavior of the ResponseWriter
// according to the implementation prompt requirements.

// ===================================================================
// Status Line Writing Benchmarks (Should be 0 allocs/op)
// ===================================================================

func BenchmarkResponseWriter_WriteStatusLine200(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Flush()
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_WriteStatusLine404(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(404)
		rw.Flush()
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_WriteStatusLine500(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(500)
		rw.Flush()
		PutResponseWriter(rw)
	}
}

// ===================================================================
// Response Writing Benchmarks
// ===================================================================

func BenchmarkResponseWriter_WriteSimpleResponse(b *testing.B) {
	var buf bytes.Buffer
	data := []byte("Hello, World!")
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteText(200, data)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_WriteJSON(b *testing.B) {
	var buf bytes.Buffer
	jsonData := []byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`)
	b.ReportAllocs()
	b.SetBytes(int64(len(jsonData)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteJSON(200, jsonData)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_WriteHTML(b *testing.B) {
	var buf bytes.Buffer
	htmlData := []byte("<html><body><h1>Hello</h1></body></html>")
	b.ReportAllocs()
	b.SetBytes(int64(len(htmlData)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHTML(200, htmlData)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_WriteError(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteError(404, "Not Found")
		PutResponseWriter(rw)
	}
}

// ===================================================================
// Header Writing Benchmarks
// ===================================================================

func BenchmarkResponseWriter_WriteWithSingleHeader(b *testing.B) {
	var buf bytes.Buffer
	data := []byte("test")
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.Header().Set(headerContentType, contentTypePlain)
		rw.WriteHeader(200)
		rw.Write(data)
		rw.Flush()
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_WriteWithMultipleHeaders(b *testing.B) {
	var buf bytes.Buffer
	data := []byte("test")
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.Header().Set(headerContentType, contentTypePlain)
		rw.Header().Set([]byte("Cache-Control"), []byte("max-age=3600"))
		rw.Header().Set([]byte("X-Request-ID"), []byte("req-123"))
		rw.Header().Set([]byte("X-Server"), []byte("Shockwave"))
		rw.WriteHeader(200)
		rw.Write(data)
		rw.Flush()
		PutResponseWriter(rw)
	}
}

// ===================================================================
// Chunked Encoding Benchmarks
// ===================================================================

func BenchmarkResponseWriter_WriteChunkedSmall(b *testing.B) {
	var buf bytes.Buffer
	chunks := [][]byte{
		[]byte("Hello"),
		[]byte(" "),
		[]byte("World"),
	}
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Header().Set([]byte("Transfer-Encoding"), []byte("chunked"))
		rw.WriteChunked(chunks)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_WriteChunkedIncremental(b *testing.B) {
	var buf bytes.Buffer
	chunks := []string{"First", "Second", "Third"}
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(200)
		for _, chunk := range chunks {
			rw.WriteChunk([]byte(chunk))
		}
		rw.FinishChunked()
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_WriteChunkedLarge(b *testing.B) {
	var buf bytes.Buffer
	chunk1KB := bytes.Repeat([]byte("x"), 1024)
	chunk10KB := bytes.Repeat([]byte("y"), 10240)
	b.ReportAllocs()
	b.SetBytes(1024 + 10240)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.WriteChunk(chunk1KB)
		rw.WriteChunk(chunk10KB)
		rw.FinishChunked()
		PutResponseWriter(rw)
	}
}

// ===================================================================
// Pooling Efficiency Benchmarks
// ===================================================================

func BenchmarkResponseWriter_PoolGetPut(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rw := GetResponseWriter(&buf)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_PoolReuse(b *testing.B) {
	var buf bytes.Buffer
	data := []byte("test data")
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rw := GetResponseWriter(&buf)
		rw.WriteText(200, data)
		buf.Reset()
		PutResponseWriter(rw)
	}
}

// ===================================================================
// Throughput Benchmarks
// ===================================================================

func BenchmarkResponseWriter_Throughput100B(b *testing.B) {
	var buf bytes.Buffer
	data := bytes.Repeat([]byte("x"), 100)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteText(200, data)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_Throughput1KB(b *testing.B) {
	var buf bytes.Buffer
	data := bytes.Repeat([]byte("x"), 1024)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteText(200, data)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_Throughput10KB(b *testing.B) {
	var buf bytes.Buffer
	data := bytes.Repeat([]byte("x"), 10*1024)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteText(200, data)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriter_Throughput100KB(b *testing.B) {
	var buf bytes.Buffer
	data := bytes.Repeat([]byte("x"), 100*1024)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteText(200, data)
		PutResponseWriter(rw)
	}
}
