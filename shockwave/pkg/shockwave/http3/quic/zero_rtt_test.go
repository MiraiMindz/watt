package quic

import (
	"testing"
	"time"
)

func TestSessionCache(t *testing.T) {
	cache := NewSessionCache(3)

	// Create test tickets
	ticket1 := &SessionTicket{
		ServerName:       "example.com",
		ReceivedAt:       time.Now(),
		MaxEarlyDataSize: 1000,
	}

	ticket2 := &SessionTicket{
		ServerName:       "test.com",
		ReceivedAt:       time.Now(),
		MaxEarlyDataSize: 2000,
	}

	// Test Put and Get
	cache.Put("example.com", ticket1)
	cache.Put("test.com", ticket2)

	retrieved, err := cache.Get("example.com")
	if err != nil {
		t.Fatalf("Failed to retrieve ticket: %v", err)
	}

	if retrieved.MaxEarlyDataSize != 1000 {
		t.Errorf("Expected MaxEarlyDataSize 1000, got %d", retrieved.MaxEarlyDataSize)
	}

	// Test cache eviction
	ticket3 := &SessionTicket{
		ServerName:       "other.com",
		ReceivedAt:       time.Now(),
		MaxEarlyDataSize: 3000,
	}
	ticket4 := &SessionTicket{
		ServerName:       "another.com",
		ReceivedAt:       time.Now(),
		MaxEarlyDataSize: 4000,
	}

	cache.Put("other.com", ticket3)
	cache.Put("another.com", ticket4)

	// First ticket should be evicted (FIFO)
	_, err = cache.Get("example.com")
	if err == nil {
		t.Error("Expected ticket to be evicted")
	}

	t.Logf("✓ Session cache working correctly")
}

func TestSessionCacheExpiration(t *testing.T) {
	cache := NewSessionCache(10)

	// Create expired ticket
	oldTicket := &SessionTicket{
		ServerName:       "expired.com",
		ReceivedAt:       time.Now().Add(-8 * 24 * time.Hour), // 8 days old
		MaxEarlyDataSize: 1000,
	}

	cache.Put("expired.com", oldTicket)

	// Should fail to retrieve expired ticket
	_, err := cache.Get("expired.com")
	if err != ErrNoSessionTicket {
		t.Errorf("Expected ErrNoSessionTicket for expired ticket, got %v", err)
	}

	t.Logf("✓ Session expiration working correctly")
}

func TestZeroRTTHandler(t *testing.T) {
	// Create mock connection
	conn := &Connection{
		localParams: DefaultTransportParameters(),
	}

	handler := NewZeroRTTHandler(conn)

	// Test initial state
	if handler.CanSendEarlyData() {
		t.Error("Should not be able to send early data initially")
	}

	// Create session ticket
	ticket := &SessionTicket{
		ServerName:         "example.com",
		ReceivedAt:         time.Now(),
		MaxEarlyDataSize:   10000,
		CipherSuite:        TLS_AES_128_GCM_SHA256,
		EarlyTrafficSecret: make([]byte, 32), // Mock secret
	}

	// Enable 0-RTT
	err := handler.Enable0RTT(ticket)
	if err != nil {
		t.Fatalf("Failed to enable 0-RTT: %v", err)
	}

	if !handler.CanSendEarlyData() {
		t.Error("Should be able to send early data after enabling")
	}

	// Test early data limits
	handler.RecordEarlyDataSent(5000)

	if !handler.CanSendEarlyData() {
		t.Error("Should still be able to send early data (5000/10000)")
	}

	handler.RecordEarlyDataSent(5000)

	if handler.CanSendEarlyData() {
		t.Error("Should not be able to send more early data (10000/10000)")
	}

	t.Logf("✓ 0-RTT handler working correctly")
}

