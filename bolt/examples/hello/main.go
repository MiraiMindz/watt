package main

import (
	"log"

	"github.com/yourusername/bolt/core"
)

// sendData is a helper function to send Data[T] responses.
// This is needed because Go doesn't support type parameters on methods.
func sendData[T any](c *core.Context, data core.Data[T]) error {
	// Set status code (default 200)
	if data.Status == 0 {
		data.Status = 200
	}

	// Handle error case
	if data.Error != nil {
		return sendErrorData(c, data)
	}

	// Set custom headers
	for key, value := range data.Headers {
		c.SetHeader(key, value)
	}

	// Build response
	response := make(map[string]interface{}, 2)
	response["data"] = data.Value

	// Include metadata if present
	if len(data.Metadata) > 0 {
		response["meta"] = data.Metadata
	}

	return c.JSON(data.Status, response)
}

// sendErrorData sends an error response.
func sendErrorData[T any](c *core.Context, data core.Data[T]) error {
	// Set custom headers (even for errors)
	for key, value := range data.Headers {
		c.SetHeader(key, value)
	}

	// Build error response
	response := make(map[string]interface{}, 2)
	response["error"] = data.Error.Error()

	// Include metadata if present
	if len(data.Metadata) > 0 {
		response["meta"] = data.Metadata
	}

	return c.JSON(data.Status, response)
}

// User represents a user in the system.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	// Create Bolt application
	app := core.New()

	// Standard API: Simple JSON response
	app.Get("/", func(c *core.Context) error {
		return c.JSON(200, map[string]string{
			"message": "Hello, Bolt!",
			"version": "1.0.0",
		})
	})

	// Data[T] API: Type-safe response using helper function
	app.Get("/users/:id", func(c *core.Context) error {
		id := c.Param("id")

		// Simulate database lookup
		if id == "" {
			data := core.BadRequest[User](core.ErrBadRequest)
			return sendData(c, data)
		}

		user := User{
			ID:    123,
			Name:  "Alice",
			Email: "alice@example.com",
		}

		data := core.OK(user).
			WithMeta("cached", true).
			WithMeta("ttl", 3600).
			WithHeader("X-Cache-Hit", "true")

		return sendData(c, data)
	})

	// Data[T] API: Type-safe request and response
	type CreateUserRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	app.Post("/users", func(c *core.Context) error {
		var req CreateUserRequest
		if err := c.BindJSON(&req); err != nil {
			data := core.BadRequest[User](core.ErrBadRequest)
			return sendData(c, data)
		}

		// Validate
		if req.Name == "" || req.Email == "" {
			data := core.BadRequest[User](core.ErrBadRequest)
			return sendData(c, data)
		}

		// Create user
		user := User{
			ID:    456,
			Name:  req.Name,
			Email: req.Email,
		}

		data := core.Created(user).
			WithMeta("created_at", "2025-11-13T00:00:00Z")

		return sendData(c, data)
	})

	// Health check endpoint
	app.Get("/health", func(c *core.Context) error {
		return c.JSON(200, map[string]string{
			"status": "healthy",
		})
	})

	// Start server
	log.Println("Starting Bolt server on :8080")
	log.Println("Try:")
	log.Println("  curl http://localhost:8080/")
	log.Println("  curl http://localhost:8080/users/123")
	log.Println("  curl -X POST http://localhost:8080/users -d '{\"name\":\"Bob\",\"email\":\"bob@example.com\"}'")

	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
