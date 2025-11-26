package http11

import (
	"strings"
	"testing"
)

func TestRequestMethod(t *testing.T) {
	req := &Request{
		MethodID: MethodGET,
	}

	if req.Method() != "GET" {
		t.Errorf("Method() = %q, want %q", req.Method(), "GET")
	}

	if !req.IsGET() {
		t.Error("IsGET() = false, want true")
	}

	if req.IsPOST() {
		t.Error("IsPOST() = true, want false")
	}
}

func TestRequestMethodBytes(t *testing.T) {
	methodBytes := []byte("POST")
	req := &Request{
		MethodID:    MethodPOST,
		methodBytes: methodBytes,
	}

	result := req.MethodBytes()
	if string(result) != "POST" {
		t.Errorf("MethodBytes() = %q, want %q", result, "POST")
	}

	// Verify it's the same slice (zero-copy)
	if &result[0] != &methodBytes[0] {
		t.Error("MethodBytes() returned a copy, expected zero-copy slice")
	}
}

func TestRequestPath(t *testing.T) {
	pathBytes := []byte("/api/users")
	req := &Request{
		pathBytes: pathBytes,
	}

	// Test Path() (allocates string)
	path := req.Path()
	if path != "/api/users" {
		t.Errorf("Path() = %q, want %q", path, "/api/users")
	}

	// Test PathBytes() (zero-copy)
	pathBytesResult := req.PathBytes()
	if string(pathBytesResult) != "/api/users" {
		t.Errorf("PathBytes() = %q, want %q", pathBytesResult, "/api/users")
	}

	// Verify zero-copy
	if &pathBytesResult[0] != &pathBytes[0] {
		t.Error("PathBytes() returned a copy, expected zero-copy slice")
	}
}

func TestRequestQuery(t *testing.T) {
	queryBytes := []byte("id=123&name=test")
	req := &Request{
		queryBytes: queryBytes,
	}

	// Test Query() (allocates string)
	query := req.Query()
	if query != "id=123&name=test" {
		t.Errorf("Query() = %q, want %q", query, "id=123&name=test")
	}

	// Test QueryBytes() (zero-copy)
	queryBytesResult := req.QueryBytes()
	if string(queryBytesResult) != "id=123&name=test" {
		t.Errorf("QueryBytes() = %q, want %q", queryBytesResult, "id=123&name=test")
	}

	// Verify zero-copy
	if &queryBytesResult[0] != &queryBytes[0] {
		t.Error("QueryBytes() returned a copy, expected zero-copy slice")
	}
}

func TestRequestNoQuery(t *testing.T) {
	req := &Request{
		pathBytes:  []byte("/api/users"),
		queryBytes: nil,
	}

	if req.Query() != "" {
		t.Errorf("Query() = %q, want empty string", req.Query())
	}

	if req.QueryBytes() != nil {
		t.Errorf("QueryBytes() = %v, want nil", req.QueryBytes())
	}
}

func TestRequestParsedURL(t *testing.T) {
	req := &Request{
		pathBytes:  []byte("/api/users"),
		queryBytes: []byte("id=123&name=test"),
	}

	// First call should parse
	parsed, err := req.ParsedURL()
	if err != nil {
		t.Fatalf("ParsedURL() error: %v", err)
	}

	if parsed.Path != "/api/users" {
		t.Errorf("parsed.Path = %q, want %q", parsed.Path, "/api/users")
	}

	if parsed.RawQuery != "id=123&name=test" {
		t.Errorf("parsed.RawQuery = %q, want %q", parsed.RawQuery, "id=123&name=test")
	}

	// Second call should return cached result (same pointer)
	parsed2, err := req.ParsedURL()
	if err != nil {
		t.Fatalf("ParsedURL() second call error: %v", err)
	}

	if parsed != parsed2 {
		t.Error("ParsedURL() second call returned different pointer (not cached)")
	}
}

func TestRequestParsedURLNoQuery(t *testing.T) {
	req := &Request{
		pathBytes:  []byte("/api/users"),
		queryBytes: nil,
	}

	parsed, err := req.ParsedURL()
	if err != nil {
		t.Fatalf("ParsedURL() error: %v", err)
	}

	if parsed.Path != "/api/users" {
		t.Errorf("parsed.Path = %q, want %q", parsed.Path, "/api/users")
	}

	if parsed.RawQuery != "" {
		t.Errorf("parsed.RawQuery = %q, want empty", parsed.RawQuery)
	}
}

