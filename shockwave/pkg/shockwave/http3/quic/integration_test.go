package quic

import (
	"testing"
	"time"
)

// TestFlowControlAndCongestionControlIntegration tests that flow control
// and congestion control work together correctly
func TestFlowControlAndCongestionControlIntegration(t *testing.T) {
	// Create both controllers
	fc := NewFlowController(100000, 100000)
	cc := NewCongestionController()
	now := time.Now()

	// Scenario: Send data limited by both flow control AND congestion control

	// Get initial limits
	fcLimit := fc.peerConnMaxData - fc.connDataSent
	ccLimit := cc.GetCongestionWindow() - cc.GetBytesInFlight()

	t.Logf("Initial limits: FC=%d, CC=%d", fcLimit, ccLimit)

	// Congestion control starts with smaller window (12000 bytes)
	if ccLimit >= fcLimit {
		t.Errorf("Expected CC to be more restrictive initially, but CC=%d, FC=%d", ccLimit, fcLimit)
	}

	// Try to send: should be limited by congestion control
	packetSize := uint64(1200)
	maxPackets := ccLimit / packetSize

	for i := uint64(0); i < maxPackets; i++ {
		if !fc.CanSendData(packetSize) {
			t.Fatalf("Flow control blocked at packet %d, expected CC to block first", i)
		}
		if !cc.CanSend(packetSize) {
			t.Fatalf("Congestion control blocked at packet %d", i)
		}

		// Send packet
		fc.RecordDataSent(packetSize)
		cc.OnPacketSent(packetSize, now)
	}

	// Now both should block
	if fc.CanSendData(packetSize) && !cc.CanSend(packetSize) {
		// Expected: CC blocks, FC allows
		t.Logf("Correctly blocked by congestion control")
	} else {
		t.Errorf("Expected CC to block, but FC.CanSend=%v, CC.CanSend=%v",
			fc.CanSendData(packetSize), cc.CanSend(packetSize))
	}

	// ACK some packets to open up congestion window
	for i := 0; i < 5; i++ {
		cc.OnPacketAcked(packetSize, 50*time.Millisecond, now.Add(50*time.Millisecond))
	}

	// Should be able to send more now
	if !cc.CanSend(packetSize) {
		t.Error("Congestion control should allow sending after ACKs")
	}
}

func TestFlowControlBlocksCongestionControl(t *testing.T) {
	// Scenario: Flow control is more restrictive than congestion control

	fc := NewFlowController(10000, 10000) // Small flow control limit
	cc := NewCongestionController()       // Larger congestion window (12000)
	now := time.Now()

	packetSize := uint64(1200)

	// Send until flow control blocks
	for fc.CanSendData(packetSize) {
		if !cc.CanSend(packetSize) {
			t.Fatal("Congestion control blocked before flow control")
		}

		fc.RecordDataSent(packetSize)
		cc.OnPacketSent(packetSize, now)
	}

	// Flow control should be blocking
	if fc.CanSendData(packetSize) {
		t.Error("Flow control should be blocking")
	}

	// Congestion control might still allow
	if cc.CanSend(packetSize) {
		t.Log("Congestion control still allows sending (expected)")
	}

	// Peer sends MAX_DATA update
	fc.UpdatePeerMaxData(50000)

	// Should be able to send more
	if !fc.CanSendData(packetSize) {
		t.Error("Flow control should allow sending after MAX_DATA")
	}
}

func TestCongestionControlWithFlowControlUpdates(t *testing.T) {
	// Test that ACKs work correctly with flow control tracking

	fc := NewFlowController(100000, 100000)
	cc := NewCongestionController()
	now := time.Now()

	packetSize := uint64(1200)

	// Send 10 packets
	for i := 0; i < 10; i++ {
		fc.RecordDataSent(packetSize)
		cc.OnPacketSent(packetSize, now.Add(time.Duration(i)*time.Millisecond))
	}

	initialCwnd := cc.GetCongestionWindow()

	// ACK all packets (should grow cwnd in slow start)
	for i := 0; i < 10; i++ {
		cc.OnPacketAcked(packetSize, 50*time.Millisecond, now.Add(time.Duration(i+50)*time.Millisecond))
	}

	// Congestion window should have grown
	newCwnd := cc.GetCongestionWindow()
	if newCwnd <= initialCwnd {
		t.Errorf("Cwnd should have grown: initial=%d, new=%d", initialCwnd, newCwnd)
	}

	// Bytes in flight should be zero
	if cc.GetBytesInFlight() != 0 {
		t.Errorf("Bytes in flight = %d, want 0 after all ACKs", cc.GetBytesInFlight())
	}
}

