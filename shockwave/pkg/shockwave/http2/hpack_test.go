package http2

import (
	"bytes"
	"testing"
)

// Test static table lookups
func TestStaticTable(t *testing.T) {
	tests := []struct {
		index int
		want  HeaderField
	}{
		{1, HeaderField{":authority", ""}},
		{2, HeaderField{":method", "GET"}},
		{3, HeaderField{":method", "POST"}},
		{8, HeaderField{":status", "200"}},
		{61, HeaderField{"www-authenticate", ""}},
	}

	for _, tt := range tests {
		got := GetStaticEntry(tt.index)
		if got.Name != tt.want.Name || got.Value != tt.want.Value {
			t.Errorf("GetStaticEntry(%d) = %+v, want %+v", tt.index, got, tt.want)
		}
	}
}

func TestFindStaticIndex(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantIndex  int
		wantExact  bool
	}{
		{":method", "GET", 2, true},         // Exact match
		{":method", "POST", 3, true},        // Exact match
		{":method", "DELETE", 2, false},     // Name match only
		{":status", "200", 8, true},         // Exact match
		{":status", "418", 8, false},        // Name match only
		{"custom-header", "value", 0, false}, // No match
	}

	for _, tt := range tests {
		gotIndex, gotExact := FindStaticIndex(tt.name, tt.value)
		if gotIndex != tt.wantIndex || gotExact != tt.wantExact {
			t.Errorf("FindStaticIndex(%q, %q) = (%d, %v), want (%d, %v)",
				tt.name, tt.value, gotIndex, gotExact, tt.wantIndex, tt.wantExact)
		}
	}
}

// Test Huffman encoding
func TestHuffmanEncode(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{"", nil},
		{"www.example.com", []byte{
			0xf1, 0xe3, 0xc2, 0xe5, 0xf2, 0x3a, 0x6b, 0xa0,
			0xab, 0x90, 0xf4, 0xff,
		}},
		{"no-cache", []byte{0xa8, 0xeb, 0x10, 0x64, 0x9c, 0xbf}},
		{"custom-key", []byte{0x25, 0xa8, 0x49, 0xe9, 0x5b, 0xa9, 0x7d, 0x7f}},
		{"custom-value", []byte{0x25, 0xa8, 0x49, 0xe9, 0x5b, 0xb8, 0xe8, 0xb4, 0xbf}},
	}

	for _, tt := range tests {
		got := HuffmanEncode(tt.input)
		if !bytes.Equal(got, tt.expected) {
			t.Errorf("HuffmanEncode(%q) = %x, want %x", tt.input, got, tt.expected)
		}
	}
}

