package quic

import (
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
)

// TLS 1.3 Handshake Integration for QUIC (RFC 9001)
// Maps TLS records to QUIC CRYPTO frames and manages encryption levels

var (
	ErrHandshakeNotComplete = errors.New("quic: TLS handshake not complete")
	ErrHandshakeFailed      = errors.New("quic: TLS handshake failed")
)

// TLSHandler manages the TLS 1.3 handshake for a QUIC connection
type TLSHandler struct {
	conn      *Connection
	tlsConn   *TLSConn
	tlsState  *tls.Conn
	config    *tls.Config
	isClient  bool
	isServer  bool

	// Handshake state
	handshakeMu       sync.Mutex
	handshakeComplete bool
	handshakeErr      error
	handshakeChan     chan struct{}

	// Encryption levels and keys
	levelMu           sync.RWMutex
	currentReadLevel  EncryptionLevel
	currentWriteLevel EncryptionLevel

	// Keys for each encryption level
	initialKeys     *CryptoKeys
	handshakeKeys   *CryptoKeys
	applicationKeys *CryptoKeys
	zeroRTTKeys     *CryptoKeys

	// Transport parameters
	localParams  *TransportParameters
	remoteParams *TransportParameters
	paramsMu     sync.RWMutex

	// Session resumption
	sessionCache  *SessionCache
	sessionTicket *SessionTicket
}

// NewTLSHandler creates a new TLS handler for client connections
func NewTLSHandler(conn *Connection, config *tls.Config, isClient bool) (*TLSHandler, error) {
	if config == nil {
		return nil, errors.New("quic: TLS config is required")
	}

	th := &TLSHandler{
		conn:              conn,
		config:            conn.TLSConfig(config),
		isClient:          isClient,
		isServer:          !isClient,
		currentReadLevel:  EncryptionLevelInitial,
		currentWriteLevel: EncryptionLevelInitial,
		handshakeChan:     make(chan struct{}),
		localParams:       conn.localParams,
	}

	// Create TLS connection wrapper
	th.tlsConn = NewTLSConn(conn)

	// Generate initial keys
	if err := th.generateInitialKeys(); err != nil {
		return nil, err
	}

	return th, nil
}

// Start initiates the TLS handshake
func (th *TLSHandler) Start() error {
	th.handshakeMu.Lock()
	if th.handshakeComplete {
		th.handshakeMu.Unlock()
		return nil
	}
	th.handshakeMu.Unlock()

	// Create TLS connection
	if th.isClient {
		th.tlsState = tls.Client(th.tlsConn, th.config)
	} else {
		th.tlsState = tls.Server(th.tlsConn, th.config)
	}

	// Run handshake in background
	go th.runHandshake()

	return nil
}

// runHandshake performs the TLS handshake
func (th *TLSHandler) runHandshake() {
	defer close(th.handshakeChan)

	// Perform TLS handshake
	err := th.tlsState.Handshake()

	th.handshakeMu.Lock()
	defer th.handshakeMu.Unlock()

	if err != nil {
		th.handshakeErr = fmt.Errorf("TLS handshake failed: %w", err)
		th.handshakeComplete = false
		return
	}

	// Extract connection state
	state := th.tlsState.ConnectionState()

	// Verify TLS 1.3
	if state.Version != tls.VersionTLS13 {
		th.handshakeErr = errors.New("QUIC requires TLS 1.3")
		return
	}

	// Derive application keys
	if err := th.deriveApplicationKeys(&state); err != nil {
		th.handshakeErr = fmt.Errorf("failed to derive application keys: %w", err)
		return
	}

	// Mark handshake complete
	th.handshakeComplete = true

	// Update connection state
	th.conn.handshakeComplete = true
}

// WaitForHandshake waits for the handshake to complete
func (th *TLSHandler) WaitForHandshake() error {
	<-th.handshakeChan

	th.handshakeMu.Lock()
	defer th.handshakeMu.Unlock()

	if th.handshakeErr != nil {
		return th.handshakeErr
	}

	if !th.handshakeComplete {
		return ErrHandshakeNotComplete
	}

	return nil
}

// generateInitialKeys generates the initial encryption keys
func (th *TLSHandler) generateInitialKeys() error {
	// Initial keys are derived from the destination connection ID
	keys, err := NewInitialKeys(th.conn.destConnID, th.isClient)
	if err != nil {
		return fmt.Errorf("failed to generate initial keys: %w", err)
	}

	th.initialKeys = keys
	th.conn.initialKeys = keys

	return nil
}

// deriveHandshakeKeys derives handshake keys from TLS secrets
func (th *TLSHandler) deriveHandshakeKeys(secret []byte, cipherSuite uint16) error {
	keys, err := deriveKeys(secret, EncryptionLevelHandshake, cipherSuite)
	if err != nil {
		return err
	}

	th.levelMu.Lock()
	th.handshakeKeys = keys
	th.conn.handshakeKeys = keys
	th.levelMu.Unlock()

	// Transition to handshake encryption level
	th.SetReadLevel(EncryptionLevelHandshake)
	th.SetWriteLevel(EncryptionLevelHandshake)

	return nil
}

