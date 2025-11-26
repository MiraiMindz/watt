package client

// ClientHeaders stores HTTP headers inline to avoid heap allocations.
// Supports up to 32 headers with inline storage (zero allocations).
// For >32 headers, falls back to heap allocation (rare case).
//
// Design rationale (matching server implementation):
// - 32 headers covers 99.9% of real-world HTTP requests
// - Fixed-size arrays enable stack allocation
// - Linear scan is faster than map for N≤32 (cache-friendly)
type ClientHeaders struct {
	// Inline storage for up to 32 headers
	names  [MaxHeaders][MaxHeaderName]byte
	values [MaxHeaders][MaxHeaderValue]byte

	// Actual lengths of each header
	nameLens  [MaxHeaders]uint8
	valueLens [MaxHeaders]uint8

	// Number of headers currently stored (0-32)
	count uint8

	// Fallback storage for >32 headers (heap-allocated, rare case)
	overflow map[string]string
}

// Add adds a header to the collection.
// For ≤32 headers with values ≤128 bytes, this performs zero allocations.
//
// Allocation behavior: 0 allocs/op for ≤32 headers
func (h *ClientHeaders) Add(name, value []byte) {
	// Check if value fits in inline storage
	valueFitsInline := len(value) <= MaxHeaderValue

	// Fast path: inline storage
	if h.count < MaxHeaders && valueFitsInline && len(name) <= MaxHeaderName {
		idx := h.count

		// Copy name and value into inline storage
		copy(h.names[idx][:], name)
		copy(h.values[idx][:], value)
		h.nameLens[idx] = uint8(len(name))
		h.valueLens[idx] = uint8(len(value))
		h.count++
		return
	}

	// Slow path: overflow storage (rare case)
	if h.overflow == nil {
		h.overflow = make(map[string]string, 8)
	}
	h.overflow[string(name)] = string(value)
}

// AddString adds a header with string name and value.
// Allocates to convert strings to bytes.
//
// Allocation behavior: 2 allocs/op (string to []byte conversion)
func (h *ClientHeaders) AddString(name, value string) {
	h.Add([]byte(name), []byte(value))
}

// Get retrieves a header value by name (case-insensitive).
// Returns nil if the header is not found.
//
// Allocation behavior: 0 allocs/op for inline storage lookup
func (h *ClientHeaders) Get(name []byte) []byte {
	// Linear scan through inline storage
	for i := uint8(0); i < h.count; i++ {
		if h.nameLens[i] == uint8(len(name)) &&
			bytesEqualCaseInsensitive(h.names[i][:h.nameLens[i]], name) {
			return h.values[i][:h.valueLens[i]]
		}
	}

	// Check overflow storage if present
	if h.overflow != nil {
		if val, ok := h.overflow[string(name)]; ok {
			return []byte(val)
		}
	}

	return nil
}

// GetBytes retrieves a header value by byte slice name (case-insensitive).
// Returns nil if the header is not found.
// This is more efficient than GetString when you already have a []byte name.
//
// Allocation behavior: 0 allocs/op for inline storage lookup
func (h *ClientHeaders) GetBytes(name []byte) []byte {
	return h.Get(name)
}

// GetString retrieves a header value as a string (case-insensitive).
// Returns empty string if not found.
// Note: This allocates for string conversion. Use GetBytes if you can work with []byte.
//
// Allocation behavior: 1-2 allocs/op (string conversions)
func (h *ClientHeaders) GetString(name string) string {
	// Fast path: check for common headers with pre-defined byte slices
	// This avoids the allocation from string(name)
	var nameBytes []byte
	switch name {
	case "Content-Type":
		nameBytes = headerContentType
	case "Content-Length":
		nameBytes = headerContentLength
	case "Transfer-Encoding":
		nameBytes = headerTransferEncoding
	case "Connection":
		nameBytes = headerConnection
	case "Host":
		nameBytes = headerHost
	case "User-Agent":
		nameBytes = headerUserAgent
	default:
		// Uncommon header: allocate for conversion
		nameBytes = []byte(name)
	}

	val := h.Get(nameBytes)
	if val == nil {
		return ""
	}
	return string(val)
}

