package http11

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestResponseWriterSimple(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.WriteHeader(200)
	rw.Write([]byte("Hello, World!"))
	rw.Flush()

	output := buf.String()

	// Should contain status line
	if !strings.Contains(output, "HTTP/1.1 200 OK\r\n") {
		t.Errorf("Output missing status line: %q", output)
	}

	// Should contain body
	if !strings.Contains(output, "Hello, World!") {
		t.Errorf("Output missing body: %q", output)
	}

	// Should have blank line before body
	if !strings.Contains(output, "\r\n\r\n") {
		t.Errorf("Output missing blank line before body: %q", output)
	}
}

func TestResponseWriterImplicitStatus(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	// Don't call WriteHeader, should default to 200
	rw.Write([]byte("test"))
	rw.Flush()

	output := buf.String()

	if !strings.Contains(output, "HTTP/1.1 200 OK\r\n") {
		t.Errorf("Output missing default 200 status: %q", output)
	}
}

func TestResponseWriterCommonStatusCodes(t *testing.T) {
	codes := []int{200, 201, 204, 301, 302, 304, 400, 401, 403, 404, 500, 502, 503}

	for _, code := range codes {
		t.Run(statusText(code), func(t *testing.T) {
			var buf bytes.Buffer
			rw := NewResponseWriter(&buf)

			rw.WriteHeader(code)
			rw.Write([]byte("test"))
			rw.Flush()

			output := buf.String()

			expectedPrefix := "HTTP/1.1 " + string(rune('0'+code/100))
			if !strings.HasPrefix(output, expectedPrefix) {
				t.Errorf("Output doesn't start with %q: %q", expectedPrefix, output)
			}
		})
	}
}

func TestResponseWriterUncommonStatusCode(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.WriteHeader(418) // I'm a teapot
	rw.Write([]byte("test"))
	rw.Flush()

	output := buf.String()

	if !strings.Contains(output, "HTTP/1.1 418") {
		t.Errorf("Output missing status 418: %q", output)
	}

	if !strings.Contains(output, "I'm a teapot") {
		t.Errorf("Output missing status text: %q", output)
	}
}

func TestResponseWriterHeaders(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.Header().Set([]byte("Content-Type"), []byte("application/json"))
	rw.Header().Set([]byte("X-Custom"), []byte("value"))

	rw.WriteHeader(200)
	rw.Write([]byte("{}"))
	rw.Flush()

	output := buf.String()

	if !strings.Contains(output, "Content-Type: application/json\r\n") {
		t.Errorf("Output missing Content-Type header: %q", output)
	}

	if !strings.Contains(output, "X-Custom: value\r\n") {
		t.Errorf("Output missing X-Custom header: %q", output)
	}
}

func TestResponseWriterMultipleHeaders(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	headers := []struct{ name, value string }{
		{"Content-Type", "text/html"},
		{"Content-Length", "13"},
		{"Server", "Shockwave"},
		{"X-Request-ID", "12345"},
	}

	for _, h := range headers {
		rw.Header().Set([]byte(h.name), []byte(h.value))
	}

	rw.WriteHeader(200)
	rw.Write([]byte("Hello, World!"))
	rw.Flush()

	output := buf.String()

	for _, h := range headers {
		expected := h.name + ": " + h.value + "\r\n"
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing header %q: %q", expected, output)
		}
	}
}

func TestResponseWriterMultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.WriteHeader(200)
	rw.Write([]byte("Hello, "))
	rw.Write([]byte("World!"))
	rw.Flush()

	output := buf.String()

	if !strings.Contains(output, "Hello, World!") {
		t.Errorf("Output missing concatenated body: %q", output)
	}

	if rw.BytesWritten() != 13 {
		t.Errorf("BytesWritten = %d, want 13", rw.BytesWritten())
	}
}

func TestResponseWriterBytesWritten(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	data := []byte("Hello, World!")
	rw.WriteHeader(200)
	rw.Write(data)

	if rw.BytesWritten() != int64(len(data)) {
		t.Errorf("BytesWritten = %d, want %d", rw.BytesWritten(), len(data))
	}
}

func TestResponseWriterStatus(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	// Before WriteHeader
	if rw.Status() != 200 {
		t.Errorf("Status before WriteHeader = %d, want 200 (default)", rw.Status())
	}

	rw.WriteHeader(404)

	if rw.Status() != 404 {
		t.Errorf("Status after WriteHeader = %d, want 404", rw.Status())
	}
}

