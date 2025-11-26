package core

import (
	"errors"
	"testing"
)

// Test types
type TestUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TestOK tests the OK constructor.
func TestOK(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	data := OK(user)

	if data.Status != 200 {
		t.Errorf("expected status 200, got %d", data.Status)
	}
	if data.Value.ID != 1 {
		t.Errorf("expected ID 1, got %d", data.Value.ID)
	}
	if data.Error != nil {
		t.Errorf("expected no error, got %v", data.Error)
	}
}

// TestCreated tests the Created constructor.
func TestCreated(t *testing.T) {
	user := TestUser{ID: 2, Name: "Bob"}
	data := Created(user)

	if data.Status != 201 {
		t.Errorf("expected status 201, got %d", data.Status)
	}
	if data.Value.Name != "Bob" {
		t.Errorf("expected name Bob, got %s", data.Value.Name)
	}
}

// TestAccepted tests the Accepted constructor.
func TestAccepted(t *testing.T) {
	user := TestUser{ID: 3, Name: "Charlie"}
	data := Accepted(user)

	if data.Status != 202 {
		t.Errorf("expected status 202, got %d", data.Status)
	}
}

// TestNoContent tests the NoContent constructor.
func TestNoContent(t *testing.T) {
	data := NoContent[TestUser]()

	if data.Status != 204 {
		t.Errorf("expected status 204, got %d", data.Status)
	}
}

// TestBadRequest tests the BadRequest constructor.
func TestBadRequest(t *testing.T) {
	err := errors.New("validation failed")
	data := BadRequest[TestUser](err)

	if data.Status != 400 {
		t.Errorf("expected status 400, got %d", data.Status)
	}
	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if data.Error.Error() != "validation failed" {
		t.Errorf("expected error 'validation failed', got '%s'", data.Error.Error())
	}
}

