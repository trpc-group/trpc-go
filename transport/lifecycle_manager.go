//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package transport

import (
	"errors"
	"hash/fnv"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"

	iatomic "trpc.group/trpc-go/trpc-go/internal/atomic"
	icontext "trpc.group/trpc-go/trpc-go/internal/context"
	ierror "trpc.group/trpc-go/trpc-go/internal/error"
	"trpc.group/trpc-go/trpc-go/log"
)

const (
	// numShardBits defines the number of bits used for sharding.
	// Using 5 bits gives us 2^5 = 32 shards.
	numShardBits = 5

	// defaultNumShards is the number of shards for connection management.
	// Using power of 2 enables efficient distribution using bitwise operations.
	// 32 shards provides good balance between parallelism and overhead.
	defaultNumShards = 1 << numShardBits

	// defaultConnChannelSize is the buffer size for new connection channel.
	// This size helps handle connection bursts without blocking.
	// Each shard has its own buffered channel of this size.
	defaultConnChannelSize = 1024

	// selectCaseStride represents the number of select cases per connection.
	// Each connection requires two select cases:
	// 1. For context done channel
	// 2. For serve done channel
	selectCaseStride = 2

	// minSelectCasesCap is the minimum capacity for selectCases slice.
	// This prevents excessive shrinking when connection count is low.
	minSelectCasesCap = 1024

	// defaultMaxConnectionsPerShard is the default maximum number of connections that can be managed per shard.
	// This helps prevent resource exhaustion by limiting the total number of connections.
	defaultMaxConnectionsPerShard = 30000
)

// maxConnectionsPerShard is the current maximum number of connections that can be managed per shard.
// This value can be modified for testing purposes.
var maxConnectionsPerShard uint32 = defaultMaxConnectionsPerShard

// tcpConnectionLifecycleManager manages TCP connection lifecycle events across multiple shards.
// It monitors connection context cancellation and completion events for graceful shutdown/restart.
// The manager uses sharding to distribute connection load and improve performance.
// Each shard runs independently to reduce contention and improve scalability.
//
// This manager is designed to reduce the excessive number of goroutines that would otherwise be needed
// for each connection. Without this manager, each connection would need a goroutine to handle cleanup
// like this:
//
//	serveDone := make(chan struct{})
//	defer close(serveDone)
//	go func() {
//		select {
//		case <-c.ctx.Done():
//			// For graceful restart, wait for connection to finish serving.
//			if errors.Is(icontext.Cause(c.ctx), ierror.GracefulRestart) {
//				<-serveDone
//			}
//		case <-serveDone:
//			// Connection finished serving normally.
//		}
//		// Close connection when no active requests remain.
//		if atomic.AddInt32(&c.activeCnt, -1) == 0 {
//			c.close()
//		}
//	}()
//
// Instead, this manager uses a sharded approach with reflect.Select to efficiently monitor
// multiple connections with minimal goroutine overhead. Each shard handles a subset of
// connections in a single goroutine.
type tcpConnectionLifecycleManager struct {
	// Array of connection lifecycle shards for distributing load.
	// Each shard manages a subset of connections independently.
	shards []*lifecycleShard

	// Bit mask for efficient shard selection via bitwise AND.
	// For example, with 32 shards, mask would be 0x1F (31).
	shardMask uint32
}

// lifecycleShard manages lifecycle events for a subset of TCP connections within a single shard.
// Each shard runs its own event loop to handle connection context and completion events independently.
// This sharding approach helps reduce contention and improves scalability.
type lifecycleShard struct {
	// inProcess tracks the number of connections currently being managed by this shard.
	// Used to enforce connection limits and prevent resource exhaustion.
	inProcess iatomic.Uint32

	// Cases for multiplexing connection events using reflect.Select.
	// Index 0 is always the new connection channel.
	// For each connection, we have 2 cases:
	// - Context done channel at odd indices (1,3,5...)
	// - Serve done channel at even indices (2,4,6...)
	selectCases []reflect.SelectCase

	// Buffered channel for new incoming connections.
	// Size is configurable via defaultConnChannelSize.
	newConns chan *tcpconn

	// Thread-safe map from select case indices to connection objects.
	// Maps both context done and serve done indices to the same connection.
	// Using sync.Map to avoid lock contention during concurrent access.
	connIndex sync.Map
}

// defaultManager is the global instance of TCP connection lifecycle manager.
// It handles all TCP connection lifecycle events for the server.
// Created at package initialization time to ensure single instance.
var defaultManager = newTCPConnectionLifecycleManager()

