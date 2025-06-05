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

// Package multiplexed provides multiplexed pool implementation.
package multiplexed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"

	"trpc.group/trpc-go/trpc-go/codec"
	inet "trpc.group/trpc-go/trpc-go/internal/net"
	"trpc.group/trpc-go/trpc-go/internal/packetbuffer"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/internal/queue"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
)

// DefaultMultiplexedPool is the default multiplexed implementation.
var DefaultMultiplexedPool = New()

const (
	defaultBufferSize        = 128 * 1024
	defaultConnNumberPerHost = 2
	defaultSendQueueSize     = 1024
	defaultDialTimeout       = time.Second
	maxBufferSize            = 65535

	defaultMaxReconnectCount = 10
	defaultInitialBackoff    = 5 * time.Millisecond
	defaultMaxBackoff        = 50 * time.Millisecond
	// defaultReconnectCountResetInterval is twice the expected total reconnect backoff time,
	// i.e. 2 * \sum_{i=1}^{maxReconnectCount}(i*initialBackoff).
	defaultReconnectCountResetInterval = 5 * time.Millisecond * (1 + 10) * 10
)

var (
	// ErrFrameBuilderNil framer builder is not set.
	ErrFrameBuilderNil = errors.New("framer builder is nil")
	// ErrDecoderNil does not implement Decoder.
	ErrDecoderNil = errors.New("framer do not implement Decoder interface")
	// ErrRecvQueueFull receive queue full.
	ErrRecvQueueFull = errors.New("virtual connection's recv queue is full")
	// ErrSendQueueFull send queue is full.
	ErrSendQueueFull = errors.New("connection's send queue is full")
	// ErrChanClose connection is closed.
	ErrChanClose = errors.New("unexpected recv chan close")
	// ErrAssertFail type assert fail.
	ErrAssertFail = errors.New("type assert fail")
	// ErrDupRequestID duplicated request id.
	ErrDupRequestID = errors.New("duplicated Request ID")
	// ErrInitPoolFail failed to initialize connection.
	ErrInitPoolFail = errors.New("init pool for specific node fail")
	// ErrWriteNotFinished write operation is not completed.
	ErrWriteNotFinished = errors.New("write not finished")
	// ErrNetworkNotSupport does not support network type.
	ErrNetworkNotSupport = errors.New("network not support")
	// ErrConnectionsHaveBeenExpelled denotes that the connections to a certain ip:port have been expelled.
	ErrConnectionsHaveBeenExpelled = errors.New("connections have been expelled")
)

// Pool is a virtual connection pool for multiplexing.
type Pool interface {
	// GetVirtualConn gets a virtual connection to the address on named network.
	GetVirtualConn(ctx context.Context, network string, address string, opts GetOptions) (VirtualConn, error)
}

// New creates a new multiplexed instance.
func New(opt ...PoolOption) *Multiplexed {
	opts := &PoolOptions{
		connectNumberPerHost: defaultConnNumberPerHost,
		sendQueueSize:        defaultSendQueueSize,
		dialTimeout:          defaultDialTimeout,
		maxReconnectCount:    defaultMaxReconnectCount,
		initialBackoff:       defaultInitialBackoff,
	}
	for _, o := range opt {
		o(opts)
	}
	// The maximum number of idle connections cannot be less than the number of pre-allocated connections.
	if opts.maxIdleConnsPerHost != 0 && opts.maxIdleConnsPerHost < opts.connectNumberPerHost {
		opts.maxIdleConnsPerHost = opts.connectNumberPerHost
	}

	if err := opts.checkReconnectParams(); err != nil {
		panic(fmt.Sprintf("fail to create a multiplexed, please verify your PoolOption: %v", err))
	}

	return &Multiplexed{
		concreteConns: make(map[string]*Connections),
		opts:          opts,
	}
}

// Multiplexed represents multiplexing.
type Multiplexed struct {
	mu sync.RWMutex
	// key(ip:port)
	//   => value(*Connections)         <-- Multiple concrete connections to a same ip:port.
	//     => (*Connection)             <-- Single concrete connection to a certain ip:port.
	//       => [](*VirtualConnection)  <-- Multiple virtual connections multiplexed on a certain concrete connection.
	concreteConns map[string]*Connections
	opts          *PoolOptions
}