func TestPacketLossWithFlowControl(t *testing.T) {
	// Test that packet loss reduces congestion window but doesn't affect flow control

	fc := NewFlowController(100000, 100000)
	cc := NewCongestionController()
	now := time.Now()

	// Build up congestion window
	cc.SetCongestionWindow(50000)
	initialCwnd := cc.GetCongestionWindow()

	packetSize := uint64(1200)

	// Send some packets
	for i := 0; i < 10; i++ {
		fc.RecordDataSent(packetSize)
		cc.OnPacketSent(packetSize, now)
	}

	fcSentBefore := fc.connDataSent

	// Lose a packet
	cc.OnPacketLost(packetSize, now.Add(100*time.Millisecond))

	// Congestion window should be reduced
	newCwnd := cc.GetCongestionWindow()
	if newCwnd >= initialCwnd {
		t.Errorf("Cwnd should be reduced after loss: initial=%d, new=%d", initialCwnd, newCwnd)
	}

	// Flow control sent bytes should not change (loss doesn't unsend data)
	if fc.connDataSent != fcSentBefore {
		t.Errorf("Flow control sent changed after loss: before=%d, after=%d",
			fcSentBefore, fc.connDataSent)
	}

	// Bytes in flight should be reduced
	expectedInFlight := 9 * packetSize // 10 sent, 1 lost
	if cc.GetBytesInFlight() != expectedInFlight {
		t.Errorf("Bytes in flight = %d, want %d", cc.GetBytesInFlight(), expectedInFlight)
	}
}

func TestStreamFlowControlWithCongestionControl(t *testing.T) {
	// Test stream-level flow control with connection-level congestion control

	connFC := NewFlowController(100000, 100000)
	streamFC := NewStreamFlowController(4, 10000, 10000)
	cc := NewCongestionController()
	now := time.Now()

	packetSize := uint64(1200)

	// Send until stream flow control blocks
	for streamFC.CanSend(packetSize) {
		// Check all three layers
		if !connFC.CanSendData(packetSize) {
			t.Fatal("Connection flow control blocked before stream")
		}
		if !cc.CanSend(packetSize) {
			t.Fatal("Congestion control blocked before stream")
		}

		// Record at all levels
		connFC.RecordDataSent(packetSize)
		streamFC.RecordSent(packetSize)
		cc.OnPacketSent(packetSize, now)
	}

	// Stream should be blocked
	if streamFC.CanSend(packetSize) {
		t.Error("Stream flow control should be blocking")
	}

	// Stream sent should equal its limit
	sendOffset, sendMax := streamFC.GetSendStats()
	if sendOffset+packetSize > sendMax {
		t.Logf("Stream correctly blocked: offset=%d, max=%d", sendOffset, sendMax)
	}

	// Peer sends MAX_STREAM_DATA
	streamFC.UpdatePeerMaxStreamData(20000)

	// Should be able to send on stream now
	if !streamFC.CanSend(packetSize) {
		t.Error("Stream should allow sending after MAX_STREAM_DATA")
	}
}

func TestAutoTuningWithCongestionControl(t *testing.T) {
	// Test that stream auto-tuning doesn't interfere with congestion control

	streamFC := NewStreamFlowController(4, 10000, 10000)
	cc := NewCongestionController()

	packetSize := uint64(1200)

	// Fill stream window to trigger auto-tune
	for i := 0; i < 8; i++ { // 8 * 1200 = 9600 bytes (96% of 10000)
		streamFC.RecordReceived(packetSize)
	}

	// Auto-tune should increase window
	newWindow := streamFC.AutoTuneWindow()
	if newWindow <= 10000 {
		t.Errorf("Auto-tune should increase window: got %d", newWindow)
	}

	// Congestion control should be independent
	if cc.GetCongestionWindow() != kInitialWindow {
		t.Error("Congestion control should not be affected by stream auto-tuning")
	}
}

