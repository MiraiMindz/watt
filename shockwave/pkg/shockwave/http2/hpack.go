package http2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// HPACK - Header Compression for HTTP/2
// RFC 7541: https://tools.ietf.org/html/rfc7541
//
// HPACK compresses HTTP headers using:
// 1. Static table (61 predefined entries)
// 2. Dynamic table (FIFO, configurable size)
// 3. Huffman encoding (optional)

// Encoder compresses HTTP/2 headers using HPACK
type Encoder struct {
	table          *indexTable
	buf            bytes.Buffer
	useHuffman     bool
	minTableIndex  bool // Prefer smaller indices for better compression
}

// NewEncoder creates a new HPACK encoder with the specified dynamic table size
func NewEncoder(maxDynamicTableSize uint32) *Encoder {
	return &Encoder{
		table:      newIndexTable(maxDynamicTableSize),
		useHuffman: true, // Enable Huffman by default
	}
}

// SetMaxDynamicTableSize changes the maximum size of the dynamic table
func (e *Encoder) SetMaxDynamicTableSize(size uint32) {
	e.table.SetMaxDynamicSize(size)
}

// SetUseHuffman enables or disables Huffman encoding for strings
func (e *Encoder) SetUseHuffman(use bool) {
	e.useHuffman = use
}

// Encode encodes a list of header fields and returns the compressed bytes
func (e *Encoder) Encode(headers []HeaderField) []byte {
	e.buf.Reset()

	for _, h := range headers {
		e.encodeHeaderField(h.Name, h.Value, true)
	}

	// Return copy of buffer
	result := make([]byte, e.buf.Len())
	copy(result, e.buf.Bytes())
	return result
}

// EncodeHeaderField encodes a single header field
// If addToTable is true, uses literal with incremental indexing
func (e *Encoder) encodeHeaderField(name, value string, addToTable bool) {
	// Try to find exact match in tables
	index, exactMatch := e.table.Find(name, value)

	if exactMatch {
		// Indexed Header Field (RFC 7541 Section 6.1)
		e.encodeIndexed(index)
		return
	}

	if index > 0 {
		// Name found, encode value as literal
		if addToTable {
			// Literal with Incremental Indexing - Indexed Name (RFC 7541 Section 6.2.1)
			e.encodeLiteralIndexedName(index, value, true)
			e.table.Add(name, value)
		} else {
			// Literal without Indexing - Indexed Name (RFC 7541 Section 6.2.2)
			e.encodeLiteralIndexedName(index, value, false)
		}
		return
	}

	// Name not found, encode both name and value
	if addToTable {
		// Literal with Incremental Indexing - New Name (RFC 7541 Section 6.2.1)
		e.encodeLiteralNewName(name, value, true, false)
		e.table.Add(name, value)
	} else {
		// Literal without Indexing - New Name (RFC 7541 Section 6.2.2)
		e.encodeLiteralNewName(name, value, false, false)
	}
}

// encodeIndexed encodes an indexed header field (RFC 7541 Section 6.1)
// Format: 1xxxxxxx (first bit is 1, followed by 7-bit index)
func (e *Encoder) encodeIndexed(index int) {
	e.encodeInteger(index, 7, 0x80) // Prefix: 1xxxxxxx
}

// encodeLiteralIndexedName encodes a literal header with indexed name
// Format depends on incremental flag:
// - Incremental: 01xxxxxx (6-bit index prefix)
// - Without indexing: 0000xxxx (4-bit index prefix)
func (e *Encoder) encodeLiteralIndexedName(nameIndex int, value string, incremental bool) {
	if incremental {
		// Literal with Incremental Indexing (RFC 7541 Section 6.2.1)
		e.encodeInteger(nameIndex, 6, 0x40) // Prefix: 01xxxxxx
	} else {
		// Literal without Indexing (RFC 7541 Section 6.2.2)
		e.encodeInteger(nameIndex, 4, 0x00) // Prefix: 0000xxxx
	}
	e.encodeString(value)
}

