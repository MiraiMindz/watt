package capacitor

import (
	"errors"
	"fmt"
)

// Additional sentinel errors for common cases.
// Base errors (ErrNotFound, ErrClosed, etc.) are defined in dal.go
var (

	// ErrReadOnly indicates an attempt to write to a read-only layer.
	// Layers can be marked read-only in their configuration.
	ErrReadOnly = errors.New("layer is read-only")

	// ErrSizeLimitExceeded indicates the value exceeds the layer's size limit.
	// This occurs when attempting to store a value larger than the configured maximum.
	ErrSizeLimitExceeded = errors.New("value size exceeds layer limit")

	// ErrEvictionFailed indicates the eviction policy failed to free space.
	// This can occur when the cache is full and no items can be evicted.
	ErrEvictionFailed = errors.New("eviction failed to free space")

	// ErrSerializationFailed indicates serialization of a value failed.
	// This occurs when converting a value to bytes for storage.
	ErrSerializationFailed = errors.New("serialization failed")

	// ErrDeserializationFailed indicates deserialization of a value failed.
	// This occurs when converting stored bytes back to a value.
	ErrDeserializationFailed = errors.New("deserialization failed")

	// ErrValidationFailed indicates value validation failed.
	// This occurs when a value doesn't meet configured validation rules.
	ErrValidationFailed = errors.New("validation failed")

	// ErrTimeout indicates an operation timed out.
	// This can occur for database operations or network requests.
	ErrTimeout = errors.New("operation timed out")

	// ErrConnectionFailed indicates a connection to a backend failed.
	// This occurs when unable to connect to a database or cache server.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrTransactionFailed indicates a transaction failed and was rolled back.
	// This occurs when a database transaction encounters an error.
	ErrTransactionFailed = errors.New("transaction failed")
)

// CacheError represents an error that occurred in a cache layer.
// It provides context about which cache layer failed and what operation was being performed.
//
// Example:
//
//	err := &CacheError{
//	    Layer: "L1",
//	    Op:    "Get",
//	    Key:   "user:123",
//	    Err:   ErrNotFound,
//	}
type CacheError struct {
	// Layer identifies which cache layer encountered the error (e.g., "L1", "L2").
	Layer string

	// Op is the operation that was being performed (e.g., "Get", "Set", "Delete").
	Op string

	// Key is the key involved in the operation (optional).
	Key string

	// Err is the underlying error that occurred.
	Err error
}

// Error implements the error interface.
func (e *CacheError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("cache error in layer %s during %s (key: %s): %v", e.Layer, e.Op, e.Key, e.Err)
	}
	return fmt.Sprintf("cache error in layer %s during %s: %v", e.Layer, e.Op, e.Err)
}

// Unwrap returns the underlying error.
// This allows errors.Is() and errors.As() to work with wrapped errors.
func (e *CacheError) Unwrap() error {
	return e.Err
}

// DatabaseError represents an error that occurred in a database layer.
// It provides context about which database encountered the error and what operation was being performed.
//
// Example:
//
//	err := &DatabaseError{
//	    Backend:   "postgres",
//	    Op:        "Set",
//	    Table:     "users",
//	    Err:       sql.ErrConnDone,
//	}
type DatabaseError struct {
	// Backend identifies which database backend encountered the error (e.g., "postgres", "mongodb").
	Backend string

	// Op is the operation that was being performed (e.g., "Get", "Set", "Delete", "Query").
	Op string

	// Table is the table/collection involved in the operation (optional).
	Table string

	// Query is the query that was being executed (optional, may be truncated for security).
	Query string

	// Err is the underlying error that occurred.
	Err error
}

// Error implements the error interface.
func (e *DatabaseError) Error() string {
	parts := []string{fmt.Sprintf("database error in %s during %s", e.Backend, e.Op)}

	if e.Table != "" {
		parts = append(parts, fmt.Sprintf("(table: %s)", e.Table))
	}

	if e.Query != "" {
		// Truncate query to avoid exposing sensitive data
		query := e.Query
		if len(query) > 100 {
			query = query[:100] + "..."
		}
		parts = append(parts, fmt.Sprintf("(query: %s)", query))
	}

	return fmt.Sprintf("%s: %v", parts[0], e.Err)
}