// GetVirtualConn gets a virtual connection to the address on named network.
func (p *Multiplexed) GetVirtualConn(
	ctx context.Context,
	network string,
	address string,
	opts GetOptions,
) (VirtualConn, error) {
	return p.getVirtualConn(ctx, network, address, opts)
}

// Get gets the virtual connection corresponding to the multiplexer.
// Deprecated: use GetVirtualConn instead.
func (p *Multiplexed) Get(
	ctx context.Context,
	network string,
	address string,
	opts GetOptions,
) (*VirtualConnection, error) {
	return p.getVirtualConn(ctx, network, address, opts)
}

func (p *Multiplexed) getVirtualConn(
	ctx context.Context,
	network string,
	address string,
	opts GetOptions,
) (*VirtualConnection, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if err := opts.update(network, address); err != nil {
		return nil, err
	}
	return p.get(ctx, &opts)
}

func (p *Multiplexed) get(ctx context.Context, opts *GetOptions) (*VirtualConnection, error) {
	// Unlike the standard double-check, a read lock is needed here because
	// the destructor of connections might cause concurrent read and write operations on the map.
	p.mu.RLock()
	// Step 1: nodeKey(ip:port) => concrete connections.
	conns, ok := p.concreteConns[opts.nodeKey]
	p.mu.RUnlock()
	if !ok {
		p.mu.Lock()
		conns, ok = p.concreteConns[opts.nodeKey]
		if !ok {
			conns = p.newConcreteConnections(opts)
			p.concreteConns[opts.nodeKey] = conns
		}
		p.mu.Unlock()
	}

	// Step 2: concrete connections => single concrete connection.
	conn, err := conns.pickSingleConcrete(nil, opts)
	if err != nil {
		return nil, fmt.Errorf(
			"multiplexed picks single concrete connection with node key %s err: %w", opts.nodeKey, err)
	}
	// Step 3: single concrete connection => virtual connection.
	return conn.newVirtualConn(ctx, opts.virtualConnID, opts.Msg), nil
}

func (p *Multiplexed) newConcreteConnections(opts *GetOptions) *Connections {
	conns := &Connections{
		nodeKey: opts.nodeKey,
		opts:    p.opts,
		conns:   make([]*Connection, 0, p.opts.connectNumberPerHost),
		maxIdle: p.opts.maxIdleConnsPerHost,
		destructor: func() {
			p.mu.Lock()
			delete(p.concreteConns, opts.nodeKey)
			p.mu.Unlock()
		},
	}
	conns.initialize(opts)
	return conns
}

func (cs *Connections) newConn(opts *GetOptions) *Connection {
	c := &Connection{
		network:          opts.network,
		address:          opts.address,
		virtualConns:     make(map[uint32]*VirtualConnection),
		done:             make(chan struct{}),
		dropFull:         cs.opts.dropFull,
		maxVirConns:      cs.opts.maxVirConnsPerConn,
		writeBuffer:      make(chan []byte, cs.opts.sendQueueSize),
		isStream:         opts.isStream,
		isIdle:           true,
		enableIdleRemove: cs.maxIdle > 0 && cs.opts.maxVirConnsPerConn > 0,
		connsAddIdle:     func() { cs.addIdle() },
		connsSubIdle:     func() { cs.subIdle() },
		connsNeedIdleRemove: func() bool {
			return int(atomic.LoadInt32(&cs.currentIdle)) > cs.maxIdle
		},

		// Reconnect params.
		maxReconnectCount:           cs.opts.maxReconnectCount,
		initialBackoff:              cs.opts.initialBackoff,
		maxBackoff:                  cs.opts.maxBackoff,
		reconnectCountResetInterval: cs.opts.reconnectCountResetInterval,
	}
	c.destroy = func() { cs.expel(c) }
	cs.conns = append(cs.conns, c)
	cs.addIdle()
	go c.startConnect(opts, cs.opts.dialTimeout)
	return c
}

func dialTCP(timeout time.Duration, opts *GetOptions) (net.Conn, *connpool.DialOptions, error) {
	dialOpts := &connpool.DialOptions{
		Network:       opts.network,
		Address:       opts.address,
		Timeout:       timeout,
		CACertFile:    opts.CACertFile,
		TLSCertFile:   opts.TLSCertFile,
		TLSKeyFile:    opts.TLSKeyFile,
		TLSServerName: opts.TLSServerName,
		LocalAddr:     opts.LocalAddr,
	}
	conn, err := tryConnect(dialOpts)
	return conn, dialOpts, err
}

