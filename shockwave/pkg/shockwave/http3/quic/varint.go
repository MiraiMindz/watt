package quic

import (
	"errors"
	"io"
)

// Variable-length integer encoding as defined in RFC 9000 Section 16.
// QUIC uses a variable-length encoding for integers up to 2^62-1.
//
// Format (most significant 2 bits indicate length):
//   00: 6-bit value (0-63)
//   01: 14-bit value (0-16383)
//   10: 30-bit value (0-1073741823)
//   11: 62-bit value (0-4611686018427387903)

const (
	// Maximum values for each encoding length
	MaxVarint1 = 63                  // 2^6 - 1
	MaxVarint2 = 16383               // 2^14 - 1
	MaxVarint4 = 1073741823          // 2^30 - 1
	MaxVarint8 = 4611686018427387903 // 2^62 - 1
)

var (
	ErrVarintTooLarge = errors.New("quic: varint too large")
	ErrVarintTrunc    = errors.New("quic: varint truncated")
)

// varintLen returns the number of bytes needed to encode v.
func varintLen(v uint64) int {
	if v <= MaxVarint1 {
		return 1
	}
	if v <= MaxVarint2 {
		return 2
	}
	if v <= MaxVarint4 {
		return 4
	}
	if v <= MaxVarint8 {
		return 8
	}
	return -1 // Too large
}

// appendVarint appends a variable-length integer to buf.
// Returns the updated buffer or error if value too large.
func appendVarint(buf []byte, v uint64) ([]byte, error) {
	switch {
	case v <= MaxVarint1:
		return append(buf, byte(v)), nil
	case v <= MaxVarint2:
		return append(buf, byte(v>>8)|0x40, byte(v)), nil
	case v <= MaxVarint4:
		return append(buf,
			byte(v>>24)|0x80,
			byte(v>>16),
			byte(v>>8),
			byte(v),
		), nil
	case v <= MaxVarint8:
		return append(buf,
			byte(v>>56)|0xC0,
			byte(v>>48),
			byte(v>>40),
			byte(v>>32),
			byte(v>>24),
			byte(v>>16),
			byte(v>>8),
			byte(v),
		), nil
	default:
		return buf, ErrVarintTooLarge
	}
}

// readVarint reads a variable-length integer from r.
// Returns (value, bytesRead, error).
func readVarint(r io.Reader) (uint64, int, error) {
	var firstByte [1]byte
	if _, err := io.ReadFull(r, firstByte[:]); err != nil {
		return 0, 0, err
	}

	// Decode length from first 2 bits
	prefix := firstByte[0] >> 6

	switch prefix {
	case 0: // 1-byte encoding (6 bits)
		return uint64(firstByte[0] & 0x3F), 1, nil

	case 1: // 2-byte encoding (14 bits)
		var buf [1]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, 1, err
		}
		v := uint64(firstByte[0]&0x3F)<<8 | uint64(buf[0])
		return v, 2, nil

	case 2: // 4-byte encoding (30 bits)
		var buf [3]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, 1, err
		}
		v := uint64(firstByte[0]&0x3F)<<24 |
			uint64(buf[0])<<16 |
			uint64(buf[1])<<8 |
			uint64(buf[2])
		return v, 4, nil

	case 3: // 8-byte encoding (62 bits)
		var buf [7]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, 1, err
		}
		v := uint64(firstByte[0]&0x3F)<<56 |
			uint64(buf[0])<<48 |
			uint64(buf[1])<<40 |
			uint64(buf[2])<<32 |
			uint64(buf[3])<<24 |
			uint64(buf[4])<<16 |
			uint64(buf[5])<<8 |
			uint64(buf[6])
		return v, 8, nil
	}

	return 0, 0, ErrVarintTrunc
}

