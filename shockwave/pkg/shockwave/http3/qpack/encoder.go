package qpack

import (
	"bytes"
)

// QPACK Encoder (RFC 9204)
// Encodes HTTP headers using QPACK compression

// Encoder encodes HTTP headers using QPACK
type Encoder struct {
	dynamicTable *DynamicTable
	maxTableSize uint64

	// Reusable buffer for encoding (Priority 1 optimization)
	buf *bytes.Buffer
}

// NewEncoder creates a new QPACK encoder
func NewEncoder(maxTableSize uint64) *Encoder {
	return &Encoder{
		dynamicTable: NewDynamicTable(maxTableSize),
		maxTableSize: maxTableSize,
		buf:          bytes.NewBuffer(make([]byte, 0, 512)), // Pre-allocate 512 bytes
	}
}

// EncodeHeaders encodes a list of headers
// Returns (encodedData, encoderStreamInstructions, error)
func (enc *Encoder) EncodeHeaders(headers []Header) ([]byte, []byte, error) {
	// Reuse buffer (Priority 1 optimization)
	enc.buf.Reset()

	// Encoded Field Section Prefix (RFC 9204 Section 4.5.1)
	// Required Insert Count = 0 (no dynamic table references for now)
	// Delta Base = 0
	requiredInsertCount := uint8(0)
	deltaBase := uint8(0)

	enc.buf.WriteByte(requiredInsertCount)
	enc.buf.WriteByte(deltaBase)

	// Encode each header
	for _, header := range headers {
		if err := enc.encodeHeader(enc.buf, header); err != nil {
			return nil, nil, err
		}
	}

	// Return copy (required for safety, but reuse buffer for next call)
	result := make([]byte, enc.buf.Len())
	copy(result, enc.buf.Bytes())
	return result, nil, nil
}

// Header represents an HTTP header name-value pair
type Header struct {
	Name  string
	Value string
}

// encodeHeader encodes a single header field
func (enc *Encoder) encodeHeader(buf *bytes.Buffer, header Header) error {
	// Try static table first
	staticIndex, exactMatch := FindStaticIndex(header.Name, header.Value)

	if exactMatch {
		// Indexed Field Line (static table, exact match)
		// RFC 9204 Section 4.5.2: 1T format (T=1 for static)
		enc.writeIndexedFieldLine(buf, uint64(staticIndex), true)
		return nil
	}

	if staticIndex != -1 {
		// Literal Field Line with Name Reference (static table)
		// RFC 9204 Section 4.5.4: 01NT format (T=1 for static, N=0 for no Huffman)
		enc.writeLiteralFieldLineWithNameRef(buf, uint64(staticIndex), header.Value, true, false)
		return nil
	}

	// Try dynamic table
	dynamicIndex, exactMatch := enc.dynamicTable.Find(header.Name, header.Value)

	if exactMatch {
		// Indexed Field Line (dynamic table)
		enc.writeIndexedFieldLine(buf, dynamicIndex, false)
		return nil
	}

	if dynamicIndex != 0 {
		// Literal Field Line with Name Reference (dynamic table)
		enc.writeLiteralFieldLineWithNameRef(buf, dynamicIndex, header.Value, false, false)
		return nil
	}

	// Literal Field Line without Name Reference
	// RFC 9204 Section 4.5.6: 001NT format
	enc.writeLiteralFieldLineWithoutNameRef(buf, header.Name, header.Value, false)

	return nil
}

// writeIndexedFieldLine writes an Indexed Field Line
// RFC 9204 Section 4.5.2: 1T followed by index
func (enc *Encoder) writeIndexedFieldLine(buf *bytes.Buffer, index uint64, isStatic bool) {
	// Format: 1T IIIIII...
	// T=0 for static, T=1 for dynamic
	var prefix byte = 0x80 // 10000000 (static)
	if !isStatic {
		prefix |= 0x40 // 11000000 (dynamic)
	}

	// Write with 6-bit prefix
	enc.writeInteger(buf, index, prefix, 6)
}

// writeLiteralFieldLineWithNameRef writes a Literal Field Line with Name Reference
// RFC 9204 Section 4.5.4: 01NT followed by index and value
func (enc *Encoder) writeLiteralFieldLineWithNameRef(buf *bytes.Buffer, index uint64, value string, isStatic bool, huffman bool) {
	// Format: 01NT IIIIII... H VVVVVV...
	// N=0 (never index hint), T=0 for static, T=1 for dynamic
	var prefix byte = 0x40 // 01000000 (static)
	if !isStatic {
		prefix |= 0x10 // 01010000 (dynamic)
	}

	// Write index with 4-bit prefix
	enc.writeInteger(buf, index, prefix, 4)

	// Write value
	enc.writeString(buf, value, huffman)
}

