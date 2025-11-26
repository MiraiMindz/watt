package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrPoolClosed is returned when attempting to get a connection from a closed pool
	ErrPoolClosed = errors.New("connection pool closed")
	// ErrNoHealthyConns is returned when no healthy connections are available
	ErrNoHealthyConns = errors.New("no healthy connections available")
	// ErrConnTimeout is returned when connection acquisition times out
	ErrConnTimeout = errors.New("connection acquisition timeout")
)

// ProtocolVersion represents the HTTP protocol version
type ProtocolVersion int

const (
	// HTTP11 represents HTTP/1.1
	HTTP11 ProtocolVersion = iota
	// HTTP2 represents HTTP/2
	HTTP2
	// HTTP3 represents HTTP/3
	HTTP3
)

// PoolConfig configures the connection pool
type PoolConfig struct {
	// MaxConnsPerHost is the maximum number of connections per host
	MaxConnsPerHost int
	// MaxIdleConnsPerHost is the maximum idle connections per host
	MaxIdleConnsPerHost int
	// MaxIdleTime is how long idle connections are kept
	MaxIdleTime time.Duration
	// ConnTimeout is the timeout for establishing new connections
	ConnTimeout time.Duration
	// IdleCheckInterval is how often to check for idle connections
	IdleCheckInterval time.Duration
	// HealthCheckInterval is how often to health check connections
	HealthCheckInterval time.Duration
	// HealthCheckTimeout is the timeout for health checks
	HealthCheckTimeout time.Duration
	// TLSConfig for secure connections
	TLSConfig *tls.Config
	// PreferredProtocol is the preferred HTTP version
	PreferredProtocol ProtocolVersion
}

// DefaultPoolConfig returns sensible defaults
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnsPerHost:     100,
		MaxIdleConnsPerHost: 10,
		MaxIdleTime:         90 * time.Second,
		ConnTimeout:         30 * time.Second,
		IdleCheckInterval:   30 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		PreferredProtocol:   HTTP2,
	}
}

// PooledConn represents a pooled connection with metadata
type PooledConn struct {
	conn           net.Conn
	host           string
	protocol       ProtocolVersion
	createdAt      time.Time
	lastUsed       time.Time
	requestCount   uint64
	healthy        atomic.Bool
	inUse          atomic.Bool
	pool           *ConnectionPool
	tlsState       *tls.ConnectionState
	mu             sync.RWMutex
}

// Conn returns the underlying connection
func (pc *PooledConn) Conn() net.Conn {
	return pc.conn
}

// Protocol returns the protocol version
func (pc *PooledConn) Protocol() ProtocolVersion {
	return pc.protocol
}

// IsHealthy returns whether the connection is healthy
func (pc *PooledConn) IsHealthy() bool {
	return pc.healthy.Load()
}

// MarkUnhealthy marks the connection as unhealthy
func (pc *PooledConn) MarkUnhealthy() {
	pc.healthy.Store(false)
}

// MarkHealthy marks the connection as healthy
func (pc *PooledConn) MarkHealthy() {
	pc.healthy.Store(true)
}

// IncrementRequests increments the request counter
func (pc *PooledConn) IncrementRequests() {
	atomic.AddUint64(&pc.requestCount, 1)
}

// RequestCount returns the number of requests made on this connection
func (pc *PooledConn) RequestCount() uint64 {
	return atomic.LoadUint64(&pc.requestCount)
}

// Age returns how long the connection has existed
func (pc *PooledConn) Age() time.Duration {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return time.Since(pc.createdAt)
}

// IdleTime returns how long the connection has been idle
func (pc *PooledConn) IdleTime() time.Duration {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return time.Since(pc.lastUsed)
}

// updateLastUsed updates the last used timestamp
func (pc *PooledConn) updateLastUsed() {
	pc.mu.Lock()
	pc.lastUsed = time.Now()
	pc.mu.Unlock()
}

// Close closes the connection and returns it to the pool
func (pc *PooledConn) Close() error {
	if pc.pool != nil {
		pc.updateLastUsed()
		pc.inUse.Store(false)
		return pc.pool.putConn(pc)
	}
	return pc.conn.Close()
}

// hostPool manages connections for a single host
type hostPool struct {
	host        string
	conns       []*PooledConn
	idleConns   chan *PooledConn
	config      *PoolConfig
	mu          sync.RWMutex
	activeCount int32
	totalCount  int32
}

// newHostPool creates a new host pool
func newHostPool(host string, config *PoolConfig) *hostPool {
	return &hostPool{
		host:      host,
		conns:     make([]*PooledConn, 0, config.MaxConnsPerHost),
		idleConns: make(chan *PooledConn, config.MaxIdleConnsPerHost),
		config:    config,
	}
}

// get retrieves an idle connection or returns nil
func (hp *hostPool) get() *PooledConn {
	select {
	case conn := <-hp.idleConns:
		if conn.IsHealthy() && conn.IdleTime() < hp.config.MaxIdleTime {
			conn.inUse.Store(true)
			atomic.AddInt32(&hp.activeCount, 1)
			return conn
		}
		// Connection is stale or unhealthy, close it
		conn.conn.Close()
		atomic.AddInt32(&hp.totalCount, -1)
		return nil
	default:
		return nil
	}
}

