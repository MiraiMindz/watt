---
name: generics-api
description: Expertise in Go generics and type-safe API design with Data[T] wrapper pattern. Use when implementing generic handlers, type-safe routing, automatic error handling, or metadata-rich responses.
allowed-tools: Read, Write, Edit
---

# Generics API Skill

## When to Use This Skill

Invoke this skill when you need to:
- Implement generic handlers with Data[T] wrapper
- Create type-safe route registration methods
- Design automatic error handling systems
- Build metadata-rich response systems
- Work with Go 1.18+ generics features
- Ensure compile-time type safety

## Core Concepts

### The Data[T] Pattern

The `Data[T]` wrapper provides a type-safe, metadata-rich response container:

```go
type Data[T any] struct {
    Value    T                      // The actual response data
    Error    error                  // Any error that occurred
    Metadata map[string]interface{} // Request metadata (optional)
    Status   int                    // HTTP status code
    Headers  map[string]string      // Custom response headers
}
```

**Benefits:**
1. **Type Safety** - Compile-time verification of response types
2. **Automatic Error Handling** - Framework handles Data.Error automatically
3. **Metadata Support** - Rich responses with caching info, pagination, etc.
4. **Status Code Control** - Explicit status codes with sensible defaults
5. **Header Management** - Custom headers per response

## Generic Handler Types

### Type 1: Simple Generic Handler

```go
// GenericHandler returns Data[T] for any type T
type GenericHandler[T any] func(*Context) Data[T]

// Registration
func (app *App) Get[T any](path string, handler GenericHandler[T]) *ChainLink {
    // Wrapper converts Data[T] to regular response
    wrappedHandler := func(c *Context) error {
        data := handler(c)
        return c.sendData(data)
    }
    return app.addRoute(MethodGet, path, wrappedHandler)
}
```

### Type 2: Request/Response Generic Handler

```go
// GenericTypedHandler receives Req, returns Data[Res]
type GenericTypedHandler[Req any, Res any] func(*Context, Req) Data[Res]

// Registration with automatic JSON parsing
func (app *App) PostJSON[Req any, Res any](
    path string,
    handler GenericTypedHandler[Req, Res],
) *ChainLink {
    wrappedHandler := func(c *Context) error {
        var req Req
        if err := c.BindJSON(&req); err != nil {
            return err
        }
        data := handler(c, req)
        return c.sendData(data)
    }
    return app.addRoute(MethodPost, path, wrappedHandler)
}
```

## Data[T] Constructors

### Success Constructors

```go
// OK returns 200 with data
func OK[T any](value T) Data[T] {
    return Data[T]{
        Value:  value,
        Status: 200,
    }
}

// Created returns 201 with data
func Created[T any](value T) Data[T] {
    return Data[T]{
        Value:  value,
        Status: 201,
    }
}

// Accepted returns 202 with data
func Accepted[T any](value T) Data[T] {
    return Data[T]{
        Value:  value,
        Status: 202,
    }
}

// NoContent returns 204 (no data)
func NoContent[T any]() Data[T] {
    return Data[T]{
        Status: 204,
    }
}
```

### Error Constructors

```go
// BadRequest returns 400 with error
func BadRequest[T any](err error) Data[T] {
    return Data[T]{
        Error:  err,
        Status: 400,
    }
}

// Unauthorized returns 401 with error
func Unauthorized[T any](err error) Data[T] {
    return Data[T]{
        Error:  err,
        Status: 401,
    }
}

// Forbidden returns 403 with error
func Forbidden[T any](err error) Data[T] {
    return Data[T]{
        Error:  err,
        Status: 403,
    }
}

// NotFound returns 404 with error
func NotFound[T any](err error) Data[T] {
    return Data[T]{
        Error:  err,
        Status: 404,
    }
}

// InternalServerError returns 500 with error
func InternalServerError[T any](err error) Data[T] {
    return Data[T]{
        Error:  err,
        Status: 500,
    }
}
```