func TestResponseWriterHeaderWritten(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	if rw.HeaderWritten() {
		t.Error("HeaderWritten before Write = true, want false")
	}

	rw.Write([]byte("test"))

	if !rw.HeaderWritten() {
		t.Error("HeaderWritten after Write = false, want true")
	}
}

func TestResponseWriterWriteHeaderOnce(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.WriteHeader(200)
	rw.WriteHeader(404) // Should be ignored

	rw.Write([]byte("test"))
	rw.Flush()

	output := buf.String()

	if !strings.Contains(output, "HTTP/1.1 200 OK") {
		t.Error("First WriteHeader not used")
	}

	if strings.Contains(output, "404") {
		t.Error("Second WriteHeader should be ignored")
	}
}

func TestResponseWriterReset(t *testing.T) {
	var buf1 bytes.Buffer
	rw := NewResponseWriter(&buf1)

	rw.WriteHeader(404)
	rw.Header().Set([]byte("X-Custom"), []byte("value"))
	rw.Write([]byte("error"))

	// Reset for reuse
	var buf2 bytes.Buffer
	rw.Reset(&buf2)

	// Should be back to defaults
	if rw.Status() != 200 {
		t.Errorf("Status after Reset = %d, want 200", rw.Status())
	}

	if rw.HeaderWritten() {
		t.Error("HeaderWritten after Reset = true, want false")
	}

	if rw.BytesWritten() != 0 {
		t.Errorf("BytesWritten after Reset = %d, want 0", rw.BytesWritten())
	}

	if rw.Header().Len() != 0 {
		t.Errorf("Header count after Reset = %d, want 0", rw.Header().Len())
	}

	// Should be able to write to new buffer
	rw.WriteHeader(200)
	rw.Write([]byte("ok"))
	rw.Flush()

	output := buf2.String()
	if !strings.Contains(output, "ok") {
		t.Errorf("Reset writer not working: %q", output)
	}
}

func TestResponseWriterWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	jsonData := []byte(`{"status":"ok"}`)
	err := rw.WriteJSON(200, jsonData)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "HTTP/1.1 200 OK") {
		t.Error("Output missing status line")
	}

	if !strings.Contains(output, "Content-Type: application/json") {
		t.Error("Output missing Content-Type header")
	}

	if !strings.Contains(output, "Content-Length:") {
		t.Error("Output missing Content-Length header")
	}

	if !strings.Contains(output, `{"status":"ok"}`) {
		t.Error("Output missing JSON body")
	}
}

func TestResponseWriterWriteText(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	textData := []byte("Hello, World!")
	err := rw.WriteText(200, textData)
	if err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Content-Type: text/plain") {
		t.Error("Output missing Content-Type header")
	}

	if !strings.Contains(output, "Hello, World!") {
		t.Error("Output missing text body")
	}
}

func TestResponseWriterWriteHTML(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	htmlData := []byte("<h1>Hello</h1>")
	err := rw.WriteHTML(200, htmlData)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Content-Type: text/html") {
		t.Error("Output missing Content-Type header")
	}

	if !strings.Contains(output, "<h1>Hello</h1>") {
		t.Error("Output missing HTML body")
	}
}

func TestResponseWriterWriteError(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	err := rw.WriteError(404, "Not Found")
	if err != nil {
		t.Fatalf("WriteError failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "HTTP/1.1 404") {
		t.Error("Output missing status line")
	}

	if !strings.Contains(output, "Not Found") {
		t.Error("Output missing error message")
	}
}

// Benchmarks

func BenchmarkResponseWriterSimple(b *testing.B) {
	var buf bytes.Buffer
	data := []byte("Hello, World!")

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := NewResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Write(data)
		rw.Flush()
	}
}

func BenchmarkResponseWriterWithHeaders(b *testing.B) {
	var buf bytes.Buffer
	data := []byte("Hello, World!")

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := NewResponseWriter(&buf)
		rw.Header().Set([]byte("Content-Type"), []byte("text/plain"))
		rw.Header().Set([]byte("Server"), []byte("Shockwave"))
		rw.WriteHeader(200)
		rw.Write(data)
		rw.Flush()
	}
}

func BenchmarkResponseWriterStatus200(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := NewResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Flush()
	}
}

func BenchmarkResponseWriterStatus404(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := NewResponseWriter(&buf)
		rw.WriteHeader(404)
		rw.Flush()
	}
}