// TestUnauthorized tests the Unauthorized constructor.
func TestUnauthorized(t *testing.T) {
	err := errors.New("invalid token")
	data := Unauthorized[TestUser](err)

	if data.Status != 401 {
		t.Errorf("expected status 401, got %d", data.Status)
	}
	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestForbidden tests the Forbidden constructor.
func TestForbidden(t *testing.T) {
	err := errors.New("insufficient permissions")
	data := Forbidden[TestUser](err)

	if data.Status != 403 {
		t.Errorf("expected status 403, got %d", data.Status)
	}
	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestNotFound tests the NotFound constructor.
func TestNotFound(t *testing.T) {
	err := errors.New("user not found")
	data := NotFound[TestUser](err)

	if data.Status != 404 {
		t.Errorf("expected status 404, got %d", data.Status)
	}
	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestConflict tests the Conflict constructor.
func TestConflict(t *testing.T) {
	err := errors.New("user already exists")
	data := Conflict[TestUser](err)

	if data.Status != 409 {
		t.Errorf("expected status 409, got %d", data.Status)
	}
	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestInternalServerError tests the InternalServerError constructor.
func TestInternalServerError(t *testing.T) {
	err := errors.New("database error")
	data := InternalServerError[TestUser](err)

	if data.Status != 500 {
		t.Errorf("expected status 500, got %d", data.Status)
	}
	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestInternalError tests the InternalError alias.
func TestInternalError(t *testing.T) {
	err := errors.New("internal error")
	data := InternalError[TestUser](err)

	if data.Status != 500 {
		t.Errorf("expected status 500, got %d", data.Status)
	}
	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if data.Error.Error() != "internal error" {
		t.Errorf("expected error 'internal error', got '%s'", data.Error.Error())
	}
}

// TestResult tests the Result[T] wrapper.
func TestResult(t *testing.T) {
	// Test success case
	successData := OK(TestUser{ID: 1, Name: "Alice"})
	successResult := Result[TestUser]{
		Data: &successData,
		Err:  nil,
	}

	if successResult.Data == nil {
		t.Fatal("expected data, got nil")
	}
	if successResult.Err != nil {
		t.Errorf("expected no error, got %v", successResult.Err)
	}
	if successResult.Data.Value.ID != 1 {
		t.Errorf("expected ID 1, got %d", successResult.Data.Value.ID)
	}

	// Test error case
	errorResult := Result[TestUser]{
		Data: nil,
		Err:  errors.New("operation failed"),
	}

	if errorResult.Data != nil {
		t.Error("expected nil data for error case")
	}
	if errorResult.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if errorResult.Err.Error() != "operation failed" {
		t.Errorf("expected error 'operation failed', got '%s'", errorResult.Err.Error())
	}
}

// TestWithError tests the WithError method.
func TestWithError(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	err := errors.New("access denied")

	// Convert success to error
	data := OK(user).
		WithError(err).
		WithStatus(403)

	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if data.Error.Error() != "access denied" {
		t.Errorf("expected error 'access denied', got '%s'", data.Error.Error())
	}
	if data.Status != 403 {
		t.Errorf("expected status 403, got %d", data.Status)
	}
	// Value should still be present even with error
	if data.Value.ID != 1 {
		t.Errorf("expected ID 1, got %d", data.Value.ID)
	}
}

// TestWithErrorChaining tests chaining WithError with other methods.
func TestWithErrorChaining(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	err := errors.New("unauthorized")

	data := OK(user).
		WithMeta("attempt", 1).
		WithError(err).
		WithStatus(401).
		WithHeader("WWW-Authenticate", "Bearer")

	if data.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if data.Status != 401 {
		t.Errorf("expected status 401, got %d", data.Status)
	}
	if len(data.Metadata) != 1 {
		t.Errorf("expected 1 metadata entry, got %d", len(data.Metadata))
	}
	if len(data.Headers) != 1 {
		t.Errorf("expected 1 header, got %d", len(data.Headers))
	}
}

// TestWithMeta tests adding single metadata entry.
func TestWithMeta(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	data := OK(user).WithMeta("cached", true)

	if len(data.Metadata) != 1 {
		t.Errorf("expected 1 metadata entry, got %d", len(data.Metadata))
	}
	if cached, ok := data.Metadata["cached"].(bool); !ok || !cached {
		t.Errorf("expected cached=true, got %v", data.Metadata["cached"])
	}
}

// TestWithMetaChaining tests chaining multiple WithMeta calls.
func TestWithMetaChaining(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	data := OK(user).
		WithMeta("cached", true).
		WithMeta("ttl", 3600).
		WithMeta("page", 1)

	if len(data.Metadata) != 3 {
		t.Errorf("expected 3 metadata entries, got %d", len(data.Metadata))
	}

	if cached, ok := data.Metadata["cached"].(bool); !ok || !cached {
		t.Error("expected cached=true")
	}
	if ttl, ok := data.Metadata["ttl"].(int); !ok || ttl != 3600 {
		t.Errorf("expected ttl=3600, got %v", data.Metadata["ttl"])
	}
	if page, ok := data.Metadata["page"].(int); !ok || page != 1 {
		t.Errorf("expected page=1, got %v", data.Metadata["page"])
	}
}

// TestWithHeader tests adding single header.
func TestWithHeader(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	data := OK(user).WithHeader("X-Cache-Hit", "true")

	if len(data.Headers) != 1 {
		t.Errorf("expected 1 header, got %d", len(data.Headers))
	}
	if data.Headers["X-Cache-Hit"] != "true" {
		t.Errorf("expected X-Cache-Hit=true, got %s", data.Headers["X-Cache-Hit"])
	}
}

// TestWithHeaderChaining tests chaining multiple WithHeader calls.
func TestWithHeaderChaining(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	data := OK(user).
		WithHeader("X-Cache-Hit", "true").
		WithHeader("X-RateLimit-Remaining", "100")

	if len(data.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(data.Headers))
	}

	if data.Headers["X-Cache-Hit"] != "true" {
		t.Error("expected X-Cache-Hit=true")
	}
	if data.Headers["X-RateLimit-Remaining"] != "100" {
		t.Error("expected X-RateLimit-Remaining=100")
	}
}

// TestWithStatus tests custom status code.
func TestWithStatus(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	data := OK(user).WithStatus(206)

	if data.Status != 206 {
		t.Errorf("expected status 206, got %d", data.Status)
	}
}

// TestWithMetadata tests setting multiple metadata at once.
func TestWithMetadata(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	metadata := map[string]interface{}{
		"page":  1,
		"limit": 10,
		"total": 100,
	}
	data := OK(user).WithMetadata(metadata)

	if len(data.Metadata) != 3 {
		t.Errorf("expected 3 metadata entries, got %d", len(data.Metadata))
	}

	if page, ok := data.Metadata["page"].(int); !ok || page != 1 {
		t.Error("expected page=1")
	}
	if limit, ok := data.Metadata["limit"].(int); !ok || limit != 10 {
		t.Error("expected limit=10")
	}
	if total, ok := data.Metadata["total"].(int); !ok || total != 100 {
		t.Error("expected total=100")
	}
}

// TestWithHeaders tests setting multiple headers at once.
func TestWithHeaders(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	headers := map[string]string{
		"X-Cache-Hit":            "true",
		"X-RateLimit-Remaining":  "100",
		"X-Request-ID":           "abc123",
	}
	data := OK(user).WithHeaders(headers)

	if len(data.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(data.Headers))
	}

	if data.Headers["X-Cache-Hit"] != "true" {
		t.Error("expected X-Cache-Hit=true")
	}
	if data.Headers["X-RateLimit-Remaining"] != "100" {
		t.Error("expected X-RateLimit-Remaining=100")
	}
	if data.Headers["X-Request-ID"] != "abc123" {
		t.Error("expected X-Request-ID=abc123")
	}
}

// TestFluentAPI tests combining all fluent methods.
func TestFluentAPI(t *testing.T) {
	user := TestUser{ID: 1, Name: "Alice"}
	data := OK(user).
		WithMeta("cached", true).
		WithMeta("ttl", 3600).
		WithHeader("X-Cache-Hit", "true").
		WithHeader("X-TTL", "3600").
		WithStatus(200)

	// Check status
	if data.Status != 200 {
		t.Errorf("expected status 200, got %d", data.Status)
	}

	// Check metadata
	if len(data.Metadata) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(data.Metadata))
	}

	// Check headers
	if len(data.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(data.Headers))
	}

	// Check value is unchanged
	if data.Value.ID != 1 || data.Value.Name != "Alice" {
		t.Error("value was modified during fluent API calls")
	}
}

