package qpack

import (
	"fmt"
	"testing"
)

func TestHuffmanTreeBuild(t *testing.T) {
	if huffmanRoot == nil {
		t.Fatal("Huffman root is nil")
	}

	// Test a few simple codes
	testCodes := []struct {
		sym   byte
		code  uint32
		nbits uint8
	}{
		{' ', 0x14, 6},
		{'a', 0x3, 5},
		{'0', 0x0, 5},
		{'1', 0x1, 5},
	}

	for _, tc := range testCodes {
		t.Logf("Testing symbol %d ('%c'): code=0x%x, nbits=%d", tc.sym, tc.sym, tc.code, tc.nbits)

		// Verify encoding
		code, nbits := getHuffmanCode(tc.sym)
		if code != tc.code || nbits != tc.nbits {
			t.Errorf("Symbol %d: expected code=0x%x nbits=%d, got code=0x%x nbits=%d",
				tc.sym, tc.code, tc.nbits, code, nbits)
		}
	}
}

func TestHuffmanSimpleDecode(t *testing.T) {
	// Test single character: 'a' = 0x3, 5 bits = 00011
	// Padded to byte: 00011111 = 0x1F
	encoded := []byte{0x1F}

	decoded, err := HuffmanDecode(encoded)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if len(decoded) != 1 || decoded[0] != 'a' {
		t.Errorf("Expected 'a', got %v", decoded)
	}

	t.Logf("Successfully decoded 'a'")
}

func TestHuffmanEncodeDecodeChar(t *testing.T) {
	testChars := []byte{'a', 'e', 'i', 'o', 's', 't', ' ', '0', '1'}

	for _, ch := range testChars {
		input := []byte{ch}
		encoded := HuffmanEncode(input)

		t.Logf("Character '%c' (0x%02x): encoded to %d bytes: %08b", ch, ch, len(encoded), encoded)

		decoded, err := HuffmanDecode(encoded)
		if err != nil {
			t.Errorf("Decode error for '%c': %v", ch, err)
			continue
		}

		if len(decoded) != 1 || decoded[0] != ch {
			t.Errorf("Character '%c': expected [%c], got %v", ch, ch, decoded)
		}
	}
}

func TestHuffmanBitPattern(t *testing.T) {
	// Test 'test' which should be:
	// 't' = 01001 (5 bits)
	// 'e' = 00101 (5 bits)
	// 's' = 01000 (5 bits)
	// 't' = 01001 (5 bits)
	// Total: 20 bits = 01001 00101 01000 01001
	// Padded: 01001001 01010000 10011111 (3 bytes)

	input := []byte("test")
	encoded := HuffmanEncode(input)

	t.Logf("'test' encoded to %d bytes:", len(encoded))
	for i, b := range encoded {
		t.Logf("  Byte %d: 0x%02x = %08b", i, b, b)
	}

	decoded, err := HuffmanDecode(encoded)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if string(decoded) != "test" {
		t.Errorf("Expected 'test', got '%s'", string(decoded))
	}
}

func TestHuffmanPadding(t *testing.T) {
	// Test that padding (all 1s) is handled correctly
	// EOS symbol is 256, with code 0x3fffffff (30 bits, all 1s)

	// Create various padding scenarios
	tests := []struct {
		name  string
		input []byte
	}{
		{"5-bit char", []byte{'a'}},  // 5 bits, 3 bits padding
		{"10-bit", []byte{'!'}},      // 10 bits, 6 bits padding
		{"15-bit", []byte{'<'}},      // 15 bits, 1 bit padding
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := HuffmanEncode(tt.input)

			t.Logf("Input: %q, Encoded bytes:", tt.input)
			for i, b := range encoded {
				t.Logf("  [%d]: 0x%02x = %08b", i, b, b)
			}

			// Check last byte padding
			lastByte := encoded[len(encoded)-1]
			t.Logf("Last byte: 0x%02x = %08b", lastByte, lastByte)

			decoded, err := HuffmanDecode(encoded)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}

			if string(decoded) != string(tt.input) {
				t.Errorf("Expected %q, got %q", tt.input, decoded)
			}
		})
	}
}

func printHuffmanTree(node *huffmanNode, prefix string, depth int) {
	if depth > 10 {
		return // Prevent excessive output
	}

	if node == nil {
		return
	}

	if node.symbol >= 0 {
		char := ""
		if node.symbol < 256 && node.symbol >= 32 && node.symbol < 127 {
			char = fmt.Sprintf(" '%c'", byte(node.symbol))
		}
		fmt.Printf("%sLeaf: symbol=%d%s\n", prefix, node.symbol, char)
		return
	}

	if node.children[0] != nil {
		fmt.Printf("%s0->\n", prefix)
		printHuffmanTree(node.children[0], prefix+"  ", depth+1)
	}
	if node.children[1] != nil {
		fmt.Printf("%s1->\n", prefix)
		printHuffmanTree(node.children[1], prefix+"  ", depth+1)
	}
}

func TestPrintHuffmanTree(t *testing.T) {
	if testing.Verbose() {
		t.Log("Huffman tree structure (first 3 levels):")
		printHuffmanTree(huffmanRoot, "", 0)
	}
}
