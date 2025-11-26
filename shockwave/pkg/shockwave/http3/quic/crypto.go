package quic

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"hash"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// QUIC uses TLS 1.3 for cryptographic handshake (RFC 9001)
// This file implements QUIC-specific packet protection

// Encryption levels as defined in RFC 9001 Section 4.1.4
type EncryptionLevel uint8

const (
	EncryptionLevelInitial EncryptionLevel = iota
	EncryptionLevelEarlyData
	EncryptionLevelHandshake
	EncryptionLevelApplication
)

func (e EncryptionLevel) String() string {
	switch e {
	case EncryptionLevelInitial:
		return "Initial"
	case EncryptionLevelEarlyData:
		return "EarlyData"
	case EncryptionLevelHandshake:
		return "Handshake"
	case EncryptionLevelApplication:
		return "Application"
	default:
		return fmt.Sprintf("Unknown(%d)", e)
	}
}

// QUIC version 1 initial salt (RFC 9001 Section 5.2)
var initialSalt = []byte{
	0x38, 0x76, 0x2c, 0xf7, 0xf5, 0x59, 0x34, 0xb3,
	0x4d, 0x17, 0x9a, 0xe6, 0xa4, 0xc8, 0x0c, 0xad,
	0xcc, 0xbb, 0x7f, 0x0a,
}

// AEAD cipher suites
const (
	// TLS 1.3 cipher suites
	TLS_AES_128_GCM_SHA256       uint16 = 0x1301
	TLS_AES_256_GCM_SHA384       uint16 = 0x1302
	TLS_CHACHA20_POLY1305_SHA256 uint16 = 0x1303
)

var (
	ErrDecryptionFailed = errors.New("quic: decryption failed")
	ErrInvalidKeyLength = errors.New("quic: invalid key length")
)

// CryptoKeys holds the keys for packet protection at a specific encryption level
type CryptoKeys struct {
	Level      EncryptionLevel
	CipherSuite uint16

	// Keys
	Key []byte // AEAD key
	IV  []byte // AEAD IV
	HP  []byte // Header protection key

	// AEAD cipher
	aead cipher.AEAD
}

// NewInitialKeys derives initial keys from the destination connection ID.
// RFC 9001 Section 5.2
func NewInitialKeys(destConnID []byte, isClient bool) (*CryptoKeys, error) {
	// Extract initial secret using HKDF-Extract
	initialSecret := hkdf.Extract(sha256.New, destConnID, initialSalt)

	var label string
	if isClient {
		label = "client in"
	} else {
		label = "server in"
	}

	// Derive client/server initial secret
	secret := hkdfExpandLabel(sha256.New, initialSecret, label, nil, 32)

	return deriveKeys(secret, EncryptionLevelInitial, TLS_AES_128_GCM_SHA256)
}

// deriveKeys derives packet protection keys from a secret.
// RFC 9001 Section 5.1
func deriveKeys(secret []byte, level EncryptionLevel, cipherSuite uint16) (*CryptoKeys, error) {
	var keyLen, ivLen, hpLen int

	switch cipherSuite {
	case TLS_AES_128_GCM_SHA256:
		keyLen, ivLen, hpLen = 16, 12, 16
	case TLS_AES_256_GCM_SHA384:
		keyLen, ivLen, hpLen = 32, 12, 32
	case TLS_CHACHA20_POLY1305_SHA256:
		keyLen, ivLen, hpLen = 32, 12, 32
	default:
		return nil, fmt.Errorf("quic: unsupported cipher suite 0x%04x", cipherSuite)
	}

	// Derive keys using HKDF-Expand-Label
	key := hkdfExpandLabel(sha256.New, secret, "quic key", nil, keyLen)
	iv := hkdfExpandLabel(sha256.New, secret, "quic iv", nil, ivLen)
	hp := hkdfExpandLabel(sha256.New, secret, "quic hp", nil, hpLen)

	keys := &CryptoKeys{
		Level:       level,
		CipherSuite: cipherSuite,
		Key:         key,
		IV:          iv,
		HP:          hp,
	}

	// Create AEAD cipher
	var err error
	switch cipherSuite {
	case TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384:
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		keys.aead, err = cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
	case TLS_CHACHA20_POLY1305_SHA256:
		keys.aead, err = chacha20poly1305.New(key)
		if err != nil {
			return nil, err
		}
	}

	return keys, nil
}