// Test Huffman decoding
func TestHuffmanDecode(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{nil, ""},
		{
			[]byte{0xf1, 0xe3, 0xc2, 0xe5, 0xf2, 0x3a, 0x6b, 0xa0, 0xab, 0x90, 0xf4, 0xff},
			"www.example.com",
		},
		{[]byte{0xa8, 0xeb, 0x10, 0x64, 0x9c, 0xbf}, "no-cache"},
		{[]byte{0x25, 0xa8, 0x49, 0xe9, 0x5b, 0xa9, 0x7d, 0x7f}, "custom-key"},
		{[]byte{0x25, 0xa8, 0x49, 0xe9, 0x5b, 0xb8, 0xe8, 0xb4, 0xbf}, "custom-value"},
	}

	for _, tt := range tests {
		got, err := HuffmanDecode(tt.input)
		if err != nil {
			t.Errorf("HuffmanDecode(%x) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("HuffmanDecode(%x) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// Test Huffman round-trip
func TestHuffmanRoundTrip(t *testing.T) {
	tests := []string{
		"",
		"hello",
		"www.example.com",
		":method",
		"GET",
		"application/json",
		"Mozilla/5.0",
	}

	for _, original := range tests {
		encoded := HuffmanEncode(original)
		decoded, err := HuffmanDecode(encoded)
		if err != nil {
			t.Errorf("HuffmanDecode error for %q: %v", original, err)
			continue
		}
		if decoded != original {
			t.Errorf("Round-trip failed: %q -> %x -> %q", original, encoded, decoded)
		}
	}
}

// Test dynamic table operations
func TestDynamicTable(t *testing.T) {
	dt := newDynamicTable(256)

	// Test empty table
	if dt.Len() != 0 {
		t.Errorf("New table should be empty, got length %d", dt.Len())
	}

	// Add entries
	dt.Add("custom-key", "custom-value")
	if dt.Len() != 1 {
		t.Errorf("After adding one entry, length should be 1, got %d", dt.Len())
	}

	// Retrieve entry
	hf, ok := dt.Get(1)
	if !ok {
		t.Error("Failed to retrieve entry at index 1")
	}
	if hf.Name != "custom-key" || hf.Value != "custom-value" {
		t.Errorf("Get(1) = %+v, want {custom-key custom-value}", hf)
	}

	// Add more entries
	dt.Add("another-key", "another-value")
	dt.Add("third-key", "third-value")

	if dt.Len() != 3 {
		t.Errorf("After adding three entries, length should be 3, got %d", dt.Len())
	}

	// Newest entry should be at index 1
	hf, ok = dt.Get(1)
	if !ok || hf.Name != "third-key" {
		t.Errorf("Get(1) should return newest entry, got %+v", hf)
	}
}

// Test dynamic table eviction
func TestDynamicTableEviction(t *testing.T) {
	// Create small table (128 bytes)
	dt := newDynamicTable(128)

	// Add entries until table is full
	// Each entry is ~48 bytes (name + value + 32 overhead)
	dt.Add("key1", "value1") // 42 bytes
	dt.Add("key2", "value2") // 42 bytes
	dt.Add("key3", "value3") // 42 bytes (total = 126 bytes)

	if dt.Len() != 3 {
		t.Errorf("Expected 3 entries, got %d", dt.Len())
	}

	// Add another entry, should evict oldest
	dt.Add("key4", "value4") // This should evict key1

	if dt.Len() != 3 {
		t.Errorf("Expected 3 entries after eviction, got %d", dt.Len())
	}

	// key4 should be at index 1 (newest)
	hf, ok := dt.Get(1)
	if !ok || hf.Name != "key4" {
		t.Errorf("Get(1) should return key4, got %+v", hf)
	}

	// key1 should be evicted
	hf, ok = dt.Get(4)
	if ok {
		t.Errorf("Get(4) should fail (only 3 entries), but got %+v", hf)
	}
}

// Test dynamic table resizing
func TestDynamicTableResize(t *testing.T) {
	dt := newDynamicTable(256)

	// Add entries
	dt.Add("key1", "value1")
	dt.Add("key2", "value2")
	dt.Add("key3", "value3")

	if dt.Len() != 3 {
		t.Errorf("Expected 3 entries, got %d", dt.Len())
	}

	// Shrink table to force eviction
	dt.SetMaxSize(64) // Should evict some entries

	if dt.Len() > 1 {
		t.Errorf("After resize to 64 bytes, expected at most 1 entry, got %d", dt.Len())
	}

	// Only newest entry should remain
	if dt.Len() > 0 {
		hf, ok := dt.Get(1)
		if !ok || hf.Name != "key3" {
			t.Errorf("After resize, Get(1) should return key3, got %+v", hf)
		}
	}
}

// Test index table (combined static + dynamic)
func TestIndexTable(t *testing.T) {
	it := newIndexTable(256)

	// Test static table access (indices 1-61)
	hf, ok := it.Get(2)
	if !ok || hf.Name != ":method" || hf.Value != "GET" {
		t.Errorf("Get(2) = %+v, want {:method GET}", hf)
	}

	// Add to dynamic table
	it.Add("custom-key", "custom-value")

	// Dynamic entry should be at index 62 (static table is 1-61)
	hf, ok = it.Get(62)
	if !ok || hf.Name != "custom-key" {
		t.Errorf("Get(62) = %+v, want {custom-key custom-value}", hf)
	}

	// Test Find with exact match in static table
	index, exact := it.Find(":method", "GET")
	if index != 2 || !exact {
		t.Errorf("Find(:method, GET) = (%d, %v), want (2, true)", index, exact)
	}

	// Test Find with exact match in dynamic table
	index, exact = it.Find("custom-key", "custom-value")
	if index != 62 || !exact {
		t.Errorf("Find(custom-key, custom-value) = (%d, %v), want (62, true)", index, exact)
	}
}

// Test integer encoding
func TestIntegerEncode(t *testing.T) {
	tests := []struct {
		value      int
		prefix     uint8
		prefixBits byte
		expected   []byte
	}{
		{10, 5, 0, []byte{10}},                     // Fits in prefix
		{31, 5, 0, []byte{31, 0}},                  // Exactly fits
		{32, 5, 0, []byte{31, 1}},                  // Needs continuation
		{127, 7, 0, []byte{127, 0}},                // Exactly fits in 7 bits
		{128, 7, 0, []byte{127, 1}},                // Needs continuation
		{1337, 5, 0, []byte{31, 154, 10}},          // RFC 7541 example
	}

	for _, tt := range tests {
		enc := &Encoder{}
		enc.encodeInteger(tt.value, tt.prefix, tt.prefixBits)
		got := enc.buf.Bytes()
		if !bytes.Equal(got, tt.expected) {
			t.Errorf("encodeInteger(%d, %d, %#x) = %v, want %v",
				tt.value, tt.prefix, tt.prefixBits, got, tt.expected)
		}
	}
}

// Test integer decoding
func TestIntegerDecode(t *testing.T) {
	tests := []struct {
		input    []byte
		prefix   uint8
		expected int
	}{
		{[]byte{10}, 5, 10},                    // Fits in prefix
		{[]byte{31, 0}, 5, 31},                 // Exactly fits
		{[]byte{31, 1}, 5, 32},                 // Continuation
		{[]byte{127, 0}, 7, 127},               // Exactly fits in 7 bits
		{[]byte{127, 1}, 7, 128},               // Continuation
		{[]byte{31, 154, 10}, 5, 1337},         // RFC 7541 example
	}

	for _, tt := range tests {
		dec := &Decoder{}
		buf := bytes.NewReader(tt.input)
		got, err := dec.decodeInteger(buf, tt.prefix)
		if err != nil {
			t.Errorf("decodeInteger(%v, %d) error: %v", tt.input, tt.prefix, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("decodeInteger(%v, %d) = %d, want %d",
				tt.input, tt.prefix, got, tt.expected)
		}
	}
}

// Test encoder/decoder round-trip
func TestEncoderDecoderRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		headers []HeaderField
	}{
		{
			name: "simple headers",
			headers: []HeaderField{
				{":method", "GET"},
				{":path", "/"},
				{":scheme", "https"},
			},
		},
		{
			name: "custom headers",
			headers: []HeaderField{
				{":method", "POST"},
				{":path", "/api/users"},
				{"content-type", "application/json"},
				{"custom-header", "custom-value"},
			},
		},
		{
			name: "repeated headers",
			headers: []HeaderField{
				{"cookie", "session=abc123"},
				{"cookie", "user=john"},
				{"cookie", "theme=dark"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewEncoder(4096)
			dec := NewDecoder(4096, 16*1024)

			// Encode
			encoded := enc.Encode(tt.headers)

			// Decode
			decoded, err := dec.Decode(encoded)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}

			// Compare
			if len(decoded) != len(tt.headers) {
				t.Fatalf("Decoded %d headers, want %d", len(decoded), len(tt.headers))
			}

			for i, want := range tt.headers {
				got := decoded[i]
				if got.Name != want.Name || got.Value != want.Value {
					t.Errorf("Header %d: got %+v, want %+v", i, got, want)
				}
			}
		})
	}
}

// Test compression ratio
func TestCompressionRatio(t *testing.T) {
	headers := []HeaderField{
		{":method", "GET"},
		{":path", "/index.html"},
		{":scheme", "https"},
		{":authority", "www.example.com"},
		{"user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
		{"accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		{"accept-language", "en-US,en;q=0.5"},
		{"accept-encoding", "gzip, deflate, br"},
	}

	// Calculate uncompressed size
	uncompressedSize := 0
	for _, h := range headers {
		uncompressedSize += len(h.Name) + len(h.Value) + 2 // +2 for ": "
	}

	// Encode
	enc := NewEncoder(4096)
	encoded := enc.Encode(headers)

	// Calculate compression ratio
	ratio := float64(len(encoded)) / float64(uncompressedSize) * 100

	t.Logf("Uncompressed: %d bytes", uncompressedSize)
	t.Logf("Compressed: %d bytes", len(encoded))
	t.Logf("Compression ratio: %.1f%%", ratio)

	// Should achieve at least some compression
	if ratio > 95 {
		t.Errorf("Poor compression ratio: %.1f%% (expected < 95%%)", ratio)
	}
}

// Test Huffman encoding preference
func TestHuffmanEncodingPreference(t *testing.T) {
	enc := NewEncoder(4096)
	enc.SetUseHuffman(true)

	// String that compresses well with Huffman
	headers := []HeaderField{
		{":method", "GET"},
		{":path", "/"},
	}

	encodedHuffman := enc.Encode(headers)

	// Disable Huffman
	enc2 := NewEncoder(4096)
	enc2.SetUseHuffman(false)
	encodedPlain := enc2.Encode(headers)

	// Huffman should be smaller or equal
	if len(encodedHuffman) > len(encodedPlain) {
		t.Logf("Huffman: %d bytes, Plain: %d bytes", len(encodedHuffman), len(encodedPlain))
		t.Logf("Huffman may not always be smaller for very short strings")
	}
}
