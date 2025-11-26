package http2

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// shardedStreamMap provides concurrent access to streams with reduced lock contention
type shardedStreamMap struct {
	shards    [16]*streamShard
	shardMask uint32
}

// streamShard is a single shard of the stream map
type streamShard struct {
	streams map[uint32]*Stream
	mu      sync.RWMutex
}

// newShardedStreamMap creates a new sharded stream map
func newShardedStreamMap() *shardedStreamMap {
	ssm := &shardedStreamMap{
		shardMask: 15, // 16 shards - 1 for masking
	}
	for i := range ssm.shards {
		ssm.shards[i] = &streamShard{
			streams: make(map[uint32]*Stream),
		}
	}
	return ssm
}

// getShard returns the shard for a given stream ID
func (ssm *shardedStreamMap) getShard(streamID uint32) *streamShard {
	return ssm.shards[streamID&ssm.shardMask]
}

// Get retrieves a stream by ID
func (ssm *shardedStreamMap) Get(streamID uint32) (*Stream, bool) {
	shard := ssm.getShard(streamID)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	stream, ok := shard.streams[streamID]
	return stream, ok
}

// Set adds or updates a stream
func (ssm *shardedStreamMap) Set(streamID uint32, stream *Stream) {
	shard := ssm.getShard(streamID)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.streams[streamID] = stream
}

// Delete removes a stream
func (ssm *shardedStreamMap) Delete(streamID uint32) {
	shard := ssm.getShard(streamID)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	delete(shard.streams, streamID)
}

// Range iterates over all streams
func (ssm *shardedStreamMap) Range(fn func(streamID uint32, stream *Stream) bool) {
	for _, shard := range ssm.shards {
		shard.mu.RLock()
		for id, stream := range shard.streams {
			if !fn(id, stream) {
				shard.mu.RUnlock()
				return
			}
		}
		shard.mu.RUnlock()
	}
}

// Len returns the total number of streams
func (ssm *shardedStreamMap) Len() int {
	count := 0
	for _, shard := range ssm.shards {
		shard.mu.RLock()
		count += len(shard.streams)
		shard.mu.RUnlock()
	}
	return count
}

// Connection represents an HTTP/2 connection (RFC 7540)
// Manages multiple concurrent streams with flow control and priority scheduling
type Connection struct {
	// Stream management
	streams      *shardedStreamMap
	nextStreamID uint32 // Atomic: next stream ID to allocate
	isClient     bool   // Client or server role

	// Flow control
	flowControl *FlowController

	// Settings
	localSettings  Settings
	remoteSettings Settings
	settingsMu     sync.RWMutex

	// HPACK encoder/decoder
	encoder *Encoder
	decoder *Decoder
	hpackMu sync.Mutex

	// Connection state
	state        ConnectionState
	stateMu      sync.RWMutex
	goAwayCode   ErrorCode
	goAwayLastID uint32

	// Priority scheduling
	priorityTree *PriorityTree
	priorityMu   sync.RWMutex

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	// Statistics
	stats      ConnectionStats
	statsMu    sync.Mutex
	created    time.Time

	// Frame handling (using interface{} for flexibility)
	frameChan   chan interface{}
	frameErrChan chan error

	// Security hardening
	config              *ConnectionConfig
	totalBufferSize     int64          // Atomic: total buffer size across all streams
	priorityRateLimiter *rateLimiter   // Rate limiter for PRIORITY frames
	lastActivity        atomic.Value   // time.Time: last activity on connection
}

// ConnectionState represents the connection state
type ConnectionState uint8

const (
	ConnectionStateOpen ConnectionState = iota
	ConnectionStateGoingAway
	ConnectionStateClosed
)

// Settings holds HTTP/2 settings (RFC 7540 Section 6.5.2)
type Settings struct {
	HeaderTableSize      uint32
	EnablePush           bool
	MaxConcurrentStreams uint32
	InitialWindowSize    uint32
	MaxFrameSize         uint32
	MaxHeaderListSize    uint32
}