func dialUDP(opts *GetOptions) (net.PacketConn, *net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr(opts.network, opts.address)
	if err != nil {
		return nil, nil, err
	}
	const defaultLocalAddr = ":"
	localAddr := defaultLocalAddr
	if opts.LocalAddr != "" {
		localAddr = opts.LocalAddr
	}
	conn, err := net.ListenPacket(opts.network, localAddr)
	if err != nil {
		return nil, nil, err
	}
	return conn, addr, nil
}

func (cs *Connections) pickSingleConcrete(_, opts *GetOptions) (*Connection, error) {
	// The lock is always needed because the length of cs.conns may be changed in another goroutine.
	// Example cases:
	//  1. During idle removal, the length of cs.conns will be reduced.
	//  2. If max retry time is reached, the length of cs.conns will be reduced.
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.expelled {
		return nil, fmt.Errorf("node key: %s, err: %w, caused by sub errors on conns: %+v",
			cs.nodeKey, ErrConnectionsHaveBeenExpelled, cs.err)
	}
	if cs.opts.maxVirConnsPerConn == 0 {
		// The number of virtual connections on each concrete connection is unlimited, do round robin.
		cs.roundRobinIndex = (cs.roundRobinIndex + 1) % cs.opts.connectNumberPerHost
		if cs.roundRobinIndex >= len(cs.conns) {
			// Current concrete connections have been reduced below the expected number.
			// Fill with a new concrete connection.
			cs.roundRobinIndex = len(cs.conns)
			return cs.newConn(opts), nil
		}
		return cs.conns[cs.roundRobinIndex], nil
	}
	for _, c := range cs.conns {
		if c.canGetVirtualConn() {
			return c, nil
		}
	}
	return cs.newConn(opts), nil
}

func (c *Connection) canGetVirtualConn() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.virtualConns) < c.maxVirConns
}

// startConnect starts to actually execute the connection logic.
func (c *Connection) startConnect(opts *GetOptions, dialTimeout time.Duration) {
	c.builder = opts.FramerBuilder
	reader, err := c.newReader(dialTimeout, opts)
	if err != nil {
		// The first time the connection fails to be established directly fails,
		// let the upper layer trigger the next time to re-establish the connection.
		c.close(err, false)
		return
	}
	// FramerBuilder builds framer.
	framer := c.builder.New(reader)
	decoder, ok := framer.(codec.Decoder)
	if !ok {
		c.close(ErrDecoderNil, false)
		return
	}
	c.copyFrame = !codec.IsSafeFramer(framer)
	c.decoder = decoder

	go c.reader()
	go c.writer()
}

func (c *Connection) decodeUDP() (codec.TransportResponseFrame, error) {
	// Reset the packet reader before reading new data.
	c.packetReader.Reset()
	n, _, err := c.packetConn.ReadFrom(c.packetReader.Bytes())
	if err != nil {
		return nil, err
	}
	c.packetReader.Advance(n)
	// Try to decode packet.
	response, err := c.decoder.Decode()
	if err != nil {
		return nil, err
	}
	// If there is still data present, it means it is an invalid packet, just skip it.
	if c.packetReader.UnRead() > 0 {
		return nil, errors.New("remaining data in buffer")
	}

	return response, nil
}

func (c *Connection) decodeTCP() (codec.TransportResponseFrame, error) {
	return c.decoder.Decode()
}

func (c *Connection) decode() (codec.TransportResponseFrame, error) {
	if c.isStream {
		return c.decodeTCP()
	}
	return c.decodeUDP()
}

func (c *Connection) reader() {
	var lastErr error
	for {
		select {
		case <-c.done:
			return
		default:
		}
		response, err := c.decode()
		if err != nil {
			// If there is an error in tcp unpacking, it may cause problems with
			// all subsequent parsing, so it is necessary to close the reconnection.
			if c.isStream {
				lastErr = err
				report.MultiplexedTCPReconnectOnReadErr.Incr()
				log.Tracef("reconnect on read err: %+v", err)
				c.close(lastErr, c.shouldReconnect(lastErr))
				return
			}
			// UDP is processed according to a single packet, receiving an illegal
			// packet does not affect the subsequent packet processing logic,
			// and can continue to receive packets.
			log.Tracef("decode packet err: %s", err)
			continue
		}
		// virtualConnID is StreamID under streaming, and each response is RequestID,
		// all obtained through GetRequestID.
		virtualConnID := response.GetRequestID()
		c.mu.RLock()
		vc, ok := c.virtualConns[virtualConnID]
		c.mu.RUnlock()
		if !ok {
			log.Tracef("multiplex connection %s->%s received invalid streamID(virtualConnID) %d, "+
				"if it is 0, please read https://git.woa.com/trpc-go/trpc-go/issues/920 "+
				"and upgrade your stream server's trpc-go version",
				c.conn.LocalAddr(), c.conn.RemoteAddr(), virtualConnID)
			continue
		}
		vc.recv(response)
	}
}