func TestRequestHeaders(t *testing.T) {
	req := &Request{}
	req.Header.Add([]byte("Content-Type"), []byte("application/json"))
	req.Header.Add([]byte("Content-Length"), []byte("123"))

	// Test GetHeader (zero-copy)
	val := req.GetHeader([]byte("Content-Type"))
	if string(val) != "application/json" {
		t.Errorf("GetHeader(Content-Type) = %q, want %q", val, "application/json")
	}

	// Test GetHeaderString
	valStr := req.GetHeaderString("Content-Length")
	if valStr != "123" {
		t.Errorf("GetHeaderString(Content-Length) = %q, want %q", valStr, "123")
	}

	// Test HasHeader
	if !req.HasHeader([]byte("Content-Type")) {
		t.Error("HasHeader(Content-Type) = false, want true")
	}

	if req.HasHeader([]byte("X-Not-Exists")) {
		t.Error("HasHeader(X-Not-Exists) = true, want false")
	}
}

func TestRequestMethodCheckers(t *testing.T) {
	tests := []struct {
		methodID uint8
		checker  func(*Request) bool
		name     string
	}{
		{MethodGET, (*Request).IsGET, "IsGET"},
		{MethodPOST, (*Request).IsPOST, "IsPOST"},
		{MethodPUT, (*Request).IsPUT, "IsPUT"},
		{MethodDELETE, (*Request).IsDELETE, "IsDELETE"},
		{MethodPATCH, (*Request).IsPATCH, "IsPATCH"},
		{MethodHEAD, (*Request).IsHEAD, "IsHEAD"},
		{MethodOPTIONS, (*Request).IsOPTIONS, "IsOPTIONS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{MethodID: tt.methodID}

			if !tt.checker(req) {
				t.Errorf("%s() = false, want true for method %d", tt.name, tt.methodID)
			}

			// Check it returns false for different method
			otherReq := &Request{MethodID: MethodGET}
			if tt.methodID != MethodGET && tt.checker(otherReq) {
				t.Errorf("%s() = true for GET, want false", tt.name)
			}
		})
	}
}

func TestRequestHasBody(t *testing.T) {
	tests := []struct {
		name              string
		contentLength     int64
		transferEncoding  []string
		expectedHasBody   bool
		expectedIsChunked bool
	}{
		{
			name:              "No body",
			contentLength:     0,
			transferEncoding:  nil,
			expectedHasBody:   false,
			expectedIsChunked: false,
		},
		{
			name:              "Content-Length body",
			contentLength:     123,
			transferEncoding:  nil,
			expectedHasBody:   true,
			expectedIsChunked: false,
		},
		{
			name:              "Chunked encoding",
			contentLength:     -1,
			transferEncoding:  []string{"chunked"},
			expectedHasBody:   true,
			expectedIsChunked: true,
		},
		{
			name:              "Chunked with gzip",
			contentLength:     -1,
			transferEncoding:  []string{"gzip", "chunked"},
			expectedHasBody:   true,
			expectedIsChunked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{
				ContentLength:    tt.contentLength,
				TransferEncoding: tt.transferEncoding,
			}

			if req.HasBody() != tt.expectedHasBody {
				t.Errorf("HasBody() = %v, want %v", req.HasBody(), tt.expectedHasBody)
			}

			if req.IsChunked() != tt.expectedIsChunked {
				t.Errorf("IsChunked() = %v, want %v", req.IsChunked(), tt.expectedIsChunked)
			}
		})
	}
}

func TestRequestShouldClose(t *testing.T) {
	req := &Request{Close: false}
	if req.ShouldClose() {
		t.Error("ShouldClose() = true, want false")
	}

	req.Close = true
	if !req.ShouldClose() {
		t.Error("ShouldClose() = false, want true")
	}
}