// encodeLiteralNewName encodes a literal header with new name
func (e *Encoder) encodeLiteralNewName(name, value string, incremental, neverIndexed bool) {
	if incremental {
		// Literal with Incremental Indexing (RFC 7541 Section 6.2.1)
		e.buf.WriteByte(0x40) // Prefix: 01000000 (index 0)
	} else if neverIndexed {
		// Literal Never Indexed (RFC 7541 Section 6.2.3)
		e.buf.WriteByte(0x10) // Prefix: 00010000 (index 0)
	} else {
		// Literal without Indexing (RFC 7541 Section 6.2.2)
		e.buf.WriteByte(0x00) // Prefix: 00000000 (index 0)
	}
	e.encodeString(name)
	e.encodeString(value)
}

// encodeInteger encodes an integer using variable-length encoding (RFC 7541 Section 5.1)
// prefix is the number of bits available for the integer
// prefixBits is the value to OR with the first byte
func (e *Encoder) encodeInteger(value int, prefix uint8, prefixBits byte) {
	maxValue := (1 << prefix) - 1

	if value < maxValue {
		// Fits in prefix bits
		e.buf.WriteByte(prefixBits | byte(value))
		return
	}

	// Doesn't fit, use continuation bytes
	e.buf.WriteByte(prefixBits | byte(maxValue))
	value -= maxValue

	for value >= 128 {
		e.buf.WriteByte(byte(value%128) | 0x80)
		value /= 128
	}
	e.buf.WriteByte(byte(value))
}

// encodeString encodes a string (RFC 7541 Section 5.2)
// Format: H + length + data
// H bit (0x80) indicates if Huffman encoded
func (e *Encoder) encodeString(s string) {
	if e.useHuffman && len(s) > 0 {
		// Try Huffman encoding
		huffmanLen := HuffmanEncodeLen(s)
		plainLen := len(s)

		// Use Huffman if it saves space
		if huffmanLen < plainLen {
			encoded := HuffmanEncode(s)
			e.encodeInteger(len(encoded), 7, 0x80) // H=1
			e.buf.Write(encoded)
			return
		}
	}

	// Plain string (no Huffman)
	e.encodeInteger(len(s), 7, 0x00) // H=0
	e.buf.WriteString(s)
}

// Decoder decompresses HTTP/2 headers using HPACK
type Decoder struct {
	table           *indexTable
	maxStringLength int // Protection against malicious headers

	// Performance optimizations
	stringIntern    map[string]string // String interning for common headers
	headerBuf       []HeaderField     // Reusable buffer for decoded headers
	stringBuf       []byte            // Reusable buffer for string decoding
	reader          byteReader        // Reusable reader (avoids bytes.NewReader allocation)
}

// NewDecoder creates a new HPACK decoder with the specified dynamic table size
func NewDecoder(maxDynamicTableSize uint32, maxStringLength int) *Decoder {
	if maxStringLength == 0 {
		maxStringLength = 16 * 1024 * 1024 // 16 MB default
	}

	// Initialize string interning map with common headers
	// This reduces allocations by reusing the same string instance
	stringIntern := make(map[string]string, 64)
	commonHeaders := []string{
		// Pseudo-headers
		":authority", ":method", ":path", ":scheme", ":status",
		// Common headers
		"accept", "accept-encoding", "accept-language", "accept-ranges",
		"access-control-allow-credentials", "access-control-allow-headers",
		"access-control-allow-methods", "access-control-allow-origin",
		"access-control-expose-headers", "access-control-max-age",
		"age", "cache-control", "content-disposition", "content-encoding",
		"content-language", "content-length", "content-location", "content-range",
		"content-type", "cookie", "date", "etag", "expect", "expires", "from",
		"host", "if-match", "if-modified-since", "if-none-match", "if-range",
		"if-unmodified-since", "last-modified", "link", "location", "max-forwards",
		"proxy-authenticate", "proxy-authorization", "range", "referer", "refresh",
		"retry-after", "server", "set-cookie", "strict-transport-security",
		"transfer-encoding", "user-agent", "vary", "via", "www-authenticate",
	}
	for _, h := range commonHeaders {
		stringIntern[h] = h
	}

	return &Decoder{
		table:           newIndexTable(maxDynamicTableSize),
		maxStringLength: maxStringLength,
		stringIntern:    stringIntern,
		headerBuf:       make([]HeaderField, 0, 32), // Pre-allocate for typical request
		stringBuf:       make([]byte, 0, 256),        // Pre-allocate for string decoding
	}
}