func (c *Connection) writer() {
	var lastErr error
	for {
		select {
		case <-c.done:
			return
		case it := <-c.writeBuffer:
			if err := c.writeAll(it); err != nil {
				if c.isStream { // If tcp fails to write data, it will cause the peer to close the connection.
					lastErr = err
					report.MultiplexedTCPReconnectOnWriteErr.Incr()
					log.Tracef("reconnect on write err: %+v", err)
					c.close(lastErr, c.shouldReconnect(lastErr))
					return
				}
				// udp failed to send packets, you can continue to send packets.
				log.Tracef("multiplexed send UDP packet failed: %v", err)
				continue
			}
		}
	}
}

// Connection represents the underlying connection.
type Connection struct {
	// mu protects the concurrency safety of virtualConnections, isIdle,
	// and also protects the connection closing process.
	mu sync.RWMutex

	// done Closes when underlying connection closed.
	done        chan struct{}
	writeBuffer chan []byte

	// Maps
	virtualConns map[uint32]*VirtualConnection

	// Network and address
	address string
	network string

	// UDP specific fields
	packetReader *packetbuffer.PacketBuffer
	addr         *net.UDPAddr
	// packetConn is the underlying udp connection.
	packetConn net.PacketConn

	// TCP/Unix stream specific fields
	isStream bool
	// conn is the underlying tcp connection.
	conn       net.Conn
	dialOpts   *connpool.DialOptions
	connLocker sync.RWMutex

	// Reconnect parameters
	// initialBackoff is the initial backoff time during the first reconnection attempt.
	initialBackoff time.Duration
	// maxBackoff is the maximum backoff time between reconnection attempts.
	maxBackoff time.Duration
	// reconnectCountResetInterval is the interval after which the reconnectCount is reset.
	reconnectCountResetInterval time.Duration
	// lastReconnectTime denotes the time at which the last reconnect happens.
	lastReconnectTime time.Time
	// maxReconnectCount is the maximum number of reconnection attempts,
	// 0 means reconnect is disable.
	maxReconnectCount int
	// reconnectCount denotes the current reconnection times.
	reconnectCount int

	// Codec and framing
	decoder codec.Decoder
	builder codec.FramerBuilder

	err              error
	copyFrame        bool
	enableIdleRemove bool
	dropFull         bool
	isIdle           bool
	closed           bool
	maxVirConns      int

	destroy             func()
	connsSubIdle        func()
	connsAddIdle        func()
	connsNeedIdleRemove func() bool
}

func (cs *Connections) initialize(opts *GetOptions) {
	for i := 0; i < cs.opts.connectNumberPerHost; i++ {
		cs.newConn(opts)
	}
}

func (c *Connection) setRawConn(conn net.Conn) {
	c.connLocker.Lock()
	defer c.connLocker.Unlock()
	c.conn = conn
}

func (c *Connection) getRawConn() net.Conn {
	c.connLocker.RLock()
	defer c.connLocker.RUnlock()
	return c.conn
}

func (c *Connection) newReader(dialTimeout time.Duration, opts *GetOptions) (io.Reader, error) {
	if c.isStream {
		return c.newTCPReader(dialTimeout, opts)
	}
	return c.newUDPReader(opts)
}

func (c *Connection) newTCPReader(dialTimeout time.Duration, opts *GetOptions) (io.Reader, error) {
	conn, dialOpts, err := dialTCP(dialTimeout, opts)
	c.dialOpts = dialOpts
	if err != nil {
		return nil, err
	}
	c.setRawConn(conn)
	return codec.NewReaderSize(conn, defaultBufferSize), nil
}