// Unwrap returns the underlying error.
func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// ValidationError represents an error that occurred during value validation.
// It can contain multiple field errors for comprehensive validation reporting.
//
// Example:
//
//	err := &ValidationError{
//	    Type:   "User",
//	    Errors: []FieldError{
//	        {Field: "Email", Message: "invalid email format"},
//	        {Field: "Age", Message: "must be at least 18"},
//	    },
//	}
type ValidationError struct {
	// Type is the type being validated (e.g., "User", "Product").
	Type string

	// Errors is a list of field-level validation errors.
	Errors []FieldError

	// Err is an optional underlying error.
	Err error
}

// FieldError represents a validation error for a specific field.
type FieldError struct {
	// Field is the name of the field that failed validation.
	Field string

	// Message is a human-readable error message.
	Message string

	// Value is the invalid value (optional, may be omitted for security).
	Value interface{}
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		if e.Err != nil {
			return fmt.Sprintf("validation failed for %s: %v", e.Type, e.Err)
		}
		return fmt.Sprintf("validation failed for %s", e.Type)
	}

	if len(e.Errors) == 1 {
		return fmt.Sprintf("validation failed for %s: %s: %s", e.Type, e.Errors[0].Field, e.Errors[0].Message)
	}

	// Multiple errors
	return fmt.Sprintf("validation failed for %s: %d errors", e.Type, len(e.Errors))
}

// Unwrap returns the underlying error.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// AddFieldError adds a field error to the validation error.
// This is useful when building up validation errors programmatically.
//
// Example:
//
//	valErr := &ValidationError{Type: "User"}
//	valErr.AddFieldError("Email", "invalid format")
//	valErr.AddFieldError("Age", "too young")
func (e *ValidationError) AddFieldError(field, message string) {
	e.Errors = append(e.Errors, FieldError{
		Field:   field,
		Message: message,
	})
}

// SerializationError represents an error that occurred during serialization.
//
// Example:
//
//	err := &SerializationError{
//	    Format: "json",
//	    Type:   "User",
//	    Err:    errors.New("unsupported type"),
//	}
type SerializationError struct {
	// Format is the serialization format (e.g., "json", "msgpack", "protobuf").
	Format string

	// Type is the type being serialized (optional).
	Type string

	// Err is the underlying error that occurred.
	Err error
}

// Error implements the error interface.
func (e *SerializationError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("serialization failed for type %s using %s: %v", e.Type, e.Format, e.Err)
	}
	return fmt.Sprintf("serialization failed using %s: %v", e.Format, e.Err)
}

// Unwrap returns the underlying error.
func (e *SerializationError) Unwrap() error {
	return e.Err
}

// DeserializationError represents an error that occurred during deserialization.
//
// Example:
//
//	err := &DeserializationError{
//	    Format: "json",
//	    Type:   "User",
//	    Err:    errors.New("unexpected EOF"),
//	}
type DeserializationError struct {
	// Format is the serialization format (e.g., "json", "msgpack", "protobuf").
	Format string

	// Type is the expected type (optional).
	Type string

	// Err is the underlying error that occurred.
	Err error
}

// Error implements the error interface.
func (e *DeserializationError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("deserialization failed for type %s using %s: %v", e.Type, e.Format, e.Err)
	}
	return fmt.Sprintf("deserialization failed using %s: %v", e.Format, e.Err)
}

// Unwrap returns the underlying error.
func (e *DeserializationError) Unwrap() error {
	return e.Err
}

// NetworkError represents an error that occurred during a network operation.
// This is used for distributed caches and remote database connections.
//
// Example:
//
//	err := &NetworkError{
//	    Host:      "cache.example.com:6379",
//	    Op:        "Get",
//	    Err:       context.DeadlineExceeded,
//	    Retryable: true,
//	}
type NetworkError struct {
	// Host is the remote host that was being contacted.
	Host string

	// Op is the operation that was being performed.
	Op string

	// Err is the underlying error that occurred.
	Err error

	// Retryable indicates whether the operation can be retried.
	Retryable bool
}

// Error implements the error interface.
func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error connecting to %s during %s: %v", e.Host, e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *NetworkError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error can be retried.
func (e *NetworkError) IsRetryable() bool {
	return e.Retryable
}