// DefaultSettings returns default HTTP/2 settings
func DefaultSettings() Settings {
	return Settings{
		HeaderTableSize:      4096,
		EnablePush:           true,
		MaxConcurrentStreams: 100,
		InitialWindowSize:    65535,
		MaxFrameSize:         16384,
		MaxHeaderListSize:    0, // Unlimited
	}
}

// ConnectionStats tracks connection statistics
type ConnectionStats struct {
	StreamsCreated   uint64
	StreamsClosed    uint64
	FramesSent       uint64
	FramesReceived   uint64
	BytesSent        uint64
	BytesReceived    uint64
	ErrorsSent       uint64
	ErrorsReceived   uint64
}

// NewConnection creates a new HTTP/2 connection
func NewConnection(isClient bool) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	initialStreamID := uint32(1)
	if !isClient {
		initialStreamID = 2 // Server uses even stream IDs
	}

	config := DefaultConnectionConfig()

	conn := &Connection{
		streams:             newShardedStreamMap(),
		nextStreamID:        initialStreamID,
		isClient:            isClient,
		flowControl:         NewFlowController(),
		localSettings:       DefaultSettings(),
		remoteSettings:      DefaultSettings(),
		encoder:             NewEncoder(4096),
		decoder:             NewDecoder(4096, 16*1024*1024),
		state:               ConnectionStateOpen,
		priorityTree:        NewPriorityTree(),
		ctx:                 ctx,
		cancel:              cancel,
		created:             time.Now(),
		frameChan:           make(chan interface{}, 256),
		frameErrChan:        make(chan error, 16),
		config:              config,
		priorityRateLimiter: newRateLimiter(config.MaxPriorityUpdatesPerSecond, config.PriorityRateLimitWindow),
	}

	conn.lastActivity.Store(time.Now())

	// Set connection reference on priority tree for rate limiting
	conn.priorityTree.conn = conn

	// Start idle timeout checker (security hardening)
	go conn.idleTimeoutChecker()

	return conn
}

// SetConfig sets the connection configuration
func (c *Connection) SetConfig(config *ConnectionConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	c.config = config
	c.priorityRateLimiter = newRateLimiter(config.MaxPriorityUpdatesPerSecond, config.PriorityRateLimitWindow)

	// Update stream buffer sizes
	c.streams.Range(func(_ uint32, stream *Stream) bool {
		stream.SetMaxBufferSize(config.MaxStreamBufferSize)
		return true
	})

	return nil
}

// trackBufferGrowth tracks buffer growth across all streams
// Returns error if connection buffer limit would be exceeded
func (c *Connection) trackBufferGrowth(delta int64) error {
	if c.config == nil {
		return nil
	}

	newTotal := atomic.AddInt64(&c.totalBufferSize, delta)
	if newTotal > c.config.MaxConnectionBuffer {
		// Rollback the addition
		atomic.AddInt64(&c.totalBufferSize, -delta)
		return ErrBufferSizeExceeded
	}

	return nil
}

// trackBufferShrink tracks buffer shrinkage when data is consumed
func (c *Connection) trackBufferShrink(delta int64) {
	if delta > 0 {
		atomic.AddInt64(&c.totalBufferSize, -delta)
	}
}

// idleTimeoutChecker runs in background to check for idle streams and connections
func (c *Connection) idleTimeoutChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.checkIdleStreams()
			c.checkIdleConnection()
		}
	}
}

// checkIdleStreams closes streams that have been idle too long
func (c *Connection) checkIdleStreams() {
	if c.config == nil {
		return
	}

	idleStreams := make([]uint32, 0)
	c.streams.Range(func(id uint32, stream *Stream) bool {
		if stream.IdleTime() > c.config.StreamIdleTimeout {
			idleStreams = append(idleStreams, id)
		}
		return true
	})

	// Close idle streams
	for _, id := range idleStreams {
		stream, exists := c.streams.Get(id)
		if exists {
			stream.Reset(ErrCodeCancel)
			c.CloseStream(id)
		}
	}
}

