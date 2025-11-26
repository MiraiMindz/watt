package core

import (
	"errors"
	"testing"
)

// TestSendDataSuccess tests sendData with valid data.
func TestSendDataSuccess(t *testing.T) {
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	ctx := &Context{}
	data := OK(User{ID: 1, Name: "Alice"})

	err := sendData(ctx, data)
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

// TestSendDataWithMetadata tests sendData with metadata.
func TestSendDataWithMetadata(t *testing.T) {
	type Product struct {
		ID    int     `json:"id"`
		Price float64 `json:"price"`
	}

	ctx := &Context{}
	data := OK(Product{ID: 1, Price: 99.99}).
		WithMeta("currency", "USD").
		WithMeta("tax", 0.08)

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}
}

// TestSendDataWithHeaders tests sendData with custom headers.
func TestSendDataWithHeaders(t *testing.T) {
	ctx := &Context{}
	data := OK("test").
		WithHeader("X-Custom-Header", "custom-value").
		WithHeader("X-Request-ID", "12345")

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify headers were set
	if val := ctx.GetResponseHeader("X-Custom-Header"); val != "custom-value" {
		t.Errorf("expected header 'custom-value', got '%s'", val)
	}

	if val := ctx.GetResponseHeader("X-Request-ID"); val != "12345" {
		t.Errorf("expected header '12345', got '%s'", val)
	}
}

// TestSendDataWithZeroStatus tests sendData defaults to 200 when status is 0.
func TestSendDataWithZeroStatus(t *testing.T) {
	ctx := &Context{}
	data := Data[string]{
		Value:  "test",
		Status: 0, // Zero status
	}

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected default status 200, got %d", ctx.StatusCode())
	}
}

// TestSendDataWithCustomStatus tests sendData with custom status.
func TestSendDataWithCustomStatus(t *testing.T) {
	ctx := &Context{}
	data := Created("resource").WithStatus(201)

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 201 {
		t.Errorf("expected status 201, got %d", ctx.StatusCode())
	}
}

// TestSendErrorDataNotFound tests sendErrorData with 404 error.
func TestSendErrorDataNotFound(t *testing.T) {
	ctx := &Context{}
	data := NotFound[string](errors.New("resource not found"))

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 404 {
		t.Errorf("expected status 404, got %d", ctx.StatusCode())
	}

	if !ctx.Written() {
		t.Error("expected response to be written")
	}
}

// TestSendErrorDataWithMetadata tests sendErrorData with metadata.
func TestSendErrorDataWithMetadata(t *testing.T) {
	ctx := &Context{}
	data := BadRequest[int](errors.New("validation failed")).
		WithMeta("field", "email").
		WithMeta("reason", "invalid format")

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 400 {
		t.Errorf("expected status 400, got %d", ctx.StatusCode())
	}
}

// TestSendErrorDataUnauthorized tests sendErrorData with 401.
func TestSendErrorDataUnauthorized(t *testing.T) {
	ctx := &Context{}
	data := Unauthorized[any](errors.New("invalid credentials"))

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 401 {
		t.Errorf("expected status 401, got %d", ctx.StatusCode())
	}
}

// TestSendErrorDataForbidden tests sendErrorData with 403.
func TestSendErrorDataForbidden(t *testing.T) {
	ctx := &Context{}
	data := Forbidden[any](errors.New("access denied"))

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 403 {
		t.Errorf("expected status 403, got %d", ctx.StatusCode())
	}
}

// TestSendErrorDataConflict tests sendErrorData with 409.
func TestSendErrorDataConflict(t *testing.T) {
	ctx := &Context{}
	data := Conflict[any](errors.New("resource already exists"))

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 409 {
		t.Errorf("expected status 409, got %d", ctx.StatusCode())
	}
}

// TestSendErrorDataInternalServerError tests sendErrorData with 500.
func TestSendErrorDataInternalServerError(t *testing.T) {
	ctx := &Context{}
	data := InternalServerError[any](errors.New("database connection failed"))

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 500 {
		t.Errorf("expected status 500, got %d", ctx.StatusCode())
	}
}

// TestSendDataNoContent tests sendData with NoContent.
func TestSendDataNoContent(t *testing.T) {
	ctx := &Context{}
	data := NoContent[any]()

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 204 {
		t.Errorf("expected status 204, got %d", ctx.StatusCode())
	}
}

// TestSendDataAccepted tests sendData with Accepted.
func TestSendDataAccepted(t *testing.T) {
	ctx := &Context{}
	data := Accepted("processing")

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 202 {
		t.Errorf("expected status 202, got %d", ctx.StatusCode())
	}
}

