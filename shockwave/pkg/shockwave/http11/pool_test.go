package http11

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// Test Request pooling

func TestRequestPool(t *testing.T) {
	req1 := GetRequest()
	if req1 == nil {
		t.Fatal("GetRequest returned nil")
	}

	// Modify request
	req1.MethodID = MethodPOST
	req1.pathBytes = []byte("/test")

	// Return to pool
	PutRequest(req1)

	// Get another request - might be the same object
	req2 := GetRequest()
	if req2 == nil {
		t.Fatal("GetRequest returned nil after Put")
	}

	// Should be reset to zero values
	if req2.MethodID != 0 {
		t.Errorf("MethodID not reset: got %d, want 0", req2.MethodID)
	}
	if req2.pathBytes != nil {
		t.Errorf("pathBytes not reset: got %v, want nil", req2.pathBytes)
	}

	PutRequest(req2)
}

func TestRequestPoolNil(t *testing.T) {
	// Should not panic
	PutRequest(nil)
}

// Test ResponseWriter pooling

func TestResponseWriterPool(t *testing.T) {
	var buf bytes.Buffer

	rw1 := GetResponseWriter(&buf)
	if rw1 == nil {
		t.Fatal("GetResponseWriter returned nil")
	}

	// Modify response writer
	rw1.WriteHeader(404)
	rw1.Header().Set([]byte("X-Test"), []byte("value"))

	// Return to pool
	PutResponseWriter(rw1)

	// Get another response writer - might be the same object
	rw2 := GetResponseWriter(&buf)
	if rw2 == nil {
		t.Fatal("GetResponseWriter returned nil after Put")
	}

	// Should be reset to defaults
	if rw2.Status() != 200 {
		t.Errorf("Status not reset: got %d, want 200", rw2.Status())
	}
	if rw2.Header().Len() != 0 {
		t.Errorf("Headers not reset: got %d headers, want 0", rw2.Header().Len())
	}

	PutResponseWriter(rw2)
}

func TestResponseWriterPoolNil(t *testing.T) {
	// Should not panic
	PutResponseWriter(nil)
}

// Test Parser pooling

func TestParserPool(t *testing.T) {
	p1 := GetParser()
	if p1 == nil {
		t.Fatal("GetParser returned nil")
	}

	// Use parser
	input := "GET / HTTP/1.1\r\n\r\n"
	_, err := p1.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Return to pool
	PutParser(p1)

	// Get another parser
	p2 := GetParser()
	if p2 == nil {
		t.Fatal("GetParser returned nil after Put")
	}

	// Buffer should be reset to length 0
	if len(p2.buf) != 0 {
		t.Errorf("Parser buffer not reset: len=%d, want 0", len(p2.buf))
	}

	PutParser(p2)
}

func TestParserPoolNil(t *testing.T) {
	// Should not panic
	PutParser(nil)
}

// Test Buffer pooling

func TestBufferPool(t *testing.T) {
	buf1 := GetBuffer()
	if buf1 == nil {
		t.Fatal("GetBuffer returned nil")
	}

	if len(buf1) != DefaultBufferSize {
		t.Errorf("Buffer length = %d, want %d", len(buf1), DefaultBufferSize)
	}

	// Modify buffer
	copy(buf1, []byte("test data"))

	// Return to pool
	PutBuffer(buf1)

	// Get another buffer
	buf2 := GetBuffer()
	if buf2 == nil {
		t.Fatal("GetBuffer returned nil after Put")
	}

	if len(buf2) != DefaultBufferSize {
		t.Errorf("Buffer length = %d, want %d", len(buf2), DefaultBufferSize)
	}

	PutBuffer(buf2)
}

func TestBufferPoolInvalidSize(t *testing.T) {
	// Should not panic with nil buffer
	PutBuffer(nil)

	// Should not panic with undersized buffer
	smallBuf := make([]byte, 10)
	PutBuffer(smallBuf)
}

// Test LargeBuffer pooling

func TestLargeBufferPool(t *testing.T) {
	buf1 := GetLargeBuffer()
	if buf1 == nil {
		t.Fatal("GetLargeBuffer returned nil")
	}

	if len(buf1) != 0 {
		t.Errorf("Buffer length = %d, want 0 (should start empty)", len(buf1))
	}

	if cap(buf1) != ParserBufferSize {
		t.Errorf("Buffer capacity = %d, want %d", cap(buf1), ParserBufferSize)
	}

	// Use buffer
	buf1 = append(buf1, []byte("test data")...)

	// Return to pool
	PutLargeBuffer(buf1)

	// Get another buffer
	buf2 := GetLargeBuffer()
	if buf2 == nil {
		t.Fatal("GetLargeBuffer returned nil after Put")
	}

	// Should be reset to length 0
	if len(buf2) != 0 {
		t.Errorf("Buffer length = %d, want 0 (should be reset)", len(buf2))
	}

	PutLargeBuffer(buf2)
}

// Test Bufio Reader pooling

func TestBufioReaderPool(t *testing.T) {
	input := strings.NewReader("test data")

	br1 := GetBufioReader(input)
	if br1 == nil {
		t.Fatal("GetBufioReader returned nil")
	}

	// Read data
	data, err := br1.ReadString('\n')
	if err != nil && err.Error() != "EOF" {
		t.Errorf("Read failed: %v", err)
	}
	if !strings.Contains(data, "test") {
		t.Errorf("Read wrong data: %q", data)
	}

	// Return to pool
	PutBufioReader(br1)

	// Get another reader
	br2 := GetBufioReader(strings.NewReader("new data"))
	if br2 == nil {
		t.Fatal("GetBufioReader returned nil after Put")
	}

	PutBufioReader(br2)
}

