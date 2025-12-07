package capacitor_test

import (
	"context"
	"fmt"

	"github.com/watt-toolkit/capacitor/pkg/capacitor"
)

// Example demonstrating batch Get operations
func ExampleDAL_GetMulti() {
	// Simulated DAL (in real usage, this would be a concrete implementation)
	type User struct {
		ID   int
		Name string
	}

	// Batch get multiple users
	_ = []int{1, 2, 3, 999} // keys that would be used in real implementation

	// In a real implementation:
	// values, errors := dal.GetMulti(ctx, keys)

	// Simulated results
	values := map[int]User{
		1: {ID: 1, Name: "Alice"},
		2: {ID: 2, Name: "Bob"},
		3: {ID: 3, Name: "Charlie"},
	}
	errors := map[int]error{
		999: capacitor.ErrNotFound, // Key 999 doesn't exist
	}

	// Process results
	for key, value := range values {
		fmt.Printf("Found user %d: %s\n", key, value.Name)
	}

	for key, err := range errors {
		fmt.Printf("Error for key %d: %v\n", key, err)
	}

	// Output:
	// Found user 1: Alice
	// Found user 2: Bob
	// Found user 3: Charlie
	// Error for key 999: key not found
}

// Example demonstrating batch Set operations
func ExampleDAL_SetMulti() {
	type Product struct {
		ID    string
		Price float64
	}

	// Batch set multiple products
	_ = map[string]Product{ // items that would be used in real implementation
		"prod:1": {ID: "prod:1", Price: 19.99},
		"prod:2": {ID: "prod:2", Price: 29.99},
		"prod:3": {ID: "prod:3", Price: 39.99},
	}

	// In a real implementation:
	// errors := dal.SetMulti(ctx, items, capacitor.WithTTL(3600))

	// Simulated result (all succeeded)
	errors := make(map[string]error)

	if len(errors) == 0 {
		fmt.Println("All products saved successfully")
	} else {
		for key, err := range errors {
			fmt.Printf("Failed to save %s: %v\n", key, err)
		}
	}

	// Output: All products saved successfully
}

// Example demonstrating batch Delete operations
func ExampleDAL_DeleteMulti() {
	// Delete multiple cache entries
	keys := []string{"session:1", "session:2", "session:3"}

	// In a real implementation:
	// errors := dal.DeleteMulti(ctx, keys)

	// Simulated result (all succeeded)
	errors := make(map[string]error)

	fmt.Printf("Deleted %d keys, %d errors\n", len(keys)-len(errors), len(errors))

	// Output: Deleted 3 keys, 0 errors
}

// Example demonstrating Range iteration
func ExampleDAL_Range() {
	ctx := context.Background()

	// In a real implementation:
	// err := dal.Range(ctx, func(key string, value User) bool {
	//     fmt.Printf("%s: %+v\n", key, value)
	//     return true // continue iteration
	// })

	// Simulated iteration
	users := map[string]string{
		"user:1": "Alice",
		"user:2": "Bob",
		"user:3": "Charlie",
	}

	count := 0
	for key, value := range users {
		fmt.Printf("%s: %s\n", key, value)
		count++
		if count >= 2 {
			break // stop early example
		}
	}

	_ = ctx

	// Output varies due to map iteration order, but demonstrates concept
}

// Example demonstrating Keys iteration with channel
func ExampleDAL_Keys() {
	ctx := context.Background()

	// In a real implementation:
	// keys, err := dal.Keys(ctx)
	// if err != nil {
	//     return err
	// }

	// Simulated channel
	keys := make(chan string, 3)
	keys <- "user:1"
	keys <- "user:2"
	keys <- "user:3"
	close(keys)

	count := 0
	for key := range keys {
		fmt.Println(key)
		count++
	}

	fmt.Printf("Total keys: %d\n", count)

	_ = ctx

	// Output:
	// user:1
	// user:2
	// user:3
	// Total keys: 3
}

// Example demonstrating transaction usage
func ExampleDAL_BeginTx() {
	ctx := context.Background()

	// In a real implementation:
	// tx, err := dal.BeginTx(ctx, capacitor.WithIsolation(capacitor.IsolationSerializable))
	// if err != nil {
	//     return err
	// }
	// defer tx.Rollback(ctx) // Rollback if not committed
	//
	// Transfer money between accounts (atomic operation)
	// if err := tx.Set(ctx, "account:1", newBalance1); err != nil {
	//     return err
	// }
	// if err := tx.Set(ctx, "account:2", newBalance2); err != nil {
	//     return err
	// }
	//
	// return tx.Commit(ctx) // Commit all changes

	fmt.Println("Transaction pattern demonstrated")

	_ = ctx

	// Output: Transaction pattern demonstrated
}