func TestRequestReset(t *testing.T) {
	// Create a request with all fields populated
	req := &Request{
		MethodID:         MethodPOST,
		methodBytes:      []byte("POST"),
		pathBytes:        []byte("/api/users"),
		queryBytes:       []byte("id=123"),
		protoBytes:       []byte("HTTP/1.1"),
		Proto:            "HTTP/1.1",
		ProtoMajor:       1,
		ProtoMinor:       1,
		ContentLength:    100,
		TransferEncoding: []string{"chunked"},
		Close:            true,
		RemoteAddr:       "192.168.1.1:1234",
		Body:             strings.NewReader("test"),
		buf:              []byte("buffer"),
	}
	req.Header.Add([]byte("Content-Type"), []byte("application/json"))

	// Reset
	req.Reset()

	// Verify all fields are cleared
	if req.MethodID != 0 {
		t.Errorf("MethodID after Reset = %d, want 0", req.MethodID)
	}
	if req.methodBytes != nil {
		t.Error("methodBytes after Reset != nil")
	}
	if req.pathBytes != nil {
		t.Error("pathBytes after Reset != nil")
	}
	if req.queryBytes != nil {
		t.Error("queryBytes after Reset != nil")
	}
	if req.protoBytes != nil {
		t.Error("protoBytes after Reset != nil")
	}
	if req.pathParsed != nil {
		t.Error("pathParsed after Reset != nil")
	}
	if req.Header.Len() != 0 {
		t.Errorf("Header.Len() after Reset = %d, want 0", req.Header.Len())
	}
	if req.Body != nil {
		t.Error("Body after Reset != nil")
	}
	if req.Proto != "" {
		t.Errorf("Proto after Reset = %q, want empty", req.Proto)
	}
	if req.ProtoMajor != 0 {
		t.Errorf("ProtoMajor after Reset = %d, want 0", req.ProtoMajor)
	}
	if req.ProtoMinor != 0 {
		t.Errorf("ProtoMinor after Reset = %d, want 0", req.ProtoMinor)
	}
	if req.ContentLength != 0 {
		t.Errorf("ContentLength after Reset = %d, want 0", req.ContentLength)
	}
	if req.TransferEncoding != nil {
		t.Error("TransferEncoding after Reset != nil")
	}
	if req.Close {
		t.Error("Close after Reset = true, want false")
	}
	if req.RemoteAddr != "" {
		t.Errorf("RemoteAddr after Reset = %q, want empty", req.RemoteAddr)
	}
	if req.buf != nil {
		t.Error("buf after Reset != nil")
	}
}

func TestRequestClone(t *testing.T) {
	// Create a request with various fields
	original := &Request{
		MethodID:      MethodPOST,
		methodBytes:   []byte("POST"),
		pathBytes:     []byte("/api/users"),
		queryBytes:    []byte("id=123"),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: 100,
		Close:         true,
		RemoteAddr:    "192.168.1.1:1234",
		Body:          strings.NewReader("test"),
	}
	original.Header.Add([]byte("Content-Type"), []byte("application/json"))
	original.Header.Add([]byte("X-Custom"), []byte("value"))

	// Clone
	clone := original.Clone()

	// Verify basic fields
	if clone.MethodID != original.MethodID {
		t.Errorf("clone.MethodID = %d, want %d", clone.MethodID, original.MethodID)
	}
	if clone.Method() != original.Method() {
		t.Errorf("clone.Method() = %q, want %q", clone.Method(), original.Method())
	}
	if clone.Path() != original.Path() {
		t.Errorf("clone.Path() = %q, want %q", clone.Path(), original.Path())
	}
	if clone.Query() != original.Query() {
		t.Errorf("clone.Query() = %q, want %q", clone.Query(), original.Query())
	}
	if clone.Proto != original.Proto {
		t.Errorf("clone.Proto = %q, want %q", clone.Proto, original.Proto)
	}
	if clone.ContentLength != original.ContentLength {
		t.Errorf("clone.ContentLength = %d, want %d", clone.ContentLength, original.ContentLength)
	}
	if clone.Close != original.Close {
		t.Errorf("clone.Close = %v, want %v", clone.Close, original.Close)
	}

	// Verify headers are cloned
	if clone.Header.Len() != original.Header.Len() {
		t.Errorf("clone.Header.Len() = %d, want %d", clone.Header.Len(), original.Header.Len())
	}
	val := clone.GetHeaderString("Content-Type")
	if val != "application/json" {
		t.Errorf("clone header Content-Type = %q, want %q", val, "application/json")
	}

	// Verify buf is NOT cloned (should be nil)
	if clone.buf != nil {
		t.Error("clone.buf should be nil (not copied)")
	}

	// Verify Body is NOT cloned
	if clone.Body != nil {
		t.Error("clone.Body should be nil (not copied)")
	}

	// Verify slices are NOT the same (new allocations)
	if len(clone.pathBytes) > 0 && len(original.pathBytes) > 0 {
		if &clone.pathBytes[0] == &original.pathBytes[0] {
			t.Error("clone.pathBytes points to same memory as original (should be new allocation)")
		}
	}
}

