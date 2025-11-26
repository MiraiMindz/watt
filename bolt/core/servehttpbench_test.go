package core

import "testing"

// BenchmarkRouter_ServeHTTP_DynamicRoute benchmarks the ACTUAL hot path
func BenchmarkRouter_ServeHTTP_DynamicRoute(b *testing.B) {
	router := NewRouter()
	handler := func(c *Context) error {
		_ = c.Param("id")
		return nil
	}
	router.Add(MethodGet, "/users/:id", handler)

	ctx := &Context{}
	ctx.methodBytes = []byte("GET")
	ctx.pathBytes = []byte("/users/123")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(ctx)
		ctx.paramsLen = 0  // Reset params
	}
}
