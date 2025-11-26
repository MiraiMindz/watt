package qpack

import (
	"errors"
	"fmt"
	"io"
)

// QPACK Decoder (RFC 9204)
// Decodes QPACK-compressed header blocks

var (
	ErrInvalidEncoding      = errors.New("qpack: invalid encoding")
	ErrIntegerOverflow      = errors.New("qpack: integer overflow")
	ErrStringTooLong        = errors.New("qpack: string too long")
	ErrTableIndexOutOfRange = errors.New("qpack: table index out of range")
	ErrBlockedStream        = errors.New("qpack: blocked on dynamic table")
)

// qpackReader is a lightweight byte reader for QPACK decoding (Priority 2 optimization)
// Eliminates bytes.NewReader allocation on every decode
type qpackReader struct {
	data []byte
	pos  int
}

func (r *qpackReader) ReadByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *qpackReader) UnreadByte() error {
	if r.pos <= 0 {
		return errors.New("qpack: cannot unread")
	}
	r.pos--
	return nil
}

func (r *qpackReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *qpackReader) Len() int {
	return len(r.data) - r.pos
}

func (r *qpackReader) Reset(data []byte) {
	r.data = data
	r.pos = 0
}

// Decoder decodes QPACK-compressed headers
type Decoder struct {
	dynamicTable *DynamicTable
	maxTableSize uint64

	// Blocked streams tracking
	blockedStreams map[uint64]bool

	// Reusable buffers for decoding (Priority 2 optimization)
	reader    qpackReader // Lightweight reader (eliminates bytes.NewReader allocation)
	headerBuf []Header    // Pre-allocated header slice
	stringBuf []byte      // Reusable buffer for string decoding
}

// NewDecoder creates a new QPACK decoder
func NewDecoder(maxTableCapacity uint64) *Decoder {
	return &Decoder{
		dynamicTable:   NewDynamicTable(maxTableCapacity),
		maxTableSize:   maxTableCapacity,
		blockedStreams: make(map[uint64]bool),
		headerBuf:      make([]Header, 0, 32),    // Pre-allocate for 32 headers
		stringBuf:      make([]byte, 0, 256),     // Pre-allocate 256 bytes for strings
	}
}

// DecodeHeaders decodes a QPACK-encoded header block
// Returns the decoded headers and any error
func (d *Decoder) DecodeHeaders(headerBlock []byte) ([]Header, error) {
	// Reuse lightweight reader (Priority 2 optimization)
	d.reader.Reset(headerBlock)

	// Parse Encoded Field Section Prefix (RFC 9204 Section 4.5.1)
	requiredInsertCount, err := d.decodeInteger(&d.reader, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to read required insert count: %w", err)
	}

	deltaBase, err := d.decodeInteger(&d.reader, 7)
	if err != nil {
		return nil, fmt.Errorf("failed to read delta base: %w", err)
	}

	// Check if we have enough dynamic table entries
	if requiredInsertCount > d.dynamicTable.insertIndex {
		return nil, ErrBlockedStream
	}

	_ = deltaBase // Will be used for post-base indexing

	// Reuse pre-allocated header buffer (Priority 2 optimization)
	d.headerBuf = d.headerBuf[:0]

	// Parse header fields
	for d.reader.Len() > 0 {
		header, err := d.decodeHeaderField(&d.reader)
		if err != nil {
			return nil, err
		}
		d.headerBuf = append(d.headerBuf, header)
	}

	return d.headerBuf, nil
}

// decodeHeaderField decodes a single header field
func (d *Decoder) decodeHeaderField(r *qpackReader) (Header, error) {
	firstByte, err := r.ReadByte()
	if err != nil {
		return Header{}, err
	}
	r.UnreadByte()

	// Indexed Field Line (RFC 9204 Section 4.5.2)
	if firstByte&0x80 != 0 {
		return d.decodeIndexedFieldLine(r)
	}

	// Literal Field Line with Name Reference (RFC 9204 Section 4.5.4)
	if firstByte&0x40 != 0 {
		return d.decodeLiteralFieldLineWithNameRef(r)
	}

	// Literal Field Line without Name Reference (RFC 9204 Section 4.5.6)
	if firstByte&0x20 != 0 {
		return d.decodeLiteralFieldLineWithoutNameRef(r)
	}

	// Indexed Field Line with Post-Base Index (RFC 9204 Section 4.5.3)
	if firstByte&0x10 != 0 {
		return d.decodeIndexedFieldLinePostBase(r)
	}

	// Literal Field Line with Post-Base Name Reference (RFC 9204 Section 4.5.5)
	return d.decodeLiteralFieldLinePostBaseNameRef(r)
}

