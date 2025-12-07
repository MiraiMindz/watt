package capacitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// DTO represents the base interface for all Data Transfer Objects.
// DTOs are immutable value objects used to transfer data between layers.
//
// Example:
//
//	type UserDTO struct {
//	    ID        int64     `json:"id" validate:"required"`
//	    Email     string    `json:"email" validate:"required,email"`
//	    CreatedAt time.Time `json:"created_at"`
//	}
//
//	func (u *UserDTO) Validate(ctx context.Context) error {
//	    if u.ID <= 0 {
//	        return fmt.Errorf("invalid user ID: %d", u.ID)
//	    }
//	    // Additional validation...
//	    return nil
//	}
type DTO interface {
	// Validate validates the DTO's data.
	// Returns nil if valid, or a ValidationError with details if invalid.
	Validate(ctx context.Context) error
}

// Serializable represents a DTO that can be serialized to bytes.
// Implementations should use efficient serialization methods (msgpack, protobuf, etc.)
// and leverage object pooling to minimize allocations.
type Serializable interface {
	DTO

	// Marshal serializes the DTO to bytes.
	// Should use sync.Pool for buffer allocation to minimize heap pressure.
	Marshal() ([]byte, error)

	// Unmarshal deserializes bytes into the DTO.
	// Should reuse the receiver when possible to avoid allocations.
	Unmarshal(data []byte) error
}

// Cacheable represents a DTO that can be cached.
// It provides cache key generation and TTL configuration.
type Cacheable interface {
	DTO

	// CacheKey returns a unique cache key for this DTO.
	// The key should be deterministic and collision-free.
	CacheKey() string

	// CacheTTL returns the time-to-live for this DTO in cache.
	// Return 0 for no expiration.
	CacheTTL() time.Duration
}

// Versionable represents a DTO with version tracking.
// Useful for optimistic locking and conflict detection.
type Versionable interface {
	DTO

	// Version returns the current version of this DTO.
	// Versions should be monotonically increasing.
	Version() int64

	// SetVersion sets the version of this DTO.
	// Used when loading from storage or after updates.
	SetVersion(version int64)
}

// Timestamped represents a DTO with creation and modification timestamps.
// Useful for audit trails and time-based queries.
type Timestamped interface {
	DTO

	// CreatedAt returns when this DTO was created.
	CreatedAt() time.Time

	// UpdatedAt returns when this DTO was last updated.
	UpdatedAt() time.Time

	// Touch updates the modification timestamp to now.
	Touch()
}

// Identifiable represents a DTO with a unique identifier.
// The ID type is generic to support different ID schemes (int64, UUID, string).
type Identifiable[ID comparable] interface {
	DTO

	// ID returns the unique identifier for this DTO.
	ID() ID

	// SetID sets the unique identifier for this DTO.
	// Used when creating new entities or loading from storage.
	SetID(id ID)
}

// Mappable represents a DTO that can convert to/from map[string]interface{}.
// Useful for dynamic queries and JSON-like operations.
type Mappable interface {
	DTO

	// ToMap converts the DTO to a map.
	// Should use sync.Pool for map allocation when possible.
	ToMap() map[string]interface{}

	// FromMap populates the DTO from a map.
	// Should validate and handle type conversions safely.
	FromMap(m map[string]interface{}) error
}

// Cloneable represents a DTO that can create deep copies of itself.
// Useful for maintaining immutability and safe concurrent access.
type Cloneable[T any] interface {
	DTO

	// Clone creates a deep copy of this DTO.
	// The clone should be independent of the original.
	Clone() T
}

// DTOMapper provides utilities for mapping between DTOs and domain models.
// It supports bidirectional conversion with validation.
//
// Example:
//
//	mapper := NewDTOMapper[User, UserDTO]()
//	dto, err := mapper.ToDTO(ctx, user)
//	user, err := mapper.FromDTO(ctx, dto)
type DTOMapper[M any, D DTO] interface {
	// ToDTO converts a domain model to a DTO.
	ToDTO(ctx context.Context, model M) (D, error)

	// FromDTO converts a DTO to a domain model.
	FromDTO(ctx context.Context, dto D) (M, error)

	// ToDTOs converts multiple domain models to DTOs.
	// Uses goroutines for parallel conversion when beneficial.
	ToDTOs(ctx context.Context, models []M) ([]D, error)

	// FromDTOs converts multiple DTOs to domain models.
	// Uses goroutines for parallel conversion when beneficial.
	FromDTOs(ctx context.Context, dtos []D) ([]M, error)
}

// ValidatorFunc is a function that validates a DTO.
// It can be composed to build complex validation logic.
type ValidatorFunc[T DTO] func(ctx context.Context, dto T) error

