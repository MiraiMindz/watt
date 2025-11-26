package quic

import (
	"crypto/tls"
	"net"
	"testing"
	"time"
)

// Test TLS connection wrapper basic functionality
func TestTLSConnBasic(t *testing.T) {
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
	}

	tlsConn := NewTLSConn(conn)

	if tlsConn == nil {
		t.Fatal("NewTLSConn returned nil")
	}

	// Test initial state
	if tlsConn.readLevel != EncryptionLevelInitial {
		t.Errorf("Expected initial read level, got %v", tlsConn.readLevel)
	}

	if tlsConn.writeLevel != EncryptionLevelInitial {
		t.Errorf("Expected initial write level, got %v", tlsConn.writeLevel)
	}

	t.Logf("✓ TLS connection wrapper initialized correctly")
}

// Test CRYPTO frame handling
func TestTLSConnCryptoFrameHandling(t *testing.T) {
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
	}

	tlsConn := NewTLSConn(conn)

	// Create CRYPTO frame with test data
	frame := &CryptoFrame{
		Offset: 0,
		Data:   []byte("test crypto data"),
	}

	// Handle CRYPTO frame at Initial level
	err := tlsConn.HandleCryptoFrame(frame, EncryptionLevelInitial)
	if err != nil {
		t.Fatalf("Failed to handle CRYPTO frame: %v", err)
	}

	// Check that data is buffered
	if tlsConn.initialBuffer.Available() != len(frame.Data) {
		t.Errorf("Expected %d bytes in initial buffer, got %d",
			len(frame.Data), tlsConn.initialBuffer.Available())
	}

	// Read from TLS connection (non-blocking test)
	readBuf := make([]byte, 100)

	// Set a short timeout and read in goroutine
	done := make(chan bool)
	go func() {
		n, _ := tlsConn.Read(readBuf)
		if n != len(frame.Data) {
			t.Errorf("Expected to read %d bytes, got %d", len(frame.Data), n)
		}
		if string(readBuf[:n]) != string(frame.Data) {
			t.Errorf("Data mismatch: expected %s, got %s",
				string(frame.Data), string(readBuf[:n]))
		}
		done <- true
	}()

	// Wait for read with timeout
	select {
	case <-done:
		t.Logf("✓ CRYPTO frame handling and reading working correctly")
	case <-time.After(100 * time.Millisecond):
		t.Error("Read timed out")
	}
}

// Test encryption level transitions
func TestTLSConnEncryptionLevels(t *testing.T) {
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
	}

	tlsConn := NewTLSConn(conn)

	// Test level transitions
	levels := []EncryptionLevel{
		EncryptionLevelInitial,
		EncryptionLevelHandshake,
		EncryptionLevelApplication,
	}

	for _, level := range levels {
		tlsConn.SetReadLevel(level)
		tlsConn.SetWriteLevel(level)

		if tlsConn.readLevel != level {
			t.Errorf("Read level not set correctly: expected %v, got %v",
				level, tlsConn.readLevel)
		}

		if tlsConn.writeLevel != level {
			t.Errorf("Write level not set correctly: expected %v, got %v",
				level, tlsConn.writeLevel)
		}
	}

	t.Logf("✓ Encryption level transitions working correctly")
}

// Test crypto buffer operations
func TestCryptoBuffer(t *testing.T) {
	buf := newCryptoBuffer()

	// Test writing at correct offset
	data1 := []byte("hello")
	err := buf.Write(data1, 0)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %v", err)
	}

	if buf.offset != uint64(len(data1)) {
		t.Errorf("Expected offset %d, got %d", len(data1), buf.offset)
	}

	// Test writing next chunk
	data2 := []byte(" world")
	err = buf.Write(data2, uint64(len(data1)))
	if err != nil {
		t.Fatalf("Failed to write second chunk: %v", err)
	}

	// Test reading
	readBuf := make([]byte, 100)
	n, _ := buf.Read(readBuf)

	expected := string(data1) + string(data2)
	if string(readBuf[:n]) != expected {
		t.Errorf("Expected %s, got %s", expected, string(readBuf[:n]))
	}

	// Test out-of-order write (should fail)
	data3 := []byte("oops")
	err = buf.Write(data3, 100) // Wrong offset
	if err == nil {
		t.Error("Expected error for out-of-order write")
	}

	t.Logf("✓ Crypto buffer operations working correctly")
}