// put returns a connection to the idle pool
func (hp *hostPool) put(conn *PooledConn) error {
	if !conn.IsHealthy() || conn.IdleTime() > hp.config.MaxIdleTime {
		hp.remove(conn)
		return conn.conn.Close()
	}

	atomic.AddInt32(&hp.activeCount, -1)

	select {
	case hp.idleConns <- conn:
		return nil
	default:
		// Idle pool full, close connection
		hp.remove(conn)
		return conn.conn.Close()
	}
}

// add adds a new connection to the pool
func (hp *hostPool) add(conn *PooledConn) {
	hp.mu.Lock()
	hp.conns = append(hp.conns, conn)
	hp.mu.Unlock()
	atomic.AddInt32(&hp.totalCount, 1)
}

// remove removes a connection from the pool
func (hp *hostPool) remove(conn *PooledConn) {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	for i, c := range hp.conns {
		if c == conn {
			hp.conns = append(hp.conns[:i], hp.conns[i+1:]...)
			atomic.AddInt32(&hp.totalCount, -1)
			return
		}
	}
}

// canCreate checks if a new connection can be created
func (hp *hostPool) canCreate() bool {
	total := atomic.LoadInt32(&hp.totalCount)
	return int(total) < hp.config.MaxConnsPerHost
}

// activeConnections returns the number of active connections
func (hp *hostPool) activeConnections() int {
	return int(atomic.LoadInt32(&hp.activeCount))
}

// totalConnections returns the total number of connections
func (hp *hostPool) totalConnections() int {
	return int(atomic.LoadInt32(&hp.totalCount))
}

// closeAll closes all connections
func (hp *hostPool) closeAll() {
	hp.mu.Lock()
	conns := hp.conns
	hp.conns = nil
	hp.mu.Unlock()

	// Drain idle connections
	close(hp.idleConns)
	for conn := range hp.idleConns {
		conn.conn.Close()
	}

	// Close all tracked connections
	for _, conn := range conns {
		conn.conn.Close()
	}

	atomic.StoreInt32(&hp.activeCount, 0)
	atomic.StoreInt32(&hp.totalCount, 0)
}

// ConnectionPool manages pooled connections to multiple hosts
type ConnectionPool struct {
	config      *PoolConfig
	pools       map[string]*hostPool
	poolsMu     sync.RWMutex
	closed      atomic.Bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	healthCheck HealthChecker
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PoolConfig) *ConnectionPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	pool := &ConnectionPool{
		config:   config,
		pools:    make(map[string]*hostPool),
		stopChan: make(chan struct{}),
	}

	// Start background workers
	pool.wg.Add(2)
	go pool.idleConnectionCleaner()
	go pool.healthCheckWorker()

	return pool
}

// SetHealthChecker sets the health checker
func (cp *ConnectionPool) SetHealthChecker(hc HealthChecker) {
	cp.healthCheck = hc
}

// GetConn acquires a connection for the given host
func (cp *ConnectionPool) GetConn(ctx context.Context, host string, protocol ProtocolVersion) (*PooledConn, error) {
	if cp.closed.Load() {
		return nil, ErrPoolClosed
	}

	hp := cp.getOrCreateHostPool(host)

	// Try to get an idle connection
	if conn := hp.get(); conn != nil {
		return conn, nil
	}

	// Try to create a new connection if allowed
	if hp.canCreate() {
		conn, err := cp.createConnection(ctx, host, protocol)
		if err != nil {
			return nil, err
		}
		hp.add(conn)
		conn.inUse.Store(true)
		atomic.AddInt32(&hp.activeCount, 1)
		return conn, nil
	}

	// Wait for a connection to become available
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	deadline, hasDeadline := ctx.Deadline()
	timeout := cp.config.ConnTimeout
	if hasDeadline {
		timeout = time.Until(deadline)
	}

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeoutTimer.C:
			return nil, ErrConnTimeout
		case <-ticker.C:
			if conn := hp.get(); conn != nil {
				return conn, nil
			}
		}
	}
}

// putConn returns a connection to the pool
func (cp *ConnectionPool) putConn(conn *PooledConn) error {
	if cp.closed.Load() {
		return conn.conn.Close()
	}

	hp := cp.getHostPool(conn.host)
	if hp == nil {
		return conn.conn.Close()
	}

	return hp.put(conn)
}

// getOrCreateHostPool gets or creates a host pool
func (cp *ConnectionPool) getOrCreateHostPool(host string) *hostPool {
	cp.poolsMu.RLock()
	hp, exists := cp.pools[host]
	cp.poolsMu.RUnlock()

	if exists {
		return hp
	}

	cp.poolsMu.Lock()
	defer cp.poolsMu.Unlock()

	// Double-check after acquiring write lock
	if hp, exists := cp.pools[host]; exists {
		return hp
	}

	hp = newHostPool(host, cp.config)
	cp.pools[host] = hp
	return hp
}

