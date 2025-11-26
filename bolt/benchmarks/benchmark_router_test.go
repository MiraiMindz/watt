package benchmarks

import (
	"testing"

	"github.com/yourusername/bolt/core"
)

// ============================================================================
// OPTION B: Simpler Router-Based Benchmarks
// ============================================================================
// These benchmarks test the router directly without the full HTTP stack.
// They measure pure routing and handler execution performance.
//
// Run with: go test -bench=BenchmarkRouter -benchmem -benchtime=10s ./benchmarks
// ============================================================================

// ============================================================================
// Scenario 1: Static Route Lookup
// ============================================================================

func BenchmarkRouter_StaticRoute(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		c.JSON(200, SimpleResponse{Message: "pong"})
		return nil
	}

	router.Add("GET", "/ping", handler)

	// Create a test context
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/ping")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.ServeHTTP(ctx)
	}
}

// ============================================================================
// Scenario 2: Dynamic Route with Parameters
// ============================================================================

func BenchmarkRouter_DynamicRoute(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		id := c.Param("id")
		c.JSON(200, UserResponse{
			ID:    123,
			Name:  "User " + id,
			Email: "user" + id + "@example.com",
		})
		return nil
	}

	router.Add("GET", "/users/:id", handler)

	// Create a test context
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/users/123")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.ServeHTTP(ctx)
	}
}

// ============================================================================
// Scenario 3: Multiple Dynamic Parameters
// ============================================================================

func BenchmarkRouter_MultipleParams(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		userID := c.Param("user_id")
		postID := c.Param("post_id")
		c.JSON(200, map[string]string{
			"user_id": userID,
			"post_id": postID,
		})
		return nil
	}

	router.Add("GET", "/users/:user_id/posts/:post_id", handler)

	// Create a test context
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/users/123/posts/456")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.ServeHTTP(ctx)
	}
}

// ============================================================================
// Scenario 4: Router with Many Routes (Scalability)
// ============================================================================

func BenchmarkRouter_ManyRoutes_StaticLookup(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		return c.JSON(200, SimpleResponse{Message: "ok"})
	}

	// Register 100 routes
	for i := 0; i < 100; i++ {
		router.Add("GET", "/route"+string(rune(i)), handler)
	}

	// Lookup a route in the middle
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/route50")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.ServeHTTP(ctx)
	}
}

func BenchmarkRouter_ManyRoutes_DynamicLookup(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		return c.JSON(200, SimpleResponse{Message: "ok"})
	}

	// Register 100 dynamic routes
	for i := 0; i < 100; i++ {
		router.Add("GET", "/users/:id/action"+string(rune(i)), handler)
	}

	// Lookup a route in the middle
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/users/123/action50")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.ServeHTTP(ctx)
	}
}

// ============================================================================
// Scenario 5: Wildcard Routes
// ============================================================================

func BenchmarkRouter_WildcardRoute(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		path := c.Param("path")
		return c.JSON(200, map[string]string{"path": path})
	}

	router.Add("GET", "/static/*path", handler)

	// Create a test context
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/static/css/style.css")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.ServeHTTP(ctx)
	}
}

// ============================================================================
// Scenario 6: Mixed Route Types
// ============================================================================

func BenchmarkRouter_MixedRoutes(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		return c.JSON(200, SimpleResponse{Message: "ok"})
	}

	// Register mix of static and dynamic routes
	router.Add("GET", "/", handler)
	router.Add("GET", "/about", handler)
	router.Add("GET", "/contact", handler)
	router.Add("GET", "/users/:id", handler)
	router.Add("GET", "/posts/:id", handler)
	router.Add("POST", "/users", handler)
	router.Add("PUT", "/users/:id", handler)
	router.Add("DELETE", "/users/:id", handler)
	router.Add("GET", "/api/v1/users/:id/posts", handler)
	router.Add("GET", "/files/*path", handler)

	b.Run("Static", func(b *testing.B) {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/about")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = router.ServeHTTP(ctx)
		}
	})

	b.Run("Dynamic", func(b *testing.B) {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/users/123")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = router.ServeHTTP(ctx)
		}
	})

	b.Run("Nested", func(b *testing.B) {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/api/v1/users/456/posts")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = router.ServeHTTP(ctx)
		}
	})

	b.Run("Wildcard", func(b *testing.B) {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/files/documents/report.pdf")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = router.ServeHTTP(ctx)
		}
	})
}

// ============================================================================
// Scenario 7: Parameter Extraction Performance
// ============================================================================

func BenchmarkRouter_ParamExtraction_Single(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		_ = c.Param("id")
		return nil
	}

	router.Add("GET", "/users/:id", handler)

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/users/123")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.ServeHTTP(ctx)
	}
}

func BenchmarkRouter_ParamExtraction_Multiple(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		_ = c.Param("user_id")
		_ = c.Param("post_id")
		_ = c.Param("comment_id")
		return nil
	}

	router.Add("GET", "/users/:user_id/posts/:post_id/comments/:comment_id", handler)

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/users/123/posts/456/comments/789")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.ServeHTTP(ctx)
	}
}

// ============================================================================
// Scenario 8: Concurrent Router Access
// ============================================================================

func BenchmarkRouter_Concurrent(b *testing.B) {
	router := core.NewRouter()
	handler := func(c *core.Context) error {
		return c.JSON(200, SimpleResponse{Message: "ok"})
	}

	// Register routes
	router.Add("GET", "/ping", handler)
	router.Add("GET", "/users/:id", handler)
	router.Add("POST", "/users", handler)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/users/123")

		for pb.Next() {
			_ = router.ServeHTTP(ctx)
		}
	})
}