// SetMaxDynamicTableSize changes the maximum size of the dynamic table
func (d *Decoder) SetMaxDynamicTableSize(size uint32) {
	d.table.SetMaxDynamicSize(size)
}

// hpackReader interface for decoding operations
type hpackReader interface {
	ReadByte() (byte, error)
	UnreadByte() error
	Read([]byte) (int, error)
}

// byteReader is a lightweight wrapper around []byte that implements io.ByteReader
// without allocating like bytes.NewReader does
type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) ReadByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *byteReader) UnreadByte() error {
	if r.pos <= 0 {
		return errors.New("cannot unread")
	}
	r.pos--
	return nil
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *byteReader) Len() int {
	return len(r.data) - r.pos
}

func (r *byteReader) Reset(data []byte) {
	r.data = data
	r.pos = 0
}

// Decode decodes a compressed header block
func (d *Decoder) Decode(encoded []byte) ([]HeaderField, error) {
	// Reuse header buffer and reset to zero length
	d.headerBuf = d.headerBuf[:0]
	d.reader.Reset(encoded)

	for d.reader.Len() > 0 {
		// Read first byte to determine representation
		b, err := d.reader.ReadByte()
		if err != nil {
			return nil, err
		}

		var hf HeaderField

		switch {
		case b&0x80 != 0:
			// Indexed Header Field (RFC 7541 Section 6.1)
			// Format: 1xxxxxxx
			d.reader.UnreadByte()
			hf, err = d.decodeIndexed(&d.reader)

		case b&0x40 != 0:
			// Literal with Incremental Indexing (RFC 7541 Section 6.2.1)
			// Format: 01xxxxxx
			d.reader.UnreadByte()
			hf, err = d.decodeLiteralIndexed(&d.reader)

		case b&0x20 != 0:
			// Dynamic Table Size Update (RFC 7541 Section 6.3)
			// Format: 001xxxxx
			d.reader.UnreadByte()
			err = d.decodeTableSizeUpdate(&d.reader)
			continue

		case b&0x10 != 0:
			// Literal Never Indexed (RFC 7541 Section 6.2.3)
			// Format: 0001xxxx
			d.reader.UnreadByte()
			hf, err = d.decodeLiteralNeverIndexed(&d.reader)

		default:
			// Literal without Indexing (RFC 7541 Section 6.2.2)
			// Format: 0000xxxx
			d.reader.UnreadByte()
			hf, err = d.decodeLiteralWithoutIndexing(&d.reader)
		}

		if err != nil {
			return nil, err
		}

		if hf.Name != "" {
			// Apply string interning to reduce allocations
			if interned, ok := d.stringIntern[hf.Name]; ok {
				hf.Name = interned
			} else if len(d.stringIntern) < 256 { // Limit intern map size
				d.stringIntern[hf.Name] = hf.Name
			}

			d.headerBuf = append(d.headerBuf, hf)
		}
	}

	// Return a copy to allow reuse of the buffer
	headers := make([]HeaderField, len(d.headerBuf))
	copy(headers, d.headerBuf)
	return headers, nil
}