// hkdfExpandLabel implements HKDF-Expand-Label from TLS 1.3
// RFC 8446 Section 7.1
func hkdfExpandLabel(hashFunc func() hash.Hash, secret []byte, label string, context []byte, length int) []byte {
	// HkdfLabel structure:
	//   uint16 length
	//   opaque label<7..255> = "tls13 " + Label
	//   opaque context<0..255> = Context

	fullLabel := "tls13 " + label
	hkdfLabel := make([]byte, 2+1+len(fullLabel)+1+len(context))

	// Length
	hkdfLabel[0] = byte(length >> 8)
	hkdfLabel[1] = byte(length)

	// Label
	hkdfLabel[2] = byte(len(fullLabel))
	copy(hkdfLabel[3:], fullLabel)

	// Context
	offset := 3 + len(fullLabel)
	hkdfLabel[offset] = byte(len(context))
	copy(hkdfLabel[offset+1:], context)

	// HKDF-Expand
	out := make([]byte, length)
	r := hkdf.Expand(hashFunc, secret, hkdfLabel)
	r.Read(out)

	return out
}

// ProtectPacket encrypts and protects a QUIC packet.
// RFC 9001 Section 5.4
func (k *CryptoKeys) ProtectPacket(packet *Packet) ([]byte, error) {
	if k.aead == nil {
		return nil, errors.New("quic: AEAD not initialized")
	}

	// Serialize packet header
	buf := packet.AppendTo(nil)

	// Find where packet number starts (header length - packet number length)
	pnOffset := len(buf) - packet.Header.PacketNumberLen - len(packet.Payload)

	// Construct nonce: IV XOR packet number
	nonce := make([]byte, len(k.IV))
	copy(nonce, k.IV)

	// XOR packet number into nonce (right-aligned)
	pn := packet.Header.PacketNumber
	for i := len(nonce) - 1; i >= len(nonce)-8 && pn > 0; i-- {
		nonce[i] ^= byte(pn)
		pn >>= 8
	}

	// Encrypt payload
	// AAD = packet header up to (and including) packet number
	aad := buf[:pnOffset+packet.Header.PacketNumberLen]

	// Replace plaintext payload with ciphertext
	ciphertext := k.aead.Seal(nil, nonce, packet.Payload, aad)
	buf = buf[:pnOffset+packet.Header.PacketNumberLen]
	buf = append(buf, ciphertext...)

	// Apply header protection
	buf = k.protectHeader(buf, pnOffset)

	return buf, nil
}

// UnprotectPacket decrypts and authenticates a QUIC packet.
// RFC 9001 Section 5.4
func (k *CryptoKeys) UnprotectPacket(data []byte, destConnIDLen int) (*Packet, error) {
	if k.aead == nil {
		return nil, errors.New("quic: AEAD not initialized")
	}

	// First, remove header protection to get packet number length
	data, pnOffset, pnLen, err := k.unprotectHeader(data, destConnIDLen)
	if err != nil {
		return nil, err
	}

	// Parse packet number
	pn := uint64(0)
	for i := 0; i < pnLen; i++ {
		pn = (pn << 8) | uint64(data[pnOffset+i])
	}

	// Construct nonce
	nonce := make([]byte, len(k.IV))
	copy(nonce, k.IV)

	// XOR packet number into nonce
	pnTemp := pn
	for i := len(nonce) - 1; i >= len(nonce)-8 && pnTemp > 0; i-- {
		nonce[i] ^= byte(pnTemp)
		pnTemp >>= 8
	}

	// AAD = header up to and including packet number
	aad := data[:pnOffset+pnLen]

	// Decrypt payload
	ciphertext := data[pnOffset+pnLen:]
	plaintext, err := k.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	// Parse packet (we need to reconstruct it)
	packet, _, err := ParsePacket(data[:pnOffset+pnLen])
	if err != nil {
		return nil, err
	}

	packet.Payload = plaintext
	packet.Header.PacketNumber = pn
	packet.Header.PacketNumberLen = pnLen

	return packet, nil
}

// protectHeader applies header protection to a packet.
// RFC 9001 Section 5.4.1
func (k *CryptoKeys) protectHeader(packet []byte, pnOffset int) []byte {
	// Sample starts 4 bytes after packet number
	sampleOffset := pnOffset + 4
	if sampleOffset+16 > len(packet) {
		return packet // Not enough data for header protection
	}

	sample := packet[sampleOffset : sampleOffset+16]

	// Generate mask using header protection key
	var mask []byte
	switch k.CipherSuite {
	case TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384:
		// AES-ECB encryption of sample
		block, _ := aes.NewCipher(k.HP)
		mask = make([]byte, 16)
		block.Encrypt(mask, sample)
	case TLS_CHACHA20_POLY1305_SHA256:
		// ChaCha20 with counter=0
		// This is simplified; real implementation would use ChaCha20
		mask = make([]byte, 5)
	}

	// Apply mask to first byte
	if packet[0]&0x80 != 0 {
		// Long header: mask bits 0-3
		packet[0] ^= mask[0] & 0x0F
	} else {
		// Short header: mask bits 0-4
		packet[0] ^= mask[0] & 0x1F
	}

	// Apply mask to packet number
	pnLen := int(packet[0]&0x03) + 1
	for i := 0; i < pnLen; i++ {
		packet[pnOffset+i] ^= mask[1+i]
	}

	return packet
}

