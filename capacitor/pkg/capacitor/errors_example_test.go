package capacitor_test

import (
	"errors"
	"fmt"

	"github.com/watt-toolkit/capacitor/pkg/capacitor"
)

// Example demonstrating basic error handling with sentinel errors
func ExampleIsNotFound() {
	// Simulate a cache Get operation that returns ErrNotFound
	var err error = capacitor.ErrNotFound

	// Check if the error is a "not found" error
	if capacitor.IsNotFound(err) {
		fmt.Println("Key not found in cache")
	}

	// Output: Key not found in cache
}

// Example demonstrating CacheError wrapping
func ExampleWrapCacheError() {
	// Simulate a cache operation that fails
	originalErr := errors.New("connection timeout")
	err := capacitor.WrapCacheError("L1", "Get", "user:123", originalErr)

	fmt.Println(err.Error())

	// Check if it's still the original error
	if errors.Is(err, originalErr) {
		fmt.Println("Original error preserved")
	}

	// Output:
	// cache error in layer L1 during Get (key: user:123): connection timeout
	// Original error preserved
}

// Example demonstrating DatabaseError wrapping
func ExampleWrapDatabaseError() {
	// Simulate a database operation that fails
	originalErr := errors.New("constraint violation")
	err := capacitor.WrapDatabaseError("postgres", "Insert", "users", "INSERT INTO users...", originalErr)

	fmt.Println(err.Error())

	// Output: database error in postgres during Insert: constraint violation
}

// Example demonstrating ValidationError with field errors
func ExampleValidationError() {
	// Create a validation error with multiple field errors
	valErr := &capacitor.ValidationError{
		Type: "User",
	}
	valErr.AddFieldError("Email", "invalid email format")
	valErr.AddFieldError("Age", "must be at least 18")

	fmt.Println(valErr.Error())

	// Extract field errors for detailed handling
	if fieldErrors := capacitor.GetValidationErrors(valErr); fieldErrors != nil {
		for _, fe := range fieldErrors {
			fmt.Printf("  - %s: %s\n", fe.Field, fe.Message)
		}
	}

	// Output:
	// validation failed for User: 2 errors
	//   - Email: invalid email format
	//   - Age: must be at least 18
}

// Example demonstrating retryable error checking
func ExampleIsRetryable() {
	// Network error that can be retried
	err1 := capacitor.WrapNetworkError("redis:6379", "Get", errors.New("timeout"), true)
	fmt.Printf("Network timeout retryable: %v\n", capacitor.IsRetryable(err1))

	// Network error that cannot be retried (e.g., auth failure)
	err2 := capacitor.WrapNetworkError("redis:6379", "Auth", errors.New("invalid password"), false)
	fmt.Printf("Auth failure retryable: %v\n", capacitor.IsRetryable(err2))

	// Timeout error (always retryable)
	fmt.Printf("Timeout retryable: %v\n", capacitor.IsRetryable(capacitor.ErrTimeout))

	// Output:
	// Network timeout retryable: true
	// Auth failure retryable: false
	// Timeout retryable: true
}

// Example demonstrating error chain handling
func ExampleCacheError_unwrap() {
	// Create a chain of errors
	original := capacitor.ErrNotFound
	wrapped := capacitor.WrapCacheError("L1", "Get", "user:123", original)

	// Check if any error in the chain is ErrNotFound
	if errors.Is(wrapped, capacitor.ErrNotFound) {
		fmt.Println("Found ErrNotFound in error chain")
	}

	// Extract the CacheError for detailed information
	var cacheErr *capacitor.CacheError
	if errors.As(wrapped, &cacheErr) {
		fmt.Printf("Layer: %s, Operation: %s, Key: %s\n", cacheErr.Layer, cacheErr.Op, cacheErr.Key)
	}

	// Output:
	// Found ErrNotFound in error chain
	// Layer: L1, Operation: Get, Key: user:123
}

// Example demonstrating serialization error handling
func ExampleWrapSerializationError() {
	// Simulate a JSON serialization failure
	originalErr := errors.New("unsupported type: chan int")
	err := capacitor.WrapSerializationError("json", "Message", originalErr)

	fmt.Println(err.Error())

	// Output: serialization failed for type Message using json: unsupported type: chan int
}

// Example demonstrating practical error handling in a DAL operation
func ExampleDatabaseError_practical() {
	// Simulated database operation
	performDatabaseOperation := func() error {
		// Simulate a query that fails
		queryErr := errors.New("deadlock detected")
		return capacitor.WrapDatabaseError("postgres", "Update", "accounts", "UPDATE accounts SET balance...", queryErr)
	}

	err := performDatabaseOperation()
	if err != nil {
		// Check if it's a retryable error
		if capacitor.IsRetryable(err) {
			fmt.Println("Error is retryable, will retry with backoff")
		}

		// Get detailed error information
		var dbErr *capacitor.DatabaseError
		if errors.As(err, &dbErr) {
			fmt.Printf("Database: %s\n", dbErr.Backend)
			fmt.Printf("Table: %s\n", dbErr.Table)
		}
	}

	// Output:
	// Database: postgres
	// Table: accounts
}