// TestDataWithSlice tests Data[T] with slice types.
func TestDataWithSlice(t *testing.T) {
	users := []TestUser{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	data := OK(users).
		WithMeta("total", 100).
		WithMeta("page", 1)

	if len(data.Value) != 2 {
		t.Errorf("expected 2 users, got %d", len(data.Value))
	}
	if data.Status != 200 {
		t.Errorf("expected status 200, got %d", data.Status)
	}
}

// TestDataWithMap tests Data[T] with map types.
func TestDataWithMap(t *testing.T) {
	userMap := map[string]TestUser{
		"alice": {ID: 1, Name: "Alice"},
		"bob":   {ID: 2, Name: "Bob"},
	}

	data := OK(userMap)

	if len(data.Value) != 2 {
		t.Errorf("expected 2 users in map, got %d", len(data.Value))
	}
	if data.Status != 200 {
		t.Errorf("expected status 200, got %d", data.Status)
	}
}

// TestDataWithPrimitiveTypes tests Data[T] with primitive types.
func TestDataWithPrimitiveTypes(t *testing.T) {
	// String
	strData := OK("hello")
	if strData.Value != "hello" {
		t.Errorf("expected 'hello', got '%s'", strData.Value)
	}

	// Int
	intData := OK(42)
	if intData.Value != 42 {
		t.Errorf("expected 42, got %d", intData.Value)
	}

	// Bool
	boolData := OK(true)
	if !boolData.Value {
		t.Error("expected true, got false")
	}
}

// BenchmarkOK benchmarks the OK constructor.
func BenchmarkOK(b *testing.B) {
	user := TestUser{ID: 1, Name: "Alice"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = OK(user)
	}
}

// BenchmarkWithMeta benchmarks adding metadata.
func BenchmarkWithMeta(b *testing.B) {
	user := TestUser{ID: 1, Name: "Alice"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = OK(user).WithMeta("cached", true)
	}
}

// BenchmarkFluentAPI benchmarks the full fluent API chain.
func BenchmarkFluentAPI(b *testing.B) {
	user := TestUser{ID: 1, Name: "Alice"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = OK(user).
			WithMeta("cached", true).
			WithMeta("ttl", 3600).
			WithHeader("X-Cache-Hit", "true").
			WithStatus(200)
	}
}
