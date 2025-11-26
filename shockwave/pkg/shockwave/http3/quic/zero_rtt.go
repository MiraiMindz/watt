package quic

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync"
	"time"
)

// 0-RTT Early Data Support (RFC 9001 Section 4.6)
// Allows sending application data in the first flight to reduce latency

var (
	ErrNoSessionTicket   = errors.New("quic: no session ticket available")
	ErrEarlyDataRejected = errors.New("quic: early data was rejected by server")
	ErrReplayDetected    = errors.New("quic: potential replay attack detected")
)

// SessionTicket represents a TLS 1.3 session ticket for 0-RTT
type SessionTicket struct {
	// Ticket data from TLS session
	Ticket []byte

	// Cryptographic state
	CipherSuite         uint16
	EarlySecret         []byte
	EarlyTrafficSecret  []byte

	// Transport parameters from previous connection
	TransportParams     *TransportParameters

	// Session metadata
	ServerName          string
	ReceivedAt          time.Time
	MaxEarlyDataSize    uint32

	// Replay protection
	ObfuscatedTicketAge uint32
}

// SessionCache manages session tickets for 0-RTT
type SessionCache struct {
	mu      sync.RWMutex
	tickets map[string]*SessionTicket // Key: server name
	maxSize int
}

// NewSessionCache creates a new session cache
func NewSessionCache(maxSize int) *SessionCache {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &SessionCache{
		tickets: make(map[string]*SessionTicket),
		maxSize: maxSize,
	}
}

// Put stores a session ticket
func (sc *SessionCache) Put(serverName string, ticket *SessionTicket) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Evict old tickets if cache is full
	if len(sc.tickets) >= sc.maxSize {
		// Simple FIFO eviction - find oldest
		var oldest string
		var oldestTime time.Time
		first := true

		for name, t := range sc.tickets {
			if first || t.ReceivedAt.Before(oldestTime) {
				oldest = name
				oldestTime = t.ReceivedAt
				first = false
			}
		}

		delete(sc.tickets, oldest)
	}

	sc.tickets[serverName] = ticket
}

// Get retrieves a session ticket
func (sc *SessionCache) Get(serverName string) (*SessionTicket, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	ticket, ok := sc.tickets[serverName]
	if !ok {
		return nil, ErrNoSessionTicket
	}

	// Check if ticket is expired (typically 7 days)
	if time.Since(ticket.ReceivedAt) > 7*24*time.Hour {
		return nil, ErrNoSessionTicket
	}

	// Check if early data is allowed
	if ticket.MaxEarlyDataSize == 0 {
		return nil, errors.New("quic: early data not supported for this ticket")
	}

	return ticket, nil
}

// Remove removes a session ticket (e.g., after it's rejected)
func (sc *SessionCache) Remove(serverName string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.tickets, serverName)
}

// ZeroRTTHandler manages 0-RTT state for a connection
type ZeroRTTHandler struct {
	conn *Connection

	// 0-RTT state
	enabled            bool
	earlyDataKeys      *CryptoKeys
	sessionTicket      *SessionTicket
	earlyDataSent      uint64
	earlyDataAccepted  bool
	earlyDataRejected  bool

	// Replay protection
	clientHelloRandom  []byte
	antiReplayWindow   *antiReplayWindow

	mu sync.RWMutex
}

// NewZeroRTTHandler creates a new 0-RTT handler
func NewZeroRTTHandler(conn *Connection) *ZeroRTTHandler {
	return &ZeroRTTHandler{
		conn:             conn,
		antiReplayWindow: newAntiReplayWindow(1000), // Track last 1000 ClientHellos
	}
}

// Enable0RTT enables 0-RTT for the connection using a session ticket
func (z *ZeroRTTHandler) Enable0RTT(ticket *SessionTicket) error {
	z.mu.Lock()
	defer z.mu.Unlock()

	if ticket == nil {
		return ErrNoSessionTicket
	}

	z.enabled = true
	z.sessionTicket = ticket

	// Derive early data keys from the ticket
	keys, err := deriveKeys(ticket.EarlyTrafficSecret, EncryptionLevelEarlyData, ticket.CipherSuite)
	if err != nil {
		return err
	}

	z.earlyDataKeys = keys
	z.conn.zeroRTTKeys = keys

	return nil
}

// CanSendEarlyData returns true if early data can be sent
func (z *ZeroRTTHandler) CanSendEarlyData() bool {
	z.mu.RLock()
	defer z.mu.RUnlock()

	if !z.enabled || z.earlyDataRejected {
		return false
	}

	if z.sessionTicket == nil {
		return false
	}

	// Check if we haven't exceeded the early data limit
	return z.earlyDataSent < uint64(z.sessionTicket.MaxEarlyDataSize)
}

// RecordEarlyDataSent records the amount of early data sent
func (z *ZeroRTTHandler) RecordEarlyDataSent(bytes uint64) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.earlyDataSent += bytes
}

// AcceptEarlyData marks early data as accepted by the server
func (z *ZeroRTTHandler) AcceptEarlyData() {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.earlyDataAccepted = true
}