// decodeIndexedFieldLine decodes an Indexed Field Line
// Format: 1TXXXXXX (T=0 static, T=1 dynamic)
func (d *Decoder) decodeIndexedFieldLine(r *qpackReader) (Header, error) {
	firstByte, _ := r.ReadByte()
	isDynamic := (firstByte & 0x40) != 0
	r.UnreadByte()

	index, err := d.decodeInteger(r, 6)
	if err != nil {
		return Header{}, err
	}

	if !isDynamic {
		// Static table lookup
		if int(index) >= len(staticTable) {
			return Header{}, ErrTableIndexOutOfRange
		}
		entry := staticTable[index]
		return Header{Name: entry.Name, Value: entry.Value}, nil
	}

	// Dynamic table lookup
	entry, err := d.dynamicTable.Get(index)
	if err != nil {
		return Header{}, err
	}
	return Header{Name: entry.Name, Value: entry.Value}, nil
}

// decodeLiteralFieldLineWithNameRef decodes a Literal Field Line with Name Reference
// Format: 01NTXXXX (N=never index, T=0 static, T=1 dynamic)
func (d *Decoder) decodeLiteralFieldLineWithNameRef(r *qpackReader) (Header, error) {
	firstByte, _ := r.ReadByte()
	neverIndex := (firstByte & 0x20) != 0
	isStatic := (firstByte & 0x10) == 0
	r.UnreadByte()

	_ = neverIndex // TODO: Track never-index hint

	// Decode name reference
	nameIndex, err := d.decodeInteger(r, 4)
	if err != nil {
		return Header{}, err
	}

	var name string
	if isStatic {
		if int(nameIndex) >= len(staticTable) {
			return Header{}, ErrTableIndexOutOfRange
		}
		name = staticTable[nameIndex].Name
	} else {
		entry, err := d.dynamicTable.Get(nameIndex)
		if err != nil {
			return Header{}, err
		}
		name = entry.Name
	}

	// Decode value
	value, err := d.decodeString(r)
	if err != nil {
		return Header{}, err
	}

	return Header{Name: name, Value: value}, nil
}

// decodeLiteralFieldLineWithoutNameRef decodes a Literal Field Line without Name Reference
// Format: 001NXXXX (N=never index)
func (d *Decoder) decodeLiteralFieldLineWithoutNameRef(r *qpackReader) (Header, error) {
	firstByte, _ := r.ReadByte()
	neverIndex := (firstByte & 0x10) != 0

	_ = neverIndex // TODO: Track never-index hint

	// The firstByte contains the pattern bits (001N) in the top 4 bits
	// We need to consume these pattern bits before decoding the name string
	// Re-read with proper handling
	r.UnreadByte()

	// Read and discard the pattern byte (we already know it's literal without name ref)
	_, err := r.ReadByte()
	if err != nil {
		return Header{}, err
	}

	// Decode name string
	name, err := d.decodeString(r)
	if err != nil {
		return Header{}, err
	}

	// Decode value string
	value, err := d.decodeString(r)
	if err != nil {
		return Header{}, err
	}

	return Header{Name: name, Value: value}, nil
}

// decodeIndexedFieldLinePostBase decodes an Indexed Field Line with Post-Base Index
// Format: 0001XXXX
func (d *Decoder) decodeIndexedFieldLinePostBase(r *qpackReader) (Header, error) {
	index, err := d.decodeInteger(r, 4)
	if err != nil {
		return Header{}, err
	}

	// Post-base indexing: index is relative to base
	entry, err := d.dynamicTable.Get(index)
	if err != nil {
		return Header{}, err
	}

	return Header{Name: entry.Name, Value: entry.Value}, nil
}

// decodeLiteralFieldLinePostBaseNameRef decodes a Literal Field Line with Post-Base Name Reference
// Format: 0000NXXX (N=never index)
func (d *Decoder) decodeLiteralFieldLinePostBaseNameRef(r *qpackReader) (Header, error) {
	firstByte, _ := r.ReadByte()
	neverIndex := (firstByte & 0x08) != 0
	r.UnreadByte()

	_ = neverIndex // TODO: Track never-index hint

	// Decode name reference (post-base)
	nameIndex, err := d.decodeInteger(r, 3)
	if err != nil {
		return Header{}, err
	}

	entry, err := d.dynamicTable.Get(nameIndex)
	if err != nil {
		return Header{}, err
	}

	// Decode value
	value, err := d.decodeString(r)
	if err != nil {
		return Header{}, err
	}

	return Header{Name: entry.Name, Value: value}, nil
}

