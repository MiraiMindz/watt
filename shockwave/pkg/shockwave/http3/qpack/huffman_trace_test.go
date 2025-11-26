package qpack

import (
	"testing"
)

func TestTraceDecodeChar(t *testing.T) {
	// Test decoding 'a' = 0x1F = 00011111
	// 'a' code is 0x3 = 00011 (5 bits)
	// Padding: 111 (3 bits)

	data := []byte{0x1F}

	t.Logf("Decoding byte 0x%02x = %08b", data[0], data[0])

	node := huffmanRoot
	for i := 7; i >= 0; i-- {
		bit := (data[0] >> uint(i)) & 1
		t.Logf("  Bit %d: %d, node.symbol=%d, has[0]=%v, has[1]=%v",
			i, bit, node.symbol,
			node.children[0] != nil, node.children[1] != nil)

		if node.children[bit] == nil {
			t.Logf("    No child[%d]! This should be padding.", bit)
			mask := byte((1 << uint(i+1)) - 1)
			remaining := data[0] & mask
			t.Logf("    Remaining bits (mask 0x%02x): 0x%02x, expected: 0x%02x",
				mask, remaining, mask)
			break
		}

		node = node.children[bit]

		if node.symbol >= 0 {
			t.Logf("    Reached leaf! Symbol = %d ('%c')", node.symbol, byte(node.symbol))
			node = huffmanRoot
		}
	}
}

func TestEOSSymbol(t *testing.T) {
	// Check if EOS (symbol 256) is in the tree
	code := huffmanTable[256]
	t.Logf("EOS symbol (256): code=0x%x, nbits=%d", code.code, code.nbits)

	// Try to traverse to EOS
	node := huffmanRoot
	for i := int(code.nbits) - 1; i >= 0; i-- {
		bit := (code.code >> uint(i)) & 1
		if node.children[bit] == nil {
			t.Fatalf("Tree incomplete at bit %d for EOS", i)
		}
		node = node.children[bit]
	}

	if node.symbol != 256 {
		t.Errorf("Expected symbol 256, got %d", node.symbol)
	} else {
		t.Log("EOS symbol found in tree correctly")
	}
}

func TestPaddingPath(t *testing.T) {
	// Test what happens if we follow all 1s from root
	// This should eventually lead to EOS (256)

	node := huffmanRoot
	path := ""

	for i := 0; i < 30; i++ { // EOS is 30 bits
		path += "1"
		if node.children[1] == nil {
			t.Logf("No child[1] after %d bits (path: ...%s)", i+1, path[max(0, len(path)-10):])
			break
		}
		node = node.children[1]
		if node.symbol >= 0 {
			t.Logf("Reached symbol %d after %d bits", node.symbol, i+1)
			break
		}
	}

	if node.symbol == 256 {
		t.Log("Following all 1s leads to EOS correctly")
	} else {
		t.Errorf("Following all 1s did not lead to EOS, got symbol %d", node.symbol)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func TestDecodeWithTrace(t *testing.T) {
	// Let's try decoding a known good sequence
	// Encode 'a', then decode with full tracing

	encoded := HuffmanEncode([]byte{'a'})
	t.Logf("Encoded 'a': %d bytes: % 08b", len(encoded), encoded)

	// Now manually decode
	out := []byte{}
	node := huffmanRoot

	for byteIdx, b := range encoded {
		t.Logf("Processing byte %d: 0x%02x = %08b", byteIdx, b, b)

		for i := 7; i >= 0; i-- {
			bit := (b >> uint(i)) & 1

			if node == huffmanRoot {
				t.Logf("  [%d] At root, bit=%d", i, bit)
			} else {
				t.Logf("  [%d] bit=%d", i, bit)
			}

			if node.children[bit] == nil {
				t.Logf("  [%d] No child[%d]", i, bit)
				// Check padding
				if byteIdx == len(encoded)-1 {
					mask := byte((1 << uint(i+1)) - 1)
					remaining := b & mask
					t.Logf("  [%d] Last byte padding check: remaining=0x%02x, mask=0x%02x", i, remaining, mask)
					if remaining == mask {
						t.Log("  Valid padding!")
						goto done
					}
				}
				t.Fatal("  Invalid code!")
			}

			node = node.children[bit]

			if node.symbol >= 0 {
				if node.symbol == 256 {
					t.Log("  Reached EOS")
					goto done
				}
				t.Logf("  [%d] Decoded symbol: %d ('%c')", i, node.symbol, byte(node.symbol))
				out = append(out, byte(node.symbol))
				node = huffmanRoot
			}
		}
	}

done:
	t.Logf("Decoded: %q", out)
	if string(out) != "a" {
		t.Errorf("Expected 'a', got %q", out)
	}
}
