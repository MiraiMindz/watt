package core

import (
	"io"
	"testing"

	"github.com/yourusername/shockwave/pkg/shockwave/http11"
)

// BenchmarkSetHeader_String benchmarks traditional string-based SetHeader
func BenchmarkSetHeader_String(b *testing.B) {
	ctx := &Context{}
	ctx.shockwaveRes = http11.NewResponseWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SetHeader("Content-Type", "application/json")
	}
}

// BenchmarkSetHeader_Bytes benchmarks zero-copy SetHeaderBytes
func BenchmarkSetHeader_Bytes(b *testing.B) {
	ctx := &Context{}
	ctx.shockwaveRes = http11.NewResponseWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SetHeaderBytes(headerContentType, contentTypeJSON)
	}
}

// BenchmarkSetHeader_PreCompiledHelper benchmarks pre-compiled helper method
func BenchmarkSetHeader_PreCompiledHelper(b *testing.B) {
	ctx := &Context{}
	ctx.shockwaveRes = http11.NewResponseWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.setContentTypeJSON()
	}
}

// BenchmarkJSON_WithPreCompiledHeaders benchmarks JSON response with pre-compiled headers
func BenchmarkJSON_WithPreCompiledHeaders(b *testing.B) {
	ctx := &Context{}
	ctx.shockwaveRes = http11.NewResponseWriter(io.Discard)

	data := map[string]string{"status": "ok", "message": "success"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ctx.JSON(200, data)
	}
}

// BenchmarkText_WithPreCompiledHeaders benchmarks Text response with pre-compiled headers
func BenchmarkText_WithPreCompiledHeaders(b *testing.B) {
	ctx := &Context{}
	ctx.shockwaveRes = http11.NewResponseWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ctx.Text(200, "Hello, World!")
	}
}

// BenchmarkHTML_WithPreCompiledHeaders benchmarks HTML response with pre-compiled headers
func BenchmarkHTML_WithPreCompiledHeaders(b *testing.B) {
	ctx := &Context{}
	ctx.shockwaveRes = http11.NewResponseWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ctx.HTML(200, "<h1>Hello, World!</h1>")
	}
}

// BenchmarkMultipleHeaders_String benchmarks multiple string headers
func BenchmarkMultipleHeaders_String(b *testing.B) {
	ctx := &Context{}
	ctx.shockwaveRes = http11.NewResponseWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SetHeader("Content-Type", "application/json")
		ctx.SetHeader("Server", "Bolt")
		ctx.SetHeader("Cache-Control", "no-cache")
	}
}

// BenchmarkMultipleHeaders_Bytes benchmarks multiple pre-compiled headers
func BenchmarkMultipleHeaders_Bytes(b *testing.B) {
	ctx := &Context{}
	ctx.shockwaveRes = http11.NewResponseWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.setContentTypeJSON()
		ctx.SetServerHeader()
		ctx.SetNoCacheHeaders()
	}
}
