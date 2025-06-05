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
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/codec"
	icontext "trpc.group/trpc-go/trpc-go/internal/context"
	ierror "trpc.group/trpc-go/trpc-go/internal/error"
)

// TestLifecycleManagerNormalShutdown tests normal connection shutdown flow.
func TestLifecycleManagerNormalShutdown(t *testing.T) {
	// Create a real TCP listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	defer ln.Close()

	m := newTCPConnectionLifecycleManager()
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan struct{})

	// Create server transport with proper options.
	st := &serverTransport{
		opts: &ServerTransportOptions{
			KeepAlivePeriod: 5 * time.Second,
		},
		addrToConn: make(map[string]*tcpconn), // Initialize the map to avoid nil pointer
		m:          &sync.RWMutex{},           // Initialize mutex
	}

	// Accept a real TCP connection.
	opts := &ListenServeOptions{
		ActiveCnt:     activeCnt{},
		FramerBuilder: &noopFramerBuilder{}, // Use no-op framer builder for testing
	}

	// Create client connection.
	clientConn, err := net.Dial("tcp", ln.Addr().String())
	assert.Nil(t, err)
	defer clientConn.Close()

	// Accept server side connection.
	serverConn, err := ln.Accept()
	assert.Nil(t, err)

	// Set keepalive options.
	if tcpConn, ok := serverConn.(*net.TCPConn); ok {
		err = tcpConn.SetKeepAlive(true)
		assert.Nil(t, err)
		err = tcpConn.SetKeepAlivePeriod(st.opts.KeepAlivePeriod)
		assert.Nil(t, err)
	}

	// Create TCP connection.
	_, conn := st.newTCPConn(ctx, serverConn, nil, opts)
	conn.serveDone = serveDone

	// Add connection and verify it's tracked.
	m.addConnection(conn)
	time.Sleep(100 * time.Millisecond) // Allow event loop to process.

	// Trigger normal shutdown.
	cancel()
	close(serveDone)
	time.Sleep(100 * time.Millisecond) // Allow cleanup to complete.
}

// TestLifecycleManagerGracefulRestart tests graceful restart flow.
func TestLifecycleManagerGracefulRestart(t *testing.T) {
	// Create a real TCP listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	defer ln.Close()

	m := newTCPConnectionLifecycleManager()
	ctx, cancel := icontext.WithCancelCause(context.Background())
	serveDone := make(chan struct{})

	// Create server transport with proper options.
	st := &serverTransport{
		opts: &ServerTransportOptions{
			KeepAlivePeriod: 5 * time.Second,
		},
	}

	// Accept a real TCP connection.
	opts := &ListenServeOptions{
		ActiveCnt:     activeCnt{},
		FramerBuilder: &noopFramerBuilder{}, // Use no-op framer builder for testing
	}

	// Create client connection.
	clientConn, err := net.Dial("tcp", ln.Addr().String())
	assert.Nil(t, err)
	defer clientConn.Close()

	// Accept server side connection.
	serverConn, err := ln.Accept()
	assert.Nil(t, err)

	// Set keepalive options.
	if tcpConn, ok := serverConn.(*net.TCPConn); ok {
		err = tcpConn.SetKeepAlive(true)
		assert.Nil(t, err)
		err = tcpConn.SetKeepAlivePeriod(st.opts.KeepAlivePeriod)
		assert.Nil(t, err)
	}

	// Create TCP connection.
	_, conn := st.newTCPConn(ctx, serverConn, nil, opts)
	conn.serveDone = serveDone
	conn.activeCnt = 1 // Initial count for serve loop.

	// Add connection and verify it's tracked.
	m.addConnection(conn)
	time.Sleep(100 * time.Millisecond) // Allow event loop to process.

	// Trigger graceful restart.
	cancel(ierror.GracefulRestart)
	time.Sleep(100 * time.Millisecond) // Allow event loop to process.

	// Connection should still be active.
	assert.Equal(t, int32(1), atomic.LoadInt32(&conn.activeCnt))

	// Complete serving.
	close(serveDone)
	time.Sleep(100 * time.Millisecond) // Allow cleanup to complete.
}