func (c *Connection) newUDPReader(opts *GetOptions) (io.Reader, error) {
	conn, addr, err := dialUDP(opts)
	if err != nil {
		return nil, err
	}
	c.addr = addr
	c.packetConn = conn
	c.packetReader = packetbuffer.New(make([]byte, maxBufferSize))
	return c.packetReader, nil
}

// Connections represents a collection of concrete connections.
type Connections struct {
	nodeKey    string
	opts       *PoolOptions
	destructor func()

	// mu protects the concurrent safety of the following fields.
	mu              sync.Mutex
	conns           []*Connection
	err             error
	roundRobinIndex int
	maxIdle         int
	currentIdle     int32
	expelled        bool
}

func (cs *Connections) addIdle() {
	if cs.maxIdle > 0 {
		atomic.AddInt32(&cs.currentIdle, 1)
	}
}

func (cs *Connections) subIdle() {
	if cs.maxIdle > 0 {
		atomic.AddInt32(&cs.currentIdle, -1)
	}
}

func (cs *Connections) expel(c *Connection) {
	cs.mu.Lock()
	cs.subIdle()
	cs.conns = filterOutConnection(cs.conns, c)
	cs.err = multierror.Append(cs.err, c.err).ErrorOrNil()
	if cs.expelled || len(cs.conns) > 0 {
		cs.mu.Unlock()
		return
	}
	cs.expelled = true
	cs.mu.Unlock()
	cs.destructor()
}