// Test TLS handler creation
func TestTLSHandlerCreation(t *testing.T) {
	// Create mock connection
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
		destConnID:  ConnectionID([]byte{5, 6, 7, 8}),
		localParams: DefaultTransportParameters(),
	}

	// Create TLS config
	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		NextProtos: []string{"h3"},
	}

	// Create TLS handler for client
	handler, err := NewTLSHandler(conn, config, true)
	if err != nil {
		t.Fatalf("Failed to create TLS handler: %v", err)
	}

	if !handler.isClient {
		t.Error("Expected client mode")
	}

	if handler.initialKeys == nil {
		t.Error("Initial keys not generated")
	}

	// Check encryption levels
	if handler.GetReadLevel() != EncryptionLevelInitial {
		t.Error("Expected initial read level")
	}

	if handler.GetWriteLevel() != EncryptionLevelInitial {
		t.Error("Expected initial write level")
	}

	t.Logf("✓ TLS handler created successfully")
}

// Test TLS handler key management
func TestTLSHandlerKeys(t *testing.T) {
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
		destConnID:  ConnectionID([]byte{5, 6, 7, 8}),
		localParams: DefaultTransportParameters(),
	}

	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}

	handler, err := NewTLSHandler(conn, config, true)
	if err != nil {
		t.Fatalf("Failed to create TLS handler: %v", err)
	}

	// Test initial keys
	initialKeys := handler.GetKeys(EncryptionLevelInitial)
	if initialKeys == nil {
		t.Error("Initial keys not available")
	}

	if initialKeys.Level != EncryptionLevelInitial {
		t.Errorf("Expected initial level, got %v", initialKeys.Level)
	}

	// Test that other keys are nil initially
	if handler.GetKeys(EncryptionLevelHandshake) != nil {
		t.Error("Handshake keys should be nil initially")
	}

	if handler.GetKeys(EncryptionLevelApplication) != nil {
		t.Error("Application keys should be nil initially")
	}

	t.Logf("✓ TLS handler key management working correctly")
}

// Test encryption level transitions in handler
func TestTLSHandlerLevelTransitions(t *testing.T) {
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
		destConnID:  ConnectionID([]byte{5, 6, 7, 8}),
		localParams: DefaultTransportParameters(),
	}

	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}

	handler, err := NewTLSHandler(conn, config, true)
	if err != nil {
		t.Fatalf("Failed to create TLS handler: %v", err)
	}

	// Test transitioning to handshake level
	handler.SetReadLevel(EncryptionLevelHandshake)
	handler.SetWriteLevel(EncryptionLevelHandshake)

	if handler.GetReadLevel() != EncryptionLevelHandshake {
		t.Error("Failed to transition read level to handshake")
	}

	if handler.GetWriteLevel() != EncryptionLevelHandshake {
		t.Error("Failed to transition write level to handshake")
	}

	// Test transitioning to application level
	handler.SetReadLevel(EncryptionLevelApplication)
	handler.SetWriteLevel(EncryptionLevelApplication)

	if handler.GetReadLevel() != EncryptionLevelApplication {
		t.Error("Failed to transition read level to application")
	}

	if handler.GetWriteLevel() != EncryptionLevelApplication {
		t.Error("Failed to transition write level to application")
	}

	t.Logf("✓ TLS handler level transitions working correctly")
}

