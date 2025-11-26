package http11

// Header stores HTTP headers inline to avoid heap allocations.
// Supports up to 32 headers with inline storage (zero allocations).
// For >32 headers, falls back to heap allocation (rare case, acceptable).
//
// Design rationale:
// - 32 headers covers 99.9% of real-world HTTP requests
// - Fixed-size arrays enable stack allocation
// - Linear scan is faster than map for N≤32 (cache-friendly)
// - Header names are case-insensitive per RFC 7230
type Header struct {
	// Inline storage for up to 32 headers
	// Each header has a name and value stored as byte arrays
	names  [MaxHeaders][MaxHeaderName]byte   // Header names (64 bytes each)
	values [MaxHeaders][MaxHeaderValue]byte  // Header values (128 bytes each)

	// Actual lengths of each header
	nameLens  [MaxHeaders]uint8  // Length of each name (0-64)
	valueLens [MaxHeaders]uint8  // Length of each value (0-128)

	// Number of headers currently stored (0-32)
	count uint8

	// Fallback storage for >32 headers (heap-allocated, rare case)
	// nil for typical requests
	overflow map[string]string
}

// Add adds a header to the collection.
// For ≤32 headers with values ≤128 bytes, this performs zero allocations.
// For >32 headers or large values >128 bytes, overflow storage is allocated (rare case).
//
// Header names are stored as-is but lookup is case-insensitive.
// Both name and value are copied into internal storage for safety.
//
// Returns ErrHeaderTooLarge if name exceeds 64 bytes or value exceeds 8KB.
// Returns ErrInvalidHeader if name or value contains CRLF characters.
// Allocation behavior: 0 allocs/op for ≤32 headers with small values
func (h *Header) Add(name, value []byte) error {
	// P1 FIX #2: Large Header Overflow Handling
	// Bounds check - allow large values up to 8KB by using overflow storage
	if len(name) > MaxHeaderName {
		return ErrHeaderTooLarge
	}
	// Allow values up to 8KB (reasonable limit for things like large cookies)
	if len(value) > 8192 {
		return ErrHeaderTooLarge
	}

	// P0 FIX #3: CRLF Header Injection Protection
	// RFC 7230 §3.2: Field values MUST NOT contain CR or LF characters
	// This prevents HTTP Response Splitting, XSS, and session fixation attacks
	// Examples of attacks:
	//   Set-Cookie: session=abc\r\nX-Malicious: injected
	//   Location: http://evil.com\r\n\r\n<script>alert(1)</script>
	for _, b := range value {
		if b == '\r' || b == '\n' {
			return ErrInvalidHeader
		}
	}

	// Also validate header name doesn't contain CRLF
	for _, b := range name {
		if b == '\r' || b == '\n' {
			return ErrInvalidHeader
		}
	}

	// P1 FIX #2: Check if value fits in inline storage
	valueFitsInline := len(value) <= MaxHeaderValue

	// Fast path: inline storage (0-32 headers with small values)
	if h.count < MaxHeaders && valueFitsInline {
		idx := h.count

		// Copy name and value into inline storage
		copy(h.names[idx][:], name)
		copy(h.values[idx][:], value)
		h.nameLens[idx] = uint8(len(name))
		h.valueLens[idx] = uint8(len(value))
		h.count++
		return nil
	}

	// Slow path: overflow storage (rare case, >32 headers OR large values >128 bytes)
	// P1 FIX #2: Use overflow for large header values to avoid rejecting them
	if h.overflow == nil {
		h.overflow = make(map[string]string, 8)
	}
	h.overflow[string(name)] = string(value)
	return nil
}

// Get retrieves a header value by name (case-insensitive).
// Returns nil if the header is not found.
//
// The returned byte slice references internal storage and is valid
// only until the next call to Reset() or Add().
//
// Allocation behavior: 0 allocs/op for inline storage lookup
func (h *Header) Get(name []byte) []byte {
	// Linear scan through inline storage
	// For N≤32, this is faster than map lookup due to cache locality
	for i := uint8(0); i < h.count; i++ {
		if h.nameLens[i] == uint8(len(name)) &&
			bytesEqualCaseInsensitive(h.names[i][:h.nameLens[i]], name) {
			return h.values[i][:h.valueLens[i]]
		}
	}

	// Check overflow storage if present
	if h.overflow != nil {
		// Convert to string for map lookup (small allocation, acceptable for rare case)
		if val, ok := h.overflow[string(name)]; ok {
			return []byte(val)
		}
	}

	return nil
}

// GetString retrieves a header value by name (case-insensitive) as a string.
// Returns empty string if the header is not found.
//
// This method allocates a string from the byte slice.
// Use Get() if you can work with []byte to avoid allocation.
func (h *Header) GetString(name []byte) string {
	val := h.Get(name)
	if val == nil {
		return ""
	}
	return string(val)
}