func (c *Connection) newVirtualConn(
	ctx context.Context,
	virtualConnID uint32,
	msg codec.Msg,
) *VirtualConnection {
	ctx, cancel := context.WithCancel(ctx)
	vc := &VirtualConnection{
		msg:        msg,
		id:         virtualConnID,
		conn:       c,
		ctx:        ctx,
		cancelFunc: cancel,
		recvQueue:  queue.New[response](ctx.Done()),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// If connection fails to establish or reconnect, close virtual connection directly.
	if c.closed {
		vc.cancel(c.err)
		return vc
	}
	// Considering the overflow of request id or the repetition of upper-level request id,
	// you need to first read and check the request id for whether it already exists, if it exists,
	// you need to return error to the original virtual connection.
	if prevConn, ok := c.virtualConns[virtualConnID]; ok {
		prevConn.cancel(ErrDupRequestID)
	}
	c.virtualConns[virtualConnID] = vc
	if c.isIdle {
		c.isIdle = false
		c.connsSubIdle()
	}
	return vc
}

func (c *Connection) send(b []byte) error {
	// If dropfull is set, the queue is full, then discard.
	if c.dropFull {
		select {
		case c.writeBuffer <- b:
			return nil
		default:
			return ErrSendQueueFull
		}
	}
	select {
	case c.writeBuffer <- b:
		return nil
	case <-c.done:
		return c.err
	}
}

func (c *Connection) writeAll(b []byte) error {
	if c.isStream {
		return c.writeTCP(b)
	}
	return c.writeUDP(b)
}

func (c *Connection) writeUDP(b []byte) error {
	num, err := c.packetConn.WriteTo(b, c.addr)
	if err != nil {
		return err
	}
	if num != len(b) {
		return ErrWriteNotFinished
	}
	return nil
}

func (c *Connection) writeTCP(b []byte) error {
	var sentNum, num int
	var err error
	conn := c.getRawConn()
	for sentNum < len(b) {
		num, err = conn.Write(b[sentNum:])
		if err != nil {
			return err
		}
		sentNum += num
	}
	return nil
}

func (c *Connection) close(lastErr error, reconnect bool) {
	if c.isStream {
		c.closeTCP(lastErr, reconnect)
		return
	}
	c.closeUDP(lastErr)
}

func (c *Connection) closeUDP(lastErr error) {
	c.destroy()
	c.err = lastErr
	close(c.done)

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, vc := range c.virtualConns {
		vc.cancel(lastErr)
	}
}

func (c *Connection) closeTCP(lastErr error, reconnect bool) {
	if lastErr == nil {
		return
	}
	if needDestroy := c.doClose(lastErr, reconnect); needDestroy {
		c.destroy()
	}
}

func (c *Connection) doClose(lastErr error, reconnect bool) (needDestroy bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Do not use c.err != nil to judge, reconnection will not clear err.
	if c.closed {
		return false
	}
	c.closed = true
	c.err = lastErr

	// when close the `c.done` channel, all Read operations will return error,
	// so we should clean all existing connections, avoiding memory leak.
	for _, vc := range c.virtualConns {
		vc.cancel(lastErr)
	}
	c.virtualConns = make(map[uint32]*VirtualConnection)
	close(c.done)
	if conn := c.getRawConn(); conn != nil {
		conn.Close()
	}
	if reconnect && c.doReconnectBackoff() {
		return !c.reconnect()
	}
	return true
}

func tryConnect(opts *connpool.DialOptions) (net.Conn, error) {
	conn, err := connpool.Dial(opts)
	if err != nil {
		return nil, err
	}
	if c, ok := conn.(*net.TCPConn); ok {
		c.SetKeepAlive(true)
	}
	return conn, nil
}

// shouldReconnect determines if a TCP connection should be re-established based on the given error.
func (c *Connection) shouldReconnect(err error) bool {
	// UDP have no need to reconnect.
	if !c.isStream {
		return false
	}
	// If server connection is closed, there's no need to reconnect.
	if errors.Is(err, io.EOF) {
		return false
	}
	return true
}

func (c *Connection) reconnect() (success bool) {
	if c.maxReconnectCount == 0 {
		return false
	}
	for {
		conn, err := tryConnect(c.dialOpts)
		if err != nil {
			report.MultiplexedTCPReconnectErr.Incr()
			log.Tracef("reconnect fail: %+v", err)
			if !c.doReconnectBackoff() {
				// If the current number of retries is greater than the maximum number of retries,
				// doReconnectBackoff will return false, so remove the corresponding connection.
				return false // A new request will trigger a reconnection.
			}
			continue
		}
		framer := c.builder.New(codec.NewReaderSize(conn, defaultBufferSize))
		// The initialization connection logic ensures that the framer implements the codec.Decoder interface.
		// The reconnection directly ignores the type assertion result.
		c.decoder = framer.(codec.Decoder)
		c.setRawConn(conn)
		c.done = make(chan struct{})
		if !c.isIdle {
			c.isIdle = true
			c.connsAddIdle()
		}
		// Successfully reconnected, remove the closed flag and reset c.err.
		c.err = nil
		c.closed = false
		go c.reader()
		go c.writer()
		return true
	}
}

func (c *Connection) doReconnectBackoff() bool {
	if c.maxReconnectCount == 0 {
		return false
	}
	cur := time.Now()
	if !c.lastReconnectTime.IsZero() && time.Since(c.lastReconnectTime) > c.reconnectCountResetInterval {
		// Clear reconnect count if reset interval is reached.
		c.reconnectCount = 0
	}
	c.reconnectCount++
	c.lastReconnectTime = cur
	if c.reconnectCount > c.maxReconnectCount {
		log.Tracef("reconnection reaches its limit: %d", c.maxReconnectCount)
		return false
	}
	currentBackoff := time.Duration(c.reconnectCount) * c.initialBackoff
	if currentBackoff > c.maxBackoff {
		currentBackoff = c.maxBackoff
	}
	time.Sleep(currentBackoff)
	return true
}

func (c *Connection) remove(vID uint32) {
	if c.doRemove(vID) {
		c.destroy()
	}
}

func (c *Connection) doRemove(vID uint32) (needDestroy bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.virtualConns, vID)
	if c.enableIdleRemove {
		return c.idleRemove()
	}
	return false
}

func (c *Connection) idleRemove() (needDestroy bool) {
	// Determine if the current connection is free.
	if len(c.virtualConns) != 0 {
		return false
	}
	// Check if the connection has been closed.
	if c.closed {
		return false
	}
	if !c.isIdle {
		c.isIdle = true
		c.connsAddIdle()
	}
	// Determine whether the current Node idle connection exceeds the maximum value.
	if !c.connsNeedIdleRemove() {
		return false
	}
	// Close the current connection.
	c.closed = true
	close(c.done)
	if conn := c.getRawConn(); conn != nil {
		conn.Close()
	}
	// Remove the current connection from the connection set.
	return true
}

var _ VirtualConn = (*VirtualConnection)(nil)

// VirtualConn is virtual connection multiplexing on a concrete connection.
type VirtualConn interface {
	// Write writes data to the virtual connection.
	Write([]byte) error

	// Read reads a packet from virtual connection.
	Read() ([]byte, error)

	// LocalAddr returns the local network address, if known.
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address, if known.
	RemoteAddr() net.Addr

	// Close closes the virtual connection.
	// Any blocked Read or Write operations will be unblocked and return errors.
	Close()
}