// checkIdleConnection checks if the entire connection has been idle too long
func (c *Connection) checkIdleConnection() {
	if c.config == nil {
		return
	}

	lastActivity, ok := c.lastActivity.Load().(time.Time)
	if !ok {
		return
	}

	if time.Since(lastActivity) > c.config.ConnectionIdleTimeout {
		// Close the connection
		c.Close()
	}
}

// CreateStream creates a new stream with the next available ID
func (c *Connection) CreateStream() (*Stream, error) {
	c.stateMu.RLock()
	if c.state != ConnectionStateOpen {
		c.stateMu.RUnlock()
		return nil, fmt.Errorf("connection not open")
	}
	c.stateMu.RUnlock()

	// Check concurrent streams limit
	activeStreams := c.countActiveStreams()
	maxStreams := c.remoteSettings.MaxConcurrentStreams

	if activeStreams >= maxStreams {
		return nil, fmt.Errorf("max concurrent streams exceeded: %d", maxStreams)
	}

	// Allocate stream ID
	streamID := atomic.AddUint32(&c.nextStreamID, 2) - 2

	// Verify stream ID parity matches role
	if c.isClient && streamID%2 == 0 {
		return nil, fmt.Errorf("client stream ID must be odd: %d", streamID)
	}
	if !c.isClient && streamID%2 == 1 {
		return nil, fmt.Errorf("server stream ID must be even: %d", streamID)
	}

	// Create stream
	initialWindowSize := int32(c.localSettings.InitialWindowSize)
	stream := NewStream(streamID, initialWindowSize)

	// Configure stream with connection reference and buffer limits
	stream.conn = c
	if c.config != nil {
		stream.SetMaxBufferSize(c.config.MaxStreamBufferSize)
	}

	// Add to stream map
	c.streams.Set(streamID, stream)

	// Update stats
	c.statsMu.Lock()
	c.stats.StreamsCreated++
	c.statsMu.Unlock()

	// Add to priority tree
	c.priorityMu.Lock()
	c.priorityTree.AddStream(streamID, 0, 15, false)
	c.priorityMu.Unlock()

	// Update connection activity
	c.lastActivity.Store(time.Now())

	return stream, nil
}

// GetStream retrieves a stream by ID
func (c *Connection) GetStream(streamID uint32) (*Stream, bool) {
	stream, exists := c.streams.Get(streamID)
	return stream, exists
}

// GetOrCreateStream gets an existing stream or creates it if allowed
func (c *Connection) GetOrCreateStream(streamID uint32) (*Stream, error) {
	// Try to get existing stream
	stream, exists := c.GetStream(streamID)
	if exists {
		return stream, nil
	}

	// Validate stream ID for peer-initiated streams
	if c.isClient && streamID%2 == 0 {
		// Server-initiated stream
		initialWindowSize := int32(c.localSettings.InitialWindowSize)
		stream = NewStream(streamID, initialWindowSize)

		// Configure stream with connection reference and buffer limits
		stream.conn = c
		if c.config != nil {
			stream.SetMaxBufferSize(c.config.MaxStreamBufferSize)
		}

		c.streams.Set(streamID, stream)

		c.statsMu.Lock()
		c.stats.StreamsCreated++
		c.statsMu.Unlock()

		// Update connection activity
		c.lastActivity.Store(time.Now())

		return stream, nil
	}

	if !c.isClient && streamID%2 == 1 {
		// Client-initiated stream
		initialWindowSize := int32(c.localSettings.InitialWindowSize)
		stream = NewStream(streamID, initialWindowSize)

		// Configure stream with connection reference and buffer limits
		stream.conn = c
		if c.config != nil {
			stream.SetMaxBufferSize(c.config.MaxStreamBufferSize)
		}

		c.streams.Set(streamID, stream)

		c.statsMu.Lock()
		c.stats.StreamsCreated++
		c.statsMu.Unlock()

		// Update connection activity
		c.lastActivity.Store(time.Now())

		return stream, nil
	}

	return nil, fmt.Errorf("invalid stream ID for role: %d", streamID)
}