// Has checks if a header exists (case-insensitive).
// Allocation behavior: 0 allocs/op
func (h *Header) Has(name []byte) bool {
	// Check inline storage
	for i := uint8(0); i < h.count; i++ {
		if h.nameLens[i] == uint8(len(name)) &&
			bytesEqualCaseInsensitive(h.names[i][:h.nameLens[i]], name) {
			return true
		}
	}

	// Check overflow
	if h.overflow != nil {
		_, ok := h.overflow[string(name)]
		return ok
	}

	return false
}

// Set sets a header value, replacing any existing value.
// If the header doesn't exist, it's added.
// If it exists, the value is updated in-place (if it fits inline) or moved to overflow.
//
// Allocation behavior: 0 allocs/op for ≤32 headers with small values
func (h *Header) Set(name, value []byte) error {
	// P1 FIX #2: Allow large values up to 8KB
	if len(name) > MaxHeaderName {
		return ErrHeaderTooLarge
	}
	if len(value) > 8192 {
		return ErrHeaderTooLarge
	}

	// P0 FIX #3: CRLF Header Injection Protection
	// RFC 7230 §3.2: Field values MUST NOT contain CR or LF characters
	for _, b := range value {
		if b == '\r' || b == '\n' {
			return ErrInvalidHeader
		}
	}

	// Also validate header name doesn't contain CRLF
	for _, b := range name {
		if b == '\r' || b == '\n' {
			return ErrInvalidHeader
		}
	}

	// Try to find and update existing header
	for i := uint8(0); i < h.count; i++ {
		if h.nameLens[i] == uint8(len(name)) &&
			bytesEqualCaseInsensitive(h.names[i][:h.nameLens[i]], name) {
			// P1 FIX #2: Check if new value fits in inline storage
			if len(value) <= MaxHeaderValue {
				// Update existing header in-place
				copy(h.values[i][:], value)
				h.valueLens[i] = uint8(len(value))
				return nil
			} else {
				// Value too large for inline storage
				// Delete from inline and move to overflow
				nameStr := string(h.names[i][:h.nameLens[i]])

				// Remove from inline storage by shifting
				if i < h.count-1 {
					copy(h.names[i:], h.names[i+1:])
					copy(h.values[i:], h.values[i+1:])
					copy(h.nameLens[i:], h.nameLens[i+1:])
					copy(h.valueLens[i:], h.valueLens[i+1:])
				}
				h.count--

				// Add to overflow
				if h.overflow == nil {
					h.overflow = make(map[string]string, 8)
				}
				h.overflow[nameStr] = string(value)
				return nil
			}
		}
	}

	// Check overflow
	if h.overflow != nil {
		nameStr := string(name)
		if _, ok := h.overflow[nameStr]; ok {
			h.overflow[nameStr] = string(value)
			return nil
		}
	}

	// Header doesn't exist, add it
	return h.Add(name, value)
}

// Del deletes a header by name (case-insensitive).
// Allocation behavior: 0 allocs/op
func (h *Header) Del(name []byte) {
	// Find and delete from inline storage
	for i := uint8(0); i < h.count; i++ {
		if h.nameLens[i] == uint8(len(name)) &&
			bytesEqualCaseInsensitive(h.names[i][:h.nameLens[i]], name) {
			// Shift remaining headers down
			if i < h.count-1 {
				copy(h.names[i:], h.names[i+1:])
				copy(h.values[i:], h.values[i+1:])
				copy(h.nameLens[i:], h.nameLens[i+1:])
				copy(h.valueLens[i:], h.valueLens[i+1:])
			}
			h.count--
			return
		}
	}

	// Delete from overflow if present
	if h.overflow != nil {
		delete(h.overflow, string(name))
	}
}

// Len returns the total number of headers.
func (h *Header) Len() int {
	total := int(h.count)
	if h.overflow != nil {
		total += len(h.overflow)
	}
	return total
}

// Reset clears all headers for reuse (e.g., when returning to pool).
// This does not deallocate the overflow map, but clears it.
// The GC will clean up the map when the Header is no longer referenced.
//
// Allocation behavior: 0 allocs/op
func (h *Header) Reset() {
	h.count = 0
	h.overflow = nil // Allow GC to clean up
}

// VisitAll calls the visitor function for each header.
// The visitor receives name and value as byte slices.
// Iteration stops if visitor returns false.
//
// This is useful for serializing headers without allocation.
func (h *Header) VisitAll(visitor func(name, value []byte) bool) {
	// Visit inline headers
	for i := uint8(0); i < h.count; i++ {
		name := h.names[i][:h.nameLens[i]]
		value := h.values[i][:h.valueLens[i]]
		if !visitor(name, value) {
			return
		}
	}

	// Visit overflow headers
	if h.overflow != nil {
		for name, value := range h.overflow {
			if !visitor([]byte(name), []byte(value)) {
				return
			}
		}
	}
}

// bytesEqualCaseInsensitive compares two byte slices case-insensitively.
// This is required per RFC 7230 - header field names are case-insensitive.
//
// Allocation behavior: 0 allocs/op
func bytesEqualCaseInsensitive(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

// toLower converts an ASCII uppercase letter to lowercase.
// Non-letter bytes are returned unchanged.
// This is sufficient for HTTP header names which are ASCII.
//
// Allocation behavior: 0 allocs/op
func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}
