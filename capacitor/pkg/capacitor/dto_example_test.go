package capacitor_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/watt-toolkit/capacitor/pkg/capacitor"
)

// Example DTO for a User entity
type UserDTO struct {
	capacitor.BaseDTO
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	version   int64
}

func (u *UserDTO) Validate(ctx context.Context) error {
	u.ClearValidationErrors()

	if err := capacitor.RequiredInt("UserID", u.UserID); err != nil {
		u.AddValidationError("UserID", err.Error())
	}
	if err := capacitor.RequiredString("Email", u.Email); err != nil {
		u.AddValidationError("Email", err.Error())
	}
	if err := capacitor.RequiredString("FirstName", u.FirstName); err != nil {
		u.AddValidationError("FirstName", err.Error())
	}
	if err := capacitor.MinValue("Age", u.Age, 0); err != nil {
		u.AddValidationError("Age", err.Error())
	}

	return u.ValidateWithType(ctx, "UserDTO")
}

func (u *UserDTO) CacheKey() string {
	return fmt.Sprintf("user:%d", u.UserID)
}

func (u *UserDTO) CacheTTL() time.Duration {
	return 30 * time.Minute
}

func (u *UserDTO) Version() int64 {
	return u.version
}

func (u *UserDTO) SetVersion(v int64) {
	u.version = v
}

// Example demonstrating basic DTO validation
func ExampleDTO_validation() {
	ctx := context.Background()

	// Valid user
	user := &UserDTO{
		UserID:    1,
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.Validate(ctx); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("User is valid")
	}

	// Invalid user (missing email)
	invalidUser := &UserDTO{
		UserID:    2,
		FirstName: "Jane",
		LastName:  "Doe",
		Age:       25,
	}

	if err := invalidUser.Validate(ctx); err != nil {
		fmt.Println("Invalid user detected")
	}

	// Output:
	// User is valid
	// Invalid user detected
}

// Example demonstrating JSON serialization
func ExampleJSONSerializer() {
	serializer := capacitor.NewJSONSerializer[*UserDTO]()

	user := &UserDTO{
		UserID:    1,
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Serialize to JSON
	data, err := serializer.Marshal(user)
	if err != nil {
		fmt.Printf("Marshal error: %v\n", err)
		return
	}

	fmt.Printf("Serialized: %d bytes\n", len(data))

	// Deserialize from JSON
	newUser := &UserDTO{}
	if err := serializer.Unmarshal(data, newUser); err != nil {
		fmt.Printf("Unmarshal error: %v\n", err)
		return
	}

	fmt.Printf("User: %s %s (ID: %d)\n", newUser.FirstName, newUser.LastName, newUser.UserID)

	// Output:
	// Serialized: 159 bytes
	// User: John Doe (ID: 1)
}

// Example demonstrating DTO pool for zero allocations
func ExampleDTOPool() {
	// Create a pool for UserDTO objects
	pool := capacitor.NewDTOPool(func() *UserDTO {
		return &UserDTO{}
	})

	// Get a DTO from the pool
	user := pool.Get()
	user.UserID = 1
	user.Email = "john@example.com"
	user.FirstName = "John"
	user.LastName = "Doe"

	fmt.Printf("User: %s %s\n", user.FirstName, user.LastName)

	// Reset and return to pool for reuse
	*user = UserDTO{} // Reset to zero value
	pool.Put(user)

	// Get another DTO (might be the same object)
	user2 := pool.Get()
	user2.UserID = 2
	user2.Email = "jane@example.com"
	user2.FirstName = "Jane"

	fmt.Printf("User: %s\n", user2.FirstName)

	pool.Put(user2)

	// Output:
	// User: John Doe
	// User: Jane
}

// Example demonstrating DTO cache
func ExampleDTOCache() {
	cache := capacitor.NewDTOCache[int64, *UserDTO]()

	// Store users in cache
	user1 := &UserDTO{UserID: 1, FirstName: "John", LastName: "Doe"}
	user2 := &UserDTO{UserID: 2, FirstName: "Jane", LastName: "Smith"}

	cache.Set(1, user1)
	cache.Set(2, user2)

	fmt.Printf("Cache size: %d\n", cache.Size())

	// Retrieve from cache
	if user, exists := cache.Get(1); exists {
		fmt.Printf("Found: %s %s\n", user.FirstName, user.LastName)
	}

	// Delete from cache
	cache.Delete(1)
	fmt.Printf("After delete: %d\n", cache.Size())

	// Output:
	// Cache size: 2
	// Found: John Doe
	// After delete: 1
}

// Example demonstrating custom validator
func ExampleValidator() {
	validator := capacitor.NewSimpleValidator[*UserDTO]()

	// Add custom validation rules
	validator.AddRule("email_required", func(ctx context.Context, dto *UserDTO) error {
		return capacitor.RequiredString("Email", dto.Email)
	})

	validator.AddRule("age_range", func(ctx context.Context, dto *UserDTO) error {
		return capacitor.InRange("Age", dto.Age, 18, 120)
	})

	validator.AddRule("name_length", func(ctx context.Context, dto *UserDTO) error {
		if err := capacitor.MinLength("FirstName", dto.FirstName, 2); err != nil {
			return err
		}
		return capacitor.MinLength("LastName", dto.LastName, 2)
	})

	ctx := context.Background()

	// Valid user
	validUser := &UserDTO{
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
	}

	if err := validator.Validate(ctx, validUser); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Valid user")
	}

	// Invalid user (age out of range)
	invalidUser := &UserDTO{
		Email:     "kid@example.com",
		FirstName: "Kid",
		LastName:  "Smith",
		Age:       10,
	}

	if err := validator.Validate(ctx, invalidUser); err != nil {
		fmt.Println("Invalid: age out of range")
	}

	// Output:
	// Valid user
	// Invalid: age out of range
}