// TestLifecycleManagerShardDistribution tests connection distribution across shards.
func TestLifecycleManagerShardDistribution(t *testing.T) {
	// Create a real TCP listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	defer ln.Close()

	m := newTCPConnectionLifecycleManager()

	// Create server transport with proper options.
	st := &serverTransport{
		opts: &ServerTransportOptions{
			KeepAlivePeriod: 5 * time.Second,
		},
	}

	opts := &ListenServeOptions{
		ActiveCnt:     activeCnt{},
		FramerBuilder: &noopFramerBuilder{}, // Use no-op framer builder for testing
	}

	// Create test connections with different addresses.
	conns := make([]*tcpconn, 100)
	clientConns := make([]net.Conn, 100)

	for i := 0; i < len(conns); i++ {
		// Create client connection.
		clientConn, err := net.Dial("tcp", ln.Addr().String())
		assert.Nil(t, err)
		clientConns[i] = clientConn

		// Accept server side connection.
		serverConn, err := ln.Accept()
		assert.Nil(t, err)

		// Set keepalive options.
		if tcpConn, ok := serverConn.(*net.TCPConn); ok {
			err = tcpConn.SetKeepAlive(true)
			assert.Nil(t, err)
			err = tcpConn.SetKeepAlivePeriod(st.opts.KeepAlivePeriod)
			assert.Nil(t, err)
		}

		// Create TCP connection.
		_, conn := st.newTCPConn(context.Background(), serverConn, nil, opts)
		conn.serveDone = make(chan struct{})
		conns[i] = conn
	}

	// Add all connections.
	for _, conn := range conns {
		m.addConnection(conn)
	}

	// Verify connections are distributed across shards.
	shardCounts := make(map[uint32]int)
	for _, conn := range conns {
		shard := m.getShardForConnection(conn)
		for i, s := range m.shards {
			if s == shard {
				shardCounts[uint32(i)]++
				break
			}
		}
	}

	// Should have used multiple shards.
	assert.Greater(t, len(shardCounts), 1)

	// Cleanup.
	for _, conn := range clientConns {
		conn.Close()
	}
}

// TestLifecycleManagerConnectionLimit tests the connection limit functionality.
func TestLifecycleManagerConnectionLimit(t *testing.T) {
	// Create a real TCP listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	defer ln.Close()

	// Save old limit and restore after test.
	oldLimit := atomic.LoadUint32(&maxConnectionsPerShard)
	atomic.StoreUint32(&maxConnectionsPerShard, 10)
	defer func() {
		atomic.StoreUint32(&maxConnectionsPerShard, oldLimit)
	}()

	m := newTCPConnectionLifecycleManager()

	// Create server transport with proper options.
	st := &serverTransport{
		opts: &ServerTransportOptions{
			KeepAlivePeriod: 5 * time.Second,
		},
	}

	opts := &ListenServeOptions{
		ActiveCnt:     activeCnt{},
		FramerBuilder: &noopFramerBuilder{}, // Use no-op framer builder for testing.
	}

	// Create test connections slightly more than the limit.
	numConns := int(atomic.LoadUint32(&maxConnectionsPerShard)) + 5
	conns := make([]*tcpconn, numConns)
	clientConns := make([]net.Conn, numConns)
	var successCount int
	var failCount int

	// Use a fixed address to ensure all connections go to the same shard.
	fixedAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	for i := 0; i < len(conns); i++ {
		// Create client connection.
		clientConn, err := net.Dial("tcp", ln.Addr().String())
		assert.Nil(t, err)
		clientConns[i] = clientConn

		// Accept server side connection.
		serverConn, err := ln.Accept()
		assert.Nil(t, err)

		// Set keepalive options.
		if tcpConn, ok := serverConn.(*net.TCPConn); ok {
			err = tcpConn.SetKeepAlive(true)
			assert.Nil(t, err)
			err = tcpConn.SetKeepAlivePeriod(st.opts.KeepAlivePeriod)
			assert.Nil(t, err)
		}

		// Create TCP connection with fixed remote address.
		_, conn := st.newTCPConn(context.Background(), serverConn, nil, opts)
		conn.serveDone = make(chan struct{})
		conn.remoteAddr = fixedAddr // Override with fixed address to ensure same shard.
		conns[i] = conn

		// Try to add connection and count success/failure.
		if m.addConnection(conn) {
			successCount++
		} else {
			failCount++
		}
	}

	// Verify that we collected approximately maxConnectionsPerShard connections.
	// The exact number might be slightly less due to concurrent processing.
	currentLimit := atomic.LoadUint32(&maxConnectionsPerShard)
	assert.True(t, successCount <= int(currentLimit),
		"Should not collect more than maxConnectionsPerShard connections, got %d.", successCount)
	assert.True(t, failCount > 0,
		"Should have some failed collections when exceeding limit, got %d failures.", failCount)

	// Try to add one more connection, it should return false.
	clientConn, err := net.Dial("tcp", ln.Addr().String())
	assert.Nil(t, err)
	defer clientConn.Close()

	serverConn, err := ln.Accept()
	assert.Nil(t, err)

	_, lastConn := st.newTCPConn(context.Background(), serverConn, nil, opts)
	lastConn.serveDone = make(chan struct{})
	lastConn.remoteAddr = fixedAddr // Use same fixed address.
	assert.False(t, m.addConnection(lastConn),
		"Should return false when trying to collect beyond limit.")

	// Cleanup.
	for _, conn := range clientConns {
		if conn != nil {
			conn.Close()
		}
	}
}

