package qpack

// Huffman encoding/decoding for QPACK (RFC 7541 Appendix B)
// Uses the same Huffman table as HPACK

import (
	"bytes"
	"errors"
	"io"
)

var (
	ErrInvalidHuffmanCode = errors.New("qpack: invalid Huffman code")
	ErrHuffmanEOS         = errors.New("qpack: unexpected EOS in Huffman data")
)

// Full Huffman code table is in huffman_table.go (RFC 7541 Appendix B)
// Contains all 257 entries (0-255 symbols + EOS)

// Huffman decoding tree node
type huffmanNode struct {
	children [2]*huffmanNode // [0] for 0 bit, [1] for 1 bit
	symbol   int              // -1 for internal nodes, 0-255 for leaf nodes
}

var huffmanRoot *huffmanNode

func init() {
	// Build Huffman decoding tree
	huffmanRoot = buildHuffmanTree()
}

// buildHuffmanTree constructs the decoding tree from the code table
func buildHuffmanTree() *huffmanNode {
	root := &huffmanNode{symbol: -1}

	// Build tree from all 257 codes (0-256)
	for sym := 0; sym < 257; sym++ {
		code := huffmanTable[sym]

		// Traverse tree, creating nodes as needed
		node := root
		for i := int(code.nbits) - 1; i >= 0; i-- {
			bit := (code.code >> uint(i)) & 1

			if node.children[bit] == nil {
				node.children[bit] = &huffmanNode{symbol: -1}
			}

			node = node.children[bit]
		}

		// Mark leaf with symbol
		node.symbol = sym
	}

	return root
}

// HuffmanEncode encodes data using Huffman coding
// Returns the encoded bytes
func HuffmanEncode(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	// Estimate output size (worst case: 30 bits per byte = ~4 bytes per input byte)
	out := make([]byte, 0, len(data)*4)
	var bits uint64  // Accumulator for bits
	var nbits uint8  // Number of bits in accumulator

	for _, b := range data {
		// Find Huffman code for this byte
		code, codelen := getHuffmanCode(b)

		// Add code to bit accumulator
		bits = (bits << codelen) | uint64(code)
		nbits += codelen

		// Write complete bytes
		for nbits >= 8 {
			nbits -= 8
			out = append(out, byte(bits>>nbits))
			bits &= (1 << nbits) - 1
		}
	}

	// Pad remaining bits with 1s (EOS pattern)
	if nbits > 0 {
		bits = (bits << (8 - nbits)) | ((1 << (8 - nbits)) - 1)
		out = append(out, byte(bits))
	}

	return out
}

// HuffmanDecode decodes Huffman-encoded data
// Returns the decoded bytes
func HuffmanDecode(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	out := bytes.NewBuffer(make([]byte, 0, len(data)*2))
	node := huffmanRoot

	for byteIdx, b := range data {
		for i := 7; i >= 0; i-- {
			bit := (b >> uint(i)) & 1

			// Traverse tree
			if node.children[bit] == nil {
				// No child - check if this is valid padding
				// Padding must be all 1s and only at the end
				if byteIdx == len(data)-1 {
					// This is the last byte, check remaining bits are all 1s
					mask := byte((1 << uint(i+1)) - 1)
					remaining := b & mask
					if remaining == mask {
						// Valid padding
						return out.Bytes(), nil
					}
				}
				// Invalid code or padding in wrong place
				return nil, ErrInvalidHuffmanCode
			}

			node = node.children[bit]

			// Check if we reached a leaf
			if node.symbol >= 0 {
				if node.symbol == 256 {
					// EOS symbol - we're done
					return out.Bytes(), nil
				}
				// Output the decoded symbol
				out.WriteByte(byte(node.symbol))
				// Reset to root for next symbol
				node = huffmanRoot
			}
		}
	}

	// After processing all bits, check if we're at root
	if node == huffmanRoot {
		return out.Bytes(), nil
	}

	// We're not at root - this means we have incomplete bits
	// This is valid only if we're on the path to EOS (all remaining bits would be 1s)
	// Check if following all 1s from here leads to EOS
	for node != nil && node.symbol < 0 {
		if node.children[1] == nil {
			// Can't continue with 1s - invalid padding
			return nil, ErrInvalidHuffmanCode
		}
		node = node.children[1]
	}

	// We should have reached EOS (256)
	if node == nil || node.symbol != 256 {
		return nil, ErrInvalidHuffmanCode
	}

	// Valid padding path to EOS
	return out.Bytes(), nil
}

// getHuffmanCode returns the Huffman code and bit length for a byte
func getHuffmanCode(b byte) (code uint32, nbits uint8) {
	return getHuffmanCodeFromTable(b)
}

// HuffmanEncodeString encodes a string using Huffman coding
func HuffmanEncodeString(s string) []byte {
	return HuffmanEncode([]byte(s))
}

// HuffmanDecodeString decodes Huffman-encoded data to a string
func HuffmanDecodeString(data []byte) (string, error) {
	decoded, err := HuffmanDecode(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// HuffmanEncodedLength estimates the encoded length for data
// Used to decide whether Huffman encoding is beneficial
func HuffmanEncodedLength(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	totalBits := 0
	for _, b := range data {
		_, nbits := getHuffmanCode(b)
		totalBits += int(nbits)
	}

	// Round up to nearest byte
	return (totalBits + 7) / 8
}

// ShouldHuffmanEncode determines if Huffman encoding would be beneficial
func ShouldHuffmanEncode(data []byte) bool {
	if len(data) < 10 {
		// For very short strings, overhead isn't worth it
		return false
	}

	encodedLen := HuffmanEncodedLength(data)
	return encodedLen < len(data)
}

// HuffmanEncodeWriter writes Huffman-encoded data to an io.Writer
type HuffmanEncodeWriter struct {
	w     io.Writer
	bits  uint64
	nbits uint8
	err   error
}

// NewHuffmanEncodeWriter creates a new Huffman encoding writer
func NewHuffmanEncodeWriter(w io.Writer) *HuffmanEncodeWriter {
	return &HuffmanEncodeWriter{w: w}
}

// WriteByte writes a single byte (Huffman-encoded)
func (w *HuffmanEncodeWriter) WriteByte(b byte) error {
	if w.err != nil {
		return w.err
	}

	code, codelen := getHuffmanCode(b)
	w.bits = (w.bits << codelen) | uint64(code)
	w.nbits += codelen

	// Flush complete bytes
	for w.nbits >= 8 {
		w.nbits -= 8
		if _, err := w.w.Write([]byte{byte(w.bits >> w.nbits)}); err != nil {
			w.err = err
			return err
		}
		w.bits &= (1 << w.nbits) - 1
	}

	return nil
}

// Write writes data (Huffman-encoded)
func (w *HuffmanEncodeWriter) Write(data []byte) (int, error) {
	for _, b := range data {
		if err := w.WriteByte(b); err != nil {
			return 0, err
		}
	}
	return len(data), nil
}

// Flush writes any remaining bits (padded with 1s)
func (w *HuffmanEncodeWriter) Flush() error {
	if w.err != nil {
		return w.err
	}

	if w.nbits > 0 {
		// Pad with 1s
		w.bits = (w.bits << (8 - w.nbits)) | ((1 << (8 - w.nbits)) - 1)
		if _, err := w.w.Write([]byte{byte(w.bits)}); err != nil {
			w.err = err
			return err
		}
		w.nbits = 0
	}

	return nil
}