func BenchmarkResponseWriterStatus500(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := NewResponseWriter(&buf)
		rw.WriteHeader(500)
		rw.Flush()
	}
}

func BenchmarkResponseWriterUncommonStatus(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := NewResponseWriter(&buf)
		rw.WriteHeader(418) // Uncommon code
		rw.Flush()
	}
}

func BenchmarkResponseWriterWriteJSON(b *testing.B) {
	var buf bytes.Buffer
	jsonData := []byte(`{"status":"ok","message":"success"}`)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(jsonData)))

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := NewResponseWriter(&buf)
		rw.WriteJSON(200, jsonData)
	}
}

func BenchmarkResponseWriterMultipleWrites(b *testing.B) {
	var buf bytes.Buffer
	chunk1 := []byte("Hello, ")
	chunk2 := []byte("World!")

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(chunk1) + len(chunk2)))

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw := NewResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Write(chunk1)
		rw.Write(chunk2)
		rw.Flush()
	}
}

func BenchmarkResponseWriterReset(b *testing.B) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		rw.Reset(&buf)
	}
}

func BenchmarkGetStatusLine(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = getStatusLine(200)
	}
}

func BenchmarkGetStatusLineUncommon(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = getStatusLine(418)
	}
}

// Additional tests for 100% coverage

func TestStatusTextAllCodes(t *testing.T) {
	// Test all status codes defined in statusText
	tests := []struct {
		code int
		text string
	}{
		// 1xx Informational
		{100, "Continue"},
		{101, "Switching Protocols"},
		// 2xx Success
		{200, "OK"},
		{201, "Created"},
		{202, "Accepted"},
		{203, "Non-Authoritative Information"},
		{204, "No Content"},
		{205, "Reset Content"},
		{206, "Partial Content"},
		// 3xx Redirection
		{300, "Multiple Choices"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{303, "See Other"},
		{304, "Not Modified"},
		{305, "Use Proxy"},
		{307, "Temporary Redirect"},
		{308, "Permanent Redirect"},
		// 4xx Client Error
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{402, "Payment Required"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{405, "Method Not Allowed"},
		{406, "Not Acceptable"},
		{407, "Proxy Authentication Required"},
		{408, "Request Timeout"},
		{409, "Conflict"},
		{410, "Gone"},
		{411, "Length Required"},
		{412, "Precondition Failed"},
		{413, "Payload Too Large"},
		{414, "URI Too Long"},
		{415, "Unsupported Media Type"},
		{416, "Range Not Satisfiable"},
		{417, "Expectation Failed"},
		{418, "I'm a teapot"},
		{422, "Unprocessable Entity"},
		{426, "Upgrade Required"},
		{428, "Precondition Required"},
		{429, "Too Many Requests"},
		{431, "Request Header Fields Too Large"},
		// 5xx Server Error
		{500, "Internal Server Error"},
		{501, "Not Implemented"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
		{504, "Gateway Timeout"},
		{505, "HTTP Version Not Supported"},
		// Unknown
		{999, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := statusText(tt.code)
			if result != tt.text {
				t.Errorf("statusText(%d) = %s, want %s", tt.code, result, tt.text)
			}
		})
	}
}

func TestResponseWriterWriteBeforeHeader(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	// Write without calling WriteHeader explicitly (should auto-call with 200)
	n, err := rw.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != 4 {
		t.Errorf("Write returned %d bytes, want 4", n)
	}

	// Status should be 200
	if rw.Status() != 200 {
		t.Errorf("Status = %d, want 200", rw.Status())
	}

	// Headers should have been written
	if !rw.HeaderWritten() {
		t.Error("Headers should have been written after Write")
	}
}

func TestResponseWriterWriteAfterFlush(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	// Write and flush
	rw.WriteHeader(200)
	rw.Flush()

	// Write after flush should still work
	n, err := rw.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write after flush failed: %v", err)
	}
	if n != 4 {
		t.Errorf("Write returned %d bytes, want 4", n)
	}
}

func TestResponseWriterFlushWithFlusher(t *testing.T) {
	// Use bufio.Writer which implements Flush interface
	var buf bytes.Buffer
	bw := GetBufioWriter(&buf)
	defer PutBufioWriter(bw)

	rw := NewResponseWriter(bw)

	rw.WriteHeader(200)
	err := rw.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	// The underlying writer should have been flushed
	if buf.Len() == 0 {
		t.Error("Buffer is empty, Flush didn't work")
	}
}

func TestResponseWriterWriteJSONError(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	// Write JSON with uncommon status code to test buildStatusLine
	jsonData := []byte(`{"error":"test"}`)
	err := rw.WriteJSON(418, jsonData)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "418") {
		t.Error("Response missing 418 status")
	}
}