### Fluent API for Metadata and Headers

```go
// WithMeta adds metadata
func (d Data[T]) WithMeta(key string, value interface{}) Data[T] {
    if d.Metadata == nil {
        d.Metadata = make(map[string]interface{})
    }
    d.Metadata[key] = value
    return d
}

// WithHeader adds custom header
func (d Data[T]) WithHeader(key, value string) Data[T] {
    if d.Headers == nil {
        d.Headers = make(map[string]string)
    }
    d.Headers[key] = value
    return d
}

// WithStatus sets custom status code
func (d Data[T]) WithStatus(status int) Data[T] {
    d.Status = status
    return d
}

// Usage:
return bolt.OK(user).
    WithMeta("cached", true).
    WithMeta("ttl", 3600).
    WithHeader("X-Cache-Hit", "true")
```

## Automatic Response Handling

### The sendData Method

```go
// sendData automatically handles Data[T] responses
func (c *Context) sendData[T any](data Data[T]) error {
    // Set status code (default 200)
    if data.Status == 0 {
        data.Status = 200
    }

    // Handle error case
    if data.Error != nil {
        return c.sendErrorData(data)
    }

    // Set custom headers
    for key, value := range data.Headers {
        c.SetHeader(key, value)
    }

    // Build response
    response := map[string]interface{}{
        "data": data.Value,
    }

    // Include metadata if present
    if len(data.Metadata) > 0 {
        response["meta"] = data.Metadata
    }

    return c.JSON(data.Status, response)
}

// sendErrorData handles error responses
func (c *Context) sendErrorData[T any](data Data[T]) error {
    response := map[string]interface{}{
        "error": data.Error.Error(),
    }

    // Include metadata in error response
    if len(data.Metadata) > 0 {
        response["meta"] = data.Metadata
    }

    return c.JSON(data.Status, response)
}
```

## Usage Examples

### Example 1: Simple GET with Type Safety

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

app.Get[User]("/users/:id", func(c *bolt.Context) bolt.Data[User] {
    id := c.Param("id")

    user, err := db.GetUser(id)
    if err != nil {
        return bolt.NotFound[User](err)
    }

    return bolt.OK(user)
})

// Compiler ensures:
// - Handler MUST return bolt.Data[User]
// - User type matches registration
// - No type casting needed
```

### Example 2: POST with Request/Response Types

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

app.PostJSON[CreateUserRequest, CreateUserResponse](
    "/users",
    func(c *bolt.Context, req CreateUserRequest) bolt.Data[CreateUserResponse] {
        // req is automatically parsed and validated

        // Validate
        if err := validate.Struct(req); err != nil {
            return bolt.BadRequest[CreateUserResponse](err)
        }

        // Create user
        user, err := db.CreateUser(req.Name, req.Email)
        if err != nil {
            return bolt.InternalServerError[CreateUserResponse](err)
        }

        // Build response
        response := CreateUserResponse{
            ID:        user.ID,
            Name:      user.Name,
            Email:     user.Email,
            CreatedAt: user.CreatedAt,
        }

        return bolt.Created(response)
    },
)

// Compiler ensures:
// - Request body must match CreateUserRequest
// - Response must be CreateUserResponse
// - No manual JSON parsing/encoding
```

### Example 3: Rich Metadata Response

```go
type Post struct {
    ID      int    `json:"id"`
    Title   string `json:"title"`
    Content string `json:"content"`
}

app.Get[[]Post]("/posts", func(c *bolt.Context) bolt.Data[[]Post] {
    page := c.QueryInt("page", 1)
    limit := c.QueryInt("limit", 10)

    posts, total, err := db.GetPosts(page, limit)
    if err != nil {
        return bolt.InternalServerError[[]Post](err)
    }

    return bolt.OK(posts).
        WithMeta("page", page).
        WithMeta("limit", limit).
        WithMeta("total", total).
        WithMeta("pages", (total + limit - 1) / limit).
        WithHeader("X-Total-Count", strconv.Itoa(total))
})

// Response:
// {
//   "data": [...posts...],
//   "meta": {
//     "page": 1,
//     "limit": 10,
//     "total": 100,
//     "pages": 10
//   }
// }
// Headers:
//   X-Total-Count: 100
```

