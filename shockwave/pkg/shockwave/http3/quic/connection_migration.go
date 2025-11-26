package quic

import (
	"crypto/rand"
	"errors"
	"net"
	"sync"
	"time"
)

// Connection Migration (RFC 9000 Section 9)
// Allows a connection to migrate to a new network path

var (
	ErrMigrationDisabled  = errors.New("quic: connection migration is disabled")
	ErrPathValidationFailed = errors.New("quic: path validation failed")
	ErrNoValidPath        = errors.New("quic: no valid path available")
)

// PathState represents the state of a network path
type PathState uint8

const (
	PathStateUnknown PathState = iota
	PathStateValidating
	PathStateValidated
	PathStateFailed
)

func (p PathState) String() string {
	switch p {
	case PathStateUnknown:
		return "Unknown"
	case PathStateValidating:
		return "Validating"
	case PathStateValidated:
		return "Validated"
	case PathStateFailed:
		return "Failed"
	default:
		return "Invalid"
	}
}

// NetworkPath represents a network path (local addr + remote addr)
type NetworkPath struct {
	LocalAddr  net.Addr
	RemoteAddr net.Addr
	State      PathState

	// Path validation
	Challenge      []byte
	ChallengeSent  time.Time
	ResponseReceived bool

	// Statistics
	RTT            time.Duration
	PacketsSent    uint64
	PacketsReceived uint64
	BytesSent      uint64
	BytesReceived  uint64

	// Timestamps
	LastUsed       time.Time
	ValidatedAt    time.Time
}

// ConnectionMigration manages connection migration and path validation
type ConnectionMigration struct {
	conn *Connection

	// Migration settings
	enabled                 bool
	disableActiveMigration  bool

	// Paths
	mu                 sync.RWMutex
	currentPath        *NetworkPath
	alternatePaths     map[string]*NetworkPath // Key: "local|remote"

	// Path validation challenges
	pendingChallenges  map[string]*NetworkPath // Key: challenge data
	challengesMu       sync.RWMutex

	// Connection IDs for migration
	availableConnIDs   []ConnectionID
	connIDsMu          sync.RWMutex
}

// NewConnectionMigration creates a new connection migration manager
func NewConnectionMigration(conn *Connection) *ConnectionMigration {
	return &ConnectionMigration{
		conn:              conn,
		enabled:           true, // Enabled by default
		alternatePaths:    make(map[string]*NetworkPath),
		pendingChallenges: make(map[string]*NetworkPath),
		availableConnIDs:  make([]ConnectionID, 0),
	}
}

// SetEnabled enables or disables connection migration
func (cm *ConnectionMigration) SetEnabled(enabled bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.enabled = enabled
}

// IsEnabled returns whether connection migration is enabled
func (cm *ConnectionMigration) IsEnabled() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.enabled
}

// SetCurrentPath sets the current network path
func (cm *ConnectionMigration) SetCurrentPath(localAddr, remoteAddr net.Addr) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.currentPath = &NetworkPath{
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		State:      PathStateValidated, // Initial path is considered validated
		LastUsed:   time.Now(),
		ValidatedAt: time.Now(),
	}
}

// GetCurrentPath returns the current network path
func (cm *ConnectionMigration) GetCurrentPath() *NetworkPath {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentPath
}

// DetectPathChange detects if the peer has migrated to a new path
func (cm *ConnectionMigration) DetectPathChange(remoteAddr net.Addr) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.currentPath == nil {
		return false
	}

	// Check if remote address has changed
	return cm.currentPath.RemoteAddr.String() != remoteAddr.String()
}

// InitiatePathValidation initiates path validation for a new path
func (cm *ConnectionMigration) InitiatePathValidation(localAddr, remoteAddr net.Addr) ([]byte, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.enabled {
		return nil, ErrMigrationDisabled
	}

	// Generate PATH_CHALLENGE data
	challenge := make([]byte, 8)
	if _, err := rand.Read(challenge); err != nil {
		return nil, err
	}

	// Create new path
	pathKey := pathKey(localAddr, remoteAddr)
	path := &NetworkPath{
		LocalAddr:     localAddr,
		RemoteAddr:    remoteAddr,
		State:         PathStateValidating,
		Challenge:     challenge,
		ChallengeSent: time.Now(),
		LastUsed:      time.Now(),
	}

	cm.alternatePaths[pathKey] = path

	// Track challenge
	cm.challengesMu.Lock()
	cm.pendingChallenges[string(challenge)] = path
	cm.challengesMu.Unlock()

	return challenge, nil
}

// ValidatePathResponse validates a PATH_RESPONSE frame
func (cm *ConnectionMigration) ValidatePathResponse(response []byte, remoteAddr net.Addr) error {
	cm.challengesMu.Lock()
	path, exists := cm.pendingChallenges[string(response)]
	if !exists {
		cm.challengesMu.Unlock()
		return errors.New("quic: unknown PATH_RESPONSE")
	}
	delete(cm.pendingChallenges, string(response))
	cm.challengesMu.Unlock()

	// Mark path as validated
	cm.mu.Lock()
	defer cm.mu.Unlock()

	path.State = PathStateValidated
	path.ResponseReceived = true
	path.ValidatedAt = time.Now()

	// Calculate RTT
	path.RTT = time.Since(path.ChallengeSent)

	return nil
}