// deriveApplicationKeys derives application keys from TLS connection state
func (th *TLSHandler) deriveApplicationKeys(state *tls.ConnectionState) error {
	// In production, we would extract the actual traffic secrets from TLS
	// For now, derive from cipher suite
	cipherSuite := state.CipherSuite

	// Generate application keys
	// NOTE: In real implementation, extract secrets from state.ExportKeyingMaterial
	// or use a custom TLS record layer to intercept secrets
	secret := make([]byte, 32) // Placeholder - would be real secret
	keys, err := deriveKeys(secret, EncryptionLevelApplication, cipherSuite)
	if err != nil {
		return err
	}

	th.levelMu.Lock()
	th.applicationKeys = keys
	th.conn.applicationKeys = keys
	th.levelMu.Unlock()

	// Transition to application encryption level
	th.SetReadLevel(EncryptionLevelApplication)
	th.SetWriteLevel(EncryptionLevelApplication)

	return nil
}

// HandleCryptoFrame processes an incoming CRYPTO frame
func (th *TLSHandler) HandleCryptoFrame(frame *CryptoFrame, level EncryptionLevel) error {
	// Pass to TLS connection wrapper
	return th.tlsConn.HandleCryptoFrame(frame, level)
}

// SetReadLevel sets the current read encryption level
func (th *TLSHandler) SetReadLevel(level EncryptionLevel) {
	th.levelMu.Lock()
	defer th.levelMu.Unlock()

	th.currentReadLevel = level
	th.tlsConn.SetReadLevel(level)
}

// SetWriteLevel sets the current write encryption level
func (th *TLSHandler) SetWriteLevel(level EncryptionLevel) {
	th.levelMu.Lock()
	defer th.levelMu.Unlock()

	th.currentWriteLevel = level
	th.tlsConn.SetWriteLevel(level)
}

// GetReadLevel returns the current read encryption level
func (th *TLSHandler) GetReadLevel() EncryptionLevel {
	th.levelMu.RLock()
	defer th.levelMu.RUnlock()
	return th.currentReadLevel
}

// GetWriteLevel returns the current write encryption level
func (th *TLSHandler) GetWriteLevel() EncryptionLevel {
	th.levelMu.RLock()
	defer th.levelMu.RUnlock()
	return th.currentWriteLevel
}

// GetKeys returns the keys for a specific encryption level
func (th *TLSHandler) GetKeys(level EncryptionLevel) *CryptoKeys {
	th.levelMu.RLock()
	defer th.levelMu.RUnlock()

	switch level {
	case EncryptionLevelInitial:
		return th.initialKeys
	case EncryptionLevelEarlyData:
		return th.zeroRTTKeys
	case EncryptionLevelHandshake:
		return th.handshakeKeys
	case EncryptionLevelApplication:
		return th.applicationKeys
	default:
		return nil
	}
}

// SetTransportParameters sets the local transport parameters
func (th *TLSHandler) SetTransportParameters(params *TransportParameters) {
	th.paramsMu.Lock()
	defer th.paramsMu.Unlock()
	th.localParams = params
}

// GetTransportParameters returns the negotiated transport parameters
func (th *TLSHandler) GetTransportParameters() *TransportParameters {
	th.paramsMu.RLock()
	defer th.paramsMu.RUnlock()
	return th.remoteParams
}

// HandleTransportParameters processes received transport parameters
func (th *TLSHandler) HandleTransportParameters(params *TransportParameters) error {
	th.paramsMu.Lock()
	defer th.paramsMu.Unlock()

	// Validate parameters
	if params.MaxUDPPayloadSize < 1200 {
		return errors.New("invalid max_udp_payload_size")
	}

	th.remoteParams = params
	return nil
}

// IsHandshakeComplete returns whether the handshake is complete
func (th *TLSHandler) IsHandshakeComplete() bool {
	th.handshakeMu.Lock()
	defer th.handshakeMu.Unlock()
	return th.handshakeComplete
}

// ConnectionState returns the TLS connection state
func (th *TLSHandler) ConnectionState() tls.ConnectionState {
	if th.tlsState == nil {
		return tls.ConnectionState{}
	}
	return th.tlsState.ConnectionState()
}

// Close closes the TLS handler
func (th *TLSHandler) Close() error {
	if th.tlsConn != nil {
		return th.tlsConn.Close()
	}
	return nil
}

// Enable0RTT enables 0-RTT with a session ticket
func (th *TLSHandler) Enable0RTT(ticket *SessionTicket) error {
	if !th.isClient {
		return errors.New("0-RTT can only be enabled on client")
	}

	// Derive 0-RTT keys
	keys, err := deriveKeys(ticket.EarlyTrafficSecret, EncryptionLevelEarlyData, ticket.CipherSuite)
	if err != nil {
		return err
	}

	th.levelMu.Lock()
	th.zeroRTTKeys = keys
	th.conn.zeroRTTKeys = keys
	th.levelMu.Unlock()

	th.sessionTicket = ticket

	return nil
}

// SupportsEarlyData returns whether 0-RTT is supported
func (th *TLSHandler) SupportsEarlyData() bool {
	return th.zeroRTTKeys != nil
}
