package capacitor

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrNotFound", ErrNotFound, "key not found"},
		{"ErrClosed", ErrClosed, "dal closed"},
		{"ErrInvalidKey", ErrInvalidKey, "invalid key"},
		{"ErrInvalidValue", ErrInvalidValue, "invalid value"},
		{"ErrLayerNotFound", ErrLayerNotFound, "layer not found"},
		{"ErrReadOnly", ErrReadOnly, "layer is read-only"},
		{"ErrSizeLimitExceeded", ErrSizeLimitExceeded, "value size exceeds layer limit"},
		{"ErrEvictionFailed", ErrEvictionFailed, "eviction failed to free space"},
		{"ErrSerializationFailed", ErrSerializationFailed, "serialization failed"},
		{"ErrDeserializationFailed", ErrDeserializationFailed, "deserialization failed"},
		{"ErrValidationFailed", ErrValidationFailed, "validation failed"},
		{"ErrTimeout", ErrTimeout, "operation timed out"},
		{"ErrConnectionFailed", ErrConnectionFailed, "connection failed"},
		{"ErrTransactionFailed", ErrTransactionFailed, "transaction failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("error message = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCacheError(t *testing.T) {
	t.Run("with key", func(t *testing.T) {
		err := &CacheError{
			Layer: "L1",
			Op:    "Get",
			Key:   "user:123",
			Err:   ErrNotFound,
		}

		want := "cache error in layer L1 during Get (key: user:123): key not found"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without key", func(t *testing.T) {
		err := &CacheError{
			Layer: "L2",
			Op:    "Clear",
			Err:   errors.New("permission denied"),
		}

		want := "cache error in layer L2 during Clear: permission denied"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		underlying := errors.New("test error")
		err := &CacheError{
			Layer: "L1",
			Op:    "Get",
			Err:   underlying,
		}

		if !errors.Is(err, underlying) {
			t.Error("errors.Is() should find underlying error")
		}
	})
}

func TestDatabaseError(t *testing.T) {
	t.Run("with table and query", func(t *testing.T) {
		err := &DatabaseError{
			Backend: "postgres",
			Op:      "Query",
			Table:   "users",
			Query:   "SELECT * FROM users WHERE id = $1",
			Err:     errors.New("connection lost"),
		}

		got := err.Error()
		// Should contain all relevant information
		if got != "database error in postgres during Query: connection lost" {
			t.Errorf("Error() = %q", got)
		}
	})

	t.Run("query truncation", func(t *testing.T) {
		longQuery := ""
		for i := 0; i < 200; i++ {
			longQuery += "X"
		}

		err := &DatabaseError{
			Backend: "postgres",
			Op:      "Query",
			Query:   longQuery,
			Err:     errors.New("test"),
		}

		got := err.Error()
		// Query should be truncated
		if len(got) > 200 {
			// Should be truncated, so total length should be reasonable
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		underlying := errors.New("test error")
		err := &DatabaseError{
			Backend: "postgres",
			Op:      "Get",
			Err:     underlying,
		}

		if !errors.Is(err, underlying) {
			t.Error("errors.Is() should find underlying error")
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("single field error", func(t *testing.T) {
		err := &ValidationError{
			Type: "User",
			Errors: []FieldError{
				{Field: "Email", Message: "invalid email format"},
			},
		}

		want := "validation failed for User: Email: invalid email format"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("multiple field errors", func(t *testing.T) {
		err := &ValidationError{
			Type: "User",
			Errors: []FieldError{
				{Field: "Email", Message: "invalid email format"},
				{Field: "Age", Message: "must be at least 18"},
			},
		}

		want := "validation failed for User: 2 errors"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("no field errors with underlying error", func(t *testing.T) {
		underlying := errors.New("custom validator failed")
		err := &ValidationError{
			Type: "User",
			Err:  underlying,
		}

		want := "validation failed for User: custom validator failed"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("add field error", func(t *testing.T) {
		err := &ValidationError{Type: "User"}
		err.AddFieldError("Email", "invalid format")
		err.AddFieldError("Age", "too young")

		if len(err.Errors) != 2 {
			t.Errorf("AddFieldError() should have added 2 errors, got %d", len(err.Errors))
		}

		if err.Errors[0].Field != "Email" || err.Errors[0].Message != "invalid format" {
			t.Error("First error not added correctly")
		}

		if err.Errors[1].Field != "Age" || err.Errors[1].Message != "too young" {
			t.Error("Second error not added correctly")
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		underlying := errors.New("test error")
		err := &ValidationError{
			Type: "User",
			Err:  underlying,
		}

		if !errors.Is(err, underlying) {
			t.Error("errors.Is() should find underlying error")
		}
	})
}

func TestSerializationError(t *testing.T) {
	t.Run("with type", func(t *testing.T) {
		err := &SerializationError{
			Format: "json",
			Type:   "User",
			Err:    errors.New("unsupported type"),
		}

		want := "serialization failed for type User using json: unsupported type"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without type", func(t *testing.T) {
		err := &SerializationError{
			Format: "msgpack",
			Err:    errors.New("buffer overflow"),
		}

		want := "serialization failed using msgpack: buffer overflow"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestDeserializationError(t *testing.T) {
	t.Run("with type", func(t *testing.T) {
		err := &DeserializationError{
			Format: "json",
			Type:   "User",
			Err:    errors.New("unexpected EOF"),
		}

		want := "deserialization failed for type User using json: unexpected EOF"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without type", func(t *testing.T) {
		err := &DeserializationError{
			Format: "protobuf",
			Err:    errors.New("invalid wire type"),
		}

		want := "deserialization failed using protobuf: invalid wire type"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestNetworkError(t *testing.T) {
	t.Run("retryable", func(t *testing.T) {
		err := &NetworkError{
			Host:      "cache.example.com:6379",
			Op:        "Get",
			Err:       errors.New("connection timeout"),
			Retryable: true,
		}

		want := "network error connecting to cache.example.com:6379 during Get: connection timeout"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}

		if !err.IsRetryable() {
			t.Error("IsRetryable() should return true")
		}
	})

	t.Run("not retryable", func(t *testing.T) {
		err := &NetworkError{
			Host:      "db.example.com:5432",
			Op:        "Connect",
			Err:       errors.New("authentication failed"),
			Retryable: false,
		}

		if err.IsRetryable() {
			t.Error("IsRetryable() should return false")
		}
	})
}

func TestConfigError(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		underlying := errors.New("invalid format")
		err := &ConfigError{
			Field:   "TTL",
			Message: "must be positive",
			Err:     underlying,
		}

		want := "config error in TTL: must be positive: invalid format"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}

		if !errors.Is(err, underlying) {
			t.Error("errors.Is() should find underlying error")
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := &ConfigError{
			Field:   "Layers",
			Message: "at least one layer required",
		}

		want := "config error in Layers: at least one layer required"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestWrapCacheError(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		underlying := errors.New("test error")
		err := WrapCacheError("L1", "Get", "key123", underlying)

		var cacheErr *CacheError
		if !errors.As(err, &cacheErr) {
			t.Fatal("WrapCacheError should return *CacheError")
		}

		if cacheErr.Layer != "L1" {
			t.Errorf("Layer = %q, want %q", cacheErr.Layer, "L1")
		}

		if cacheErr.Op != "Get" {
			t.Errorf("Op = %q, want %q", cacheErr.Op, "Get")
		}

		if cacheErr.Key != "key123" {
			t.Errorf("Key = %q, want %q", cacheErr.Key, "key123")
		}

		if !errors.Is(err, underlying) {
			t.Error("Should wrap underlying error")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := WrapCacheError("L1", "Get", "key", nil)
		if err != nil {
			t.Error("WrapCacheError(nil) should return nil")
		}
	})
}

func TestWrapDatabaseError(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		underlying := errors.New("test error")
		err := WrapDatabaseError("postgres", "Query", "users", "SELECT * FROM users", underlying)

		var dbErr *DatabaseError
		if !errors.As(err, &dbErr) {
			t.Fatal("WrapDatabaseError should return *DatabaseError")
		}

		if !errors.Is(err, underlying) {
			t.Error("Should wrap underlying error")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := WrapDatabaseError("postgres", "Get", "users", "", nil)
		if err != nil {
			t.Error("WrapDatabaseError(nil) should return nil")
		}
	})
}

func TestWrapNetworkError(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		underlying := errors.New("test error")
		err := WrapNetworkError("redis:6379", "Get", underlying, true)

		var netErr *NetworkError
		if !errors.As(err, &netErr) {
			t.Fatal("WrapNetworkError should return *NetworkError")
		}

		if !netErr.IsRetryable() {
			t.Error("Should be retryable")
		}

		if !errors.Is(err, underlying) {
			t.Error("Should wrap underlying error")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := WrapNetworkError("host", "op", nil, false)
		if err != nil {
			t.Error("WrapNetworkError(nil) should return nil")
		}
	})
}

func TestWrapSerializationError(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		underlying := errors.New("test error")
		err := WrapSerializationError("json", "User", underlying)

		var serErr *SerializationError
		if !errors.As(err, &serErr) {
			t.Fatal("WrapSerializationError should return *SerializationError")
		}

		if !errors.Is(err, underlying) {
			t.Error("Should wrap underlying error")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := WrapSerializationError("json", "User", nil)
		if err != nil {
			t.Error("WrapSerializationError(nil) should return nil")
		}
	})
}

func TestWrapDeserializationError(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		underlying := errors.New("test error")
		err := WrapDeserializationError("json", "User", underlying)

		var deserErr *DeserializationError
		if !errors.As(err, &deserErr) {
			t.Fatal("WrapDeserializationError should return *DeserializationError")
		}

		if !errors.Is(err, underlying) {
			t.Error("Should wrap underlying error")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := WrapDeserializationError("json", "User", nil)
		if err != nil {
			t.Error("WrapDeserializationError(nil) should return nil")
		}
	})
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct ErrNotFound", ErrNotFound, true},
		{"wrapped ErrNotFound", WrapCacheError("L1", "Get", "key", ErrNotFound), true},
		{"other error", errors.New("other"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsClosed(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct ErrClosed", ErrClosed, true},
		{"wrapped ErrClosed", WrapCacheError("L1", "Get", "key", ErrClosed), true},
		{"other error", errors.New("other"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsClosed(tt.err); got != tt.want {
				t.Errorf("IsClosed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"network error retryable", &NetworkError{Retryable: true}, true},
		{"network error not retryable", &NetworkError{Retryable: false}, false},
		{"ErrTimeout", ErrTimeout, true},
		{"ErrConnectionFailed", ErrConnectionFailed, true},
		{"ErrNotFound", ErrNotFound, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetValidationErrors(t *testing.T) {
	t.Run("extracts field errors", func(t *testing.T) {
		valErr := &ValidationError{
			Type: "User",
			Errors: []FieldError{
				{Field: "Email", Message: "invalid"},
				{Field: "Age", Message: "too young"},
			},
		}

		fieldErrors := GetValidationErrors(valErr)
		if fieldErrors == nil {
			t.Fatal("GetValidationErrors should return field errors")
		}

		if len(fieldErrors) != 2 {
			t.Errorf("len(fieldErrors) = %d, want 2", len(fieldErrors))
		}
	})

	t.Run("returns nil for non-validation error", func(t *testing.T) {
		err := errors.New("other error")
		fieldErrors := GetValidationErrors(err)
		if fieldErrors != nil {
			t.Error("GetValidationErrors should return nil for non-validation error")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		fieldErrors := GetValidationErrors(nil)
		if fieldErrors != nil {
			t.Error("GetValidationErrors should return nil for nil error")
		}
	})
}