// addConnection adds a new TCP connection to the default manager for lifecycle management.
// It protects against nil connections and delegates to the default manager instance.
// This is the main entry point for connection lifecycle management.
func addConnection(conn *tcpconn) {
	if conn == nil {
		return // Protect against nil connection.
	}
	// First try connection lifecycle manager, true means success.
	if defaultManager.addConnection(conn) {
		return
	}

	// Fallback to manual connection management.
	// Use the connection's own serveDone channel.
	go func() {
		select {
		case <-conn.ctx.Done():
			if errors.Is(icontext.Cause(conn.ctx), ierror.GracefulRestart) {
				<-conn.serveDone
			}
		case <-conn.serveDone:
		}
		if atomic.AddInt32(&conn.activeCnt, -1) == 0 {
			conn.close()
		}
	}()
}

// newTCPConnectionLifecycleManager creates and initializes a new connection lifecycle manager.
// It sets up the sharding infrastructure and launches event processing goroutines.
// Each shard gets its own event loop goroutine for independent processing.
func newTCPConnectionLifecycleManager() *tcpConnectionLifecycleManager {
	mgr := &tcpConnectionLifecycleManager{
		shards:    make([]*lifecycleShard, defaultNumShards),
		shardMask: uint32(defaultNumShards - 1), // Creates mask like 0x1F for 32 shards.
	}

	// Initialize each shard with its own event loop and connection handling.
	for i := 0; i < defaultNumShards; i++ {
		shard := &lifecycleShard{
			// Pre-allocate capacity for select cases.
			// Each connection needs 2 cases (context done + serve done).
			selectCases: make([]reflect.SelectCase, 0, minSelectCasesCap),
			newConns:    make(chan *tcpconn, defaultConnChannelSize),
		}

		// First select case is always the new connection channel.
		shard.selectCases = append(shard.selectCases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(shard.newConns),
		})

		mgr.shards[i] = shard
		// Launch goroutine for shard's event processing.
		go shard.eventLoop()
	}
	return mgr
}

// addConnection adds a new TCP connection to the appropriate shard based on its address.
// If the shard's connection channel is full, the connection is closed to prevent resource exhaustion.
// The connection will be handled by the shard's event loop goroutine.
func (m *tcpConnectionLifecycleManager) addConnection(conn *tcpconn) bool {
	shard := m.getShardForConnection(conn)
	if shard.inProcess.Add(1) >= maxConnectionsPerShard {
		shard.inProcess.Add(^uint32(0))
		return false
	}
	select {
	case shard.newConns <- conn: // Try non-blocking send.
	default:
		// Channel is full, ignore the connection to prevent resource exhaustion.
		// This is acceptable since the connection will be closed at the end of serve loop.
		shard.inProcess.Add(^uint32(0))
		return false
	}
	return true
}

// getShardForConnection determines which shard should handle a given connection.
// It uses FNV hash of the remote address for even distribution across shards.
// The hash is masked to get a shard index in the valid range.
func (m *tcpConnectionLifecycleManager) getShardForConnection(conn *tcpconn) *lifecycleShard {
	// FNV hash provides good distribution properties for network addresses.
	hasher := fnv.New32a()
	hasher.Write([]byte(conn.remoteAddr.String()))
	// Fast modulo for power of 2 using bitwise AND with mask.
	shardIndex := hasher.Sum32() & m.shardMask
	return m.shards[shardIndex]
}

// eventLoop processes connection lifecycle events for a single shard.
// It handles new connections, context cancellations, and connection completions.
// This runs in its own goroutine for each shard and recovers from panics.
func (s *lifecycleShard) eventLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("lifecycleShard eventLoop panic: %v\nstack: %s\n", r, debug.Stack())
		}
	}()

	// newConnSelectIndex is the index for new connection channel in selectCases.
	// Always the first case (index 0) in the select statement.
	const newConnSelectIndex = 0

	// contextDoneSelectIndex is the base index for context done channel in selectCases.
	// Used to identify context cancellation events in the select statement.
	// Context done channels are at odd indices (1,3,5...).
	const contextDoneSelectIndex = 1

	for {
		// Since all operations happen in the same goroutine, no need for mutex.
		caseIndex, value, ok := reflect.Select(s.selectCases)

		// Handle new connection events (index 0).
		if caseIndex == newConnSelectIndex {
			if !ok {
				return // Shard shutdown initiated by channel close.
			}
			conn, ok := value.Interface().(*tcpconn)
			if !ok || conn == nil {
				continue // Skip invalid connection.
			}
			s.handleNewConnection(conn)
			continue
		}

		// Handle existing connection events.
		// First retrieve the connection object for this event.
		conn, exists := s.connIndex.Load(caseIndex)
		if !exists {
			continue // Skip if connection was already removed.
		}
		tc, ok := conn.(*tcpconn)
		if !ok || tc == nil {
			continue // Skip invalid connection.
		}

		// Route event based on case index parity.
		// Odd indices are context done events.
		// Even indices are serve done events.
		if caseIndex%selectCaseStride == contextDoneSelectIndex {
			s.handleContextDone(tc, caseIndex)
		} else {
			s.handleServeDone(tc, caseIndex)
		}
	}
}