// getHostPool gets a host pool if it exists
func (cp *ConnectionPool) getHostPool(host string) *hostPool {
	cp.poolsMu.RLock()
	defer cp.poolsMu.RUnlock()
	return cp.pools[host]
}

// createConnection creates a new connection
func (cp *ConnectionPool) createConnection(ctx context.Context, host string, protocol ProtocolVersion) (*PooledConn, error) {
	dialer := &net.Dialer{
		Timeout: cp.config.ConnTimeout,
	}

	var conn net.Conn
	var err error

	if cp.config.TLSConfig != nil {
		conn, err = tls.DialWithDialer(dialer, "tcp", host, cp.config.TLSConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", host)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", host, err)
	}

	pooledConn := &PooledConn{
		conn:      conn,
		host:      host,
		protocol:  protocol,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		pool:      cp,
	}
	pooledConn.healthy.Store(true)

	// Get TLS state if available
	if tlsConn, ok := conn.(*tls.Conn); ok {
		state := tlsConn.ConnectionState()
		pooledConn.tlsState = &state
	}

	return pooledConn, nil
}

// idleConnectionCleaner periodically removes idle connections
func (cp *ConnectionPool) idleConnectionCleaner() {
	defer cp.wg.Done()

	ticker := time.NewTicker(cp.config.IdleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cp.stopChan:
			return
		case <-ticker.C:
			cp.cleanIdleConnections()
		}
	}
}

// cleanIdleConnections removes stale idle connections
func (cp *ConnectionPool) cleanIdleConnections() {
	cp.poolsMu.RLock()
	pools := make([]*hostPool, 0, len(cp.pools))
	for _, hp := range cp.pools {
		pools = append(pools, hp)
	}
	cp.poolsMu.RUnlock()

	for _, hp := range pools {
		hp.mu.RLock()
		conns := make([]*PooledConn, len(hp.conns))
		copy(conns, hp.conns)
		hp.mu.RUnlock()

		for _, conn := range conns {
			if !conn.inUse.Load() && conn.IdleTime() > cp.config.MaxIdleTime {
				hp.remove(conn)
				conn.conn.Close()
			}
		}
	}
}

// healthCheckWorker periodically health checks connections
func (cp *ConnectionPool) healthCheckWorker() {
	defer cp.wg.Done()

	if cp.config.HealthCheckInterval == 0 {
		return
	}

	ticker := time.NewTicker(cp.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cp.stopChan:
			return
		case <-ticker.C:
			cp.performHealthChecks()
		}
	}
}

// performHealthChecks runs health checks on all connections
func (cp *ConnectionPool) performHealthChecks() {
	if cp.healthCheck == nil {
		return
	}

	cp.poolsMu.RLock()
	pools := make([]*hostPool, 0, len(cp.pools))
	for _, hp := range cp.pools {
		pools = append(pools, hp)
	}
	cp.poolsMu.RUnlock()

	for _, hp := range pools {
		hp.mu.RLock()
		conns := make([]*PooledConn, len(hp.conns))
		copy(conns, hp.conns)
		hp.mu.RUnlock()

		for _, conn := range conns {
			if !conn.inUse.Load() {
				ctx, cancel := context.WithTimeout(context.Background(), cp.config.HealthCheckTimeout)
				if err := cp.healthCheck.Check(ctx, conn); err != nil {
					conn.MarkUnhealthy()
					hp.remove(conn)
					conn.conn.Close()
				}
				cancel()
			}
		}
	}
}

// Stats returns pool statistics
func (cp *ConnectionPool) Stats() PoolStats {
	cp.poolsMu.RLock()
	defer cp.poolsMu.RUnlock()

	stats := PoolStats{
		Hosts: make(map[string]HostStats),
	}

	for host, hp := range cp.pools {
		stats.Hosts[host] = HostStats{
			Total:  hp.totalConnections(),
			Active: hp.activeConnections(),
			Idle:   hp.totalConnections() - hp.activeConnections(),
		}
		stats.TotalConns += hp.totalConnections()
		stats.ActiveConns += hp.activeConnections()
		stats.IdleConns += hp.totalConnections() - hp.activeConnections()
	}

	return stats
}

// PoolStats contains pool statistics
type PoolStats struct {
	TotalConns  int
	ActiveConns int
	IdleConns   int
	Hosts       map[string]HostStats
}

// HostStats contains per-host statistics
type HostStats struct {
	Total  int
	Active int
	Idle   int
}

// Close closes the connection pool
func (cp *ConnectionPool) Close() error {
	if !cp.closed.CompareAndSwap(false, true) {
		return ErrPoolClosed
	}

	close(cp.stopChan)
	cp.wg.Wait()

	cp.poolsMu.Lock()
	defer cp.poolsMu.Unlock()

	for _, hp := range cp.pools {
		hp.closeAll()
	}

	cp.pools = nil
	return nil
}