// VirtualConnection multiplexes virtual connections.
type VirtualConnection struct {
	conn       *Connection
	msg        codec.Msg
	recvQueue  *queue.Queue[response]
	ctx        context.Context
	cancelFunc context.CancelFunc
	err        error

	mu     sync.RWMutex
	id     uint32
	closed uint32
}

// RemoteAddr gets the peer address of the connection.
func (vc *VirtualConnection) RemoteAddr() net.Addr {
	if vc.conn == nil {
		return nil
	}
	if !vc.conn.isStream {
		return vc.conn.addr
	}
	conn := vc.conn.getRawConn()
	if conn != nil {
		return conn.RemoteAddr()
	}
	return inet.ResolveAddress(vc.conn.network, vc.conn.address)
}

// LocalAddr gets the local address of the connection.
func (vc *VirtualConnection) LocalAddr() net.Addr {
	if vc.conn == nil {
		return nil
	}
	conn := vc.conn.getRawConn()
	if conn == nil {
		return nil
	}
	return conn.LocalAddr()
}

// recv receives the returned data.
func (vc *VirtualConnection) recv(rsp codec.TransportResponseFrame) {
	rspBuf := rsp.GetResponseBuf()
	if vc.conn.copyFrame {
		copyBuf := make([]byte, len(rspBuf))
		copy(copyBuf, rspBuf)
		rspBuf = copyBuf
	}
	vc.recvQueue.Put(response{raw: rsp, copiedBuf: rspBuf})
}

// Write writes request packet.
// Write and Read can be concurrent, multiple Write can be concurrent.
func (vc *VirtualConnection) Write(b []byte) error {
	if err := vc.loadErr(); err != nil {
		return err
	}
	select {
	case <-vc.ctx.Done():
		// clean the virtual connection when context timeout or cancelled.
		vc.Close()
		return vc.ctx.Err()
	default:
	}
	if err := vc.conn.send(b); err != nil {
		// clean the virtual connection when send fail.
		vc.Close()
		return err
	}
	return nil
}

// Read reads back the packet.
// Write and Read can be concurrent, but not concurrent Read.
func (vc *VirtualConnection) Read() ([]byte, error) {
	if err := vc.loadErr(); err != nil {
		return nil, err
	}
	rsp, ok := vc.recvQueue.Get()
	if !ok {
		vc.Close()
		if err := vc.loadErr(); err != nil {
			return nil, err
		}
		return nil, vc.ctx.Err()
	}
	if err := vc.conn.decoder.UpdateMsg(rsp.raw, vc.msg); err != nil {
		vc.Close()
		return nil, fmt.Errorf("virtual connection update message failed: %w", err)
	}
	return rsp.copiedBuf, nil
}

// Close puts connection back into the connection pool.
func (vc *VirtualConnection) Close() {
	if atomic.CompareAndSwapUint32(&vc.closed, 0, 1) {
		vc.cancel(nil)
		vc.conn.remove(vc.id)
	}
}

func (vc *VirtualConnection) loadErr() error {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.err
}

func (vc *VirtualConnection) storeErr(err error) {
	if vc.loadErr() != nil {
		return
	}
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.err = err
}

func (vc *VirtualConnection) cancel(err error) {
	vc.storeErr(err)
	vc.cancelFunc()
}

type response struct {
	raw       codec.TransportResponseFrame
	copiedBuf []byte
}

func makeNodeKey(network, address string) string {
	var key strings.Builder
	key.Grow(len(network) + len(address) + 1)
	key.WriteString(network)
	key.WriteByte('_')
	key.WriteString(address)
	return key.String()
}

func isStream(network string) (bool, error) {
	switch network {
	case protocol.TCP, protocol.TCP4, protocol.TCP6, protocol.UNIX:
		return true, nil
	case protocol.UDP, protocol.UDP4, protocol.UDP6:
		return false, nil
	default:
		return false, ErrNetworkNotSupport
	}
}

func filterOutConnection(in []*Connection, exclude *Connection) []*Connection {
	for i, v := range in {
		if v == exclude {
			in[i] = nil
			copy(in[i:], in[i+1:])
			in = in[:len(in)-1]
			return in
		}
	}
	return in
}