// handleNewConnection sets up monitoring for a new connection's lifecycle events.
// It adds select cases for both context cancellation and completion events.
// The connection is mapped to both event indices for later lookup.
func (s *lifecycleShard) handleNewConnection(conn *tcpconn) {
	idx := len(s.selectCases)

	// Add monitoring for both connection lifecycle events:
	// 1. Context done channel for cancellation
	// 2. Serve done channel for completion
	s.selectCases = append(s.selectCases,
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(conn.ctx.Done()),
		},
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(conn.serveDone),
		})

	// Map both event indices to the connection object.
	// This allows us to look up the connection from either event.
	s.connIndex.Store(idx, conn)
	s.connIndex.Store(idx+1, conn)
}

// handleContextDone processes context cancellation events for a connection.
// It handles graceful restart scenarios and cleans up inactive connections.
// During graceful restart, connections are preserved until serve done.
func (s *lifecycleShard) handleContextDone(conn *tcpconn, idx int) {
	// During graceful restart, preserve the connection and wait for serve done.
	if errors.Is(icontext.Cause(conn.ctx), ierror.GracefulRestart) {
		return
	}

	// For normal shutdown, clean up the connection.
	s.closeConnectionIfInactive(conn)
	s.removeConnection(idx)
}

// handleServeDone processes connection completion events.
// It cleans up the connection and removes it from the shard.
// This is called when a connection's serve loop completes.
func (s *lifecycleShard) handleServeDone(conn *tcpconn, idx int) {
	s.closeConnectionIfInactive(conn)
	s.removeConnection(idx)
}

// closeConnectionIfInactive closes a connection if it has no active requests.
// It uses atomic operations to safely check and update the active request count.
// The connection is closed only when the last reference is removed.
func (s *lifecycleShard) closeConnectionIfInactive(conn *tcpconn) {
	// Decrement active count and close if no active requests.
	// This is thread-safe due to atomic operation.
	if atomic.AddInt32(&conn.activeCnt, -1) <= 0 {
		conn.close()
	}
}

// removeConnection removes a connection's event handlers from the shard.
// It updates the select cases and connection index mappings to maintain consistency.
// This handles cleanup of both context done and serve done handlers.
func (s *lifecycleShard) removeConnection(idx int) {
	// Normalize to even index for pair removal.
	// This ensures we remove both handlers for the connection.
	baseIdx := idx - (idx+1)%selectCaseStride
	if baseIdx < 0 || baseIdx >= len(s.selectCases) {
		return // Protect against invalid indices.
	}

	// Clean up connection index mappings for both handlers.
	s.connIndex.Delete(baseIdx)
	s.connIndex.Delete(baseIdx + 1)

	// Remove select cases for both event handlers.
	s.selectCases = append(s.selectCases[:baseIdx], s.selectCases[baseIdx+selectCaseStride:]...)

	// Update indices for remaining connections to maintain consistency.
	// This shifts all higher indices down by 2 to fill the gap.
	for i := baseIdx; i < len(s.selectCases); i++ {
		if conn, ok := s.connIndex.Load(i + selectCaseStride); ok {
			s.connIndex.Delete(i + selectCaseStride)
			s.connIndex.Store(i, conn)
		}
	}

	// Shrink capacity if length is less than half of capacity.
	// But don't shrink below minSelectCasesCap.
	currentCap := cap(s.selectCases)
	currentLen := len(s.selectCases)
	if currentLen < currentCap/2 && currentCap/2 >= minSelectCasesCap {
		// Create new slice with reduced capacity but not less than minSelectCasesCap.
		newCap := currentCap / 2
		newSlice := make([]reflect.SelectCase, currentLen, newCap)
		copy(newSlice, s.selectCases)
		s.selectCases = newSlice
	}

	// Decrement the in-process counter since we've removed a connection.
	s.inProcess.Add(^uint32(0))
}