// unprotectHeader removes header protection from a packet.
// RFC 9001 Section 5.4.2
func (k *CryptoKeys) unprotectHeader(packet []byte, destConnIDLen int) ([]byte, int, int, error) {
	// Estimate packet number offset
	// For Initial packets: 1 (flags) + 4 (version) + 1 (dcid len) + dcid + 1 (scid len) + scid + token len + token + length
	// This is complex, so we'll use a simplified approach

	// For now, assume we know where packet number is
	// In a real implementation, this would need to parse the header structure

	firstByte := packet[0]
	isLongHeader := (firstByte & 0x80) != 0

	var pnOffset int
	if isLongHeader {
		// Long header: estimate offset (simplified)
		offset := 1 + 4 // flags + version

		// DCID
		dcidLen := int(packet[offset])
		offset += 1 + dcidLen

		// SCID
		scidLen := int(packet[offset])
		offset += 1 + scidLen

		// For Initial: token length + token
		if (firstByte & 0x30) == 0x00 {
			tokenLen, n, _ := parseVarint(packet[offset:])
			offset += n + int(tokenLen)
		}

		// Length field
		_, n, _ := parseVarint(packet[offset:])
		offset += n

		pnOffset = offset
	} else {
		// Short header: 1 (flags) + destConnIDLen
		pnOffset = 1 + destConnIDLen
	}

	// Sample starts 4 bytes after packet number
	sampleOffset := pnOffset + 4
	if sampleOffset+16 > len(packet) {
		return nil, 0, 0, errors.New("quic: packet too short for header protection")
	}

	sample := packet[sampleOffset : sampleOffset+16]

	// Generate mask
	var mask []byte
	switch k.CipherSuite {
	case TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384:
		block, _ := aes.NewCipher(k.HP)
		mask = make([]byte, 16)
		block.Encrypt(mask, sample)
	case TLS_CHACHA20_POLY1305_SHA256:
		mask = make([]byte, 5)
	}

	// Remove mask from first byte
	data := make([]byte, len(packet))
	copy(data, packet)

	if isLongHeader {
		data[0] ^= mask[0] & 0x0F
	} else {
		data[0] ^= mask[0] & 0x1F
	}

	// Get packet number length from unmasked first byte
	pnLen := int(data[0]&0x03) + 1

	// Remove mask from packet number
	for i := 0; i < pnLen; i++ {
		data[pnOffset+i] ^= mask[1+i]
	}

	return data, pnOffset, pnLen, nil
}

// TLSConfig creates a TLS configuration for QUIC
func NewQUICTLSConfig(isClient bool) *tls.Config {
	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		NextProtos: []string{"h3"}, // HTTP/3 ALPN
	}

	if !isClient {
		// Server configuration
		config.ClientAuth = tls.NoClientCert
	}

	return config
}

// Transport parameters that need to be exchanged during handshake
type TransportParameters struct {
	// Connection limits
	MaxIdleTimeout                 uint64
	MaxUDPPayloadSize              uint64
	InitialMaxData                 uint64
	InitialMaxStreamDataBidiLocal  uint64
	InitialMaxStreamDataBidiRemote uint64
	InitialMaxStreamDataUni        uint64
	InitialMaxStreamsBidi          uint64
	InitialMaxStreamsUni           uint64

	// Other parameters
	AckDelayExponent               uint64
	MaxAckDelay                    uint64
	DisableActiveMigration         bool
	ActiveConnectionIDLimit        uint64
	InitialSourceConnectionID      []byte

	// 0-RTT support
	MaxEarlyDataSize               uint64
}

// Default transport parameters
func DefaultTransportParameters() *TransportParameters {
	return &TransportParameters{
		MaxIdleTimeout:                 30000, // 30 seconds
		MaxUDPPayloadSize:              1200,
		InitialMaxData:                 10 * 1024 * 1024, // 10 MB
		InitialMaxStreamDataBidiLocal:  1 * 1024 * 1024,  // 1 MB
		InitialMaxStreamDataBidiRemote: 1 * 1024 * 1024,  // 1 MB
		InitialMaxStreamDataUni:        1 * 1024 * 1024,  // 1 MB
		InitialMaxStreamsBidi:          100,
		InitialMaxStreamsUni:           100,
		AckDelayExponent:               3,
		MaxAckDelay:                    25, // 25ms
		ActiveConnectionIDLimit:        2,
	}
}
