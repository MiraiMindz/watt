package qpack

import (
	"bytes"
	"testing"
)

func TestDecoderInteger(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		prefix uint8
		want   uint64
	}{
		{"1-byte, 5-bit prefix", []byte{0x0A}, 5, 10},
		{"1-byte, max value", []byte{0x1E}, 5, 30},
		{"2-byte", []byte{0x1F, 0x00}, 5, 31},
		{"2-byte with value", []byte{0x1F, 0x09}, 5, 40},
		{"3-byte", []byte{0x1F, 0x80, 0x01}, 5, 159},
		{"6-bit prefix", []byte{0x3F, 0x00}, 6, 63},
		{"8-bit prefix", []byte{0xFF, 0x00}, 8, 255},
	}

	decoder := NewDecoder(4096)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r qpackReader
			r.Reset(tt.data)
			got, err := decoder.decodeInteger(&r, tt.prefix)
			if err != nil {
				t.Fatalf("decodeInteger() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("decodeInteger() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDecoderString(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"empty string", []byte{0x00}, ""},
		{"simple string", []byte{0x05, 'h', 'e', 'l', 'l', 'o'}, "hello"},
		{"with special chars", []byte{0x0B, '/', 'i', 'n', 'd', 'e', 'x', '.', 'h', 't', 'm', 'l'}, "/index.html"},
	}

	decoder := NewDecoder(4096)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r qpackReader
			r.Reset(tt.data)
			got, err := decoder.decodeString(&r)
			if err != nil {
				t.Fatalf("decodeString() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("decodeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeIndexedFieldLineStatic(t *testing.T) {
	decoder := NewDecoder(4096)

	// Test static table lookup
	// Indexed Field Line format: 1T (T=0 for static, T=1 for dynamic)
	// So pattern is 10XXXXXX for static indexed field
	// Index 0 = ":authority" = ""
	// Index 17 = ":method" = "GET"
	// Index 25 = ":status" = "200"

	tests := []struct {
		name      string
		data      []byte
		wantName  string
		wantValue string
	}{
		{"authority", []byte{0x80}, ":authority", ""},      // 10000000 = static index 0
		{"method GET", []byte{0x91}, ":method", "GET"},     // 10010001 = static index 17
		{"status 200", []byte{0x99}, ":status", "200"},     // 10011001 = static index 25
		{"path /", []byte{0x81}, ":path", "/"},             // 10000001 = static index 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r qpackReader
			r.Reset(tt.data)
			header, err := decoder.decodeIndexedFieldLine(&r)
			if err != nil {
				t.Fatalf("decodeIndexedFieldLine() error = %v", err)
			}
			if header.Name != tt.wantName || header.Value != tt.wantValue {
				t.Errorf("decodeIndexedFieldLine() = {%q, %q}, want {%q, %q}",
					header.Name, header.Value, tt.wantName, tt.wantValue)
			}
		})
	}
}

func TestDecodeLiteralWithNameRef(t *testing.T) {
	decoder := NewDecoder(4096)

	// Literal with name reference to static table
	// 01NTXXXX format, N=0 (can index), T=0 (static), XXXX=name index
	// Example: :path with value "/test"
	// Index 1 in static table = ":path"
	// 01000001 (0x41) = literal, not never-index, static, index=1
	// Then string length and value

	data := []byte{
		0x41,                               // Literal, static, index=1 (:path)
		0x05, '/', 't', 'e', 's', 't',     // Value = "/test"
	}

	var r qpackReader
	r.Reset(data)
	header, err := decoder.decodeLiteralFieldLineWithNameRef(&r)
	if err != nil {
		t.Fatalf("decodeLiteralFieldLineWithNameRef() error = %v", err)
	}

	if header.Name != ":path" {
		t.Errorf("Name = %q, want %q", header.Name, ":path")
	}
	if header.Value != "/test" {
		t.Errorf("Value = %q, want %q", header.Value, "/test")
	}
}

func TestDecodeLiteralWithoutNameRef(t *testing.T) {
	decoder := NewDecoder(4096)

	// Literal without name reference
	// 001NXXXX format, N=0 (can index)
	// 00100000 (0x20) = literal without name ref, not never-index
	// Then name length, name, value length, value

	data := []byte{
		0x20,                                           // Literal without name ref
		0x0A, 'c', 'u', 's', 't', 'o', 'm', '-', 'k', 'e', 'y', // Name = "custom-key"
		0x0C, 'c', 'u', 's', 't', 'o', 'm', '-', 'v', 'a', 'l', 'u', 'e', // Value = "custom-value"
	}

	var r qpackReader
	r.Reset(data)
	header, err := decoder.decodeLiteralFieldLineWithoutNameRef(&r)
	if err != nil {
		t.Fatalf("decodeLiteralFieldLineWithoutNameRef() error = %v", err)
	}

	if header.Name != "custom-key" {
		t.Errorf("Name = %q, want %q", header.Name, "custom-key")
	}
	if header.Value != "custom-value" {
		t.Errorf("Value = %q, want %q", header.Value, "custom-value")
	}
}

func TestDecodeHeaders(t *testing.T) {
	decoder := NewDecoder(4096)

	// Encode a simple header block with only static table references
	// Prefix: RequiredInsertCount=0, DeltaBase=0
	// Headers: :method GET, :path /, :status 200

	var buf bytes.Buffer

	// Encoded Field Section Prefix
	buf.WriteByte(0x00) // RequiredInsertCount = 0 (8-bit prefix)
	buf.WriteByte(0x00) // DeltaBase = 0 (7-bit prefix with sign bit)

	// :method GET (static index 17)
	buf.WriteByte(0x91) // 10010001 = indexed, static, index=17

	// :path / (static index 1)
	buf.WriteByte(0x81) // 10000001 = indexed, static, index=1

	// :status 200 (static index 25)
	buf.WriteByte(0x99) // 10011001 = indexed, static, index=25

	headers, err := decoder.DecodeHeaders(buf.Bytes())
	if err != nil {
		t.Fatalf("DecodeHeaders() error = %v", err)
	}

	if len(headers) != 3 {
		t.Fatalf("DecodeHeaders() returned %d headers, want 3", len(headers))
	}

	expected := []Header{
		{":method", "GET"},
		{":path", "/"},
		{":status", "200"},
	}

	for i, want := range expected {
		if headers[i].Name != want.Name || headers[i].Value != want.Value {
			t.Errorf("Header[%d] = {%q, %q}, want {%q, %q}",
				i, headers[i].Name, headers[i].Value, want.Name, want.Value)
		}
	}
}

func TestDecodeHeadersWithLiterals(t *testing.T) {
	decoder := NewDecoder(4096)

	var buf bytes.Buffer

	// Encoded Field Section Prefix
	buf.WriteByte(0x00) // RequiredInsertCount = 0
	buf.WriteByte(0x00) // DeltaBase = 0

	// :method GET (static index 17)
	buf.WriteByte(0x91) // 10010001 = indexed, static, index=17

	// :path /custom (literal with name ref to static index 1)
	buf.WriteByte(0x41) // Literal, static, index=1
	buf.WriteByte(0x07) // Length = 7
	buf.WriteString("/custom")

	// custom-header: custom-value (literal without name ref)
	buf.WriteByte(0x20)                            // Literal without name ref
	buf.WriteByte(0x0D)                            // Name length = 13
	buf.WriteString("custom-header")
	buf.WriteByte(0x0C)                            // Value length = 12
	buf.WriteString("custom-value")

	headers, err := decoder.DecodeHeaders(buf.Bytes())
	if err != nil {
		t.Fatalf("DecodeHeaders() error = %v", err)
	}

	if len(headers) != 3 {
		t.Fatalf("DecodeHeaders() returned %d headers, want 3", len(headers))
	}

	expected := []Header{
		{":method", "GET"},
		{":path", "/custom"},
		{"custom-header", "custom-value"},
	}

	for i, want := range expected {
		if headers[i].Name != want.Name || headers[i].Value != want.Value {
			t.Errorf("Header[%d] = {%q, %q}, want {%q, %q}",
				i, headers[i].Name, headers[i].Value, want.Name, want.Value)
		}
	}
}

func TestEncoderDecoderRoundTrip(t *testing.T) {
	encoder := NewEncoder(4096)
	decoder := NewDecoder(4096)

	headers := []Header{
		{":method", "GET"},
		{":scheme", "https"},
		{":authority", "example.com"},
		{":path", "/index.html"},
		{"content-type", "text/html"},
		{"cache-control", "no-cache"},
	}

	// Encode headers
	encoded, _, err := encoder.EncodeHeaders(headers)
	if err != nil {
		t.Fatalf("EncodeHeaders() error = %v", err)
	}

	// Decode headers
	decoded, err := decoder.DecodeHeaders(encoded)
	if err != nil {
		t.Fatalf("DecodeHeaders() error = %v", err)
	}

	if len(decoded) != len(headers) {
		t.Fatalf("DecodeHeaders() returned %d headers, want %d", len(decoded), len(headers))
	}

	for i, want := range headers {
		if decoded[i].Name != want.Name || decoded[i].Value != want.Value {
			t.Errorf("Header[%d] = {%q, %q}, want {%q, %q}",
				i, decoded[i].Name, decoded[i].Value, want.Name, want.Value)
		}
	}
}

func TestEncoderDecoderRoundTripLargeHeaders(t *testing.T) {
	encoder := NewEncoder(16384)
	decoder := NewDecoder(16384)

	headers := []Header{
		{":method", "POST"},
		{":scheme", "https"},
		{":authority", "api.example.com"},
		{":path", "/v1/users/12345/profile"},
		{"content-type", "application/json"},
		{"content-length", "1234"},
		{"authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"},
		{"accept", "application/json, text/plain, */*"},
		{"accept-encoding", "gzip, deflate, br"},
		{"accept-language", "en-US,en;q=0.9"},
		{"x-request-id", "550e8400-e29b-41d4-a716-446655440000"},
		{"x-custom-header-1", "value1"},
		{"x-custom-header-2", "value2"},
		{"x-custom-header-3", "value3"},
	}

	// Encode headers
	encoded, _, err := encoder.EncodeHeaders(headers)
	if err != nil {
		t.Fatalf("EncodeHeaders() error = %v", err)
	}

	t.Logf("Encoded %d headers into %d bytes", len(headers), len(encoded))

	// Decode headers
	decoded, err := decoder.DecodeHeaders(encoded)
	if err != nil {
		t.Fatalf("DecodeHeaders() error = %v", err)
	}

	if len(decoded) != len(headers) {
		t.Fatalf("DecodeHeaders() returned %d headers, want %d", len(decoded), len(headers))
	}

	for i, want := range headers {
		if decoded[i].Name != want.Name || decoded[i].Value != want.Value {
			t.Errorf("Header[%d] = {%q, %q}, want {%q, %q}",
				i, decoded[i].Name, decoded[i].Value, want.Name, want.Value)
		}
	}
}

func TestDecodeInvalidInteger(t *testing.T) {
	decoder := NewDecoder(4096)

	// Test integer overflow
	data := make([]byte, 100)
	data[0] = 0x1F // Start multi-byte
	for i := 1; i < 100; i++ {
		data[i] = 0x80 // All continuation bits set
	}

	var r qpackReader
	r.Reset(data)
	_, err := decoder.decodeInteger(&r, 5)
	if err != ErrIntegerOverflow {
		t.Errorf("decodeInteger() error = %v, want ErrIntegerOverflow", err)
	}
}

func TestDecodeInvalidPrefix(t *testing.T) {
	decoder := NewDecoder(4096)

	var r qpackReader
	r.Reset([]byte{0x00})
	_, err := decoder.decodeInteger(&r, 0)
	if err == nil {
		t.Error("decodeInteger() with prefix=0 should return error")
	}

	r.Reset([]byte{0x00})
	_, err = decoder.decodeInteger(&r, 9)
	if err == nil {
		t.Error("decodeInteger() with prefix=9 should return error")
	}
}

func TestDecodeStringTooLong(t *testing.T) {
	decoder := NewDecoder(4096)

	// Create string with length > 1MB
	data := []byte{0x7F, 0xFF, 0xFF, 0x7F} // Large length
	var r qpackReader
	r.Reset(data)
	_, err := decoder.decodeString(&r)
	if err != ErrStringTooLong {
		t.Errorf("decodeString() error = %v, want ErrStringTooLong", err)
	}
}

func BenchmarkDecodeIndexedFieldLine(b *testing.B) {
	decoder := NewDecoder(4096)
	data := []byte{0x91} // :method GET (10010001 = static index 17)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var r qpackReader
		r.Reset(data)
		_, err := decoder.decodeIndexedFieldLine(&r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeHeaders(b *testing.B) {
	decoder := NewDecoder(4096)

	var buf bytes.Buffer
	buf.WriteByte(0x00) // RequiredInsertCount
	buf.WriteByte(0x00) // DeltaBase
	buf.WriteByte(0x91) // :method GET (10010001 = static index 17)
	buf.WriteByte(0x81) // :path / (10000001 = static index 1)
	buf.WriteByte(0x99) // :status 200 (10011001 = static index 25)

	data := buf.Bytes()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := decoder.DecodeHeaders(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncoderDecoderRoundTrip(b *testing.B) {
	encoder := NewEncoder(4096)
	decoder := NewDecoder(4096)

	headers := []Header{
		{":method", "GET"},
		{":scheme", "https"},
		{":authority", "example.com"},
		{":path", "/"},
		{"content-type", "application/json"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		encoded, _, err := encoder.EncodeHeaders(headers)
		if err != nil {
			b.Fatal(err)
		}

		_, err = decoder.DecodeHeaders(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}