// CloseStream closes a stream and removes it from the active set
// Now returns streams to the pool for reuse
func (c *Connection) CloseStream(streamID uint32) error {
	stream, exists := c.streams.Get(streamID)
	if !exists {
		return fmt.Errorf("stream not found: %d", streamID)
	}

	c.streams.Delete(streamID)

	// Update stats
	c.statsMu.Lock()
	c.stats.StreamsClosed++
	c.statsMu.Unlock()

	// Remove from priority tree
	c.priorityMu.Lock()
	c.priorityTree.RemoveStream(streamID)
	c.priorityMu.Unlock()

	// Return stream to pool (cancels context internally)
	putPooledStream(stream)

	return nil
}

// countActiveStreams counts active (non-closed) streams
func (c *Connection) countActiveStreams() uint32 {
	count := uint32(0)
	c.streams.Range(func(_ uint32, stream *Stream) bool {
		if !stream.IsClosed() {
			count++
		}
		return true
	})
	return count
}

// ActiveStreams returns the number of active streams
func (c *Connection) ActiveStreams() uint32 {
	return c.countActiveStreams()
}

// UpdateSettings updates connection settings
func (c *Connection) UpdateSettings(settings Settings) error {
	c.settingsMu.Lock()
	defer c.settingsMu.Unlock()

	// Update initial window size affects existing streams
	// RFC 7540 Section 6.9.2: Must adjust all stream windows by delta
	if settings.InitialWindowSize != c.localSettings.InitialWindowSize {
		delta := int32(settings.InitialWindowSize) - int32(c.localSettings.InitialWindowSize)

		var updateErr error
		c.streams.Range(func(_ uint32, stream *Stream) bool {
			if delta > 0 {
				// Increase window size
				if err := stream.IncrementSendWindow(delta); err != nil {
					updateErr = err
					return false
				}
			} else if delta < 0 {
				// Decrease window size (can go negative per RFC 7540 Section 6.9.2)
				stream.windowMu.Lock()
				newWindow := stream.sendWindow + delta

				// Check for underflow (more negative than -MaxWindowSize)
				if newWindow < -MaxWindowSize {
					stream.windowMu.Unlock()
					updateErr = ErrWindowUnderflow
					return false
				}

				stream.sendWindow = newWindow
				stream.windowMu.Unlock()
			}
			return true
		})

		if updateErr != nil {
			return updateErr
		}
	}

	// Update flow control max frame size
	if settings.MaxFrameSize != c.localSettings.MaxFrameSize {
		if err := c.flowControl.SetMaxFrameSize(settings.MaxFrameSize); err != nil {
			return err
		}
	}

	// Update HPACK table size
	if settings.HeaderTableSize != c.localSettings.HeaderTableSize {
		c.hpackMu.Lock()
		c.encoder.SetMaxDynamicTableSize(settings.HeaderTableSize)
		c.decoder.SetMaxDynamicTableSize(settings.HeaderTableSize)
		c.hpackMu.Unlock()
	}

	c.localSettings = settings
	return nil
}

// RemoteSettings returns the remote peer's settings
func (c *Connection) RemoteSettings() Settings {
	c.settingsMu.RLock()
	defer c.settingsMu.RUnlock()

	return c.remoteSettings
}

// SetRemoteSettings updates the remote peer's settings
func (c *Connection) SetRemoteSettings(settings Settings) {
	c.settingsMu.Lock()
	defer c.settingsMu.Unlock()

	c.remoteSettings = settings
}

// GoAway initiates graceful connection shutdown
func (c *Connection) GoAway(lastStreamID uint32, code ErrorCode) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.state == ConnectionStateClosed {
		return fmt.Errorf("connection already closed")
	}

	c.state = ConnectionStateGoingAway
	c.goAwayLastID = lastStreamID
	c.goAwayCode = code

	// Cancel context to signal shutdown
	c.cancel()

	return nil
}

