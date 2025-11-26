package benchmarks

import (
	"testing"

	"github.com/yourusername/shockwave/pkg/shockwave/http3/qpack"
)

// BenchmarkShockwaveHeaderEncoding benchmarks Shockwave's header encoding
func BenchmarkShockwaveHeaderEncoding(b *testing.B) {
	encoder := qpack.NewEncoder(4096)

	headers := []qpack.Header{
		{Name: ":method", Value: "GET"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "example.com"},
		{Name: ":path", Value: "/api/v1/users"},
		{Name: "user-agent", Value: "shockwave/1.0"},
		{Name: "accept", Value: "application/json"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := encoder.EncodeHeaders(headers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNghttp3HeaderEncoding benchmarks nghttp3's header encoding
func BenchmarkNghttp3HeaderEncoding(b *testing.B) {
	encoder := NewNgHttp3Encoder()
	defer encoder.Close()

	headers := map[string]string{
		":method":    "GET",
		":scheme":    "https",
		":authority": "example.com",
		":path":      "/api/v1/users",
		"user-agent": "shockwave/1.0",
		"accept":     "application/json",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := encoder.EncodeHeaders(headers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShockwaveHeaderDecodingSmall benchmarks decoding small header blocks
func BenchmarkShockwaveHeaderDecodingSmall(b *testing.B) {
	encoder := qpack.NewEncoder(4096)
	decoder := qpack.NewDecoder(4096)

	headers := []qpack.Header{
		{Name: ":status", Value: "200"},
		{Name: "content-type", Value: "application/json"},
		{Name: "content-length", Value: "1234"},
	}

	encodedHeaders, _, _ := encoder.EncodeHeaders(headers)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := decoder.DecodeHeaders(encodedHeaders)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShockwaveHeaderDecodingLarge benchmarks decoding large header blocks
func BenchmarkShockwaveHeaderDecodingLarge(b *testing.B) {
	encoder := qpack.NewEncoder(16384)
	decoder := qpack.NewDecoder(16384)

	headers := []qpack.Header{
		{Name: ":status", Value: "200"},
		{Name: "content-type", Value: "application/json"},
		{Name: "content-length", Value: "1234"},
	}

	// Add many custom headers
	for i := 0; i < 50; i++ {
		headers = append(headers, qpack.Header{
			Name:  "x-custom-header",
			Value: "value-with-some-length-to-test-compression",
		})
	}

	encodedHeaders, _, _ := encoder.EncodeHeaders(headers)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := decoder.DecodeHeaders(encodedHeaders)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShockwaveStaticTableLookup benchmarks static table lookup
func BenchmarkShockwaveStaticTableLookup(b *testing.B) {
	encoder := qpack.NewEncoder(4096)

	// These headers should all be in the static table
	headers := []qpack.Header{
		{Name: ":method", Value: "GET"},
		{Name: ":method", Value: "POST"},
		{Name: ":path", Value: "/"},
		{Name: ":scheme", Value: "https"},
		{Name: "accept", Value: "*/*"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := encoder.EncodeHeaders(headers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShockwaveDynamicTableInsertion benchmarks dynamic table insertion
func BenchmarkShockwaveDynamicTableInsertion(b *testing.B) {
	encoder := qpack.NewEncoder(4096)

	// Custom headers that will go into dynamic table
	headers := []qpack.Header{
		{Name: "x-custom-header-1", Value: "value1"},
		{Name: "x-custom-header-2", Value: "value2"},
		{Name: "x-custom-header-3", Value: "value3"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := encoder.EncodeHeaders(headers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShockwaveHeaderReuse benchmarks reusing headers from dynamic table
func BenchmarkShockwaveHeaderReuse(b *testing.B) {
	encoder := qpack.NewEncoder(4096)
	decoder := qpack.NewDecoder(4096)

	// First request with custom headers
	headers1 := []qpack.Header{
		{Name: ":method", Value: "GET"},
		{Name: ":path", Value: "/api/v1/users"},
		{Name: "x-api-key", Value: "secret-key-12345"},
		{Name: "x-request-id", Value: "req-12345"},
	}

	// Second request reuses same headers
	headers2 := []qpack.Header{
		{Name: ":method", Value: "GET"},
		{Name: ":path", Value: "/api/v1/posts"},
		{Name: "x-api-key", Value: "secret-key-12345"}, // Should be in dynamic table
		{Name: "x-request-id", Value: "req-67890"},
	}

	// Warm up dynamic table
	encoded1, instructions1, _ := encoder.EncodeHeaders(headers1)
	decoder.ProcessEncoderInstruction(instructions1)
	decoder.DecodeHeaders(encoded1)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		encoded2, instructions2, _ := encoder.EncodeHeaders(headers2)
		decoder.ProcessEncoderInstruction(instructions2)
		_, err := decoder.DecodeHeaders(encoded2)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShockwaveCompressionRatio measures compression effectiveness
func BenchmarkShockwaveCompressionRatio(b *testing.B) {
	encoder := qpack.NewEncoder(4096)

	headers := []qpack.Header{
		{Name: ":method", Value: "POST"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "api.example.com"},
		{Name: ":path", Value: "/api/v1/users/12345/profile"},
		{Name: "content-type", Value: "application/json"},
		{Name: "content-length", Value: "1024"},
		{Name: "user-agent", Value: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"},
		{Name: "accept", Value: "application/json, text/plain, */*"},
		{Name: "accept-encoding", Value: "gzip, deflate, br"},
		{Name: "accept-language", Value: "en-US,en;q=0.9"},
	}

	// Calculate uncompressed size
	uncompressedSize := 0
	for _, h := range headers {
		uncompressedSize += len(h.Name) + len(h.Value)
	}

	b.ResetTimer()
	b.ReportAllocs()

	var compressedSize int
	for i := 0; i < b.N; i++ {
		encoded, _, _ := encoder.EncodeHeaders(headers)
		compressedSize = len(encoded)
	}

	b.StopTimer()
	ratio := float64(uncompressedSize) / float64(compressedSize)
	b.ReportMetric(ratio, "compression_ratio")
	b.ReportMetric(float64(uncompressedSize), "uncompressed_bytes")
	b.ReportMetric(float64(compressedSize), "compressed_bytes")
}