// MigratePath migrates the connection to a new validated path
func (cm *ConnectionMigration) MigratePath(localAddr, remoteAddr net.Addr) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.enabled {
		return ErrMigrationDisabled
	}

	pathK := pathKey(localAddr, remoteAddr)
	path, exists := cm.alternatePaths[pathK]

	if !exists {
		return errors.New("quic: path not found")
	}

	if path.State != PathStateValidated {
		return ErrPathValidationFailed
	}

	// Perform migration
	oldPath := cm.currentPath
	cm.currentPath = path

	// Update connection addresses
	cm.conn.remoteAddr = remoteAddr

	// Move old path to alternates if still valid
	if oldPath != nil {
		oldKey := pathKey(oldPath.LocalAddr, oldPath.RemoteAddr)
		cm.alternatePaths[oldKey] = oldPath
	}

	// Remove from alternates
	delete(cm.alternatePaths, pathK)

	return nil
}

// HandlePathChallenge handles a received PATH_CHALLENGE frame
// Returns PATH_RESPONSE data to send back
func (cm *ConnectionMigration) HandlePathChallenge(challenge []byte) []byte {
	// Simply echo back the challenge data as response
	response := make([]byte, len(challenge))
	copy(response, challenge)
	return response
}

// ProbeTimeout returns the timeout for path validation
func (cm *ConnectionMigration) ProbeTimeout() time.Duration {
	// Use 3x PTO (Probe Timeout)
	// For now, return a fixed value
	return 3 * time.Second
}

// CheckPathValidationTimeouts checks for path validation timeouts
func (cm *ConnectionMigration) CheckPathValidationTimeouts() []*NetworkPath {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	timeout := cm.ProbeTimeout()
	timedOut := make([]*NetworkPath, 0)

	for pathK, path := range cm.alternatePaths {
		if path.State == PathStateValidating {
			if time.Since(path.ChallengeSent) > timeout {
				// Path validation timed out
				path.State = PathStateFailed

				// Remove from pending challenges
				cm.challengesMu.Lock()
				delete(cm.pendingChallenges, string(path.Challenge))
				cm.challengesMu.Unlock()

				timedOut = append(timedOut, path)

				// Remove failed path
				delete(cm.alternatePaths, pathK)
			}
		}
	}

	return timedOut
}

// GetAlternatePaths returns all alternate paths
func (cm *ConnectionMigration) GetAlternatePaths() []*NetworkPath {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	paths := make([]*NetworkPath, 0, len(cm.alternatePaths))
	for _, path := range cm.alternatePaths {
		paths = append(paths, path)
	}

	return paths
}

// RecordPacketSent records a packet sent on the current path
func (cm *ConnectionMigration) RecordPacketSent(bytes uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.currentPath != nil {
		cm.currentPath.PacketsSent++
		cm.currentPath.BytesSent += bytes
		cm.currentPath.LastUsed = time.Now()
	}
}

// RecordPacketReceived records a packet received on the current path
func (cm *ConnectionMigration) RecordPacketReceived(bytes uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.currentPath != nil {
		cm.currentPath.PacketsReceived++
		cm.currentPath.BytesReceived += bytes
		cm.currentPath.LastUsed = time.Now()
	}
}

// GenerateNewConnectionID generates a new connection ID for migration
func (cm *ConnectionMigration) GenerateNewConnectionID() (ConnectionID, error) {
	// Generate random connection ID
	id := make([]byte, 8) // 8-byte connection ID
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}

	connID := ConnectionID(id)

	// Add to available connection IDs
	cm.connIDsMu.Lock()
	cm.availableConnIDs = append(cm.availableConnIDs, connID)
	cm.connIDsMu.Unlock()

	return connID, nil
}

// RetireConnectionID retires a connection ID
func (cm *ConnectionMigration) RetireConnectionID(connID ConnectionID) {
	cm.connIDsMu.Lock()
	defer cm.connIDsMu.Unlock()

	// Remove from available IDs
	for i, id := range cm.availableConnIDs {
		if string(id) == string(connID) {
			cm.availableConnIDs = append(cm.availableConnIDs[:i], cm.availableConnIDs[i+1:]...)
			break
		}
	}
}

// GetAvailableConnectionIDs returns available connection IDs
func (cm *ConnectionMigration) GetAvailableConnectionIDs() []ConnectionID {
	cm.connIDsMu.RLock()
	defer cm.connIDsMu.RUnlock()

	ids := make([]ConnectionID, len(cm.availableConnIDs))
	copy(ids, cm.availableConnIDs)
	return ids
}

// pathKey generates a unique key for a path
func pathKey(localAddr, remoteAddr net.Addr) string {
	return localAddr.String() + "|" + remoteAddr.String()
}

// UpdateRTT updates the RTT estimate for the current path
func (cm *ConnectionMigration) UpdateRTT(rtt time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.currentPath != nil {
		cm.currentPath.RTT = rtt
	}
}

// GetPathStatistics returns statistics for the current path
func (cm *ConnectionMigration) GetPathStatistics() (sent, received uint64, rtt time.Duration) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.currentPath != nil {
		return cm.currentPath.PacketsSent, cm.currentPath.PacketsReceived, cm.currentPath.RTT
	}

	return 0, 0, 0
}

// CanMigrate returns whether migration is currently possible
func (cm *ConnectionMigration) CanMigrate() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.enabled || cm.disableActiveMigration {
		return false
	}

	// Check if there are validated alternate paths
	for _, path := range cm.alternatePaths {
		if path.State == PathStateValidated {
			return true
		}
	}

	return false
}

// SelectBestPath selects the best validated alternate path
func (cm *ConnectionMigration) SelectBestPath() *NetworkPath {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var bestPath *NetworkPath
	var bestRTT time.Duration

	for _, path := range cm.alternatePaths {
		if path.State == PathStateValidated {
			if bestPath == nil || path.RTT < bestRTT {
				bestPath = path
				bestRTT = path.RTT
			}
		}
	}

	return bestPath
}