func TestRequestCloneModification(t *testing.T) {
	// Create original
	original := &Request{
		MethodID:   MethodGET,
		pathBytes:  []byte("/original"),
		queryBytes: []byte("original=true"),
	}
	original.Header.Add([]byte("X-Original"), []byte("original-value"))

	// Clone
	clone := original.Clone()

	// Modify clone's header
	clone.Header.Set([]byte("X-Original"), []byte("modified-value"))
	clone.Header.Add([]byte("X-New"), []byte("new-value"))

	// Original should be unchanged
	originalVal := original.GetHeaderString("X-Original")
	if originalVal != "original-value" {
		t.Errorf("original header modified after clone change: got %q, want %q", originalVal, "original-value")
	}

	if original.HasHeader([]byte("X-New")) {
		t.Error("original has X-New header after adding to clone (clone not independent)")
	}
}

// Benchmarks

func BenchmarkRequestMethod(b *testing.B) {
	req := &Request{MethodID: MethodGET}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = req.Method()
	}
}

func BenchmarkRequestPathBytes(b *testing.B) {
	req := &Request{pathBytes: []byte("/api/users/123")}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = req.PathBytes()
	}
}

func BenchmarkRequestPath(b *testing.B) {
	req := &Request{pathBytes: []byte("/api/users/123")}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = req.Path()
	}
}

func BenchmarkRequestIsGET(b *testing.B) {
	req := &Request{MethodID: MethodGET}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = req.IsGET()
	}
}

func BenchmarkRequestGetHeader(b *testing.B) {
	req := &Request{}
	req.Header.Add([]byte("Content-Type"), []byte("application/json"))

	name := []byte("Content-Type")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = req.GetHeader(name)
	}
}

func BenchmarkRequestParsedURL(b *testing.B) {
	req := &Request{
		pathBytes:  []byte("/api/users"),
		queryBytes: []byte("id=123&name=test"),
	}

	// Pre-parse to test cached access
	req.ParsedURL()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = req.ParsedURL()
	}
}

func BenchmarkRequestParsedURLFirstCall(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		req := &Request{
			pathBytes:  []byte("/api/users"),
			queryBytes: []byte("id=123&name=test"),
		}
		b.StartTimer()

		_, _ = req.ParsedURL()
	}
}

func BenchmarkRequestReset(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := &Request{
			MethodID:      MethodPOST,
			methodBytes:   []byte("POST"),
			pathBytes:     []byte("/api/users"),
			queryBytes:    []byte("id=123"),
			Proto:         "HTTP/1.1",
			ContentLength: 100,
		}
		req.Header.Add([]byte("Content-Type"), []byte("application/json"))

		req.Reset()
	}
}

func BenchmarkRequestClone(b *testing.B) {
	original := &Request{
		MethodID:      MethodPOST,
		methodBytes:   []byte("POST"),
		pathBytes:     []byte("/api/users"),
		queryBytes:    []byte("id=123"),
		Proto:         "HTTP/1.1",
		ContentLength: 100,
	}
	original.Header.Add([]byte("Content-Type"), []byte("application/json"))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = original.Clone()
	}
}

// Additional tests for 100% coverage

func TestRequestCloneWithTransferEncoding(t *testing.T) {
	original := &Request{
		MethodID:         MethodPOST,
		methodBytes:      []byte("POST"),
		pathBytes:        []byte("/api/test"),
		TransferEncoding: []string{"chunked", "gzip"},
	}

	clone := original.Clone()

	if len(clone.TransferEncoding) != len(original.TransferEncoding) {
		t.Errorf("Clone TransferEncoding length = %d, want %d",
			len(clone.TransferEncoding), len(original.TransferEncoding))
	}

	for i, enc := range original.TransferEncoding {
		if clone.TransferEncoding[i] != enc {
			t.Errorf("Clone TransferEncoding[%d] = %s, want %s",
				i, clone.TransferEncoding[i], enc)
		}
	}

	// Note: TransferEncoding is shallow copied (shares underlying array)
	// This is intentional behavior for read-only usage
}

func TestRequestParsedURLInvalidPath(t *testing.T) {
	// Create a request with invalid URL path
	req := &Request{
		pathBytes: []byte("://invalid"),
	}

	_, err := req.ParsedURL()
	if err == nil {
		t.Error("ParsedURL should return error for invalid path")
	}
}

func TestRequestCloneWithParsedURL(t *testing.T) {
	req := &Request{
		pathBytes:  []byte("/api/test"),
		queryBytes: []byte("id=123"),
	}

	// Parse URL first
	_, err := req.ParsedURL()
	if err != nil {
		t.Fatalf("ParsedURL failed: %v", err)
	}

	// Now clone - should also clone the parsed URL
	clone := req.Clone()

	// Clone should have parsed URL
	if clone.pathParsed == nil {
		t.Error("Clone should have parsed URL")
	}

	// Verify they're different objects
	if req.pathParsed == clone.pathParsed {
		t.Error("Clone pathParsed should be different object")
	}
}