// parseVarint parses a variable-length integer from a byte slice.
// Returns (value, bytesConsumed, error).
func parseVarint(data []byte) (uint64, int, error) {
	if len(data) == 0 {
		return 0, 0, ErrVarintTrunc
	}

	prefix := data[0] >> 6

	switch prefix {
	case 0: // 1-byte encoding
		return uint64(data[0] & 0x3F), 1, nil

	case 1: // 2-byte encoding
		if len(data) < 2 {
			return 0, 0, ErrVarintTrunc
		}
		v := uint64(data[0]&0x3F)<<8 | uint64(data[1])
		return v, 2, nil

	case 2: // 4-byte encoding
		if len(data) < 4 {
			return 0, 0, ErrVarintTrunc
		}
		v := uint64(data[0]&0x3F)<<24 |
			uint64(data[1])<<16 |
			uint64(data[2])<<8 |
			uint64(data[3])
		return v, 4, nil

	case 3: // 8-byte encoding
		if len(data) < 8 {
			return 0, 0, ErrVarintTrunc
		}
		v := uint64(data[0]&0x3F)<<56 |
			uint64(data[1])<<48 |
			uint64(data[2])<<40 |
			uint64(data[3])<<32 |
			uint64(data[4])<<24 |
			uint64(data[5])<<16 |
			uint64(data[6])<<8 |
			uint64(data[7])
		return v, 8, nil
	}

	return 0, 0, ErrVarintTrunc
}

// writeVarint writes a variable-length integer to w.
func writeVarint(w io.Writer, v uint64) (int, error) {
	var buf [8]byte
	n := putVarint(buf[:], v)
	if n == 0 {
		return 0, ErrVarintTooLarge
	}
	return w.Write(buf[:n])
}

// putVarint encodes v into buf and returns the number of bytes written.
// Returns 0 if the value is too large.
func putVarint(buf []byte, v uint64) int {
	switch {
	case v <= MaxVarint1:
		buf[0] = byte(v)
		return 1
	case v <= MaxVarint2:
		buf[0] = byte(v>>8) | 0x40
		buf[1] = byte(v)
		return 2
	case v <= MaxVarint4:
		buf[0] = byte(v>>24) | 0x80
		buf[1] = byte(v >> 16)
		buf[2] = byte(v >> 8)
		buf[3] = byte(v)
		return 4
	case v <= MaxVarint8:
		buf[0] = byte(v>>56) | 0xC0
		buf[1] = byte(v >> 48)
		buf[2] = byte(v >> 40)
		buf[3] = byte(v >> 32)
		buf[4] = byte(v >> 24)
		buf[5] = byte(v >> 16)
		buf[6] = byte(v >> 8)
		buf[7] = byte(v)
		return 8
	default:
		return 0
	}
}

// ConnectionID represents a QUIC connection ID (0-20 bytes)
type ConnectionID []byte

// IsEmpty returns true if the connection ID is empty (0 bytes).
func (c ConnectionID) IsEmpty() bool {
	return len(c) == 0
}

// Equal returns true if two connection IDs are equal.
func (c ConnectionID) Equal(other ConnectionID) bool {
	if len(c) != len(other) {
		return false
	}
	for i := range c {
		if c[i] != other[i] {
			return false
		}
	}
	return true
}

// Len returns the length of the connection ID in bytes.
func (c ConnectionID) Len() int {
	return len(c)
}

// parseConnectionID parses a connection ID from data.
// The first byte indicates the length (0-20).
func parseConnectionID(data []byte) (ConnectionID, int, error) {
	if len(data) == 0 {
		return nil, 0, io.ErrUnexpectedEOF
	}

	cidLen := int(data[0])
	if cidLen > 20 {
		return nil, 0, errors.New("quic: connection ID too long")
	}

	if len(data) < 1+cidLen {
		return nil, 0, io.ErrUnexpectedEOF
	}

	cid := make([]byte, cidLen)
	copy(cid, data[1:1+cidLen])
	return ConnectionID(cid), 1 + cidLen, nil
}

// appendConnectionID appends a connection ID to buf with length prefix.
func appendConnectionID(buf []byte, cid ConnectionID) []byte {
	buf = append(buf, byte(len(cid)))
	return append(buf, cid...)
}