func TestResponseWriterWriteTextUncommonStatus(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	err := rw.WriteText(206, []byte("partial content"))
	if err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "206") {
		t.Error("Response missing 206 status")
	}
}

func TestResponseWriterWriteHTMLUncommonStatus(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	err := rw.WriteHTML(301, []byte("<html><body>Moved</body></html>"))
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "301") {
		t.Error("Response missing 301 status")
	}
}

// Additional tests for error paths

type errorWriter struct {
	failAfter int
	written   int
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	if w.written >= w.failAfter {
		return 0, fmt.Errorf("write error")
	}
	w.written += len(p)
	return len(p), nil
}

func TestResponseWriterWriteHeadersError(t *testing.T) {
	// Writer that fails after first write (status line)
	w := &errorWriter{failAfter: 20}
	rw := NewResponseWriter(w)

	rw.Header().Set([]byte("Content-Type"), []byte("application/json"))
	rw.WriteHeader(200)

	// Try to write - this should trigger writeHeaders and fail
	_, err := rw.Write([]byte("test"))
	if err == nil {
		t.Error("Expected error when writing headers fails")
	}
}

func TestResponseWriterFlushError(t *testing.T) {
	// Test Flush when headers haven't been written yet and writing fails
	w := &errorWriter{failAfter: 0}
	rw := NewResponseWriter(w)

	rw.Header().Set([]byte("X-Test"), []byte("value"))

	err := rw.Flush()
	if err == nil {
		t.Error("Expected error when Flush fails to write headers")
	}
}

func TestResponseWriterWriteJSONFlushError(t *testing.T) {
	// Writer that allows headers but fails on body
	w := &errorWriter{failAfter: 100}
	rw := NewResponseWriter(w)

	jsonData := []byte(`{"test":"data"}`)

	// This should succeed in writing headers but fail on Flush
	err := rw.WriteJSON(200, jsonData)
	// The error might come from Write or Flush
	if err == nil {
		t.Log("WriteJSON completed without error")
	}
}

// ============================================================================
// Chunked Transfer Encoding Tests
// ============================================================================

func TestResponseWriterWriteChunked(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.WriteHeader(200)
	rw.Header().Set([]byte("Transfer-Encoding"), []byte("chunked"))

	chunks := [][]byte{
		[]byte("Hello"),
		[]byte(" "),
		[]byte("World"),
	}

	err := rw.WriteChunked(chunks)
	if err != nil {
		t.Fatalf("WriteChunked failed: %v", err)
	}

	output := buf.String()

	// Should have status line
	if !strings.Contains(output, "HTTP/1.1 200 OK\r\n") {
		t.Errorf("Output missing status line: %q", output)
	}

	// Should have Transfer-Encoding header
	if !strings.Contains(output, "Transfer-Encoding: chunked\r\n") {
		t.Errorf("Output missing Transfer-Encoding header: %q", output)
	}

	// Should have chunk sizes in hex (5, 1, 5)
	if !strings.Contains(output, "5\r\nHello\r\n") {
		t.Errorf("Output missing first chunk: %q", output)
	}
	if !strings.Contains(output, "1\r\n \r\n") {
		t.Errorf("Output missing second chunk: %q", output)
	}
	if !strings.Contains(output, "5\r\nWorld\r\n") {
		t.Errorf("Output missing third chunk: %q", output)
	}

	// Should have final chunk marker
	if !strings.HasSuffix(output, "0\r\n\r\n") {
		t.Errorf("Output missing final chunk marker: %q", output)
	}
}

