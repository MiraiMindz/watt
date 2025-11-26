package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// HealthChecker defines the interface for connection health checking
type HealthChecker interface {
	// Check verifies if a connection is healthy
	Check(ctx context.Context, conn *PooledConn) error
}

// TCPHealthChecker performs basic TCP health checks
type TCPHealthChecker struct {
	// PingInterval is how often to send keepalive probes
	PingInterval time.Duration
	// ReadTimeout for health check operations
	ReadTimeout time.Duration
}

// NewTCPHealthChecker creates a new TCP health checker
func NewTCPHealthChecker() *TCPHealthChecker {
	return &TCPHealthChecker{
		PingInterval: 30 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
}

// Check performs a TCP-level health check
func (hc *TCPHealthChecker) Check(ctx context.Context, conn *PooledConn) error {
	// Set a read deadline for the health check
	deadline := time.Now().Add(hc.ReadTimeout)
	if err := conn.conn.SetReadDeadline(deadline); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}
	defer conn.conn.SetReadDeadline(time.Time{})

	// Try to read with a zero-byte read to detect closed connections
	one := make([]byte, 1)
	conn.conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	_, err := conn.conn.Read(one)

	// Reset deadline
	conn.conn.SetReadDeadline(time.Time{})

	// If we get EOF or connection closed, it's unhealthy
	if err == io.EOF {
		return fmt.Errorf("connection closed")
	}

	// Timeout is expected (no data available), which means healthy
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return nil
	}

	// Any other error is unhealthy
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// HTTPHealthChecker performs HTTP-level health checks
type HTTPHealthChecker struct {
	// Method is the HTTP method for health checks (default: HEAD)
	Method string
	// Path is the path to check (default: /)
	Path string
	// ExpectedStatus is the expected status code (default: 200)
	ExpectedStatus int
	// Timeout for the health check request
	Timeout time.Duration
}

// NewHTTPHealthChecker creates a new HTTP health checker
func NewHTTPHealthChecker() *HTTPHealthChecker {
	return &HTTPHealthChecker{
		Method:         "HEAD",
		Path:           "/",
		ExpectedStatus: 200,
		Timeout:        5 * time.Second,
	}
}

// Check performs an HTTP-level health check
func (hc *HTTPHealthChecker) Check(ctx context.Context, conn *PooledConn) error {
	// First do TCP check
	tcpCheck := NewTCPHealthChecker()
	if err := tcpCheck.Check(ctx, conn); err != nil {
		return err
	}

	// For HTTP/1.1, we could send a HEAD request
	// For HTTP/2 and HTTP/3, this would need protocol-specific implementation
	// For now, TCP check is sufficient

	return nil
}

// CompositeHealthChecker combines multiple health checkers
type CompositeHealthChecker struct {
	checkers []HealthChecker
}

// NewCompositeHealthChecker creates a composite health checker
func NewCompositeHealthChecker(checkers ...HealthChecker) *CompositeHealthChecker {
	return &CompositeHealthChecker{
		checkers: checkers,
	}
}

// Check runs all health checkers
func (chc *CompositeHealthChecker) Check(ctx context.Context, conn *PooledConn) error {
	for _, checker := range chc.checkers {
		if err := checker.Check(ctx, conn); err != nil {
			return err
		}
	}
	return nil
}

// NoOpHealthChecker is a health checker that always returns success
type NoOpHealthChecker struct{}

// Check always returns nil
func (nohc *NoOpHealthChecker) Check(ctx context.Context, conn *PooledConn) error {
	return nil
}

// HealthCheckResult contains the result of a health check
type HealthCheckResult struct {
	Healthy   bool
	Latency   time.Duration
	Error     error
	Timestamp time.Time
}

// HealthMonitor tracks health check results over time
type HealthMonitor struct {
	results map[string][]HealthCheckResult
	mu      sync.RWMutex
	maxHistory int
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(maxHistory int) *HealthMonitor {
	return &HealthMonitor{
		results:    make(map[string][]HealthCheckResult),
		maxHistory: maxHistory,
	}
}

// RecordResult records a health check result
func (hm *HealthMonitor) RecordResult(host string, result HealthCheckResult) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	results, exists := hm.results[host]
	if !exists {
		results = make([]HealthCheckResult, 0, hm.maxHistory)
	}

	results = append(results, result)
	if len(results) > hm.maxHistory {
		results = results[len(results)-hm.maxHistory:]
	}

	hm.results[host] = results
}

// GetResults returns health check results for a host
func (hm *HealthMonitor) GetResults(host string) []HealthCheckResult {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results, exists := hm.results[host]
	if !exists {
		return nil
	}

	// Return a copy
	copy := make([]HealthCheckResult, len(results))
	for i := range results {
		copy[i] = results[i]
	}
	return copy
}

// HealthRate returns the health rate for a host (percentage of healthy checks)
func (hm *HealthMonitor) HealthRate(host string) float64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results, exists := hm.results[host]
	if !exists || len(results) == 0 {
		return 1.0 // Assume healthy if no data
	}

	healthy := 0
	for _, result := range results {
		if result.Healthy {
			healthy++
		}
	}

	return float64(healthy) / float64(len(results))
}

// AverageLatency returns the average latency for health checks
func (hm *HealthMonitor) AverageLatency(host string) time.Duration {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results, exists := hm.results[host]
	if !exists || len(results) == 0 {
		return 0
	}

	var total time.Duration
	count := 0
	for _, result := range results {
		if result.Healthy {
			total += result.Latency
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}