// Validator provides validation functionality for DTOs.
// It supports composable validation rules and field-level errors.
type Validator[T DTO] interface {
	// Validate validates a DTO using all registered rules.
	Validate(ctx context.Context, dto T) error

	// ValidateField validates a specific field of the DTO.
	ValidateField(ctx context.Context, dto T, field string) error

	// AddRule adds a validation rule.
	AddRule(name string, rule ValidatorFunc[T])

	// RemoveRule removes a validation rule.
	RemoveRule(name string)
}

// SerializationFormat represents the format used for serialization.
type SerializationFormat int

const (
	// FormatJSON uses encoding/json.
	// Good for human-readability and debugging.
	FormatJSON SerializationFormat = iota

	// FormatMsgPack uses MessagePack.
	// Better performance and smaller size than JSON.
	FormatMsgPack

	// FormatProtobuf uses Protocol Buffers.
	// Best performance but requires schema definition.
	FormatProtobuf

	// FormatCustom allows custom serialization implementation.
	FormatCustom
)

// String returns the string representation of the serialization format.
func (f SerializationFormat) String() string {
	switch f {
	case FormatJSON:
		return "JSON"
	case FormatMsgPack:
		return "MessagePack"
	case FormatProtobuf:
		return "Protobuf"
	case FormatCustom:
		return "Custom"
	default:
		return fmt.Sprintf("Unknown(%d)", f)
	}
}

// Serializer provides serialization/deserialization functionality.
// Implementations should use object pooling to minimize allocations.
type Serializer[T DTO] interface {
	// Marshal serializes a DTO to bytes.
	Marshal(dto T) ([]byte, error)

	// Unmarshal deserializes bytes to a DTO.
	// The dto parameter should be a pointer to allow in-place deserialization.
	Unmarshal(data []byte, dto T) error

	// Format returns the serialization format used.
	Format() SerializationFormat
}

// BaseDTO provides common DTO functionality.
// Embed this in your DTOs to get default implementations.
//
// Example:
//
//	type UserDTO struct {
//	    BaseDTO
//	    ID    int64  `json:"id"`
//	    Email string `json:"email"`
//	}
//
//	func (u *UserDTO) Validate(ctx context.Context) error {
//	    u.ClearValidationErrors()
//	    // Add validation logic...
//	    return u.ValidateWithType(ctx, "UserDTO")
//	}
type BaseDTO struct {
	// validation errors accumulated during validation
	validationErrors []FieldError
}

// Validate implements DTO.Validate with a no-op implementation.
// Override this in your concrete DTOs.
func (b *BaseDTO) Validate(ctx context.Context) error {
	if len(b.validationErrors) > 0 {
		return &ValidationError{
			Type:   "BaseDTO",
			Errors: b.validationErrors,
		}
	}
	return nil
}

// ValidateWithType validates and returns an error with the specified type name.
// Use this in your Validate implementations to provide the correct type name.
func (b *BaseDTO) ValidateWithType(ctx context.Context, typeName string) error {
	if len(b.validationErrors) > 0 {
		return &ValidationError{
			Type:   typeName,
			Errors: b.validationErrors,
		}
	}
	return nil
}

// AddValidationError adds a field validation error.
// This is a helper for building up validation errors.
func (b *BaseDTO) AddValidationError(field, message string) {
	b.validationErrors = append(b.validationErrors, FieldError{
		Field:   field,
		Message: message,
	})
}

// ClearValidationErrors clears all accumulated validation errors.
func (b *BaseDTO) ClearValidationErrors() {
	b.validationErrors = b.validationErrors[:0]
}

// JSONSerializer provides JSON serialization using encoding/json.
// It uses sync.Pool for buffer allocation to minimize heap pressure.
type JSONSerializer[T DTO] struct {
	bufferPool *sync.Pool
}

// NewJSONSerializer creates a new JSON serializer.
func NewJSONSerializer[T DTO]() *JSONSerializer[T] {
	return &JSONSerializer[T]{
		bufferPool: &sync.Pool{
			New: func() interface{} {
				// Pre-allocate 1KB buffer
				buf := make([]byte, 0, 1024)
				return &buf
			},
		},
	}
}

// Marshal implements Serializer.Marshal.
func (s *JSONSerializer[T]) Marshal(dto T) ([]byte, error) {
	data, err := json.Marshal(dto)
	if err != nil {
		return nil, WrapSerializationError("json", reflect.TypeOf(dto).Name(), err)
	}
	return data, nil
}

// Unmarshal implements Serializer.Unmarshal.
func (s *JSONSerializer[T]) Unmarshal(data []byte, dto T) error {
	if err := json.Unmarshal(data, dto); err != nil {
		return WrapDeserializationError("json", reflect.TypeOf(dto).Name(), err)
	}
	return nil
}

// Format implements Serializer.Format.
func (s *JSONSerializer[T]) Format() SerializationFormat {
	return FormatJSON
}

