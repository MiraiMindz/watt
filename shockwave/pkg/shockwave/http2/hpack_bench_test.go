package http2

import (
	"bytes"
	"testing"
)

// Benchmark Huffman encoding
func BenchmarkHuffmanEncode(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{"short", "GET"},
		{"medium", "www.example.com"},
		{"long", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tt.input)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = HuffmanEncode(tt.input)
			}
		})
	}
}

// Benchmark Huffman decoding
func BenchmarkHuffmanDecode(b *testing.B) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"short", HuffmanEncode("GET")},
		{"medium", HuffmanEncode("www.example.com")},
		{"long", HuffmanEncode("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tt.input)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = HuffmanDecode(tt.input)
			}
		})
	}
}

// Benchmark static table lookups
func BenchmarkStaticTableLookup(b *testing.B) {
	tests := []struct {
		name  string
		value string
	}{
		{":method", "GET"},
		{":status", "200"},
		{"content-type", "application/json"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = FindStaticIndex(tt.name, tt.value)
			}
		})
	}
}

// Benchmark dynamic table operations
func BenchmarkDynamicTableAdd(b *testing.B) {
	dt := newDynamicTable(4096)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dt.Add("custom-header", "custom-value")
	}
}

func BenchmarkDynamicTableGet(b *testing.B) {
	dt := newDynamicTable(4096)
	for i := 0; i < 10; i++ {
		dt.Add("header", "value")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = dt.Get(1)
	}
}

func BenchmarkDynamicTableFind(b *testing.B) {
	dt := newDynamicTable(4096)
	for i := 0; i < 10; i++ {
		dt.Add("header", "value")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = dt.Find("header", "value")
	}
}

// Benchmark integer encoding
func BenchmarkIntegerEncode(b *testing.B) {
	tests := []struct {
		name  string
		value int
	}{
		{"small", 10},
		{"medium", 127},
		{"large", 1337},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			enc := &Encoder{}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				enc.buf.Reset()
				enc.encodeInteger(tt.value, 7, 0)
			}
		})
	}
}

// Benchmark integer decoding
func BenchmarkIntegerDecode(b *testing.B) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"small", []byte{10}},
		{"medium", []byte{127, 0}},
		{"large", []byte{127, 154, 10}},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			dec := &Decoder{}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := bytes.NewReader(tt.input)
				_, _ = dec.decodeInteger(buf, 7)
			}
		})
	}
}

// Benchmark header encoding
func BenchmarkEncode(b *testing.B) {
	tests := []struct {
		name    string
		headers []HeaderField
	}{
		{
			name: "small",
			headers: []HeaderField{
				{":method", "GET"},
				{":path", "/"},
			},
		},
		{
			name: "medium",
			headers: []HeaderField{
				{":method", "GET"},
				{":path", "/index.html"},
				{":scheme", "https"},
				{":authority", "www.example.com"},
				{"accept", "text/html"},
			},
		},
		{
			name: "large",
			headers: []HeaderField{
				{":method", "GET"},
				{":path", "/api/users/123/profile"},
				{":scheme", "https"},
				{":authority", "api.example.com"},
				{"user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
				{"accept", "application/json,text/html,*/*;q=0.8"},
				{"accept-language", "en-US,en;q=0.9"},
				{"accept-encoding", "gzip, deflate, br"},
				{"cookie", "session=abc123; user=john; theme=dark"},
				{"authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			enc := NewEncoder(4096)

			// Calculate uncompressed size
			size := 0
			for _, h := range tt.headers {
				size += len(h.Name) + len(h.Value)
			}
			b.SetBytes(int64(size))

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = enc.Encode(tt.headers)
			}
		})
	}
}

// Benchmark header decoding
func BenchmarkDecode(b *testing.B) {
	tests := []struct {
		name    string
		headers []HeaderField
	}{
		{
			name: "small",
			headers: []HeaderField{
				{":method", "GET"},
				{":path", "/"},
			},
		},
		{
			name: "medium",
			headers: []HeaderField{
				{":method", "GET"},
				{":path", "/index.html"},
				{":scheme", "https"},
				{":authority", "www.example.com"},
				{"accept", "text/html"},
			},
		},
		{
			name: "large",
			headers: []HeaderField{
				{":method", "GET"},
				{":path", "/api/users/123/profile"},
				{":scheme", "https"},
				{":authority", "api.example.com"},
				{"user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
				{"accept", "application/json,text/html,*/*;q=0.8"},
				{"accept-language", "en-US,en;q=0.9"},
				{"accept-encoding", "gzip, deflate, br"},
				{"cookie", "session=abc123; user=john; theme=dark"},
				{"authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			enc := NewEncoder(4096)
			encoded := enc.Encode(tt.headers)

			dec := NewDecoder(4096, 16*1024)

			// Calculate compressed size
			b.SetBytes(int64(len(encoded)))

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = dec.Decode(encoded)
			}
		})
	}
}

// Benchmark round-trip (encode + decode)
func BenchmarkRoundTrip(b *testing.B) {
	headers := []HeaderField{
		{":method", "GET"},
		{":path", "/index.html"},
		{":scheme", "https"},
		{":authority", "www.example.com"},
		{"user-agent", "Mozilla/5.0"},
		{"accept", "text/html"},
	}

	enc := NewEncoder(4096)
	dec := NewDecoder(4096, 16*1024)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		encoded := enc.Encode(headers)
		_, _ = dec.Decode(encoded)
	}
}

// Benchmark with/without Huffman encoding
func BenchmarkEncodeHuffman(b *testing.B) {
	headers := []HeaderField{
		{":method", "GET"},
		{":path", "/index.html"},
		{":scheme", "https"},
		{":authority", "www.example.com"},
	}

	b.Run("with_huffman", func(b *testing.B) {
		enc := NewEncoder(4096)
		enc.SetUseHuffman(true)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = enc.Encode(headers)
		}
	})

	b.Run("without_huffman", func(b *testing.B) {
		enc := NewEncoder(4096)
		enc.SetUseHuffman(false)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = enc.Encode(headers)
		}
	})
}

// Benchmark dynamic table updates
func BenchmarkEncoderWithDynamicTable(b *testing.B) {
	headers := []HeaderField{
		{"custom-header-1", "custom-value-1"},
		{"custom-header-2", "custom-value-2"},
		{"custom-header-3", "custom-value-3"},
	}

	enc := NewEncoder(4096)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = enc.Encode(headers)
	}
}

// Benchmark sequential requests (simulating HTTP/2 connection)
func BenchmarkSequentialRequests(b *testing.B) {
	requests := [][]HeaderField{
		{
			{":method", "GET"},
			{":path", "/"},
			{":scheme", "https"},
			{":authority", "www.example.com"},
		},
		{
			{":method", "GET"},
			{":path", "/style.css"},
			{":scheme", "https"},
			{":authority", "www.example.com"},
		},
		{
			{":method", "GET"},
			{":path", "/script.js"},
			{":scheme", "https"},
			{":authority", "www.example.com"},
		},
	}

	enc := NewEncoder(4096)
	dec := NewDecoder(4096, 16*1024)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, headers := range requests {
			encoded := enc.Encode(headers)
			_, _ = dec.Decode(encoded)
		}
	}
}
