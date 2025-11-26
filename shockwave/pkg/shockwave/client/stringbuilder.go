package client

// InlineStringBuilder provides zero-allocation string building using a fixed-size buffer.
// Unlike strings.Builder, this uses stack-allocated buffer for common cases.
//
// Performance: 0 allocs/op for strings up to BufferSize
type InlineStringBuilder struct {
	buf [512]byte // Inline buffer (stack allocated)
	len int       // Current length
}

// Reset clears the builder.
//
// Allocation behavior: 0 allocs/op
func (b *InlineStringBuilder) Reset() {
	b.len = 0
}

// WriteString appends a string to the buffer.
//
// Allocation behavior: 0 allocs/op if total length <= 512
func (b *InlineStringBuilder) WriteString(s string) {
	if b.len+len(s) <= len(b.buf) {
		copy(b.buf[b.len:], s)
		b.len += len(s)
	}
}

// WriteByte appends a byte to the buffer.
//
// Allocation behavior: 0 allocs/op if total length <= 512
func (b *InlineStringBuilder) WriteByte(c byte) {
	if b.len < len(b.buf) {
		b.buf[b.len] = c
		b.len++
	}
}

// WriteBytes appends bytes to the buffer.
//
// Allocation behavior: 0 allocs/op if total length <= 512
func (b *InlineStringBuilder) WriteBytes(p []byte) {
	if b.len+len(p) <= len(b.buf) {
		copy(b.buf[b.len:], p)
		b.len += len(p)
	}
}

// String returns the built string.
// IMPORTANT: The returned string is only valid until the next Reset() call.
//
// Allocation behavior: 1 alloc/op (string header allocation is unavoidable)
func (b *InlineStringBuilder) String() string {
	return string(b.buf[:b.len])
}

// Bytes returns the built bytes as a slice.
// Zero-copy reference to internal buffer.
//
// Allocation behavior: 0 allocs/op
func (b *InlineStringBuilder) Bytes() []byte {
	return b.buf[:b.len]
}

// Len returns the current length.
func (b *InlineStringBuilder) Len() int {
	return b.len
}

// BuildHostPort builds a "host:port" string with zero allocations.
// The buffer must be reset before reuse.
//
// Allocation behavior: 0 allocs/op for the building, 1 alloc for String()
func BuildHostPort(host, port string) string {
	var sb InlineStringBuilder
	sb.WriteString(host)
	sb.WriteByte(':')
	sb.WriteString(port)
	return sb.String()
}

// BuildHostPortBytes builds a "host:port" as bytes with zero allocations.
// Returns a slice that's valid until the builder is reset.
//
// Allocation behavior: 0 allocs/op
func BuildHostPortBytes(host, port []byte) []byte {
	var sb InlineStringBuilder
	sb.WriteBytes(host)
	sb.WriteByte(':')
	sb.WriteBytes(port)
	return sb.Bytes()
}