func TestBufioReaderPoolNil(t *testing.T) {
	// Should not panic
	PutBufioReader(nil)
}

// Test Bufio Writer pooling

func TestBufioWriterPool(t *testing.T) {
	var buf1 bytes.Buffer

	bw1 := GetBufioWriter(&buf1)
	if bw1 == nil {
		t.Fatal("GetBufioWriter returned nil")
	}

	// Write data
	bw1.WriteString("test data")
	bw1.Flush()

	if buf1.String() != "test data" {
		t.Errorf("Write failed: got %q", buf1.String())
	}

	// Return to pool
	PutBufioWriter(bw1)

	// Get another writer
	var buf2 bytes.Buffer
	bw2 := GetBufioWriter(&buf2)
	if bw2 == nil {
		t.Fatal("GetBufioWriter returned nil after Put")
	}

	// Should work with new buffer
	bw2.WriteString("new data")
	bw2.Flush()

	if buf2.String() != "new data" {
		t.Errorf("Write failed: got %q", buf2.String())
	}

	PutBufioWriter(bw2)
}

func TestBufioWriterPoolNil(t *testing.T) {
	// Should not panic
	PutBufioWriter(nil)
}

// Test WarmupPools

func TestWarmupPools(t *testing.T) {
	// Should not panic
	WarmupPools(10)

	// After warmup, pools should have objects
	req := GetRequest()
	if req == nil {
		t.Error("No request available after warmup")
	}
	PutRequest(req)

	rw := GetResponseWriter(nil)
	if rw == nil {
		t.Error("No response writer available after warmup")
	}
	PutResponseWriter(rw)
}

// Test GetPoolStats

func TestGetPoolStats(t *testing.T) {
	stats := GetPoolStats()

	if len(stats) == 0 {
		t.Error("GetPoolStats returned empty slice")
	}

	// Should have stats for all pools
	expectedPools := []string{
		"Request",
		"ResponseWriter",
		"Parser",
		"Buffer",
		"LargeBuffer",
		"BufioReader",
		"BufioWriter",
	}

	for _, expected := range expectedPools {
		found := false
		for _, stat := range stats {
			if stat.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing pool stats for %s", expected)
		}
	}
}

// Concurrent access tests

func TestRequestPoolConcurrent(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				req := GetRequest()
				req.MethodID = MethodGET
				PutRequest(req)
			}
		}()
	}

	wg.Wait()
}

func TestResponseWriterPoolConcurrent(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				var buf bytes.Buffer
				rw := GetResponseWriter(&buf)
				rw.WriteHeader(200)
				PutResponseWriter(rw)
			}
		}()
	}

	wg.Wait()
}

func TestBufferPoolConcurrent(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				buf := GetBuffer()
				copy(buf, []byte("test"))
				PutBuffer(buf)
			}
		}()
	}

	wg.Wait()
}

// Benchmarks

func BenchmarkRequestPoolGetPut(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := GetRequest()
		PutRequest(req)
	}
}

func BenchmarkRequestPoolGetUsePut(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := GetRequest()
		req.MethodID = MethodGET
		req.pathBytes = []byte("/test")
		req.Header.Add([]byte("Host"), []byte("example.com"))
		PutRequest(req)
	}
}

func BenchmarkResponseWriterPoolGetPut(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rw := GetResponseWriter(&buf)
		PutResponseWriter(rw)
	}
}

func BenchmarkResponseWriterPoolGetUsePut(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		rw.Flush()
		PutResponseWriter(rw)
	}
}

func BenchmarkParserPoolGetPut(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p := GetParser()
		PutParser(p)
	}
}

func BenchmarkBufferPoolGetPut(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := GetBuffer()
		PutBuffer(buf)
	}
}

func BenchmarkLargeBufferPoolGetPut(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := GetLargeBuffer()
		PutLargeBuffer(buf)
	}
}

func BenchmarkBufioReaderPoolGetPut(b *testing.B) {
	input := strings.NewReader("test data")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		br := GetBufioReader(input)
		PutBufioReader(br)
	}
}

func BenchmarkBufioWriterPoolGetPut(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		bw := GetBufioWriter(&buf)
		PutBufioWriter(bw)
	}
}

func BenchmarkWarmupPools(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		WarmupPools(10)
	}
}

// Compare pooled vs non-pooled allocations

func BenchmarkRequestNonPooled(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := &Request{}
		req.Reset()
		_ = req
	}
}

func BenchmarkRequestPooled(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := GetRequest()
		PutRequest(req)
	}
}

func BenchmarkResponseWriterNonPooled(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rw := &ResponseWriter{}
		rw.Reset(&buf)
		_ = rw
	}
}

func BenchmarkResponseWriterPooled(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rw := GetResponseWriter(&buf)
		PutResponseWriter(rw)
	}
}

// Additional tests for 100% coverage

func TestPutLargeBufferCapacityCheck(t *testing.T) {
	// Test that PutLargeBuffer doesn't pool buffers smaller than ParserBufferSize
	smallBuf := make([]byte, ParserBufferSize-1)

	// This should be a no-op (not panic)
	PutLargeBuffer(smallBuf)

	// Get a new buffer - should not be the small one we just "put"
	buf := GetLargeBuffer()
	if cap(buf) != ParserBufferSize {
		t.Errorf("GetLargeBuffer returned buffer with cap=%d, want %d",
			cap(buf), ParserBufferSize)
	}
}