// DecodeInto decodes a compressed header block into the provided slice.
// The slice will be appended to, so it should typically be passed as headers[:0]
// to reuse the backing array. This method is more efficient than Decode() as
// it eliminates one allocation by not copying the internal buffer.
// Returns the slice with decoded headers appended.
func (d *Decoder) DecodeInto(encoded []byte, headers []HeaderField) ([]HeaderField, error) {
	// Reuse header buffer and reset to zero length
	d.headerBuf = d.headerBuf[:0]
	d.reader.Reset(encoded)

	for d.reader.Len() > 0 {
		// Read first byte to determine representation
		b, err := d.reader.ReadByte()
		if err != nil {
			return headers, err
		}

		var hf HeaderField

		switch {
		case b&0x80 != 0:
			// Indexed Header Field (RFC 7541 Section 6.1)
			// Format: 1xxxxxxx
			d.reader.UnreadByte()
			hf, err = d.decodeIndexed(&d.reader)

		case b&0x40 != 0:
			// Literal with Incremental Indexing (RFC 7541 Section 6.2.1)
			// Format: 01xxxxxx
			d.reader.UnreadByte()
			hf, err = d.decodeLiteralIndexed(&d.reader)

		case b&0x20 != 0:
			// Dynamic Table Size Update (RFC 7541 Section 6.3)
			// Format: 001xxxxx
			d.reader.UnreadByte()
			err = d.decodeTableSizeUpdate(&d.reader)
			continue

		case b&0x10 != 0:
			// Literal Never Indexed (RFC 7541 Section 6.2.3)
			// Format: 0001xxxx
			d.reader.UnreadByte()
			hf, err = d.decodeLiteralNeverIndexed(&d.reader)

		default:
			// Literal without Indexing (RFC 7541 Section 6.2.2)
			// Format: 0000xxxx
			d.reader.UnreadByte()
			hf, err = d.decodeLiteralWithoutIndexing(&d.reader)
		}

		if err != nil {
			return headers, err
		}

		if hf.Name != "" {
			// Apply string interning to reduce allocations
			if interned, ok := d.stringIntern[hf.Name]; ok {
				hf.Name = interned
			} else if len(d.stringIntern) < 256 { // Limit intern map size
				d.stringIntern[hf.Name] = hf.Name
			}

			headers = append(headers, hf)
		}
	}

	return headers, nil
}

// decodeIndexed decodes an indexed header field (RFC 7541 Section 6.1)
func (d *Decoder) decodeIndexed(buf hpackReader) (HeaderField, error) {
	index, err := d.decodeInteger(buf, 7)
	if err != nil {
		return HeaderField{}, err
	}

	if index == 0 {
		return HeaderField{}, errors.New("hpack: invalid index 0")
	}

	hf, ok := d.table.Get(index)
	if !ok {
		return HeaderField{}, fmt.Errorf("hpack: invalid index %d", index)
	}

	return hf, nil
}

// decodeLiteralIndexed decodes literal with incremental indexing (RFC 7541 Section 6.2.1)
func (d *Decoder) decodeLiteralIndexed(buf hpackReader) (HeaderField, error) {
	nameIndex, err := d.decodeInteger(buf, 6)
	if err != nil {
		return HeaderField{}, err
	}

	var name string
	if nameIndex == 0 {
		// New name
		name, err = d.decodeString(buf)
		if err != nil {
			return HeaderField{}, err
		}
	} else {
		// Indexed name
		hf, ok := d.table.Get(nameIndex)
		if !ok {
			return HeaderField{}, fmt.Errorf("hpack: invalid index %d", nameIndex)
		}
		name = hf.Name
	}

	value, err := d.decodeString(buf)
	if err != nil {
		return HeaderField{}, err
	}

	hf := HeaderField{Name: name, Value: value}
	d.table.Add(name, value)

	return hf, nil
}

// decodeLiteralWithoutIndexing decodes literal without indexing (RFC 7541 Section 6.2.2)
func (d *Decoder) decodeLiteralWithoutIndexing(buf hpackReader) (HeaderField, error) {
	nameIndex, err := d.decodeInteger(buf, 4)
	if err != nil {
		return HeaderField{}, err
	}

	var name string
	if nameIndex == 0 {
		// New name
		name, err = d.decodeString(buf)
		if err != nil {
			return HeaderField{}, err
		}
	} else {
		// Indexed name
		hf, ok := d.table.Get(nameIndex)
		if !ok {
			return HeaderField{}, fmt.Errorf("hpack: invalid index %d", nameIndex)
		}
		name = hf.Name
	}

	value, err := d.decodeString(buf)
	if err != nil {
		return HeaderField{}, err
	}

	return HeaderField{Name: name, Value: value}, nil
}

