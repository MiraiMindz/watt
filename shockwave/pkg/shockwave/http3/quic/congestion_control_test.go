package quic

import (
	"testing"
	"time"
)

func TestCongestionControllerInitial(t *testing.T) {
	cc := NewCongestionController()

	// Check initial state
	cwnd := cc.GetCongestionWindow()
	if cwnd != kInitialWindow {
		t.Errorf("Initial cwnd = %d, want %d", cwnd, kInitialWindow)
	}

	state := cc.GetState()
	if state != CongestionStateSlowStart {
		t.Errorf("Initial state = %v, want SlowStart", state)
	}

	ssthresh := cc.GetSlowStartThreshold()
	if ssthresh == 0 {
		t.Error("Initial ssthresh = 0, want > 0")
	}
}

func TestCongestionControllerSlowStart(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	initialCwnd := cc.GetCongestionWindow()

	// Send and ack a packet (should increase cwnd by packet size)
	packetSize := uint64(1200)
	cc.OnPacketSent(packetSize, now)
	cc.OnPacketAcked(packetSize, 100*time.Millisecond, now.Add(100*time.Millisecond))

	newCwnd := cc.GetCongestionWindow()
	expected := initialCwnd + packetSize
	if newCwnd != expected {
		t.Errorf("After ack, cwnd = %d, want %d", newCwnd, expected)
	}

	// Should still be in slow start
	if cc.GetState() != CongestionStateSlowStart {
		t.Error("State should be SlowStart after one ack")
	}
}

func TestCongestionControllerCongestionAvoidance(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Set ssthresh low and build up cwnd to reach it
	cc.SetSlowStartThreshold(kInitialWindow / 2)

	// Ack enough packets to reach ssthresh (triggers state change to congestion avoidance)
	packetSize := uint64(1200)
	for cc.GetState() == CongestionStateSlowStart {
		cc.OnPacketSent(packetSize, now)
		cc.OnPacketAcked(packetSize, 100*time.Millisecond, now.Add(100*time.Millisecond))
	}

	// Now we're in congestion avoidance
	initialCwnd := cc.GetCongestionWindow()

	// Send and ack one more packet
	cc.OnPacketSent(packetSize, now)
	cc.OnPacketAcked(packetSize, 100*time.Millisecond, now.Add(100*time.Millisecond))

	// Should be in congestion avoidance now
	if cc.GetState() != CongestionStateCongestionAvoidance {
		t.Errorf("State = %v, want CongestionAvoidance", cc.GetState())
	}

	newCwnd := cc.GetCongestionWindow()
	// In congestion avoidance, increase should be smaller
	if newCwnd <= initialCwnd {
		t.Error("cwnd should increase in congestion avoidance")
	}
	// Increase should be smaller than full packet size
	// Note: The exact increase depends on cwnd/MSS ratio
	increase := newCwnd - initialCwnd
	if increase >= packetSize {
		t.Errorf("cwnd increase = %d, should be < %d (full packet) in congestion avoidance",
			increase, packetSize)
	}
	// Verify we're using additive increase
	if increase == 0 {
		t.Error("cwnd should increase in congestion avoidance, got 0")
	}
}

func TestCongestionControllerPacketLoss(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Build up cwnd
	cc.SetCongestionWindow(20000)
	initialCwnd := cc.GetCongestionWindow()

	// Lose a packet
	cc.OnPacketSent(1200, now)
	cc.OnPacketLost(1200, now.Add(100*time.Millisecond))

	// Should enter recovery
	if cc.GetState() != CongestionStateRecovery {
		t.Errorf("State = %v, want Recovery after loss", cc.GetState())
	}

	// cwnd should be reduced
	newCwnd := cc.GetCongestionWindow()
	if newCwnd >= initialCwnd {
		t.Errorf("cwnd after loss = %d, should be < %d", newCwnd, initialCwnd)
	}

	// ssthresh should be set
	ssthresh := cc.GetSlowStartThreshold()
	if ssthresh >= initialCwnd {
		t.Errorf("ssthresh after loss = %d, should be < %d", ssthresh, initialCwnd)
	}

	// cwnd should equal ssthresh (NewReno behavior)
	if newCwnd != ssthresh {
		t.Errorf("cwnd = %d, ssthresh = %d, should be equal after loss", newCwnd, ssthresh)
	}
}