func TestEarlyDataAcceptReject(t *testing.T) {
	conn := &Connection{
		localParams: DefaultTransportParameters(),
	}

	handler := NewZeroRTTHandler(conn)

	ticket := &SessionTicket{
		ServerName:         "example.com",
		ReceivedAt:         time.Now(),
		MaxEarlyDataSize:   10000,
		CipherSuite:        TLS_AES_128_GCM_SHA256,
		EarlyTrafficSecret: make([]byte, 32),
	}

	handler.Enable0RTT(ticket)

	// Test acceptance
	handler.AcceptEarlyData()

	if !handler.IsEarlyDataAccepted() {
		t.Error("Early data should be marked as accepted")
	}

	if handler.IsEarlyDataRejected() {
		t.Error("Early data should not be marked as rejected")
	}

	// Test rejection
	handler2 := NewZeroRTTHandler(conn)
	handler2.Enable0RTT(ticket)
	handler2.RejectEarlyData()

	if !handler2.IsEarlyDataRejected() {
		t.Error("Early data should be marked as rejected")
	}

	if handler2.CanSendEarlyData() {
		t.Error("Should not be able to send early data after rejection")
	}

	t.Logf("✓ Early data accept/reject working correctly")
}

func TestAntiReplayWindow(t *testing.T) {
	window := newAntiReplayWindow(3)

	random1 := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	random2 := []byte{9, 10, 11, 12, 13, 14, 15, 16}
	random3 := []byte{17, 18, 19, 20, 21, 22, 23, 24}
	random4 := []byte{25, 26, 27, 28, 29, 30, 31, 32}

	// Add first random
	if window.Contains(random1) {
		t.Error("Window should not contain random1 initially")
	}

	window.Add(random1)

	if !window.Contains(random1) {
		t.Error("Window should contain random1 after adding")
	}

	// Add more randoms
	window.Add(random2)
	window.Add(random3)

	if !window.Contains(random1) || !window.Contains(random2) || !window.Contains(random3) {
		t.Error("Window should contain all added randoms")
	}

	// Adding fourth should evict first (FIFO with size 3)
	window.Add(random4)

	if window.Contains(random1) {
		t.Error("random1 should be evicted")
	}

	if !window.Contains(random4) {
		t.Error("random4 should be in window")
	}

	t.Logf("✓ Anti-replay window working correctly")
}

func TestValidateClientHello(t *testing.T) {
	conn := &Connection{
		localParams: DefaultTransportParameters(),
	}

	handler := NewZeroRTTHandler(conn)

	random1 := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	random2 := []byte{1, 2, 3, 4, 5, 6, 7, 8} // Same as random1

	// First validation should succeed
	err := handler.ValidateClientHello(random1)
	if err != nil {
		t.Fatalf("First validation should succeed: %v", err)
	}

	// Second validation with same random should fail (replay)
	err = handler.ValidateClientHello(random2)
	if err != ErrReplayDetected {
		t.Errorf("Expected ErrReplayDetected, got %v", err)
	}

	t.Logf("✓ ClientHello validation working correctly")
}

func TestCreateSessionTicket(t *testing.T) {
	conn := &Connection{
		localParams: DefaultTransportParameters(),
	}

	ticket, err := CreateSessionTicket(conn)
	if err != nil {
		t.Fatalf("Failed to create session ticket: %v", err)
	}

	if len(ticket.Ticket) != 32 {
		t.Errorf("Expected ticket length 32, got %d", len(ticket.Ticket))
	}

	if ticket.MaxEarlyDataSize == 0 {
		t.Error("MaxEarlyDataSize should be set")
	}

	if ticket.TransportParams == nil {
		t.Error("TransportParams should be set")
	}

	t.Logf("✓ Session ticket creation working correctly")
}

func BenchmarkSessionCachePut(b *testing.B) {
	cache := NewSessionCache(1000)

	ticket := &SessionTicket{
		ServerName:       "example.com",
		ReceivedAt:       time.Now(),
		MaxEarlyDataSize: 1000,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Put("example.com", ticket)
	}
}

func BenchmarkSessionCacheGet(b *testing.B) {
	cache := NewSessionCache(1000)

	ticket := &SessionTicket{
		ServerName:       "example.com",
		ReceivedAt:       time.Now(),
		MaxEarlyDataSize: 1000,
	}

	cache.Put("example.com", ticket)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Get("example.com")
	}
}

func BenchmarkAntiReplayCheck(b *testing.B) {
	window := newAntiReplayWindow(1000)

	random := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	window.Add(random)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		window.Contains(random)
	}
}