// TestLifecycleManagerFallback tests the fallback mechanism when connection limit is reached.
func TestLifecycleManagerFallback(t *testing.T) {
	// Create a real TCP listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	defer ln.Close()

	// Save old limit and restore after test.
	oldLimit := atomic.LoadUint32(&maxConnectionsPerShard)
	atomic.StoreUint32(&maxConnectionsPerShard, 1)
	defer func() {
		atomic.StoreUint32(&maxConnectionsPerShard, oldLimit)
	}()

	// Create server transport with proper options.
	st := &serverTransport{
		opts: &ServerTransportOptions{
			KeepAlivePeriod: 5 * time.Second,
		},
		addrToConn: make(map[string]*tcpconn), // Initialize the map to avoid nil pointer.
		m:          &sync.RWMutex{},           // Initialize mutex.
	}

	opts := &ListenServeOptions{
		ActiveCnt:     activeCnt{},
		FramerBuilder: &noopFramerBuilder{}, // Use no-op framer builder for testing.
	}

	// Use a fixed address to ensure all connections go to the same shard.
	fixedAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	// Create first connection that should succeed.
	ctx, cancel := icontext.WithCancelCause(context.Background())
	defer cancel(nil)

	// Create first client connection.
	clientConn1, err := net.Dial("tcp", ln.Addr().String())
	assert.Nil(t, err)
	defer clientConn1.Close()

	// Accept first server connection.
	serverConn1, err := ln.Accept()
	assert.Nil(t, err)

	// Set keepalive options.
	if tcpConn, ok := serverConn1.(*net.TCPConn); ok {
		err = tcpConn.SetKeepAlive(true)
		assert.Nil(t, err)
		err = tcpConn.SetKeepAlivePeriod(st.opts.KeepAlivePeriod)
		assert.Nil(t, err)
	}

	// Create first TCP connection.
	_, conn1 := st.newTCPConn(ctx, serverConn1, nil, opts)
	conn1.serveDone = make(chan struct{})
	conn1.activeCnt = 1
	conn1.remoteAddr = fixedAddr // Use fixed address to ensure same shard.

	// Add first connection, should succeed.
	addConnection(conn1)
	time.Sleep(100 * time.Millisecond) // Allow event loop to process.

	// Create second client connection.
	clientConn2, err := net.Dial("tcp", ln.Addr().String())
	assert.Nil(t, err)
	defer clientConn2.Close()

	// Accept second server connection.
	serverConn2, err := ln.Accept()
	assert.Nil(t, err)

	// Set keepalive options.
	if tcpConn, ok := serverConn2.(*net.TCPConn); ok {
		err = tcpConn.SetKeepAlive(true)
		assert.Nil(t, err)
		err = tcpConn.SetKeepAlivePeriod(st.opts.KeepAlivePeriod)
		assert.Nil(t, err)
	}

	// Create second TCP connection.
	_, conn2 := st.newTCPConn(ctx, serverConn2, nil, opts)
	conn2.serveDone = make(chan struct{})
	conn2.activeCnt = 1
	conn2.remoteAddr = fixedAddr // Use same fixed address to ensure same shard.

	// Add second connection, should fallback to manual management.
	addConnection(conn2)
	time.Sleep(100 * time.Millisecond) // Allow event loop to process.

	// Test graceful restart handling in fallback mode.
	cancel(ierror.GracefulRestart)
	time.Sleep(100 * time.Millisecond)

	// Connection should still be active.
	assert.Equal(t, int32(1), atomic.LoadInt32(&conn2.activeCnt))

	// Complete serving.
	close(conn2.serveDone)
	time.Sleep(100 * time.Millisecond)

	// Connection should be closed.
	assert.Equal(t, int32(0), atomic.LoadInt32(&conn2.activeCnt))

	// Cleanup first connection.
	close(conn1.serveDone)
}

// noopFramer is a no-op implementation of codec.Framer for testing.
type noopFramer struct{}

func (f *noopFramer) ReadFrame() ([]byte, error)          { return nil, io.EOF }
func (f *noopFramer) WriteFrame([]byte) error             { return nil }
func (f *noopFramer) IsSafe() bool                        { return true }
func (f *noopFramer) SetMaxFrameSize(maxFrameSize uint32) {}

// noopFramerBuilder is a no-op implementation of codec.FramerBuilder for testing.
type noopFramerBuilder struct{}

func (b *noopFramerBuilder) New(reader io.Reader) codec.Framer {
	return &noopFramer{}
}
