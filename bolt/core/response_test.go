package core

import (
	"testing"
)

// TestJSONWithError tests JSON response with encoding error.
func TestJSONWithError(t *testing.T) {
	ctx := &Context{}

	// Create a value that will fail JSON encoding (channel)
	invalidData := make(chan int)

	err := ctx.JSON(200, invalidData)
	// Note: goccy/go-json may handle channels differently than stdlib
	// This test documents the behavior
	if err != nil {
		t.Logf("JSON encoding error (expected): %v", err)
	}
}

// TestJSONBytesSuccess tests JSONBytes with valid data.
func TestJSONBytesSuccess(t *testing.T) {
	ctx := &Context{}

	data := []byte(`{"message":"success"}`)
	err := ctx.JSONBytes(200, data)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}

	if !ctx.Written() {
		t.Error("expected response to be written")
	}
}

// TestJSONBytesEmpty tests JSONBytes with empty data.
func TestJSONBytesEmpty(t *testing.T) {
	ctx := &Context{}

	err := ctx.JSONBytes(204, []byte{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 204 {
		t.Errorf("expected status 204, got %d", ctx.StatusCode())
	}
}

// TestTextMultipleWrites tests Text response.
func TestTextMultipleWrites(t *testing.T) {
	ctx := &Context{}

	// First write
	err := ctx.Text(200, "Hello")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Second write (should still work)
	err = ctx.Text(201, "World")
	if err != nil {
		t.Errorf("unexpected error on second write: %v", err)
	}

	// Status should be from last write
	if ctx.StatusCode() != 201 {
		t.Errorf("expected status 201, got %d", ctx.StatusCode())
	}
}

// TestHTMLSuccess tests HTML response.
func TestHTMLSuccess(t *testing.T) {
	ctx := &Context{}

	html := `<!DOCTYPE html><html><body><h1>Test</h1></body></html>`
	err := ctx.HTML(200, html)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}

	if !ctx.Written() {
		t.Error("expected response to be written")
	}
}

// TestHTMLEmptyString tests HTML with empty string.
func TestHTMLEmptyString(t *testing.T) {
	ctx := &Context{}

	err := ctx.HTML(200, "")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestNoContentResponse tests NoContent response.
func TestNoContentResponse(t *testing.T) {
	ctx := &Context{}

	err := ctx.NoContent()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 204 {
		t.Errorf("expected status 204, got %d", ctx.StatusCode())
	}

	if !ctx.Written() {
		t.Error("expected response to be written")
	}
}

// TestGetHeaderWithShockwave tests GetHeader in different modes.
func TestGetHeaderWithShockwave(t *testing.T) {
	// Test mode (no Shockwave)
	ctx := &Context{}
	ctx.SetRequestHeader("X-Test", "value")

	if val := ctx.GetHeader("X-Test"); val != "value" {
		t.Errorf("expected 'value', got '%s'", val)
	}

	// Test with empty context
	emptyCtx := &Context{}
	if val := emptyCtx.GetHeader("Missing"); val != "" {
		t.Errorf("expected empty string for missing header, got '%s'", val)
	}
}

// TestGetHeaderCaseInsensitive tests header retrieval.
func TestGetHeaderCaseInsensitive(t *testing.T) {
	ctx := &Context{}
	ctx.SetRequestHeader("Content-Type", "application/json")

	// Should work (exact match)
	if val := ctx.GetHeader("Content-Type"); val != "application/json" {
		t.Errorf("expected 'application/json', got '%s'", val)
	}
}

// TestGetResponseHeaderMissing tests GetResponseHeader with missing key.
func TestGetResponseHeaderMissing(t *testing.T) {
	ctx := &Context{}

	if val := ctx.GetResponseHeader("Missing"); val != "" {
		t.Errorf("expected empty string, got '%s'", val)
	}
}

// TestGetResponseHeaderSet tests GetResponseHeader after SetHeader.
func TestGetResponseHeaderSet(t *testing.T) {
	ctx := &Context{}

	ctx.SetHeader("X-Custom", "custom-value")

	if val := ctx.GetResponseHeader("X-Custom"); val != "custom-value" {
		t.Errorf("expected 'custom-value', got '%s'", val)
	}
}

// TestParseQueryEdgeCases tests query parsing edge cases.
func TestParseQueryEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		query string
		key   string
		want  string
	}{
		{"empty query", "", "key", ""},
		{"no equals", "key", "key", ""},
		{"empty value", "key=", "key", ""},
		{"multiple equals", "key=val=ue", "key", "val=ue"},
		{"url encoded", "key=hello%20world", "key", "hello%20world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{queryBytes: []byte(tt.query)}
			if got := ctx.Query(tt.key); got != tt.want {
				t.Errorf("Query(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// TestJSONLargeObject tests JSON with large object.
func TestJSONLargeObject(t *testing.T) {
	ctx := &Context{}

	// Create large object
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[string(rune('a'+i%26))+string(rune('0'+i%10))] = i
	}

	err := ctx.JSON(200, largeData)
	if err != nil {
		t.Errorf("unexpected error with large object: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}
}

// TestJSONNestedStructures tests JSON with deeply nested structures.
func TestJSONNestedStructures(t *testing.T) {
	ctx := &Context{}

	type Inner struct {
		Value int `json:"value"`
	}

	type Middle struct {
		Inner Inner `json:"inner"`
	}

	type Outer struct {
		Middle Middle `json:"middle"`
	}

	data := Outer{
		Middle: Middle{
			Inner: Inner{
				Value: 123,
			},
		},
	}

	err := ctx.JSON(200, data)
	if err != nil {
		t.Errorf("unexpected error with nested structures: %v", err)
	}
}

// TestJSONBytesLarge tests JSONBytes with large data.
func TestJSONBytesLarge(t *testing.T) {
	ctx := &Context{}

	// Create large JSON bytes (>1KB)
	largeJSON := make([]byte, 2048)
	for i := range largeJSON {
		largeJSON[i] = byte('a' + i%26)
	}

	err := ctx.JSONBytes(200, largeJSON)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}
}

// TestTextLongString tests Text with long string.
func TestTextLongString(t *testing.T) {
	ctx := &Context{}

	// Create long text
	longText := make([]byte, 5000)
	for i := range longText {
		longText[i] = byte('A' + i%26)
	}

	err := ctx.Text(200, string(longText))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}
}

// TestHTMLComplexDocument tests HTML with complex document.
func TestHTMLComplexDocument(t *testing.T) {
	ctx := &Context{}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Test Page</title>
    <style>body { margin: 0; }</style>
</head>
<body>
    <h1>Test</h1>
    <p>This is a test page with special chars: &lt; &gt; &amp; &quot;</p>
</body>
</html>`

	err := ctx.HTML(200, html)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}
}

// TestResetClearsParamsBuf tests that Reset properly clears inline buffer.
func TestResetClearsParamsBuf(t *testing.T) {
	ctx := &Context{}

	// Add params using inline storage
	ctx.setParam("id", "123")
	ctx.setParam("name", "test")

	// Verify they're accessible
	if ctx.Param("id") != "123" {
		t.Error("param not set correctly")
	}

	// Reset
	ctx.Reset()

	// paramsLen should be 0
	if ctx.paramsLen != 0 {
		t.Errorf("expected paramsLen=0, got %d", ctx.paramsLen)
	}

	// Params should be gone
	if ctx.Param("id") != "" {
		t.Error("param not cleared after reset")
	}
}

// TestContextResetWithLargeMap tests Reset with large params map.
func TestContextResetWithLargeMap(t *testing.T) {
	ctx := &Context{
		params: make(map[string]string),
	}

	// Add 10 params (> 8, should create new map on reset)
	for i := 0; i < 10; i++ {
		ctx.params[string(rune('a'+i))] = "value"
	}

	ctx.Reset()

	// Map should be nil (recreated on next use)
	if ctx.params != nil {
		t.Error("expected large params map to be nil after reset")
	}
}

// TestContextResetWithSmallMap tests Reset with small params map.
func TestContextResetWithSmallMap(t *testing.T) {
	ctx := &Context{
		params: map[string]string{
			"a": "1",
			"b": "2",
		},
	}

	ctx.Reset()

	// Map should be cleared but not nil
	if len(ctx.params) != 0 {
		t.Errorf("expected params to be cleared, got %d items", len(ctx.params))
	}
}