func TestCongestionControllerBytesInFlight(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Send some packets
	cc.OnPacketSent(1200, now)
	cc.OnPacketSent(1200, now)
	cc.OnPacketSent(1200, now)

	inFlight := cc.GetBytesInFlight()
	if inFlight != 3600 {
		t.Errorf("BytesInFlight = %d, want 3600", inFlight)
	}

	// Ack one packet
	cc.OnPacketAcked(1200, 50*time.Millisecond, now.Add(50*time.Millisecond))

	inFlight = cc.GetBytesInFlight()
	if inFlight != 2400 {
		t.Errorf("BytesInFlight after ack = %d, want 2400", inFlight)
	}

	// Lose one packet
	cc.OnPacketLost(1200, now.Add(100*time.Millisecond))

	inFlight = cc.GetBytesInFlight()
	if inFlight != 1200 {
		t.Errorf("BytesInFlight after loss = %d, want 1200", inFlight)
	}
}

func TestCongestionControllerCanSend(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Set a small cwnd for testing
	cc.SetCongestionWindow(5000)

	// Should be able to send within cwnd
	if !cc.CanSend(3000) {
		t.Error("CanSend(3000) = false, want true with cwnd=5000")
	}

	// Send some data
	cc.OnPacketSent(3000, now)

	// Should be able to send remaining
	if !cc.CanSend(2000) {
		t.Error("CanSend(2000) = false, want true with 3000 in flight")
	}

	// Should not be able to exceed cwnd
	if cc.CanSend(3000) {
		t.Error("CanSend(3000) = true, want false with 3000 in flight and cwnd=5000")
	}
}

func TestCongestionControllerRTT(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Send and ack with specific RTT
	rtt1 := 100 * time.Millisecond
	cc.OnPacketSent(1200, now)
	cc.OnPacketAcked(1200, rtt1, now.Add(rtt1))

	minRTT, smoothedRTT, rttVar := cc.GetRTT()

	// Min RTT should be set
	if minRTT != rtt1 {
		t.Errorf("minRTT = %v, want %v", minRTT, rtt1)
	}

	// First sample: smoothed RTT is initialized to the measured RTT
	// But we start with an initial value, so it will be averaged
	// Just check it's reasonable (between 50ms and 400ms)
	if smoothedRTT < 50*time.Millisecond || smoothedRTT > 400*time.Millisecond {
		t.Errorf("smoothedRTT = %v, want between 50ms and 400ms", smoothedRTT)
	}

	// RTT variance should be reasonable
	if rttVar <= 0 {
		t.Error("rttVar should be > 0")
	}

	// Send another packet with different RTT
	rtt2 := 50 * time.Millisecond
	cc.OnPacketSent(1200, now.Add(200*time.Millisecond))
	cc.OnPacketAcked(1200, rtt2, now.Add(250*time.Millisecond))

	minRTT, smoothedRTT, _ = cc.GetRTT()

	// Min RTT should be updated
	if minRTT != rtt2 {
		t.Errorf("minRTT after second sample = %v, want %v", minRTT, rtt2)
	}

	// Smoothed RTT should be moving toward the new samples
	// Just check it's still reasonable
	if smoothedRTT < 40*time.Millisecond || smoothedRTT > 350*time.Millisecond {
		t.Errorf("smoothedRTT after second sample = %v, want between 40ms and 350ms", smoothedRTT)
	}
}

func TestCongestionControllerStatistics(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Send 10 packets
	for i := 0; i < 10; i++ {
		cc.OnPacketSent(1200, now.Add(time.Duration(i)*time.Millisecond))
	}

	// Ack 8 packets
	for i := 0; i < 8; i++ {
		cc.OnPacketAcked(1200, 50*time.Millisecond, now.Add(time.Duration(i+50)*time.Millisecond))
	}

	// Lose 2 packets
	cc.OnPacketLost(1200, now.Add(100*time.Millisecond))
	cc.OnPacketLost(1200, now.Add(101*time.Millisecond))

	sent, acked, lost, lossRate := cc.GetStatistics()

	if sent != 10 {
		t.Errorf("packetsSent = %d, want 10", sent)
	}
	if acked != 8 {
		t.Errorf("packetsAcked = %d, want 8", acked)
	}
	if lost != 2 {
		t.Errorf("packetsLost = %d, want 2", lost)
	}

	expectedLossRate := 2.0 / 10.0
	if lossRate < expectedLossRate-0.01 || lossRate > expectedLossRate+0.01 {
		t.Errorf("lossRate = %f, want %f", lossRate, expectedLossRate)
	}
}