// Close closes the connection and all streams
func (c *Connection) Close() error {
	c.stateMu.Lock()
	if c.state == ConnectionStateClosed {
		c.stateMu.Unlock()
		return nil
	}

	c.state = ConnectionStateClosed
	c.stateMu.Unlock()

	// Cancel context
	c.cancel()

	// Close all streams
	streams := make([]*Stream, 0)
	c.streams.Range(func(_ uint32, stream *Stream) bool {
		streams = append(streams, stream)
		return true
	})

	// Clear all shards
	c.streams = newShardedStreamMap()

	// Return all streams to pool
	for _, stream := range streams {
		putPooledStream(stream)
	}

	// Close channels
	close(c.frameChan)
	close(c.frameErrChan)

	return nil
}

// IsClosed returns true if the connection is closed
func (c *Connection) IsClosed() bool {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()

	return c.state == ConnectionStateClosed
}

// Context returns the connection context
func (c *Connection) Context() context.Context {
	return c.ctx
}

// Stats returns connection statistics
func (c *Connection) Stats() ConnectionStats {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	return c.stats
}

// EncodeHeaders encodes headers using HPACK
func (c *Connection) EncodeHeaders(headers []HeaderField) []byte {
	c.hpackMu.Lock()
	defer c.hpackMu.Unlock()

	return c.encoder.Encode(headers)
}

// DecodeHeaders decodes headers using HPACK
func (c *Connection) DecodeHeaders(encoded []byte) ([]HeaderField, error) {
	c.hpackMu.Lock()
	defer c.hpackMu.Unlock()

	return c.decoder.Decode(encoded)
}

// FlowController returns the connection's flow controller
func (c *Connection) FlowController() *FlowController {
	return c.flowControl
}

// PriorityTree represents a stream priority tree (RFC 7540 Section 5.3)
type PriorityTree struct {
	streams map[uint32]*PriorityNode
	mu      sync.RWMutex
	conn    *Connection // Parent connection for rate limiting
}

// PriorityNode represents a node in the priority tree
type PriorityNode struct {
	streamID   uint32
	weight     uint8
	dependency uint32
	exclusive  bool
	children   []uint32
}

// NewPriorityTree creates a new priority tree
func NewPriorityTree() *PriorityTree {
	return &PriorityTree{
		streams: make(map[uint32]*PriorityNode),
	}
}

// AddStream adds a stream to the priority tree
func (pt *PriorityTree) AddStream(streamID, dependency uint32, weight uint8, exclusive bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	node := &PriorityNode{
		streamID:   streamID,
		weight:     weight,
		dependency: dependency,
		exclusive:  exclusive,
		children:   make([]uint32, 0),
	}

	pt.streams[streamID] = node

	// Update parent's children
	if dependency != 0 {
		if parent, exists := pt.streams[dependency]; exists {
			parent.children = append(parent.children, streamID)
		}
	}
}

// RemoveStream removes a stream from the priority tree
func (pt *PriorityTree) RemoveStream(streamID uint32) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	node, exists := pt.streams[streamID]
	if !exists {
		return
	}

	// Remove from parent's children
	if node.dependency != 0 {
		if parent, exists := pt.streams[node.dependency]; exists {
			for i, childID := range parent.children {
				if childID == streamID {
					parent.children = append(parent.children[:i], parent.children[i+1:]...)
					break
				}
			}
		}
	}

	// Reparent children to this stream's parent
	for _, childID := range node.children {
		if child, exists := pt.streams[childID]; exists {
			child.dependency = node.dependency

			// Add to new parent's children
			if node.dependency != 0 {
				if newParent, exists := pt.streams[node.dependency]; exists {
					newParent.children = append(newParent.children, childID)
				}
			}
		}
	}

	delete(pt.streams, streamID)
}