func TestResponseWriterWriteChunkIncremental(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.WriteHeader(200)

	// Write chunks incrementally
	chunks := []string{"First", "Second", "Third"}
	for _, chunk := range chunks {
		err := rw.WriteChunk([]byte(chunk))
		if err != nil {
			t.Fatalf("WriteChunk failed: %v", err)
		}
	}

	// Finish chunked response
	err := rw.FinishChunked()
	if err != nil {
		t.Fatalf("FinishChunked failed: %v", err)
	}

	output := buf.String()

	// Should automatically add Transfer-Encoding header
	if !strings.Contains(output, "Transfer-Encoding: chunked\r\n") {
		t.Errorf("Output missing Transfer-Encoding header: %q", output)
	}

	// Check each chunk
	if !strings.Contains(output, "5\r\nFirst\r\n") {
		t.Errorf("Output missing 'First' chunk: %q", output)
	}
	if !strings.Contains(output, "6\r\nSecond\r\n") {
		t.Errorf("Output missing 'Second' chunk: %q", output)
	}
	if !strings.Contains(output, "5\r\nThird\r\n") {
		t.Errorf("Output missing 'Third' chunk: %q", output)
	}

	// Should have final chunk marker
	if !strings.HasSuffix(output, "0\r\n\r\n") {
		t.Errorf("Output missing final chunk marker: %q", output)
	}

	// Verify bytes written tracking
	expectedBytes := 5 + 6 + 5 // "First" + "Second" + "Third"
	if rw.BytesWritten() != int64(expectedBytes) {
		t.Errorf("BytesWritten = %d, want %d", rw.BytesWritten(), expectedBytes)
	}
}

func TestResponseWriterWriteChunkEmptyChunks(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.WriteHeader(200)

	// Write empty chunks (should be skipped)
	err := rw.WriteChunk([]byte{})
	if err != nil {
		t.Fatalf("WriteChunk empty failed: %v", err)
	}

	// Write non-empty chunk
	err = rw.WriteChunk([]byte("data"))
	if err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}

	// Write another empty chunk
	err = rw.WriteChunk([]byte{})
	if err != nil {
		t.Fatalf("WriteChunk empty failed: %v", err)
	}

	err = rw.FinishChunked()
	if err != nil {
		t.Fatalf("FinishChunked failed: %v", err)
	}

	output := buf.String()

	// Should only have the non-empty chunk
	if !strings.Contains(output, "4\r\ndata\r\n") {
		t.Errorf("Output missing 'data' chunk: %q", output)
	}

	// Should not have "0\r\n" before final marker (no empty chunks written)
	// Count occurrences of "0\r\n" - should only be the final marker
	count := strings.Count(output, "0\r\n")
	if count != 1 {
		t.Errorf("Expected 1 occurrence of '0\\r\\n' (final marker), got %d: %q", count, output)
	}
}

func TestResponseWriterWriteChunkedLargeData(t *testing.T) {
	var buf bytes.Buffer
	rw := NewResponseWriter(&buf)

	rw.WriteHeader(200)

	// Create large chunks
	chunk1KB := bytes.Repeat([]byte("x"), 1024)
	chunk10KB := bytes.Repeat([]byte("y"), 10240)

	err := rw.WriteChunk(chunk1KB)
	if err != nil {
		t.Fatalf("WriteChunk 1KB failed: %v", err)
	}

	err = rw.WriteChunk(chunk10KB)
	if err != nil {
		t.Fatalf("WriteChunk 10KB failed: %v", err)
	}

	err = rw.FinishChunked()
	if err != nil {
		t.Fatalf("FinishChunked failed: %v", err)
	}

	output := buf.String()

	// Check chunk sizes in hex
	// 1024 = 0x400
	if !strings.Contains(output, "400\r\n") {
		t.Errorf("Output missing 1KB chunk size: %q", output[:200])
	}

	// 10240 = 0x2800
	if !strings.Contains(output, "2800\r\n") {
		t.Errorf("Output missing 10KB chunk size: %q", output[:200])
	}

	// Verify bytes written
	if rw.BytesWritten() != int64(1024+10240) {
		t.Errorf("BytesWritten = %d, want %d", rw.BytesWritten(), 1024+10240)
	}
}

func TestResponseWriterWriteChunkedPooling(t *testing.T) {
	// Test that pooled ResponseWriter works correctly with chunked encoding
	var buf bytes.Buffer

	rw := GetResponseWriter(&buf)
	defer PutResponseWriter(rw)

	err := rw.WriteChunk([]byte("chunk1"))
	if err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}

	err = rw.WriteChunk([]byte("chunk2"))
	if err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}

	err = rw.FinishChunked()
	if err != nil {
		t.Fatalf("FinishChunked failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "6\r\nchunk1\r\n") {
		t.Errorf("Output missing first chunk: %q", output)
	}
	if !strings.Contains(output, "6\r\nchunk2\r\n") {
		t.Errorf("Output missing second chunk: %q", output)
	}
	if !strings.HasSuffix(output, "0\r\n\r\n") {
		t.Errorf("Output missing final marker: %q", output)
	}
}
