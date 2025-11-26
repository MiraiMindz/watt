package quic

import (
	"net"
	"testing"
	"time"
)

func TestConnectionMigrationBasic(t *testing.T) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	// Test enabled by default
	if !cm.IsEnabled() {
		t.Error("Connection migration should be enabled by default")
	}

	// Test disable
	cm.SetEnabled(false)
	if cm.IsEnabled() {
		t.Error("Connection migration should be disabled")
	}

	cm.SetEnabled(true)

	t.Logf("✓ Basic connection migration working")
}

func TestPathValidation(t *testing.T) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	// Create mock addresses
	local1 := &net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 5000}
	remote1 := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 443}

	// Set initial path
	cm.SetCurrentPath(local1, remote1)

	currentPath := cm.GetCurrentPath()
	if currentPath == nil {
		t.Fatal("Current path should be set")
	}

	if currentPath.State != PathStateValidated {
		t.Error("Initial path should be validated")
	}

	// Initiate validation for new path
	local2 := &net.UDPAddr{IP: net.ParseIP("192.168.2.1"), Port: 5001}
	remote2 := &net.UDPAddr{IP: net.ParseIP("10.0.0.2"), Port: 443}

	challenge, err := cm.InitiatePathValidation(local2, remote2)
	if err != nil {
		t.Fatalf("Failed to initiate path validation: %v", err)
	}

	if len(challenge) != 8 {
		t.Errorf("Expected challenge length 8, got %d", len(challenge))
	}

	// Validate response
	err = cm.ValidatePathResponse(challenge, remote2)
	if err != nil {
		t.Fatalf("Failed to validate path response: %v", err)
	}

	// Check alternate paths
	altPaths := cm.GetAlternatePaths()
	if len(altPaths) == 0 {
		t.Error("Should have alternate paths")
	}

	var validatedPath *NetworkPath
	for _, p := range altPaths {
		if p.State == PathStateValidated {
			validatedPath = p
			break
		}
	}

	if validatedPath == nil {
		t.Fatal("Should have validated alternate path")
	}

	t.Logf("✓ Path validation working correctly (RTT: %v)", validatedPath.RTT)
}

func TestPathMigration(t *testing.T) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	// Setup initial path
	local1 := &net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 5000}
	remote1 := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 443}
	cm.SetCurrentPath(local1, remote1)

	// Validate new path
	local2 := &net.UDPAddr{IP: net.ParseIP("192.168.2.1"), Port: 5001}
	remote2 := &net.UDPAddr{IP: net.ParseIP("10.0.0.2"), Port: 443}

	challenge, _ := cm.InitiatePathValidation(local2, remote2)
	cm.ValidatePathResponse(challenge, remote2)

	// Migrate to new path
	err := cm.MigratePath(local2, remote2)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Check current path is now the new path
	currentPath := cm.GetCurrentPath()
	if currentPath.RemoteAddr.String() != remote2.String() {
		t.Errorf("Expected current path to be remote2, got %s", currentPath.RemoteAddr.String())
	}

	// Old path should be in alternates
	altPaths := cm.GetAlternatePaths()
	foundOldPath := false
	for _, p := range altPaths {
		if p.RemoteAddr.String() == remote1.String() {
			foundOldPath = true
			break
		}
	}

	if !foundOldPath {
		t.Error("Old path should be in alternate paths")
	}

	t.Logf("✓ Path migration working correctly")
}

func TestPathChallengeResponse(t *testing.T) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	// Generate challenge
	challenge := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	// Handle challenge (should echo back)
	response := cm.HandlePathChallenge(challenge)

	if len(response) != len(challenge) {
		t.Errorf("Response length mismatch: expected %d, got %d", len(challenge), len(response))
	}

	for i := range challenge {
		if response[i] != challenge[i] {
			t.Errorf("Response byte %d mismatch: expected %d, got %d", i, challenge[i], response[i])
		}
	}

	t.Logf("✓ PATH_CHALLENGE/RESPONSE working correctly")
}

func TestPathValidationTimeout(t *testing.T) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	// Create path with old challenge time
	local := &net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 5000}
	remote := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 443}

	challenge, _ := cm.InitiatePathValidation(local, remote)

	// Manually set challenge time to the past
	cm.mu.Lock()
	pathKey := pathKey(local, remote)
	if path, exists := cm.alternatePaths[pathKey]; exists {
		path.ChallengeSent = time.Now().Add(-10 * time.Second)
	}
	cm.mu.Unlock()

	// Check timeouts
	timedOut := cm.CheckPathValidationTimeouts()

	if len(timedOut) == 0 {
		t.Error("Path should have timed out")
	}

	// Verify challenge is removed
	err := cm.ValidatePathResponse(challenge, remote)
	if err == nil {
		t.Error("Should fail to validate timed-out path")
	}

	t.Logf("✓ Path validation timeout working correctly")
}