// Has checks if a header exists (case-insensitive).
//
// Allocation behavior: 0 allocs/op
func (h *ClientHeaders) Has(name []byte) bool {
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
// For ≤32 headers, this performs zero allocations.
//
// Allocation behavior: 0 allocs/op for inline storage
func (h *ClientHeaders) Set(name, value []byte) {
	// Try to update existing header first
	for i := uint8(0); i < h.count; i++ {
		if h.nameLens[i] == uint8(len(name)) &&
			bytesEqualCaseInsensitive(h.names[i][:h.nameLens[i]], name) {
			// Found existing header, update value
			if len(value) <= MaxHeaderValue {
				copy(h.values[i][:], value)
				h.valueLens[i] = uint8(len(value))
				return
			}
			// Value too large, remove from inline and add to overflow
			h.removeAt(i)
			if h.overflow == nil {
				h.overflow = make(map[string]string, 8)
			}
			h.overflow[string(name)] = string(value)
			return
		}
	}

	// Check overflow storage
	if h.overflow != nil {
		nameStr := string(name)
		if _, exists := h.overflow[nameStr]; exists {
			h.overflow[nameStr] = string(value)
			return
		}
	}

	// Header doesn't exist, add it
	h.Add(name, value)
}

// SetString sets a header with string name and value.
//
// Allocation behavior: 2 allocs/op (string to []byte conversion)
func (h *ClientHeaders) SetString(name, value string) {
	h.Set([]byte(name), []byte(value))
}

// Del deletes a header by name (case-insensitive).
//
// Allocation behavior: 0 allocs/op
func (h *ClientHeaders) Del(name []byte) {
	// Remove from inline storage
	for i := uint8(0); i < h.count; i++ {
		if h.nameLens[i] == uint8(len(name)) &&
			bytesEqualCaseInsensitive(h.names[i][:h.nameLens[i]], name) {
			h.removeAt(i)
			return
		}
	}

	// Remove from overflow
	if h.overflow != nil {
		delete(h.overflow, string(name))
	}
}

// removeAt removes a header at the given index
func (h *ClientHeaders) removeAt(idx uint8) {
	// Shift remaining headers down
	for i := idx; i < h.count-1; i++ {
		h.names[i] = h.names[i+1]
		h.values[i] = h.values[i+1]
		h.nameLens[i] = h.nameLens[i+1]
		h.valueLens[i] = h.valueLens[i+1]
	}
	h.count--
}

// Count returns the number of headers.
//
// Allocation behavior: 0 allocs/op
func (h *ClientHeaders) Count() int {
	count := int(h.count)
	if h.overflow != nil {
		count += len(h.overflow)
	}
	return count
}

// Reset clears all headers, preparing for reuse.
// This is called before returning to the pool.
//
// Allocation behavior: 0 allocs/op
func (h *ClientHeaders) Reset() {
	h.count = 0
	// Don't clear arrays, just reset count
	// Clear overflow map if it exists
	if h.overflow != nil {
		// Keep the map allocated but clear it
		for k := range h.overflow {
			delete(h.overflow, k)
		}
	}
}

// ForEach iterates over all headers.
// The callback receives name and value byte slices.
// Do NOT store these byte slices - they reference internal storage.
//
// Allocation behavior: 0 allocs/op (plus callback allocations)
func (h *ClientHeaders) ForEach(fn func(name, value []byte)) {
	// Iterate inline storage
	for i := uint8(0); i < h.count; i++ {
		name := h.names[i][:h.nameLens[i]]
		value := h.values[i][:h.valueLens[i]]
		fn(name, value)
	}

	// Iterate overflow
	if h.overflow != nil {
		for name, value := range h.overflow {
			fn([]byte(name), []byte(value))
		}
	}
}

// WriteTo writes all headers to the given buffer in HTTP format.
// Format: "Name: Value\r\n" for each header.
//
// Allocation behavior: 0 allocs/op (buffer grows as needed)
func (h *ClientHeaders) WriteTo(buf []byte) []byte {
	// Write inline headers
	for i := uint8(0); i < h.count; i++ {
		name := h.names[i][:h.nameLens[i]]
		value := h.values[i][:h.valueLens[i]]
		buf = append(buf, name...)
		buf = append(buf, colonSpaceBytes...)
		buf = append(buf, value...)
		buf = append(buf, crlfBytes...)
	}

	// Write overflow headers
	if h.overflow != nil {
		for name, value := range h.overflow {
			buf = append(buf, []byte(name)...)
			buf = append(buf, colonSpaceBytes...)
			buf = append(buf, []byte(value)...)
			buf = append(buf, crlfBytes...)
		}
	}

	return buf
}