// writeLiteralFieldLineWithoutNameRef writes a Literal Field Line without Name Reference
// RFC 9204 Section 4.5.6: 001NT followed by name and value
func (enc *Encoder) writeLiteralFieldLineWithoutNameRef(buf *bytes.Buffer, name string, value string, huffman bool) {
	// Format: 001NT NNNNN... H VVVVVV...
	var prefix byte = 0x20 // 00100000

	// Write prefix
	buf.WriteByte(prefix)

	// Write name
	enc.writeString(buf, name, huffman)

	// Write value
	enc.writeString(buf, value, huffman)
}

// writeInteger writes an integer with N-bit prefix encoding
// RFC 9204 Section 4.1.1 (uses QUIC variable-length integers)
func (enc *Encoder) writeInteger(buf *bytes.Buffer, value uint64, prefix byte, prefixBits int) {
	maxValue := uint64((1 << prefixBits) - 1)

	if value < maxValue {
		// Fits in prefix
		buf.WriteByte(prefix | byte(value))
	} else {
		// Doesn't fit in prefix, use variable-length encoding
		buf.WriteByte(prefix | byte(maxValue))
		value -= maxValue

		// Write variable-length integer
		for value >= 128 {
			buf.WriteByte(byte((value & 0x7F) | 0x80))
			value >>= 7
		}
		buf.WriteByte(byte(value))
	}
}

// writeString writes a string (with optional Huffman encoding)
// RFC 9204 Section 4.1.2
func (enc *Encoder) writeString(buf *bytes.Buffer, s string, huffman bool) {
	if huffman {
		// TODO: Implement Huffman encoding
		// For now, fall back to literal
		huffman = false
	}

	// Format: H LLLLLL... SSSSSS...
	// H=1 for Huffman, H=0 for literal
	var prefix byte = 0x00
	if huffman {
		prefix = 0x80
	}

	// Write length with 7-bit prefix
	length := uint64(len(s))
	enc.writeInteger(buf, length, prefix, 7)

	// Write string data
	buf.WriteString(s)
}

// SetDynamicTableCapacity updates the dynamic table capacity
func (enc *Encoder) SetDynamicTableCapacity(capacity uint64) {
	enc.dynamicTable.SetMaxSize(capacity)
	enc.maxTableSize = capacity
}

// GetDynamicTable returns the encoder's dynamic table (for testing)
func (enc *Encoder) GetDynamicTable() *DynamicTable {
	return enc.dynamicTable
}

// InsertWithNameRef inserts an entry with a name reference
// This generates an encoder stream instruction
func (enc *Encoder) InsertWithNameRef(index uint64, value string, isStatic bool) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Format: 1T IIIIII... H VVVVV...
	var prefix byte = 0x80
	if isStatic {
		prefix |= 0x40
	}

	enc.writeInteger(buf, index, prefix, 6)
	enc.writeString(buf, value, false)

	// Also insert into dynamic table
	var name string
	if isStatic {
		entry, ok := GetStaticEntry(int(index))
		if !ok {
			return nil, ErrInvalidIndex
		}
		name = entry.Name
	} else {
		entry, err := enc.dynamicTable.Get(index)
		if err != nil {
			return nil, err
		}
		name = entry.Name
	}

	if err := enc.dynamicTable.Insert(name, value); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// InsertWithoutNameRef inserts an entry without a name reference
func (enc *Encoder) InsertWithoutNameRef(name, value string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Format: 01NH NNNNN... H VVVVV...
	var prefix byte = 0x40
	enc.writeInteger(buf, 0, prefix, 5)

	enc.writeString(buf, name, false)
	enc.writeString(buf, value, false)

	// Insert into dynamic table
	if err := enc.dynamicTable.Insert(name, value); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Duplicate duplicates an existing dynamic table entry
func (enc *Encoder) Duplicate(index uint64) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Format: 000 IIIII...
	var prefix byte = 0x00
	enc.writeInteger(buf, index, prefix, 5)

	// Duplicate in dynamic table
	if err := enc.dynamicTable.Duplicate(index); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// SetDynamicTableCapacityInstruction creates a Set Dynamic Table Capacity instruction
func (enc *Encoder) SetDynamicTableCapacityInstruction(capacity uint64) []byte {
	buf := bytes.NewBuffer(nil)

	// Format: 001 CCCCC...
	var prefix byte = 0x20
	enc.writeInteger(buf, capacity, prefix, 5)

	return buf.Bytes()
}

// EncodeSimpleHeaders is a simplified header encoding for basic use cases
func EncodeSimpleHeaders(headers map[string]string) ([]byte, error) {
	enc := NewEncoder(4096) // 4KB dynamic table

	headerList := make([]Header, 0, len(headers))
	for name, value := range headers {
		headerList = append(headerList, Header{Name: name, Value: value})
	}

	encoded, _, err := enc.EncodeHeaders(headerList)
	return encoded, err
}
