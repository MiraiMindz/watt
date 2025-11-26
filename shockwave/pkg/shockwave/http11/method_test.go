package http11

import (
	"testing"
)

func TestParseMethodID(t *testing.T) {
	tests := []struct {
		name     string
		method   []byte
		expected uint8
	}{
		// Valid methods
		{"GET", []byte("GET"), MethodGET},
		{"POST", []byte("POST"), MethodPOST},
		{"PUT", []byte("PUT"), MethodPUT},
		{"DELETE", []byte("DELETE"), MethodDELETE},
		{"PATCH", []byte("PATCH"), MethodPATCH},
		{"HEAD", []byte("HEAD"), MethodHEAD},
		{"OPTIONS", []byte("OPTIONS"), MethodOPTIONS},
		{"CONNECT", []byte("CONNECT"), MethodCONNECT},
		{"TRACE", []byte("TRACE"), MethodTRACE},

		// Invalid methods
		{"Invalid", []byte("INVALID"), MethodUnknown},
		{"Lowercase get", []byte("get"), MethodUnknown},
		{"Empty", []byte(""), MethodUnknown},
		{"Partial", []byte("GE"), MethodUnknown},
		{"Too long", []byte("GETPOST"), MethodUnknown},
		{"Numbers", []byte("123"), MethodUnknown},
		{"Mixed case", []byte("GeT"), MethodUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMethodID(tt.method)
			if result != tt.expected {
				t.Errorf("ParseMethodID(%q) = %d, want %d", tt.method, result, tt.expected)
			}
		})
	}
}

func TestMethodString(t *testing.T) {
	tests := []struct {
		name     string
		id       uint8
		expected string
	}{
		{"GET", MethodGET, "GET"},
		{"POST", MethodPOST, "POST"},
		{"PUT", MethodPUT, "PUT"},
		{"DELETE", MethodDELETE, "DELETE"},
		{"PATCH", MethodPATCH, "PATCH"},
		{"HEAD", MethodHEAD, "HEAD"},
		{"OPTIONS", MethodOPTIONS, "OPTIONS"},
		{"CONNECT", MethodCONNECT, "CONNECT"},
		{"TRACE", MethodTRACE, "TRACE"},
		{"Unknown", MethodUnknown, ""},
		{"Invalid ID", uint8(99), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MethodString(tt.id)
			if result != tt.expected {
				t.Errorf("MethodString(%d) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

func TestMethodBytes(t *testing.T) {
	tests := []struct {
		name     string
		id       uint8
		expected []byte
	}{
		{"GET", MethodGET, []byte("GET")},
		{"POST", MethodPOST, []byte("POST")},
		{"PUT", MethodPUT, []byte("PUT")},
		{"DELETE", MethodDELETE, []byte("DELETE")},
		{"PATCH", MethodPATCH, []byte("PATCH")},
		{"HEAD", MethodHEAD, []byte("HEAD")},
		{"OPTIONS", MethodOPTIONS, []byte("OPTIONS")},
		{"CONNECT", MethodCONNECT, []byte("CONNECT")},
		{"TRACE", MethodTRACE, []byte("TRACE")},
		{"Unknown", MethodUnknown, nil},
		{"Invalid ID", uint8(99), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MethodBytes(tt.id)
			if !bytesEqual(result, tt.expected) {
				t.Errorf("MethodBytes(%d) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

func TestIsValidMethodID(t *testing.T) {
	tests := []struct {
		name     string
		id       uint8
		expected bool
	}{
		{"GET", MethodGET, true},
		{"POST", MethodPOST, true},
		{"TRACE", MethodTRACE, true},
		{"Unknown", MethodUnknown, false},
		{"Invalid high", uint8(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidMethodID(tt.id)
			if result != tt.expected {
				t.Errorf("IsValidMethodID(%d) = %v, want %v", tt.id, result, tt.expected)
			}
		})
	}
}

// TestRoundTrip verifies that ParseMethodID and MethodString are inverses
func TestRoundTrip(t *testing.T) {
	methods := [][]byte{
		[]byte("GET"),
		[]byte("POST"),
		[]byte("PUT"),
		[]byte("DELETE"),
		[]byte("PATCH"),
		[]byte("HEAD"),
		[]byte("OPTIONS"),
		[]byte("CONNECT"),
		[]byte("TRACE"),
	}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			id := ParseMethodID(method)
			if id == MethodUnknown {
				t.Fatalf("ParseMethodID(%q) returned MethodUnknown", method)
			}

			str := MethodString(id)
			if str != string(method) {
				t.Errorf("Round trip failed: %q -> %d -> %q", method, id, str)
			}

			bytes := MethodBytes(id)
			if !bytesEqual(bytes, method) {
				t.Errorf("Round trip bytes failed: %q -> %d -> %q", method, id, bytes)
			}
		})
	}
}

// Helper function for byte slice comparison
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Benchmarks

// BenchmarkParseMethodID measures method parsing performance
// Target: 0 allocs/op, <10 ns/op
func BenchmarkParseMethodID(b *testing.B) {
	method := []byte("GET")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ParseMethodID(method)
	}
}

func BenchmarkParseMethodIDPOST(b *testing.B) {
	method := []byte("POST")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ParseMethodID(method)
	}
}

func BenchmarkParseMethodIDDELETE(b *testing.B) {
	method := []byte("DELETE")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ParseMethodID(method)
	}
}

func BenchmarkParseMethodIDOPTIONS(b *testing.B) {
	method := []byte("OPTIONS")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ParseMethodID(method)
	}
}

func BenchmarkParseMethodIDUnknown(b *testing.B) {
	method := []byte("UNKNOWN")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ParseMethodID(method)
	}
}

func BenchmarkMethodString(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = MethodString(MethodGET)
	}
}

func BenchmarkMethodBytes(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = MethodBytes(MethodGET)
	}
}

// BenchmarkRoundTrip benchmarks the full parse-to-string conversion
func BenchmarkRoundTrip(b *testing.B) {
	method := []byte("POST")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		id := ParseMethodID(method)
		_ = MethodString(id)
	}
}