### Example 4: Conditional Response with Caching

```go
app.Get[User]("/users/:id", func(c *bolt.Context) bolt.Data[User] {
    id := c.Param("id")

    // Check cache
    if cached, found := cache.Get(id); found {
        return bolt.OK(cached).
            WithMeta("cached", true).
            WithMeta("ttl", cache.TTL(id)).
            WithHeader("X-Cache-Hit", "true").
            WithHeader("Cache-Control", "max-age=300")
    }

    // Fetch from database
    user, err := db.GetUser(id)
    if err != nil {
        return bolt.NotFound[User](err)
    }

    // Cache for next time
    cache.Set(id, user, 5*time.Minute)

    return bolt.OK(user).
        WithMeta("cached", false).
        WithHeader("X-Cache-Hit", "false").
        WithHeader("Cache-Control", "max-age=300")
})
```

### Example 5: Generic Collection Endpoint

```go
// Generic list response
type ListResponse[T any] struct {
    Items []T `json:"items"`
    Total int `json:"total"`
}

app.Get[ListResponse[User]]("/users", func(c *bolt.Context) bolt.Data[ListResponse[User]] {
    users, total, err := db.ListUsers()
    if err != nil {
        return bolt.InternalServerError[ListResponse[User]](err)
    }

    response := ListResponse[User]{
        Items: users,
        Total: total,
    }

    return bolt.OK(response)
})

// Works for any type!
app.Get[ListResponse[Post]]("/posts", func(c *bolt.Context) bolt.Data[ListResponse[Post]] {
    posts, total, err := db.ListPosts()
    if err != nil {
        return bolt.InternalServerError[ListResponse[Post]](err)
    }

    response := ListResponse[Post]{
        Items: posts,
        Total: total,
    }

    return bolt.OK(response)
})
```

## Advanced Patterns

### Pattern 1: Optional Response Data

```go
// Option[T] for nullable responses
type Option[T any] struct {
    Value T
    Valid bool
}

app.Get[Option[User]]("/users/:id", func(c *bolt.Context) bolt.Data[Option[User]] {
    id := c.Param("id")

    user, err := db.GetUser(id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            // Return valid response with no data
            return bolt.OK(Option[User]{Valid: false})
        }
        return bolt.InternalServerError[Option[User]](err)
    }

    return bolt.OK(Option[User]{
        Value: user,
        Valid: true,
    })
})
```

### Pattern 2: Result Type (Railway-Oriented Programming)

```go
// Result[T] encapsulates success or error
type Result[T any] struct {
    Value T
    Error error
}

func (r Result[T]) IsOK() bool {
    return r.Error == nil
}

func (r Result[T]) ToData(status int) Data[T] {
    if r.Error != nil {
        return Data[T]{Error: r.Error, Status: status}
    }
    return Data[T]{Value: r.Value, Status: status}
}

// Service layer returns Result[T]
func (s *UserService) GetUser(id string) Result[User] {
    user, err := s.db.GetUser(id)
    return Result[User]{Value: user, Error: err}
}

// Handler converts Result[T] to Data[T]
app.Get[User]("/users/:id", func(c *bolt.Context) bolt.Data[User] {
    id := c.Param("id")
    result := userService.GetUser(id)

    if !result.IsOK() {
        return bolt.NotFound[User](result.Error)
    }

    return bolt.OK(result.Value)
})
```

### Pattern 3: Generic Middleware