// Test transport parameter handling
func TestTLSHandlerTransportParams(t *testing.T) {
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
		destConnID:  ConnectionID([]byte{5, 6, 7, 8}),
		localParams: DefaultTransportParameters(),
	}

	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}

	handler, err := NewTLSHandler(conn, config, true)
	if err != nil {
		t.Fatalf("Failed to create TLS handler: %v", err)
	}

	// Create remote transport parameters
	remoteParams := &TransportParameters{
		MaxIdleTimeout:    60000,
		MaxUDPPayloadSize: 1500,
		InitialMaxData:    1048576,
	}

	// Handle transport parameters
	err = handler.HandleTransportParameters(remoteParams)
	if err != nil {
		t.Fatalf("Failed to handle transport parameters: %v", err)
	}

	// Verify parameters were stored
	storedParams := handler.GetTransportParameters()
	if storedParams == nil {
		t.Fatal("Transport parameters not stored")
	}

	if storedParams.MaxUDPPayloadSize != 1500 {
		t.Errorf("Expected MaxUDPPayloadSize 1500, got %d",
			storedParams.MaxUDPPayloadSize)
	}

	// Test invalid parameters
	invalidParams := &TransportParameters{
		MaxUDPPayloadSize: 1000, // Less than minimum 1200
	}

	err = handler.HandleTransportParameters(invalidParams)
	if err == nil {
		t.Error("Expected error for invalid transport parameters")
	}

	t.Logf("✓ Transport parameter handling working correctly")
}

// Test net.Conn interface compliance
func TestTLSConnInterface(t *testing.T) {
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
		remoteAddr:  &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4433},
	}

	// Ensure localAddr is set
	conn.localAddr = &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4434}

	tlsConn := NewTLSConn(conn)

	// Test that TLSConn implements net.Conn
	var _ net.Conn = tlsConn

	// Test address methods
	localAddr := tlsConn.LocalAddr()
	if localAddr == nil {
		t.Error("LocalAddr returned nil")
	}

	remoteAddr := tlsConn.RemoteAddr()
	if remoteAddr == nil {
		t.Error("RemoteAddr returned nil")
	}

	// Test deadline methods (should not error)
	if err := tlsConn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("SetDeadline failed: %v", err)
	}

	if err := tlsConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("SetReadDeadline failed: %v", err)
	}

	if err := tlsConn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("SetWriteDeadline failed: %v", err)
	}

	// Test close
	if err := tlsConn.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	t.Logf("✓ TLS connection implements net.Conn interface correctly")
}

// Test TLS config creation
func TestTLSConfigCreation(t *testing.T) {
	conn := &Connection{
		localConnID: ConnectionID([]byte{1, 2, 3, 4}),
	}

	baseConfig := &tls.Config{
		ServerName: "example.com",
	}

	quicConfig := conn.TLSConfig(baseConfig)

	// Verify TLS 1.3 is enforced
	if quicConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("Expected MinVersion TLS 1.3, got 0x%x", quicConfig.MinVersion)
	}

	if quicConfig.MaxVersion != tls.VersionTLS13 {
		t.Errorf("Expected MaxVersion TLS 1.3, got 0x%x", quicConfig.MaxVersion)
	}

	// Verify ALPN is set
	if len(quicConfig.NextProtos) == 0 {
		t.Error("NextProtos (ALPN) not set")
	}

	if quicConfig.NextProtos[0] != "h3" {
		t.Errorf("Expected ALPN 'h3', got '%s'", quicConfig.NextProtos[0])
	}

	// Verify server name is preserved
	if quicConfig.ServerName != "example.com" {
		t.Errorf("ServerName not preserved: got '%s'", quicConfig.ServerName)
	}

	t.Logf("✓ TLS config creation working correctly")
}

// Benchmark TLS connection operations
func BenchmarkCryptoBufferWrite(b *testing.B) {
	buf := newCryptoBuffer()
	data := []byte("test data for benchmarking")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Write(data, uint64(i*len(data)))
	}
}

func BenchmarkCryptoBufferRead(b *testing.B) {
	buf := newCryptoBuffer()
	data := []byte("test data for benchmarking")
	readBuf := make([]byte, len(data))

	// Pre-fill buffer
	for i := 0; i < 1000; i++ {
		buf.Write(data, uint64(i*len(data)))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Read(readBuf)
	}
}