// RejectEarlyData marks early data as rejected by the server
func (z *ZeroRTTHandler) RejectEarlyData() {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.earlyDataRejected = true
	z.earlyDataKeys = nil
	z.conn.zeroRTTKeys = nil
}

// IsEarlyDataAccepted returns true if early data was accepted
func (z *ZeroRTTHandler) IsEarlyDataAccepted() bool {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.earlyDataAccepted
}

// IsEarlyDataRejected returns true if early data was rejected
func (z *ZeroRTTHandler) IsEarlyDataRejected() bool {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.earlyDataRejected
}

// GetSessionTicket returns the session ticket if available
func (z *ZeroRTTHandler) GetSessionTicket() *SessionTicket {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.sessionTicket
}

// GetTransportParameters returns cached transport parameters for 0-RTT
func (z *ZeroRTTHandler) GetTransportParameters() *TransportParameters {
	z.mu.RLock()
	defer z.mu.RUnlock()

	if z.sessionTicket == nil {
		return nil
	}

	return z.sessionTicket.TransportParams
}

// ValidateClientHello validates a ClientHello for replay protection
// Server-side only
func (z *ZeroRTTHandler) ValidateClientHello(random []byte) error {
	z.mu.Lock()
	defer z.mu.Unlock()

	// Check anti-replay window
	if z.antiReplayWindow.Contains(random) {
		return ErrReplayDetected
	}

	z.antiReplayWindow.Add(random)
	z.clientHelloRandom = random

	return nil
}

// antiReplayWindow implements a sliding window for replay protection
type antiReplayWindow struct {
	mu      sync.RWMutex
	window  map[string]time.Time
	maxSize int
}

func newAntiReplayWindow(maxSize int) *antiReplayWindow {
	return &antiReplayWindow{
		window:  make(map[string]time.Time),
		maxSize: maxSize,
	}
}

func (w *antiReplayWindow) Contains(random []byte) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	key := string(random)
	_, exists := w.window[key]
	return exists
}

func (w *antiReplayWindow) Add(random []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Evict old entries if window is full
	if len(w.window) >= w.maxSize {
		// Find oldest entry
		var oldest string
		var oldestTime time.Time
		first := true

		for k, t := range w.window {
			if first || t.Before(oldestTime) {
				oldest = k
				oldestTime = t
				first = false
			}
		}

		delete(w.window, oldest)
	}

	key := string(random)
	w.window[key] = time.Now()
}

// CreateSessionTicket creates a session ticket from a completed connection
// Server-side only
func CreateSessionTicket(conn *Connection) (*SessionTicket, error) {
	// Generate random ticket data
	ticketData := make([]byte, 32)
	if _, err := rand.Read(ticketData); err != nil {
		return nil, err
	}

	// Calculate obfuscated ticket age for replay protection
	ageAdd := make([]byte, 4)
	rand.Read(ageAdd)
	obfuscatedAge := binary.BigEndian.Uint32(ageAdd)

	ticket := &SessionTicket{
		Ticket:              ticketData,
		CipherSuite:         TLS_AES_128_GCM_SHA256, // Use connection's cipher suite
		TransportParams:     conn.localParams,
		ReceivedAt:          time.Now(),
		MaxEarlyDataSize:    0xFFFFFFFF, // 4GB max early data
		ObfuscatedTicketAge: obfuscatedAge,
	}

	return ticket, nil
}

// SendEarlyData sends application data in 0-RTT packets
// Client-side only
func (z *ZeroRTTHandler) SendEarlyData(data []byte) error {
	z.mu.Lock()
	defer z.mu.Unlock()

	if !z.enabled {
		return errors.New("quic: 0-RTT not enabled")
	}

	if z.earlyDataRejected {
		return ErrEarlyDataRejected
	}

	// Check early data limit
	if z.earlyDataSent+uint64(len(data)) > uint64(z.sessionTicket.MaxEarlyDataSize) {
		return errors.New("quic: early data size limit exceeded")
	}

	// Send data in 0-RTT packet
	// This would integrate with the connection's packet sending logic
	z.earlyDataSent += uint64(len(data))

	return nil
}

// Handle0RTTPacket processes a received 0-RTT packet
// Server-side only
func (z *ZeroRTTHandler) Handle0RTTPacket(packet *Packet) error {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.earlyDataRejected {
		return ErrEarlyDataRejected
	}

	// Decrypt packet with early data keys
	if z.earlyDataKeys == nil {
		return errors.New("quic: no early data keys available")
	}

	// Process packet...
	// This would integrate with the connection's packet processing logic

	return nil
}

// GetEarlyDataKeys returns the early data encryption keys
func (z *ZeroRTTHandler) GetEarlyDataKeys() *CryptoKeys {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.earlyDataKeys
}

// Clear clears the 0-RTT state
func (z *ZeroRTTHandler) Clear() {
	z.mu.Lock()
	defer z.mu.Unlock()

	z.enabled = false
	z.sessionTicket = nil
	z.earlyDataKeys = nil
	z.earlyDataSent = 0
	z.earlyDataAccepted = false
	z.earlyDataRejected = false
}