// Example demonstrating validation helpers
func ExampleRequiredString() {
	if err := capacitor.RequiredString("Email", ""); err != nil {
		fmt.Println("Email is required")
	}

	if err := capacitor.RequiredString("Email", "john@example.com"); err != nil {
		fmt.Println("Should not reach here")
	} else {
		fmt.Println("Email is valid")
	}

	// Output:
	// Email is required
	// Email is valid
}

func ExampleMinLength() {
	if err := capacitor.MinLength("Username", "ab", 3); err != nil {
		fmt.Println("Username too short")
	}

	if err := capacitor.MinLength("Username", "john", 3); err != nil {
		fmt.Println("Should not reach here")
	} else {
		fmt.Println("Username length is valid")
	}

	// Output:
	// Username too short
	// Username length is valid
}

func ExampleInRange() {
	if err := capacitor.InRange("Age", 150, 0, 120); err != nil {
		fmt.Println("Age out of range")
	}

	if err := capacitor.InRange("Age", 30, 0, 120); err != nil {
		fmt.Println("Should not reach here")
	} else {
		fmt.Println("Age in valid range")
	}

	// Output:
	// Age out of range
	// Age in valid range
}

// Example demonstrating Cacheable interface
func ExampleCacheable() {
	user := &UserDTO{
		UserID:    1,
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	// Get cache key
	key := user.CacheKey()
	fmt.Printf("Cache key: %s\n", key)

	// Get TTL
	ttl := user.CacheTTL()
	fmt.Printf("TTL: %v\n", ttl)

	// Output:
	// Cache key: user:1
	// TTL: 30m0s
}

// Example demonstrating Versionable interface for optimistic locking
func ExampleVersionable() {
	user := &UserDTO{
		UserID: 1,
		Email:  "john@example.com",
	}

	// Set initial version
	user.SetVersion(1)
	fmt.Printf("Initial version: %d\n", user.Version())

	// Simulate an update
	user.Email = "john.doe@example.com"
	user.SetVersion(user.Version() + 1)
	fmt.Printf("After update: %d\n", user.Version())

	// Output:
	// Initial version: 1
	// After update: 2
}

// Example demonstrating SerializationFormat
func ExampleSerializationFormat_String() {
	formats := []capacitor.SerializationFormat{
		capacitor.FormatJSON,
		capacitor.FormatMsgPack,
		capacitor.FormatProtobuf,
		capacitor.FormatCustom,
	}

	for _, format := range formats {
		fmt.Println(format.String())
	}

	// Output:
	// JSON
	// MessagePack
	// Protobuf
	// Custom
}

// Example demonstrating complex validation with multiple errors
func ExampleValidationError_multiple() {
	ctx := context.Background()

	user := &UserDTO{
		// Missing UserID, Email, FirstName
		Age: -5, // Invalid age
	}

	err := user.Validate(ctx)
	if err != nil {
		if fieldErrors := capacitor.GetValidationErrors(err); fieldErrors != nil {
			fmt.Printf("Found %d validation errors:\n", len(fieldErrors))
			for _, fe := range fieldErrors {
				fmt.Printf("  - %s: %s\n", fe.Field, fe.Message)
			}
		}
	}

	// Output:
	// Found 4 validation errors:
	//   - UserID: UserID is required
	//   - Email: Email is required
	//   - FirstName: FirstName is required
	//   - Age: Age must be at least 0
}

// Example demonstrating DTO with all interfaces implemented
func ExampleDTO_complete() {
	ctx := context.Background()

	// Create a user
	user := &UserDTO{
		UserID:    1,
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Validate
	if err := user.Validate(ctx); err != nil {
		fmt.Printf("Invalid: %v\n", err)
		return
	}

	// Get cache information
	fmt.Printf("Cache key: %s\n", user.CacheKey())
	fmt.Printf("TTL: %v\n", user.CacheTTL())

	// Version tracking
	user.SetVersion(1)
	fmt.Printf("Version: %d\n", user.Version())

	// Serialize
	serializer := capacitor.NewJSONSerializer[*UserDTO]()
	data, _ := serializer.Marshal(user)
	fmt.Printf("Serialized: %d bytes\n", len(data))

	// Output:
	// Cache key: user:1
	// TTL: 30m0s
	// Version: 1
	// Serialized: 189 bytes
}

// Example demonstrating validation error handling
func ExampleValidationError_handling() {
	ctx := context.Background()

	user := &UserDTO{
		UserID: 0, // Invalid: required
		Email:  "", // Invalid: required
		Age:    -1, // Invalid: must be >= 0
		// FirstName missing: also invalid
	}

	err := user.Validate(ctx)
	if err != nil {
		// Check if it's a validation error
		var valErr *capacitor.ValidationError
		if errors.As(err, &valErr) {
			fmt.Printf("Validation failed for %s\n", valErr.Type)
			fmt.Printf("Total errors: %d\n", len(valErr.Errors))
		}
	}

	// Output:
	// Validation failed for UserDTO
	// Total errors: 4
}

// Example demonstrating DTO pool pattern for high performance
func ExampleDTOPool_pattern() {
	// Create pool with pre-allocated DTOs
	pool := capacitor.NewDTOPool(func() *UserDTO {
		return &UserDTO{
			CreatedAt: time.Now(), // Set defaults
		}
	})

	// Simulate processing multiple users
	for i := 1; i <= 3; i++ {
		// Get from pool
		user := pool.Get()

		// Use the DTO
		user.UserID = int64(i)
		user.Email = fmt.Sprintf("user%d@example.com", i)
		user.FirstName = fmt.Sprintf("User%d", i)

		fmt.Printf("Processing: %s (%s)\n", user.FirstName, user.Email)

		// Reset and return to pool
		*user = UserDTO{CreatedAt: time.Now()}
		pool.Put(user)
	}

	// Output:
	// Processing: User1 (user1@example.com)
	// Processing: User2 (user2@example.com)
	// Processing: User3 (user3@example.com)
}

// Example demonstrating field-level validation
func ExampleValidator_ValidateField() {
	validator := capacitor.NewSimpleValidator[*UserDTO]()

	validator.AddRule("email", func(ctx context.Context, dto *UserDTO) error {
		return capacitor.RequiredString("Email", dto.Email)
	})

	validator.AddRule("age", func(ctx context.Context, dto *UserDTO) error {
		return capacitor.InRange("Age", dto.Age, 18, 120)
	})

	ctx := context.Background()
	user := &UserDTO{
		Email: "john@example.com",
		Age:   15, // Invalid
	}

	// Validate specific field
	if err := validator.ValidateField(ctx, user, "age"); err != nil {
		fmt.Println("Age validation failed")
	}

	if err := validator.ValidateField(ctx, user, "email"); err != nil {
		fmt.Println("Should not reach here")
	} else {
		fmt.Println("Email is valid")
	}

	// Output:
	// Age validation failed
	// Email is valid
}