// ConfigError represents an error in DAL configuration.
//
// Example:
//
//	err := &ConfigError{
//	    Field:   "Layers",
//	    Message: "at least one layer must be configured",
//	}
type ConfigError struct {
	// Field is the configuration field that is invalid.
	Field string

	// Message is a human-readable error message.
	Message string

	// Err is an optional underlying error.
	Err error
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("config error in %s: %s: %v", e.Field, e.Message, e.Err)
	}
	return fmt.Sprintf("config error in %s: %s", e.Field, e.Message)
}

// Unwrap returns the underlying error.
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// Error Wrapping Utilities

// WrapCacheError wraps an error with cache context.
// This is a convenience function for creating CacheError instances.
//
// Example:
//
//	if err := cache.Get(ctx, key); err != nil {
//	    return WrapCacheError("L1", "Get", key, err)
//	}
func WrapCacheError(layer, op string, key string, err error) error {
	if err == nil {
		return nil
	}
	return &CacheError{
		Layer: layer,
		Op:    op,
		Key:   key,
		Err:   err,
	}
}

// WrapDatabaseError wraps an error with database context.
// This is a convenience function for creating DatabaseError instances.
//
// Example:
//
//	if err := db.Query(ctx, query); err != nil {
//	    return WrapDatabaseError("postgres", "Query", "users", query, err)
//	}
func WrapDatabaseError(backend, op, table, query string, err error) error {
	if err == nil {
		return nil
	}
	return &DatabaseError{
		Backend: backend,
		Op:      op,
		Table:   table,
		Query:   query,
		Err:     err,
	}
}

// WrapNetworkError wraps an error with network context.
// This is a convenience function for creating NetworkError instances.
//
// Example:
//
//	if err := conn.Do(ctx, cmd); err != nil {
//	    return WrapNetworkError("redis.example.com:6379", "Do", err, true)
//	}
func WrapNetworkError(host, op string, err error, retryable bool) error {
	if err == nil {
		return nil
	}
	return &NetworkError{
		Host:      host,
		Op:        op,
		Err:       err,
		Retryable: retryable,
	}
}

// WrapSerializationError wraps an error with serialization context.
//
// Example:
//
//	if err := json.Marshal(value); err != nil {
//	    return WrapSerializationError("json", "User", err)
//	}
func WrapSerializationError(format, typ string, err error) error {
	if err == nil {
		return nil
	}
	return &SerializationError{
		Format: format,
		Type:   typ,
		Err:    err,
	}
}

// WrapDeserializationError wraps an error with deserialization context.
//
// Example:
//
//	if err := json.Unmarshal(data, &value); err != nil {
//	    return WrapDeserializationError("json", "User", err)
//	}
func WrapDeserializationError(format, typ string, err error) error {
	if err == nil {
		return nil
	}
	return &DeserializationError{
		Format: format,
		Type:   typ,
		Err:    err,
	}
}

// IsNotFound returns true if the error is or wraps ErrNotFound.
// This is a convenience function for the common case of checking for missing keys.
//
// Example:
//
//	value, err := dal.Get(ctx, key)
//	if IsNotFound(err) {
//	    // Key doesn't exist, create it
//	    err = dal.Set(ctx, key, defaultValue)
//	}
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsClosed returns true if the error is or wraps ErrClosed.
//
// Example:
//
//	_, err := dal.Get(ctx, key)
//	if IsClosed(err) {
//	    // DAL was closed, cannot use it
//	    return err
//	}
func IsClosed(err error) bool {
	return errors.Is(err, ErrClosed)
}

// IsRetryable returns true if the error indicates a retryable operation.
// This checks for network errors and other transient failures.
//
// Example:
//
//	err := dal.Set(ctx, key, value)
//	if IsRetryable(err) {
//	    // Retry with exponential backoff
//	    time.Sleep(backoff)
//	    err = dal.Set(ctx, key, value)
//	}
func IsRetryable(err error) bool {
	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return netErr.IsRetryable()
	}

	// Add other retryable error types here
	return errors.Is(err, ErrTimeout) || errors.Is(err, ErrConnectionFailed)
}

// GetValidationErrors extracts field errors from a ValidationError.
// Returns nil if the error is not a ValidationError.
//
// Example:
//
//	if fieldErrors := GetValidationErrors(err); fieldErrors != nil {
//	    for _, fe := range fieldErrors {
//	        log.Printf("Field %s: %s", fe.Field, fe.Message)
//	    }
//	}
func GetValidationErrors(err error) []FieldError {
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return valErr.Errors
	}
	return nil
}