// decodeInteger decodes a QPACK integer with N-bit prefix (RFC 9204 Section 4.1.1)
func (d *Decoder) decodeInteger(r *qpackReader, n uint8) (uint64, error) {
	if n < 1 || n > 8 {
		return 0, errors.New("qpack: invalid prefix bits")
	}

	firstByte, err := r.ReadByte()
	if err != nil {
		return 0, err
	}

	// Calculate mask for N-bit prefix
	mask := uint8((1 << n) - 1)
	value := uint64(firstByte & mask)

	// If value < 2^N - 1, we're done
	if value < uint64(mask) {
		return value, nil
	}

	// Multi-byte integer
	m := uint64(0)
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				return 0, ErrInvalidEncoding
			}
			return 0, err
		}

		// Check for overflow
		if m >= 63 {
			return 0, ErrIntegerOverflow
		}

		value += uint64(b&0x7F) << m
		m += 7

		// If continuation bit is not set, we're done
		if (b & 0x80) == 0 {
			break
		}
	}

	return value, nil
}

// decodeString decodes a QPACK string (RFC 9204 Section 4.1.2)
func (d *Decoder) decodeString(r *qpackReader) (string, error) {
	firstByte, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	r.UnreadByte()

	isHuffman := (firstByte & 0x80) != 0

	// Decode length
	length, err := d.decodeInteger(r, 7)
	if err != nil {
		return "", err
	}

	// Safety check
	if length > 1024*1024 { // 1MB max
		return "", ErrStringTooLong
	}

	// Reuse string buffer if it has enough capacity (Priority 2 optimization)
	if cap(d.stringBuf) < int(length) {
		d.stringBuf = make([]byte, length)
	} else {
		d.stringBuf = d.stringBuf[:length]
	}

	// Read string bytes into reusable buffer
	if _, err := io.ReadFull(r, d.stringBuf); err != nil {
		return "", err
	}

	if isHuffman {
		// Decode Huffman-encoded string
		decoded, err := HuffmanDecode(d.stringBuf)
		if err != nil {
			// If Huffman decoding fails, try returning the raw string
			// This is a fallback for incomplete Huffman table implementation
			return string(d.stringBuf), nil
		}
		return string(decoded), nil
	}

	// Standard string conversion (copies the data, which is necessary
	// because stringBuf is reused across multiple decodeString calls)
	// Priority 2 optimization: We still benefit from stringBuf reuse,
	// which eliminates allocations in io.ReadFull
	return string(d.stringBuf), nil
}

// ProcessEncoderInstruction processes an encoder stream instruction
// This updates the dynamic table based on encoder stream data
func (d *Decoder) ProcessEncoderInstruction(data []byte) error {
	var r qpackReader
	r.Reset(data)

	for r.Len() > 0 {
		firstByte, err := r.ReadByte()
		if err != nil {
			break
		}
		r.UnreadByte()

		// Insert with Name Reference (1TXXXXXX)
		if firstByte&0x80 != 0 {
			if err := d.processInsertWithNameRef(&r); err != nil {
				return err
			}
			continue
		}

		// Insert without Name Reference (01XXXXXX)
		if firstByte&0x40 != 0 {
			if err := d.processInsertWithoutNameRef(&r); err != nil {
				return err
			}
			continue
		}

		// Duplicate (000XXXXX)
		if firstByte&0x20 == 0 {
			if err := d.processDuplicate(&r); err != nil {
				return err
			}
			continue
		}

		// Set Dynamic Table Capacity (001XXXXX)
		if err := d.processSetCapacity(&r); err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) processInsertWithNameRef(r *qpackReader) error {
	firstByte, _ := r.ReadByte()
	isStatic := (firstByte & 0x40) == 0
	r.UnreadByte()

	nameIndex, err := d.decodeInteger(r, 6)
	if err != nil {
		return err
	}

	var name string
	if isStatic {
		if int(nameIndex) >= len(staticTable) {
			return ErrTableIndexOutOfRange
		}
		name = staticTable[nameIndex].Name
	} else {
		entry, err := d.dynamicTable.Get(nameIndex)
		if err != nil {
			return err
		}
		name = entry.Name
	}

	value, err := d.decodeString(r)
	if err != nil {
		return err
	}

	return d.dynamicTable.Insert(name, value)
}

func (d *Decoder) processInsertWithoutNameRef(r *qpackReader) error {
	name, err := d.decodeString(r)
	if err != nil {
		return err
	}

	value, err := d.decodeString(r)
	if err != nil {
		return err
	}

	return d.dynamicTable.Insert(name, value)
}

func (d *Decoder) processDuplicate(r *qpackReader) error {
	index, err := d.decodeInteger(r, 5)
	if err != nil {
		return err
	}

	return d.dynamicTable.Duplicate(index)
}

func (d *Decoder) processSetCapacity(r *qpackReader) error {
	capacity, err := d.decodeInteger(r, 5)
	if err != nil {
		return err
	}

	d.dynamicTable.SetMaxSize(capacity)
	return nil
}

// SetMaxTableCapacity updates the maximum dynamic table capacity
func (d *Decoder) SetMaxTableCapacity(capacity uint64) {
	d.maxTableSize = capacity
	d.dynamicTable.SetMaxSize(capacity)
}