// TestSendDataWithBothValueAndError tests behavior when both are set.
func TestSendDataWithBothValueAndError(t *testing.T) {
	ctx := &Context{}
	data := OK("value").WithError(errors.New("but also error"))

	// When error is set, should use error handling path
	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Error was set, so should have been handled as error
	if ctx.StatusCode() != 200 {
		t.Errorf("expected status from error handling, got %d", ctx.StatusCode())
	}
}

// TestSendDataComplexType tests sendData with complex nested types.
func TestSendDataComplexType(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type User struct {
		ID      int     `json:"id"`
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	ctx := &Context{}
	user := User{
		ID:   1,
		Name: "Alice",
		Address: Address{
			Street: "123 Main St",
			City:   "NYC",
		},
	}

	data := OK(user).WithMeta("source", "database")

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSendDataSliceType tests sendData with slice.
func TestSendDataSliceType(t *testing.T) {
	ctx := &Context{}
	data := OK([]int{1, 2, 3, 4, 5}).WithMeta("count", 5)

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSendDataMapType tests sendData with map.
func TestSendDataMapType(t *testing.T) {
	ctx := &Context{}
	data := OK(map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	})

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSendDataWithAllFeatures tests sendData with metadata, headers, and custom status.
func TestSendDataWithAllFeatures(t *testing.T) {
	ctx := &Context{}
	data := OK("test").
		WithMeta("key1", "value1").
		WithMeta("key2", 123).
		WithHeader("X-Custom", "custom").
		WithHeader("X-Another", "another").
		WithStatus(201)

	err := sendData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 201 {
		t.Errorf("expected status 201, got %d", ctx.StatusCode())
	}

	if val := ctx.GetResponseHeader("X-Custom"); val != "custom" {
		t.Errorf("expected header 'custom', got '%s'", val)
	}

	if val := ctx.GetResponseHeader("X-Another"); val != "another" {
		t.Errorf("expected header 'another', got '%s'", val)
	}
}

// TestSendErrorDataNoMetadata tests sendErrorData without metadata.
func TestSendErrorDataNoMetadata(t *testing.T) {
	ctx := &Context{}
	data := BadRequest[string](errors.New("validation failed"))

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 400 {
		t.Errorf("expected status 400, got %d", ctx.StatusCode())
	}
}

// TestSendErrorDataWithHeaders tests sendErrorData with custom headers.
func TestSendErrorDataWithHeaders(t *testing.T) {
	ctx := &Context{}
	data := NotFound[string](errors.New("not found")).
		WithHeader("X-Error-ID", "12345").
		WithHeader("X-Trace-ID", "trace-123")

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 404 {
		t.Errorf("expected status 404, got %d", ctx.StatusCode())
	}

	if val := ctx.GetResponseHeader("X-Error-ID"); val != "12345" {
		t.Errorf("expected header '12345', got '%s'", val)
	}

	if val := ctx.GetResponseHeader("X-Trace-ID"); val != "trace-123" {
		t.Errorf("expected header 'trace-123', got '%s'", val)
	}
}

// TestSendErrorDataAllStatusCodes tests sendErrorData with all error status codes.
func TestSendErrorDataAllStatusCodes(t *testing.T) {
	tests := []struct {
		name   string
		data   Data[any]
		status int
	}{
		{"BadRequest", BadRequest[any](errors.New("bad request")), 400},
		{"Unauthorized", Unauthorized[any](errors.New("unauthorized")), 401},
		{"Forbidden", Forbidden[any](errors.New("forbidden")), 403},
		{"NotFound", NotFound[any](errors.New("not found")), 404},
		{"Conflict", Conflict[any](errors.New("conflict")), 409},
		{"InternalServerError", InternalServerError[any](errors.New("server error")), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			err := sendErrorData(ctx, tt.data)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if ctx.StatusCode() != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, ctx.StatusCode())
			}
		})
	}
}

// TestSendErrorDataWithMetadataAndHeaders tests sendErrorData with both metadata and headers.
func TestSendErrorDataWithMetadataAndHeaders(t *testing.T) {
	ctx := &Context{}
	data := BadRequest[int](errors.New("validation failed")).
		WithMeta("field", "email").
		WithMeta("reason", "invalid format").
		WithMeta("suggestion", "use valid email").
		WithHeader("X-Error-Code", "VALIDATION_ERROR").
		WithHeader("X-Request-ID", "req-123")

	err := sendErrorData(ctx, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 400 {
		t.Errorf("expected status 400, got %d", ctx.StatusCode())
	}

	// Verify headers
	if val := ctx.GetResponseHeader("X-Error-Code"); val != "VALIDATION_ERROR" {
		t.Errorf("expected header 'VALIDATION_ERROR', got '%s'", val)
	}

	if val := ctx.GetResponseHeader("X-Request-ID"); val != "req-123" {
		t.Errorf("expected header 'req-123', got '%s'", val)
	}
}