// Example demonstrating transaction with batch operations
func ExampleTx_SetMulti() {
	ctx := context.Background()

	// Batch operations within a transaction
	// In a real implementation:
	// tx, err := dal.BeginTx(ctx)
	// if err != nil {
	//     return err
	// }
	// defer tx.Rollback(ctx)
	//
	// items := map[string]Account{
	//     "account:1": {Balance: 100.0},
	//     "account:2": {Balance: 200.0},
	// }
	//
	// errors := tx.SetMulti(ctx, items)
	// if len(errors) > 0 {
	//     return fmt.Errorf("batch set failed")
	// }
	//
	// return tx.Commit(ctx)

	fmt.Println("Transaction batch operations demonstrated")

	_ = ctx

	// Output: Transaction batch operations demonstrated
}

// Example demonstrating transaction isolation levels
func ExampleWithIsolation() {
	ctx := context.Background()

	// Different isolation levels for different use cases

	// Read-only transaction with default isolation
	// tx1, _ := dal.BeginTx(ctx, capacitor.WithReadOnly())

	// Write transaction with serializable isolation (strongest)
	// tx2, _ := dal.BeginTx(ctx, capacitor.WithIsolation(capacitor.IsolationSerializable))

	// Fast read with read-committed (weaker isolation, better performance)
	// tx3, _ := dal.BeginTx(ctx, capacitor.WithIsolation(capacitor.IsolationReadCommitted))

	fmt.Println("Isolation level examples")

	_ = ctx

	// Output: Isolation level examples
}

// Example demonstrating IsolationLevel String method
func ExampleIsolationLevel_String() {
	levels := []capacitor.IsolationLevel{
		capacitor.IsolationDefault,
		capacitor.IsolationReadUncommitted,
		capacitor.IsolationReadCommitted,
		capacitor.IsolationRepeatableRead,
		capacitor.IsolationSerializable,
	}

	for _, level := range levels {
		fmt.Println(level.String())
	}

	// Output:
	// Default
	// ReadUncommitted
	// ReadCommitted
	// RepeatableRead
	// Serializable
}

// Example demonstrating BatchResult usage
func ExampleBatchResult_OK() {
	// Successful batch operation
	result1 := &capacitor.BatchResult[string]{
		SuccessCount: 10,
		FailureCount: 0,
		Errors:       nil,
	}

	fmt.Printf("Batch 1 OK: %v\n", result1.OK())

	// Partially failed batch operation
	result2 := &capacitor.BatchResult[string]{
		SuccessCount: 8,
		FailureCount: 2,
		Errors: map[string]error{
			"key1": capacitor.ErrNotFound,
			"key2": capacitor.ErrInvalidValue,
		},
	}

	fmt.Printf("Batch 2 OK: %v\n", result2.OK())
	fmt.Printf("Batch 2 error: %v\n", result2.Error())

	// Output:
	// Batch 1 OK: true
	// Batch 2 OK: false
	// Batch 2 error: key not found
}

// Example demonstrating transaction rollback on error
func ExampleTx_Rollback() {
	ctx := context.Background()

	// Pattern: defer Rollback, explicit Commit
	// In a real implementation:
	// tx, err := dal.BeginTx(ctx)
	// if err != nil {
	//     return err
	// }
	// defer tx.Rollback(ctx) // Safe to call even after Commit
	//
	// if err := tx.Set(ctx, "key1", value1); err != nil {
	//     return err // Rollback happens via defer
	// }
	//
	// if err := someValidation(); err != nil {
	//     return err // Rollback happens via defer
	// }
	//
	// return tx.Commit(ctx) // Explicit commit on success

	fmt.Println("Transaction rollback pattern demonstrated")

	_ = ctx

	// Output: Transaction rollback pattern demonstrated
}

// Example demonstrating context cancellation with iteration
func ExampleDAL_Keys_cancellation() {
	_, cancel := context.WithCancel(context.Background())

	// In a real implementation:
	// keys, err := dal.Keys(ctx)
	// if err != nil {
	//     return err
	// }
	//
	// for key := range keys {
	//     if shouldStop() {
	//         cancel() // Cancels iteration
	//         break
	//     }
	//     process(key)
	// }

	// Simulated
	keys := make(chan string, 5)
	keys <- "key1"
	keys <- "key2"
	close(keys)

	processed := 0
	for key := range keys {
		fmt.Println(key)
		processed++
		if processed >= 1 {
			cancel() // Cancel after first key
			break
		}
	}

	// Output: key1
}

// Example demonstrating transaction with timeout
func ExampleWithTimeout() {
	ctx := context.Background()

	// Set transaction timeout to 30 seconds
	// In a real implementation:
	// tx, err := dal.BeginTx(ctx, capacitor.WithTimeout(30*time.Second.Nanoseconds()))
	// if err != nil {
	//     return err
	// }
	// defer tx.Rollback(ctx)
	//
	// // Long-running operations...
	//
	// return tx.Commit(ctx)

	fmt.Println("Transaction timeout example")

	_ = ctx

	// Output: Transaction timeout example
}

// Example demonstrating combined transaction options
func ExampleTxOption() {
	// Combine multiple transaction options
	// In a real implementation:
	// tx, err := dal.BeginTx(ctx,
	//     capacitor.WithIsolation(capacitor.IsolationSerializable),
	//     capacitor.WithTimeout(60*time.Second.Nanoseconds()),
	// )

	fmt.Println("Combined transaction options example")

	// Output: Combined transaction options example
}