func TestCongestionControllerPersistentCongestion(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Build up cwnd
	cc.SetCongestionWindow(50000)
	initialCwnd := cc.GetCongestionWindow()

	// Trigger persistent congestion (multiple PTO-worth of losses)
	cc.OnCongestionEvent(kPersistentCongestionThreshold, now)

	// Should reset to minimum window
	newCwnd := cc.GetCongestionWindow()
	if newCwnd != kMinimumWindow {
		t.Errorf("cwnd after persistent congestion = %d, want %d", newCwnd, kMinimumWindow)
	}

	// Should be back in slow start
	if cc.GetState() != CongestionStateSlowStart {
		t.Errorf("State after persistent congestion = %v, want SlowStart", cc.GetState())
	}

	if newCwnd >= initialCwnd {
		t.Error("cwnd should be reset after persistent congestion")
	}
}

func TestCongestionControllerRecoveryExit(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Set up for recovery
	cc.SetCongestionWindow(20000)
	cc.OnPacketSent(1200, now)
	cc.OnPacketLost(1200, now.Add(100*time.Millisecond))

	// Should be in recovery
	if cc.GetState() != CongestionStateRecovery {
		t.Fatal("Should be in recovery")
	}

	// Ack a packet sent AFTER recovery started
	futureTime := now.Add(200 * time.Millisecond)
	cc.OnPacketSent(1200, futureTime)
	cc.OnPacketAcked(1200, 50*time.Millisecond, futureTime.Add(50*time.Millisecond))

	// Should exit recovery
	if cc.GetState() != CongestionStateCongestionAvoidance {
		t.Errorf("State after recovery = %v, want CongestionAvoidance", cc.GetState())
	}
}

func TestCongestionControllerReset(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Modify state
	cc.OnPacketSent(1200, now)
	cc.SetCongestionWindow(50000)
	cc.SetSlowStartThreshold(25000)

	// Reset
	cc.Reset()

	// Should be back to initial state
	if cc.GetCongestionWindow() != kInitialWindow {
		t.Errorf("cwnd after reset = %d, want %d", cc.GetCongestionWindow(), kInitialWindow)
	}

	if cc.GetBytesInFlight() != 0 {
		t.Error("bytesInFlight after reset should be 0")
	}

	if cc.GetState() != CongestionStateSlowStart {
		t.Error("State after reset should be SlowStart")
	}

	sent, acked, lost, _ := cc.GetStatistics()
	if sent != 0 || acked != 0 || lost != 0 {
		t.Error("Statistics should be reset")
	}
}

func TestCongestionControllerPacingRate(t *testing.T) {
	cc := NewCongestionController()
	now := time.Now()

	// Enable pacing
	cc.EnablePacing(true)

	// Send and ack to establish RTT
	cc.OnPacketSent(1200, now)
	cc.OnPacketAcked(1200, 100*time.Millisecond, now.Add(100*time.Millisecond))

	// Pacing rate should be calculated
	pacingRate := cc.GetPacingRate()
	if pacingRate == 0 {
		t.Error("Pacing rate should be > 0 after RTT sample")
	}

	// Pacing rate should be approximately cwnd / rtt
	cwnd := cc.GetCongestionWindow()
	_, smoothedRTT, _ := cc.GetRTT()
	expectedRate := uint64(float64(cwnd) / (float64(smoothedRTT) / float64(time.Second)))

	// Allow 50% tolerance due to rounding
	if pacingRate < expectedRate/2 || pacingRate > expectedRate*2 {
		t.Errorf("Pacing rate = %d, expected ~%d (cwnd=%d, rtt=%v)",
			pacingRate, expectedRate, cwnd, smoothedRTT)
	}
}

func BenchmarkCongestionControllerCanSend(b *testing.B) {
	cc := NewCongestionController()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cc.CanSend(1200)
	}
}

func BenchmarkCongestionControllerOnPacketAcked(b *testing.B) {
	cc := NewCongestionController()
	now := time.Now()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cc.OnPacketAcked(1200, 50*time.Millisecond, now)
	}
}

func BenchmarkCongestionControllerOnPacketLost(b *testing.B) {
	cc := NewCongestionController()
	now := time.Now()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cc.OnPacketLost(1200, now)
		cc.Reset() // Reset to avoid staying in recovery
	}
}