func TestPathStatistics(t *testing.T) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	local := &net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 5000}
	remote := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 443}
	cm.SetCurrentPath(local, remote)

	// Record packets
	cm.RecordPacketSent(1200)
	cm.RecordPacketSent(1200)
	cm.RecordPacketReceived(800)

	sent, received, _ := cm.GetPathStatistics()

	if sent != 2 {
		t.Errorf("Expected 2 packets sent, got %d", sent)
	}

	if received != 1 {
		t.Errorf("Expected 1 packet received, got %d", received)
	}

	// Check bytes
	currentPath := cm.GetCurrentPath()
	if currentPath.BytesSent != 2400 {
		t.Errorf("Expected 2400 bytes sent, got %d", currentPath.BytesSent)
	}

	if currentPath.BytesReceived != 800 {
		t.Errorf("Expected 800 bytes received, got %d", currentPath.BytesReceived)
	}

	t.Logf("✓ Path statistics working correctly")
}

func TestConnectionIDManagement(t *testing.T) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	// Generate new connection ID
	connID1, err := cm.GenerateNewConnectionID()
	if err != nil {
		t.Fatalf("Failed to generate connection ID: %v", err)
	}

	if len(connID1) != 8 {
		t.Errorf("Expected connection ID length 8, got %d", len(connID1))
	}

	// Generate another
	connID2, err := cm.GenerateNewConnectionID()
	if err != nil {
		t.Fatalf("Failed to generate connection ID: %v", err)
	}

	// Should have 2 available IDs
	availableIDs := cm.GetAvailableConnectionIDs()
	if len(availableIDs) != 2 {
		t.Errorf("Expected 2 available IDs, got %d", len(availableIDs))
	}

	// Retire first ID
	cm.RetireConnectionID(connID1)

	availableIDs = cm.GetAvailableConnectionIDs()
	if len(availableIDs) != 1 {
		t.Errorf("Expected 1 available ID after retirement, got %d", len(availableIDs))
	}

	if string(availableIDs[0]) != string(connID2) {
		t.Error("Remaining ID should be connID2")
	}

	t.Logf("✓ Connection ID management working correctly")
}

func TestPathSelection(t *testing.T) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	// Set initial path
	local1 := &net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 5000}
	remote1 := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 443}
	cm.SetCurrentPath(local1, remote1)

	// Create multiple validated paths with different RTTs
	paths := []struct {
		local  *net.UDPAddr
		remote *net.UDPAddr
		rtt    time.Duration
	}{
		{
			local:  &net.UDPAddr{IP: net.ParseIP("192.168.2.1"), Port: 5001},
			remote: &net.UDPAddr{IP: net.ParseIP("10.0.0.2"), Port: 443},
			rtt:    50 * time.Millisecond,
		},
		{
			local:  &net.UDPAddr{IP: net.ParseIP("192.168.3.1"), Port: 5002},
			remote: &net.UDPAddr{IP: net.ParseIP("10.0.0.3"), Port: 443},
			rtt:    30 * time.Millisecond, // Best path
		},
		{
			local:  &net.UDPAddr{IP: net.ParseIP("192.168.4.1"), Port: 5003},
			remote: &net.UDPAddr{IP: net.ParseIP("10.0.0.4"), Port: 443},
			rtt:    70 * time.Millisecond,
		},
	}

	for _, p := range paths {
		challenge, _ := cm.InitiatePathValidation(p.local, p.remote)
		cm.ValidatePathResponse(challenge, p.remote)

		// Set RTT manually for testing
		pathKey := pathKey(p.local, p.remote)
		cm.mu.Lock()
		if path, exists := cm.alternatePaths[pathKey]; exists {
			path.RTT = p.rtt
		}
		cm.mu.Unlock()
	}

	// Select best path (lowest RTT)
	bestPath := cm.SelectBestPath()
	if bestPath == nil {
		t.Fatal("Should have selected a best path")
	}

	if bestPath.RTT != 30*time.Millisecond {
		t.Errorf("Expected best path RTT to be 30ms, got %v", bestPath.RTT)
	}

	t.Logf("✓ Path selection working correctly (selected RTT: %v)", bestPath.RTT)
}

func BenchmarkPathValidation(b *testing.B) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	local := &net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 5000}
	remote := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 443}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		challenge, _ := cm.InitiatePathValidation(local, remote)
		cm.ValidatePathResponse(challenge, remote)
	}
}

func BenchmarkPathStatisticsUpdate(b *testing.B) {
	conn := &Connection{}
	cm := NewConnectionMigration(conn)

	local := &net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 5000}
	remote := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 443}
	cm.SetCurrentPath(local, remote)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cm.RecordPacketSent(1200)
	}
}
