// Package core provides the core types and functionality for the Bolt web framework.
//
// Bolt achieves 1.7-3.7x better performance than standard library through:
//   - Shockwave HTTP server integration (zero-copy parsing)
//   - Dual API design (Sugared + Unsugared + Generics)
//   - Aggressive object pooling
//   - Zero-allocation code paths
//
// # Generics API Usage
//
// IMPORTANT - Go Limitation:
// Methods cannot have type parameters independent of the receiver type.
// This means we CANNOT create generic methods like app.Get[T]() or app.Post[T]().
//
// Instead, use the Data[T] wrapper with standalone sendData helper function:
//
//	app := bolt.New()
//	app.Get("/users/:id", func(c *bolt.Context) error {
//	    user, err := db.GetUser(c.Param("id"))
//	    if err != nil {
//	        return sendData(c, bolt.NotFound[User](err))
//	    }
//	    data := bolt.OK(user).
//	        WithMeta("cached", true).
//	        WithHeader("X-Cache-Hit", "true")
//	    return sendData(c, data)
//	})
//	app.Listen(":8080")
//
// See examples/hello/main.go for complete working example.
package core

// Data wraps a response value with metadata and error handling.
//
// The Data[T] pattern provides:
//   - Type safety: Compile-time verification of response types
//   - Automatic error handling: Framework handles Data.Error automatically
//   - Metadata support: Rich responses with caching info, pagination, etc.
//   - Status code control: Explicit status codes with sensible defaults
//   - Header management: Custom headers per response
//
// Example:
//
//	return bolt.OK(user).
//	    WithMeta("cached", true).
//	    WithMeta("ttl", 3600).
//	    WithHeader("X-Cache-Hit", "true")
type Data[T any] struct {
	// Value is the actual response data
	Value T `json:"data,omitempty"`

	// Error is any error that occurred during request processing
	Error error `json:"error,omitempty"`

	// Metadata contains additional information about the response
	// (e.g., pagination, caching, timing information)
	Metadata map[string]interface{} `json:"meta,omitempty"`

	// Status is the HTTP status code (default: 200 for GET, 201 for POST)
	Status int `json:"-"`

	// Headers contains custom response headers
	Headers map[string]string `json:"-"`
}

// Result wraps a Data[T] with an additional error field.
//
// This is useful for operations that may return either data or an error,
// providing a type-safe alternative to (Data[T], error) tuple returns.
//
// Example:
//
//	func getUser(id string) Result[User] {
//	    user, err := db.GetUser(id)
//	    if err != nil {
//	        return Result[User]{Err: err}
//	    }
//	    return Result[User]{Data: &Data[User]{Value: user, Status: 200}}
//	}
type Result[T any] struct {
	// Data contains the successful response data
	Data *Data[T]

	// Err contains any error that occurred
	Err error
}

// GenericHandler defines a handler that returns Data[T].
//
// The framework automatically:
//   - Serializes Data.Value to JSON
//   - Handles Data.Error (returns error response)
//   - Sets Data.Status (defaults to 200)
//   - Includes Data.Metadata in response
//   - Sets Data.Headers in HTTP response
//
// Example:
//
//	app.Get[User]("/users/:id", func(c *bolt.Context) bolt.Data[User] {
//	    return bolt.OK(user)
//	})
type GenericHandler[T any] func(*Context) Data[T]

// GenericTypedHandler defines a handler with automatic JSON parsing.
//
// The framework automatically:
//   - Parses request body into Req type
//   - Validates JSON structure
//   - Passes parsed Req to handler
//   - Handles response as with GenericHandler
//
// Example:
//
//	app.PostJSON[CreateUserRequest, User]("/users",
//	    func(c *bolt.Context, req CreateUserRequest) bolt.Data[User] {
//	        user := createUser(req)
//	        return bolt.Created(user)
//	    })
type GenericTypedHandler[Req any, Res any] func(*Context, Req) Data[Res]

// OK returns a 200 OK response with the given value.
//
// Example:
//
//	return bolt.OK(user)
func OK[T any](value T) Data[T] {
	return Data[T]{
		Value:  value,
		Status: 200,
	}
}

// Created returns a 201 Created response with the given value.
//
// Example:
//
//	return bolt.Created(newUser)
func Created[T any](value T) Data[T] {
	return Data[T]{
		Value:  value,
		Status: 201,
	}
}

// Accepted returns a 202 Accepted response with the given value.
//
// Use for asynchronous operations that have been accepted but not completed.
//
// Example:
//
//	return bolt.Accepted(job)
func Accepted[T any](value T) Data[T] {
	return Data[T]{
		Value:  value,
		Status: 202,
	}
}

// NoContent returns a 204 No Content response.
//
// Use for successful operations that return no data (e.g., DELETE).
//
// Example:
//
//	return bolt.NoContent[any]()
func NoContent[T any]() Data[T] {
	return Data[T]{
		Status: 204,
	}
}

// BadRequest returns a 400 Bad Request error response.
//
// Example:
//
//	if err := validate(req); err != nil {
//	    return bolt.BadRequest[User](err)
//	}
func BadRequest[T any](err error) Data[T] {
	return Data[T]{
		Error:  err,
		Status: 400,
	}
}

// Unauthorized returns a 401 Unauthorized error response.
//
// Example:
//
//	if !authenticated {
//	    return bolt.Unauthorized[User](errors.New("invalid token"))
//	}
func Unauthorized[T any](err error) Data[T] {
	return Data[T]{
		Error:  err,
		Status: 401,
	}
}