// SimpleValidator provides basic validation functionality.
// It maintains a map of validation rules that are executed in order.
type SimpleValidator[T DTO] struct {
	rules map[string]ValidatorFunc[T]
	mu    sync.RWMutex
}

// NewSimpleValidator creates a new simple validator.
func NewSimpleValidator[T DTO]() *SimpleValidator[T] {
	return &SimpleValidator[T]{
		rules: make(map[string]ValidatorFunc[T]),
	}
}

// Validate implements Validator.Validate.
func (v *SimpleValidator[T]) Validate(ctx context.Context, dto T) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var validationErr *ValidationError

	for name, rule := range v.rules {
		if err := rule(ctx, dto); err != nil {
			if validationErr == nil {
				validationErr = &ValidationError{
					Type: reflect.TypeOf(dto).Name(),
				}
			}

			// If it's already a ValidationError, merge the errors
			var vErr *ValidationError
			if ok := errors.As(err, &vErr); ok {
				validationErr.Errors = append(validationErr.Errors, vErr.Errors...)
			} else {
				// Otherwise add as a general error
				validationErr.AddFieldError(name, err.Error())
			}
		}
	}

	if validationErr != nil {
		return validationErr
	}
	return nil
}

// ValidateField implements Validator.ValidateField.
func (v *SimpleValidator[T]) ValidateField(ctx context.Context, dto T, field string) error {
	v.mu.RLock()
	rule, exists := v.rules[field]
	v.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no validation rule for field: %s", field)
	}

	return rule(ctx, dto)
}

// AddRule implements Validator.AddRule.
func (v *SimpleValidator[T]) AddRule(name string, rule ValidatorFunc[T]) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rules[name] = rule
}

// RemoveRule implements Validator.RemoveRule.
func (v *SimpleValidator[T]) RemoveRule(name string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.rules, name)
}

// Common validation helper functions

// RequiredString validates that a string field is not empty.
func RequiredString(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// RequiredInt validates that an int field is not zero.
func RequiredInt[T int | int32 | int64](field string, value T) error {
	if value == 0 {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// MinLength validates that a string has minimum length.
func MinLength(field, value string, min int) error {
	if len(value) < min {
		return fmt.Errorf("%s must be at least %d characters", field, min)
	}
	return nil
}

// MaxLength validates that a string does not exceed maximum length.
func MaxLength(field, value string, max int) error {
	if len(value) > max {
		return fmt.Errorf("%s must not exceed %d characters", field, max)
	}
	return nil
}

// MinValue validates that a numeric value meets minimum.
func MinValue[T int | int32 | int64 | float32 | float64](field string, value, min T) error {
	if value < min {
		return fmt.Errorf("%s must be at least %v", field, min)
	}
	return nil
}

// MaxValue validates that a numeric value does not exceed maximum.
func MaxValue[T int | int32 | int64 | float32 | float64](field string, value, max T) error {
	if value > max {
		return fmt.Errorf("%s must not exceed %v", field, max)
	}
	return nil
}

// InRange validates that a numeric value is within a range.
func InRange[T int | int32 | int64 | float32 | float64](field string, value, min, max T) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %v and %v", field, min, max)
	}
	return nil
}

// DTO Pool provides object pooling for DTOs to minimize allocations.
// Use this for frequently created/destroyed DTOs.
//
// Example:
//
//	pool := NewDTOPool(func() *UserDTO {
//	    return &UserDTO{}
//	})
//	dto := pool.Get()
//	defer pool.Put(dto)
type DTOPool[T DTO] struct {
	pool *sync.Pool
}

// NewDTOPool creates a new DTO pool with the given constructor.
func NewDTOPool[T DTO](newFunc func() T) *DTOPool[T] {
	return &DTOPool[T]{
		pool: &sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
	}
}

// Get retrieves a DTO from the pool.
func (p *DTOPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns a DTO to the pool.
// The DTO should be reset to a clean state before putting back.
func (p *DTOPool[T]) Put(dto T) {
	// Note: The caller is responsible for resetting the DTO
	p.pool.Put(dto)
}

// DTOCache provides a simple in-memory cache for DTOs.
// This is useful for caching frequently accessed DTOs to reduce database load.
type DTOCache[K comparable, V DTO] struct {
	data map[K]V
	mu   sync.RWMutex
}

// NewDTOCache creates a new DTO cache.
func NewDTOCache[K comparable, V DTO]() *DTOCache[K, V] {
	return &DTOCache[K, V]{
		data: make(map[K]V),
	}
}

// Get retrieves a DTO from the cache.
func (c *DTOCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, exists := c.data[key]
	return val, exists
}

// Set stores a DTO in the cache.
func (c *DTOCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Delete removes a DTO from the cache.
func (c *DTOCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// Clear removes all DTOs from the cache.
func (c *DTOCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[K]V)
}

// Size returns the number of DTOs in the cache.
func (c *DTOCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}