// UpdatePriority updates a stream's priority
// Returns error if cycle detected or stream tries to depend on itself (RFC 7540 Section 5.3.1)
func (pt *PriorityTree) UpdatePriority(streamID, dependency uint32, weight uint8, exclusive bool) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Check rate limit for PRIORITY frame updates (security hardening)
	if pt.conn != nil && pt.conn.priorityRateLimiter != nil {
		if !pt.conn.priorityRateLimiter.allow() {
			return ErrRateLimitExceeded
		}
	}

	node, exists := pt.streams[streamID]
	if !exists {
		return nil // Stream doesn't exist, nothing to do
	}

	// RFC 7540 Section 5.3.1: A stream cannot depend on itself
	if streamID == dependency {
		return ErrStreamSelfDependency
	}

	// RFC 7540 Section 5.3.1: Detect dependency cycles
	// Traverse the dependency chain to ensure no cycle is created
	if dependency != 0 {
		visited := make(map[uint32]bool)
		current := dependency
		for current != 0 {
			if current == streamID {
				// Cycle detected! Break it by making streamID depend on root (0)
				// This follows RFC 7540 Section 5.3.1: "If a stream is made dependent
				// on one of its own dependencies, the formerly dependent stream is
				// first moved to be dependent on the reprioritized stream's previous parent."
				dependency = 0
				break
			}
			if visited[current] {
				// Existing cycle in the tree, break out
				return ErrPriorityCycleDetected
			}
			visited[current] = true
			if currentNode, exists := pt.streams[current]; exists {
				current = currentNode.dependency
			} else {
				break
			}
		}
	}

	// Remove from old parent
	if node.dependency != 0 && node.dependency != dependency {
		if oldParent, exists := pt.streams[node.dependency]; exists {
			for i, childID := range oldParent.children {
				if childID == streamID {
					oldParent.children = append(oldParent.children[:i], oldParent.children[i+1:]...)
					break
				}
			}
		}
	}

	// Update node
	node.weight = weight
	node.dependency = dependency
	node.exclusive = exclusive

	// Add to new parent
	if dependency != 0 {
		if parent, exists := pt.streams[dependency]; exists {
			if exclusive {
				// Move parent's other children under this stream
				for _, childID := range parent.children {
					if childID != streamID {
						if child, exists := pt.streams[childID]; exists {
							child.dependency = streamID
							node.children = append(node.children, childID)
						}
					}
				}
				parent.children = []uint32{streamID}
			} else {
				parent.children = append(parent.children, streamID)
			}
		}
	}

	return nil
}

// CalculateWeight calculates the effective weight for scheduling
func (pt *PriorityTree) CalculateWeight(streamID uint32) uint32 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	node, exists := pt.streams[streamID]
	if !exists {
		return 16 // Default weight
	}

	// Weight is 1-256, stored as 0-255
	return uint32(node.weight) + 1
}

// CleanupIdleStreams removes closed streams from the tree
func (pt *PriorityTree) CleanupIdleStreams(conn *Connection, maxIdleTime time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	toRemove := make([]uint32, 0)

	for streamID := range pt.streams {
		stream, exists := conn.GetStream(streamID)
		if !exists || stream.IsClosed() || stream.IdleTime() > maxIdleTime {
			toRemove = append(toRemove, streamID)
		}
	}

	for _, streamID := range toRemove {
		pt.mu.Unlock()
		pt.RemoveStream(streamID)
		pt.mu.Lock()
	}
}

// SendFrame sends a frame (to be implemented with actual I/O)
func (c *Connection) SendFrame(frame interface{}) error {
	c.stateMu.RLock()
	if c.state == ConnectionStateClosed {
		c.stateMu.RUnlock()
		return io.EOF
	}
	c.stateMu.RUnlock()

	// Update stats
	c.statsMu.Lock()
	c.stats.FramesSent++
	c.statsMu.Unlock()

	// In a real implementation, this would write to the underlying connection
	// For now, we'll just queue it
	select {
	case c.frameChan <- frame:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

// ReceiveFrame receives a frame (to be implemented with actual I/O)
func (c *Connection) ReceiveFrame() (interface{}, error) {
	select {
	case frame := <-c.frameChan:
		c.statsMu.Lock()
		c.stats.FramesReceived++
		c.statsMu.Unlock()
		return frame, nil
	case err := <-c.frameErrChan:
		return nil, err
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	}
}