// Forbidden returns a 403 Forbidden error response.
//
// Example:
//
//	if !authorized {
//	    return bolt.Forbidden[User](errors.New("insufficient permissions"))
//	}
func Forbidden[T any](err error) Data[T] {
	return Data[T]{
		Error:  err,
		Status: 403,
	}
}

// NotFound returns a 404 Not Found error response.
//
// Example:
//
//	user, err := db.GetUser(id)
//	if err != nil {
//	    return bolt.NotFound[User](err)
//	}
func NotFound[T any](err error) Data[T] {
	return Data[T]{
		Error:  err,
		Status: 404,
	}
}

// Conflict returns a 409 Conflict error response.
//
// Use for operations that conflict with current state (e.g., duplicate entry).
//
// Example:
//
//	if exists {
//	    return bolt.Conflict[User](errors.New("user already exists"))
//	}
func Conflict[T any](err error) Data[T] {
	return Data[T]{
		Error:  err,
		Status: 409,
	}
}

// InternalServerError returns a 500 Internal Server Error response.
//
// Example:
//
//	if err != nil {
//	    return bolt.InternalServerError[User](err)
//	}
func InternalServerError[T any](err error) Data[T] {
	return Data[T]{
		Error:  err,
		Status: 500,
	}
}

// InternalError is an alias for InternalServerError.
//
// Example:
//
//	if err != nil {
//	    return bolt.InternalError[User](err)
//	}
func InternalError[T any](err error) Data[T] {
	return InternalServerError[T](err)
}

// WithMeta adds metadata to the Data response.
//
// Metadata is included in the response as a "meta" field and can contain
// pagination info, caching details, timing information, etc.
//
// Example:
//
//	return bolt.OK(users).
//	    WithMeta("page", 1).
//	    WithMeta("total", 100).
//	    WithMeta("cached", true)
//
// Response:
//
//	{
//	  "data": [...users...],
//	  "meta": {
//	    "page": 1,
//	    "total": 100,
//	    "cached": true
//	  }
//	}
func (d Data[T]) WithMeta(key string, value interface{}) Data[T] {
	if d.Metadata == nil {
		d.Metadata = make(map[string]interface{}, 4)
	}
	d.Metadata[key] = value
	return d
}

// WithHeader adds a custom HTTP header to the response.
//
// Example:
//
//	return bolt.OK(data).
//	    WithHeader("X-Cache-Hit", "true").
//	    WithHeader("X-RateLimit-Remaining", "100")
func (d Data[T]) WithHeader(key, value string) Data[T] {
	if d.Headers == nil {
		d.Headers = make(map[string]string, 4)
	}
	d.Headers[key] = value
	return d
}

// WithStatus sets a custom HTTP status code.
//
// Use sparingly; prefer the built-in constructors (OK, Created, etc.).
//
// Example:
//
//	return bolt.OK(data).WithStatus(206) // Partial Content
func (d Data[T]) WithStatus(status int) Data[T] {
	d.Status = status
	return d
}

// WithError sets an error on the Data response.
//
// This converts a successful response into an error response.
// Use when you need to conditionally return an error.
//
// Example:
//
//	data := bolt.OK(user)
//	if !canAccess {
//	    data = data.WithError(errors.New("access denied")).WithStatus(403)
//	}
//	return data
func (d Data[T]) WithError(err error) Data[T] {
	d.Error = err
	return d
}

// WithMetadata sets multiple metadata entries at once.
//
// Example:
//
//	metadata := map[string]interface{}{
//	    "page": 1,
//	    "limit": 10,
//	    "total": 100,
//	}
//	return bolt.OK(users).WithMetadata(metadata)
func (d Data[T]) WithMetadata(metadata map[string]interface{}) Data[T] {
	if d.Metadata == nil {
		d.Metadata = make(map[string]interface{}, len(metadata))
	}
	for k, v := range metadata {
		d.Metadata[k] = v
	}
	return d
}

// WithHeaders sets multiple HTTP headers at once.
//
// Example:
//
//	headers := map[string]string{
//	    "X-Cache-Hit": "true",
//	    "X-RateLimit-Remaining": "100",
//	}
//	return bolt.OK(data).WithHeaders(headers)
func (d Data[T]) WithHeaders(headers map[string]string) Data[T] {
	if d.Headers == nil {
		d.Headers = make(map[string]string, len(headers))
	}
	for k, v := range headers {
		d.Headers[k] = v
	}
	return d
}

// sendData is a helper function (not a method) that sends a Data[T] response to the client.
//
// This is used internally to handle Data[T] responses.
//
// Behavior:
//   - If Data.Error != nil: Returns error response with Data.Status
//   - If Data.Status == 0: Defaults to 200
//   - Serializes Data.Value to JSON
//   - Includes Data.Metadata if present
//   - Sets Data.Headers in HTTP response
//
// Response format (success):
//
//	{
//	  "data": <value>,
//	  "meta": <metadata>  // if present
//	}
//
// Response format (error):
//
//	{
//	  "error": "<error message>",
//	  "meta": <metadata>  // if present
//	}
func sendData[T any](c *Context, data Data[T]) error {
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

// sendErrorData is a helper function that sends an error response to the client.
func sendErrorData[T any](c *Context, data Data[T]) error {
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
