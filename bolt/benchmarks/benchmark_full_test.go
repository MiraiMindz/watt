package benchmarks

import (
	"fmt"
	"net/http/httptest"
	"testing"

	// Bolt framework
	"github.com/yourusername/bolt/core"

	// Competitor frameworks
	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
)

// ============================================================================
// OPTION A: Full Benchmarks with http.Handler Interface
// ============================================================================
// These benchmarks use the standard http.Handler interface (ServeHTTP method)
// for fair comparison across all frameworks.
//
// Run with: go test -bench=BenchmarkFull -benchmem -benchtime=10s ./benchmarks
// ============================================================================

// ============================================================================
// Test Data Structures
// ============================================================================

type SimpleResponse struct {
	Message string `json:"message"`
}

type UserResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type LargeDataItem struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Price       float64                `json:"price"`
	Stock       int                    `json:"stock"`
	Categories  []string               `json:"categories"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type LargeDataResponse struct {
	Total int             `json:"total"`
	Items []LargeDataItem `json:"items"`
}

type SearchResponse struct {
	Query   string   `json:"query"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
	Sort    string   `json:"sort"`
	Filters []string `json:"filters"`
	Results int      `json:"results"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// generateLargeData creates a ~10KB JSON response with 100 items
func generateLargeData() LargeDataResponse {
	items := make([]LargeDataItem, 100)
	for i := 0; i < 100; i++ {
		items[i] = LargeDataItem{
			ID:          i + 1,
			Name:        fmt.Sprintf("Product %d", i+1),
			Description: "This is a detailed product description that adds some size to the response payload for realistic testing scenarios",
			Price:       99.99 + float64(i),
			Stock:       100 + i*10,
			Categories:  []string{"Electronics", "Computers", "Laptops"},
			Metadata: map[string]interface{}{
				"brand":      "TechBrand",
				"model":      fmt.Sprintf("Model-%d", i),
				"year":       2024,
				"featured":   i%2 == 0,
				"rating":     4.5,
				"reviews":    i * 10,
				"warranty":   "2 years",
				"shipping":   "Free",
				"in_stock":   true,
				"popularity": i * 100,
			},
		}
	}
	return LargeDataResponse{
		Total: 100,
		Items: items,
	}
}

// ============================================================================
// Scenario 1: Static Route - Simple JSON Response
// ============================================================================

func BenchmarkFull_Bolt_StaticRoute(b *testing.B) {
	app := core.New()
	app.Get("/ping", func(c *core.Context) error {
		return c.JSON(200, SimpleResponse{Message: "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Gin_StaticRoute(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, SimpleResponse{Message: "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Echo_StaticRoute(b *testing.B) {
	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.JSON(200, SimpleResponse{Message: "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		e.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Fiber_StaticRoute(b *testing.B) {
	app := fiber.New()
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(SimpleResponse{Message: "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

// ============================================================================
// Scenario 2: Dynamic Route - Path Parameters
// ============================================================================

func BenchmarkFull_Bolt_DynamicRoute(b *testing.B) {
	app := core.New()
	app.Get("/users/:id", func(c *core.Context) error {
		id := c.Param("id")
		return c.JSON(200, UserResponse{
			ID:    123,
			Name:  "User " + id,
			Email: "user" + id + "@example.com",
		})
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Gin_DynamicRoute(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, UserResponse{
			ID:    123,
			Name:  "User " + id,
			Email: "user" + id + "@example.com",
		})
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Echo_DynamicRoute(b *testing.B) {
	e := echo.New()
	e.GET("/users/:id", func(c echo.Context) error {
		id := c.Param("id")
		return c.JSON(200, UserResponse{
			ID:    123,
			Name:  "User " + id,
			Email: "user" + id + "@example.com",
		})
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		e.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Fiber_DynamicRoute(b *testing.B) {
	app := fiber.New()
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		return c.JSON(UserResponse{
			ID:    123,
			Name:  "User " + id,
			Email: "user" + id + "@example.com",
		})
	})

	req := httptest.NewRequest("GET", "/users/123", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

// ============================================================================
// Scenario 3: Middleware Chain (5 middleware)
// ============================================================================

func BenchmarkFull_Bolt_MiddlewareChain(b *testing.B) {
	app := core.New()

	// Add 5 no-op middleware
	for i := 0; i < 5; i++ {
		app.Use(func(next core.Handler) core.Handler {
			return func(c *core.Context) error {
				return next(c)
			}
		})
	}

	app.Get("/test", func(c *core.Context) error {
		return c.JSON(200, SimpleResponse{Message: "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Gin_MiddlewareChain(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Add 5 no-op middleware
	for i := 0; i < 5; i++ {
		r.Use(func(c *gin.Context) {
			c.Next()
		})
	}

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, SimpleResponse{Message: "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Echo_MiddlewareChain(b *testing.B) {
	e := echo.New()

	// Add 5 no-op middleware
	for i := 0; i < 5; i++ {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		})
	}

	e.GET("/test", func(c echo.Context) error {
		return c.JSON(200, SimpleResponse{Message: "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		e.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Fiber_MiddlewareChain(b *testing.B) {
	app := fiber.New()

	// Add 5 no-op middleware
	for i := 0; i < 5; i++ {
		app.Use(func(c *fiber.Ctx) error {
			return c.Next()
		})
	}

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(SimpleResponse{Message: "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

// ============================================================================
// Scenario 4: Large JSON Encoding (~10KB)
// ============================================================================

func BenchmarkFull_Bolt_LargeJSON(b *testing.B) {
	largeData := generateLargeData()
	app := core.New()
	app.Get("/data", func(c *core.Context) error {
		return c.JSONLarge(200, largeData) // Use JSONLarge for large payloads (>8KB)
	})

	req := httptest.NewRequest("GET", "/data", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Gin_LargeJSON(b *testing.B) {
	largeData := generateLargeData()
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/data", func(c *gin.Context) {
		c.JSON(200, largeData)
	})

	req := httptest.NewRequest("GET", "/data", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Echo_LargeJSON(b *testing.B) {
	largeData := generateLargeData()
	e := echo.New()
	e.GET("/data", func(c echo.Context) error {
		return c.JSON(200, largeData)
	})

	req := httptest.NewRequest("GET", "/data", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		e.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Fiber_LargeJSON(b *testing.B) {
	largeData := generateLargeData()
	app := fiber.New()
	app.Get("/data", func(c *fiber.Ctx) error {
		return c.JSON(largeData)
	})

	req := httptest.NewRequest("GET", "/data", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

// ============================================================================
// Scenario 5: Query Parameters (10 parameters)
// ============================================================================

const queryString = "?q=golang&limit=10&offset=0&sort=date&order=desc&category=books&price_range=10-50&author=john&year=2024&format=pdf"

func BenchmarkFull_Bolt_QueryParams(b *testing.B) {
	app := core.New()
	app.Get("/search", func(c *core.Context) error {
		return c.JSON(200, SearchResponse{
			Query:   c.Query("q"),
			Limit:   10,
			Offset:  0,
			Sort:    c.Query("sort"),
			Filters: []string{c.Query("category"), c.Query("price_range")},
			Results: 42,
		})
	})

	req := httptest.NewRequest("GET", "/search"+queryString, nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Gin_QueryParams(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/search", func(c *gin.Context) {
		c.JSON(200, SearchResponse{
			Query:   c.Query("q"),
			Limit:   10,
			Offset:  0,
			Sort:    c.Query("sort"),
			Filters: []string{c.Query("category"), c.Query("price_range")},
			Results: 42,
		})
	})

	req := httptest.NewRequest("GET", "/search"+queryString, nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Echo_QueryParams(b *testing.B) {
	e := echo.New()
	e.GET("/search", func(c echo.Context) error {
		return c.JSON(200, SearchResponse{
			Query:   c.QueryParam("q"),
			Limit:   10,
			Offset:  0,
			Sort:    c.QueryParam("sort"),
			Filters: []string{c.QueryParam("category"), c.QueryParam("price_range")},
			Results: 42,
		})
	})

	req := httptest.NewRequest("GET", "/search"+queryString, nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		e.ServeHTTP(w, req)
	}
}

func BenchmarkFull_Fiber_QueryParams(b *testing.B) {
	app := fiber.New()
	app.Get("/search", func(c *fiber.Ctx) error {
		return c.JSON(SearchResponse{
			Query:   c.Query("q"),
			Limit:   10,
			Offset:  0,
			Sort:    c.Query("sort"),
			Filters: []string{c.Query("category"), c.Query("price_range")},
			Results: 42,
		})
	})

	req := httptest.NewRequest("GET", "/search"+queryString, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

// ============================================================================
// Scenario 6: Concurrent Throughput
// ============================================================================

func BenchmarkFull_Bolt_Concurrent(b *testing.B) {
	app := core.New()
	app.Get("/ping", func(c *core.Context) error {
		return c.JSON(200, SimpleResponse{Message: "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		w := httptest.NewRecorder()
		for pb.Next() {
			w.Body.Reset()
			app.ServeHTTP(w, req)
		}
	})
}

func BenchmarkFull_Gin_Concurrent(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, SimpleResponse{Message: "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		w := httptest.NewRecorder()
		for pb.Next() {
			w.Body.Reset()
			r.ServeHTTP(w, req)
		}
	})
}

func BenchmarkFull_Echo_Concurrent(b *testing.B) {
	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.JSON(200, SimpleResponse{Message: "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		w := httptest.NewRecorder()
		for pb.Next() {
			w.Body.Reset()
			e.ServeHTTP(w, req)
		}
	})
}

func BenchmarkFull_Fiber_Concurrent(b *testing.B) {
	app := fiber.New()
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(SimpleResponse{Message: "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = app.Test(req, -1)
		}
	})
}