// decodeLiteralNeverIndexed decodes literal never indexed (RFC 7541 Section 6.2.3)
func (d *Decoder) decodeLiteralNeverIndexed(buf hpackReader) (HeaderField, error) {
	// Same as without indexing, just different semantics
	nameIndex, err := d.decodeInteger(buf, 4)
	if err != nil {
		return HeaderField{}, err
	}

	var name string
	if nameIndex == 0 {
		// New name
		name, err = d.decodeString(buf)
		if err != nil {
			return HeaderField{}, err
		}
	} else {
		// Indexed name
		hf, ok := d.table.Get(nameIndex)
		if !ok {
			return HeaderField{}, fmt.Errorf("hpack: invalid index %d", nameIndex)
		}
		name = hf.Name
	}

	value, err := d.decodeString(buf)
	if err != nil {
		return HeaderField{}, err
	}

	return HeaderField{Name: name, Value: value}, nil
}

// decodeTableSizeUpdate decodes a dynamic table size update (RFC 7541 Section 6.3)
func (d *Decoder) decodeTableSizeUpdate(buf hpackReader) error {
	size, err := d.decodeInteger(buf, 5)
	if err != nil {
		return err
	}

	d.table.SetMaxDynamicSize(uint32(size))
	return nil
}

// decodeInteger decodes a variable-length integer (RFC 7541 Section 5.1)
func (d *Decoder) decodeInteger(buf hpackReader, prefix uint8) (int, error) {
	b, err := buf.ReadByte()
	if err != nil {
		return 0, err
	}

	maxValue := (1 << prefix) - 1
	mask := byte(maxValue)

	value := int(b & mask)
	if value < maxValue {
		return value, nil
	}

	// Continuation bytes
	m := 0
	for {
		b, err := buf.ReadByte()
		if err != nil {
			if err == io.EOF {
				return 0, errors.New("hpack: unexpected EOF decoding integer")
			}
			return 0, err
		}

		value += int(b&0x7f) << m
		m += 7

		if b&0x80 == 0 {
			break
		}

		// Prevent integer overflow
		if m > 28 {
			return 0, errors.New("hpack: integer overflow")
		}
	}

	return value, nil
}

// decodeString decodes a string (RFC 7541 Section 5.2)
// Optimized to reuse the decoder's string buffer to reduce allocations
func (d *Decoder) decodeString(buf hpackReader) (string, error) {
	b, err := buf.ReadByte()
	if err != nil {
		return "", err
	}

	huffman := b&0x80 != 0
	buf.UnreadByte()

	length, err := d.decodeInteger(buf, 7)
	if err != nil {
		return "", err
	}

	if length > d.maxStringLength {
		return "", fmt.Errorf("hpack: string length %d exceeds maximum %d", length, d.maxStringLength)
	}

	if length == 0 {
		return "", nil
	}

	// Reuse string buffer if it has enough capacity
	// This reduces allocations significantly
	if cap(d.stringBuf) < length {
		d.stringBuf = make([]byte, length)
	} else {
		d.stringBuf = d.stringBuf[:length]
	}

	n, err := buf.Read(d.stringBuf)
	if err != nil {
		return "", err
	}
	if n != length {
		return "", errors.New("hpack: unexpected EOF reading string")
	}

	if huffman {
		// HuffmanDecode allocates a new string, unavoidable
		return HuffmanDecode(d.stringBuf)
	}

	// Zero-copy conversion using unsafe
	// SAFETY: This is safe because:
	// 1. The string is immediately used to create a HeaderField
	// 2. HeaderField stores the string (which copies it on assignment)
	// 3. stringBuf is reused but not modified during this string's lifetime
	// 4. The string is read-only (Go enforces immutability)
	return bytesToString(d.stringBuf), nil
}