```go
// WithValidation adds validation to generic handlers
func WithValidation[Req any, Res any](
    handler GenericTypedHandler[Req, Res],
) GenericTypedHandler[Req, Res] {
    return func(c *Context, req Req) Data[Res] {
        // Validate request
        if err := validate.Struct(req); err != nil {
            return BadRequest[Res](err)
        }

        // Call original handler
        return handler(c, req)
    }
}

// Usage:
app.PostJSON[CreateUserRequest, User](
    "/users",
    WithValidation(createUserHandler),
)
```

### Pattern 4: Generic Error Handling

```go
// Automatically convert panics to error responses
func WithRecovery[T any](handler GenericHandler[T]) GenericHandler[T] {
    return func(c *Context) Data[T] {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Panic: %v", r)
            }
        }()

        return handler(c)
    }
}
```

## Type Inference Examples

### Inference from Return Type

```go
// Compiler infers T from return statement
app.Get("/user", func(c *bolt.Context) bolt.Data[User] {
    // Type T is inferred as User
    return bolt.OK(User{ID: 1, Name: "Alice"})
})
```

### Inference from Variable

```go
// Compiler infers from variable type
var getUserHandler bolt.GenericHandler[User] = func(c *bolt.Context) bolt.Data[User] {
    return bolt.OK(User{ID: 1})
}

app.Get("/user", getUserHandler)
```

### Explicit Type Arguments

```go
// Explicit when inference isn't possible
app.Get[map[string]interface{}]("/data", func(c *bolt.Context) bolt.Data[map[string]interface{}] {
    return bolt.OK(map[string]interface{}{
        "key": "value",
    })
})
```

## Performance Considerations

### Zero-Allocation Data Construction

```go
// Pre-allocate maps when possible
func OK[T any](value T) Data[T] {
    return Data[T]{
        Value:    value,
        Status:   200,
        Metadata: nil,  // Allocate only when used
        Headers:  nil,  // Allocate only when used
    }
}

// Lazy allocation in WithMeta
func (d Data[T]) WithMeta(key string, value interface{}) Data[T] {
    if d.Metadata == nil {
        d.Metadata = make(map[string]interface{}, 4)  // Pre-size
    }
    d.Metadata[key] = value
    return d
}
```

### Avoid Generic Instantiation Overhead

```go
// ❌ BAD: Generic instantiation in hot path
func hotPath() {
    for i := 0; i < 1000000; i++ {
        data := bolt.OK(i)  // Instantiates Data[int] every time
        _ = data
    }
}

// ✅ GOOD: Instantiate once
func hotPath() {
    for i := 0; i < 1000000; i++ {
        // Compiler optimizes this
        data := Data[int]{Value: i, Status: 200}
        _ = data
    }
}
```

## Testing Generic Handlers

```go
func TestGenericHandler(t *testing.T) {
    app := New()

    // Register generic handler
    app.Get[User]("/user", func(c *Context) Data[User] {
        return OK(User{ID: 1, Name: "Test"})
    })

    // Test
    req := createTestRequest("GET", "/user")
    resp := httptest.NewRecorder()

    app.ServeHTTP(resp, req)

    // Verify
    if resp.Code != 200 {
        t.Errorf("Expected 200, got %d", resp.Code)
    }

    var result struct {
        Data User `json:"data"`
    }
    json.Unmarshal(resp.Body.Bytes(), &result)

    if result.Data.ID != 1 {
        t.Errorf("Expected ID 1, got %d", result.Data.ID)
    }
}
```

## Best Practices

1. **Use Type-Specific Constructors** - `OK[User](user)` not `Data[User]{...}`
2. **Leverage Fluent API** - Chain `WithMeta()` and `WithHeader()` for clarity
3. **Explicit Type Arguments When Ambiguous** - Help the compiler when it can't infer
4. **Pre-allocate Metadata Maps** - Use capacity hint when creating maps
5. **Return Concrete Errors** - Use sentinel errors for common cases
6. **Test Type Safety** - Verify compile-time type checking works

---

**This skill makes you an expert in Go generics and the Data[T] pattern. Use it to build type-safe, elegant APIs.**