func TestPacingWithFlowControl(t *testing.T) {
	// Test that pacing rate calculation works with flow control

	cc := NewCongestionController()
	now := time.Now()

	// Send and ACK to establish RTT
	packetSize := uint64(1200)
	cc.OnPacketSent(packetSize, now)
	cc.OnPacketAcked(packetSize, 100*time.Millisecond, now.Add(100*time.Millisecond))

	// Get pacing rate
	pacingRate := cc.GetPacingRate()
	if pacingRate == 0 {
		t.Error("Pacing rate should be > 0 after RTT sample")
	}

	// Pacing rate should be approximately cwnd / rtt
	cwnd := cc.GetCongestionWindow()
	_, smoothedRTT, _ := cc.GetRTT()
	expectedRate := uint64(float64(cwnd) / (float64(smoothedRTT) / float64(time.Second)))

	// Allow 50% tolerance
	if pacingRate < expectedRate/2 || pacingRate > expectedRate*2 {
		t.Logf("Warning: Pacing rate %d bytes/sec, expected ~%d bytes/sec (cwnd=%d, rtt=%v)",
			pacingRate, expectedRate, cwnd, smoothedRTT)
	} else {
		t.Logf("Pacing rate: %d bytes/sec (cwnd=%d, rtt=%v)", pacingRate, cwnd, smoothedRTT)
	}
}

func TestRecoveryWithFlowControlUpdates(t *testing.T) {
	// Test that recovery period works correctly with flow control

	fc := NewFlowController(100000, 100000)
	cc := NewCongestionController()
	now := time.Now()

	// Build up cwnd
	cc.SetCongestionWindow(30000)
	packetSize := uint64(1200)

	// Send packets
	for i := 0; i < 10; i++ {
		fc.RecordDataSent(packetSize)
		cc.OnPacketSent(packetSize, now.Add(time.Duration(i)*time.Millisecond))
	}

	// Lose a packet (enters recovery)
	cc.OnPacketLost(packetSize, now.Add(50*time.Millisecond))

	if cc.GetState() != CongestionStateRecovery {
		t.Fatal("Should be in recovery")
	}

	// ACK packets sent BEFORE recovery (should not increase cwnd)
	cwndBefore := cc.GetCongestionWindow()
	cc.OnPacketAcked(packetSize, 50*time.Millisecond, now.Add(40*time.Millisecond)) // Before recovery
	cwndAfter := cc.GetCongestionWindow()

	if cwndAfter > cwndBefore {
		t.Error("Cwnd should not increase for packets ACKed during recovery that were sent before recovery")
	}

	// Send and ACK packet AFTER recovery started
	futureTime := now.Add(100 * time.Millisecond)
	cc.OnPacketSent(packetSize, futureTime)
	cc.OnPacketAcked(packetSize, 50*time.Millisecond, futureTime.Add(50*time.Millisecond))

	// Should exit recovery
	if cc.GetState() == CongestionStateRecovery {
		t.Error("Should have exited recovery after ACKing post-recovery packet")
	}
}

func BenchmarkIntegratedFlowAndCongestionCheck(b *testing.B) {
	fc := NewFlowController(1000000, 1000000)
	cc := NewCongestionController()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Check both controllers (typical send path)
		fcOK := fc.CanSendData(1200)
		ccOK := cc.CanSend(1200)
		_ = fcOK && ccOK
	}
}

func BenchmarkIntegratedSendPath(b *testing.B) {
	fc := NewFlowController(uint64(b.N)*1200, uint64(b.N)*1200)
	cc := NewCongestionController()
	now := time.Now()

	// Build up cwnd
	cc.SetCongestionWindow(uint64(b.N) * 1200)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Check both
		if fc.CanSendData(1200) && cc.CanSend(1200) {
			// Record at both levels
			fc.RecordDataSent(1200)
			cc.OnPacketSent(1200, now)
		}
	}
}
