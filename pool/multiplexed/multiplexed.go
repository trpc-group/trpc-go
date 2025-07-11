//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
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
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"trpc.group/trpc-go/trpc-go/internal/packetbuffer"
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
)

// The following needs to be variables according to some test cases.
var (
	initialBackoff    = 5 * time.Millisecond
	maxBackoff        = 50 * time.Millisecond
	maxReconnectCount = 10
	// reconnectCountResetInterval is twice the expected total reconnect backoff time,
	// i.e. 2 * \sum_{i=1}^{maxReconnectCount}(i*initialBackoff).
	reconnectCountResetInterval = 5 * time.Millisecond * (1 + 10) * 10
)

var (
	// ErrFrameParserNil indicates that frame parse is nil.
	ErrFrameParserNil = errors.New("frame parser is nil")
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

// Pool is a connection pool for multiplexing.
type Pool interface {
	// GetMuxConn gets a multiplexing connection to the address on named network.
	GetMuxConn(ctx context.Context, network string, address string, opts GetOptions) (MuxConn, error)
}

// New creates a new multiplexed instance.
func New(opt ...PoolOption) *Multiplexed {
	opts := &PoolOptions{
		connectNumberPerHost: defaultConnNumberPerHost,
		sendQueueSize:        defaultSendQueueSize,
		dialTimeout:          defaultDialTimeout,
	}
	for _, o := range opt {
		o(opts)
	}
	// The maximum number of idle connections cannot be less than the number of pre-allocated connections.
	if opts.maxIdleConnsPerHost != 0 && opts.maxIdleConnsPerHost < opts.connectNumberPerHost {
		opts.maxIdleConnsPerHost = opts.connectNumberPerHost
	}
	return &Multiplexed{
		concreteConns: new(sync.Map),
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
	concreteConns *sync.Map
	opts          *PoolOptions
}

// GetMuxConn gets a multiplexing connection to the address on named network.
func (p *Multiplexed) GetMuxConn(
	ctx context.Context,
	network string,
	address string,
	opts GetOptions,
) (MuxConn, error) {
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
	// Step 1: nodeKey(ip:port) => concrete connections.
	value, ok := p.concreteConns.Load(opts.nodeKey)
	if !ok {
		p.initPoolForNode(opts)
		value, ok = p.concreteConns.Load(opts.nodeKey)
		if !ok {
			return nil, ErrInitPoolFail
		}
	}
	conns, ok := value.(*Connections)
	if !ok {
		return nil, fmt.Errorf("%w, expected: *Connections, actual: %T", ErrAssertFail, value)
	}
	// Step 2: concrete connections => single concrete connection.
	conn, err := conns.pickSingleConcrete(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf(
			"multiplexed pick single concreate connection with node key %s err: %w", opts.nodeKey, err)
	}
	// Step 3: single concrete connection => virtual connection.
	return conn.newVirConn(ctx, opts.VID), nil
}

func (p *Multiplexed) initPoolForNode(opts *GetOptions) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Check again in case another goroutine has initialized the pool just ahead of us.
	if _, ok := p.concreteConns.Load(opts.nodeKey); ok {
		return
	}
	p.concreteConns.Store(opts.nodeKey, p.newConcreteConnections(opts))
}

func (p *Multiplexed) newConcreteConnections(opts *GetOptions) *Connections {
	conns := &Connections{
		nodeKey: opts.nodeKey,
		opts:    p.opts,
		conns:   make([]*Connection, 0, p.opts.connectNumberPerHost),
		maxIdle: p.opts.maxIdleConnsPerHost,
		destructor: func() {
			p.concreteConns.Delete(opts.nodeKey)
		},
	}
	conns.initialize(opts)
	return conns
}

func (cs *Connections) newConn(opts *GetOptions) *Connection {
	c := &Connection{
		network:          opts.network,
		address:          opts.address,
		virConns:         make(map[uint32]*VirtualConnection),
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

func (cs *Connections) pickSingleConcrete(ctx context.Context, opts *GetOptions) (*Connection, error) {
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
		if c.canGetVirConn() {
			return c, nil
		}
	}
	return cs.newConn(opts), nil
}

func (c *Connection) canGetVirConn() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxVirConns == 0 || // 0 means unlimited.
		len(c.virConns) < c.maxVirConns
}

// startConnect starts to actually execute the connection logic.
func (c *Connection) startConnect(opts *GetOptions, dialTimeout time.Duration) {
	c.fp = opts.FP
	if err := c.dial(dialTimeout, opts); err != nil {
		// The first time the connection fails to be established directly fails,
		// let the upper layer trigger the next time to re-establish the connection.
		c.close(err, false)
		return
	}
	go c.reading()
	go c.writing()
}

func (c *Connection) dial(timeout time.Duration, opts *GetOptions) error {
	if c.isStream {
		conn, dialOpts, err := dialTCP(timeout, opts)
		c.dialOpts = dialOpts
		if err != nil {
			return err
		}
		c.setRawConn(conn)
	} else {
		conn, addr, err := dialUDP(opts)
		if err != nil {
			return err
		}
		c.addr = addr
		c.packetConn = conn
		c.packetBuffer = packetbuffer.New(conn, maxBufferSize)
	}
	return nil
}

func (c *Connection) reading() {
	var lastErr error
	for {
		select {
		case <-c.done:
			return
		default:
		}
		vid, buf, err := c.parse()
		if err != nil {
			// If there is an error in tcp unpacking, it may cause problems with
			// all subsequent parsing, so it is necessary to close the reconnection.
			if c.isStream {
				lastErr = err
				report.MultiplexedTCPReconnectOnReadErr.Incr()
				log.Tracef("reconnect on read err: %+v", err)
				break
			}
			// udp is processed according to a single packet, receiving an illegal
			// packet does not affect the subsequent packet processing logic, and can continue to receive packets.
			log.Tracef("decode packet err: %s", err)
			continue
		}

		c.mu.RLock()
		vc, ok := c.virConns[vid]
		c.mu.RUnlock()
		if !ok {
			continue
		}
		vc.recvQueue.Put(buf)
	}
	c.close(lastErr, true)
}

func (c *Connection) writing() {
	var lastErr error
L:
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
					break L
				}
				// udp failed to send packets, you can continue to send packets.
				log.Tracef("multiplexed send UDP packet failed: %v", err)
				continue
			}
		}
	}
	c.close(lastErr, true)
}

func (c *Connection) parse() (vid uint32, buf []byte, err error) {
	if c.isStream {
		return c.fp.Parse(c.getRawConn())
	}
	defer func() {
		closeErr := c.packetBuffer.Next()
		if closeErr == nil {
			return
		}
		if err == nil {
			err = closeErr
			return
		}
		err = fmt.Errorf("parse error %w, close packet error %s", err, closeErr)
	}()
	return c.fp.Parse(c.packetBuffer)
}

// Connection represents the underlying tcp connection.
type Connection struct {
	err                 error
	address             string
	network             string
	enableIdleRemove    bool
	destroy             func()
	connsSubIdle        func()
	connsAddIdle        func()
	connsNeedIdleRemove func() bool

	// reconnectCount denotes the current reconnection times.
	reconnectCount int
	// lastReconnectTime denotes the time at which the last reconnect happens.
	lastReconnectTime time.Time

	// mu protects the concurrency safety of virtualConnections, isIdle,
	// and also protects the connection closing process.
	mu       sync.RWMutex
	virConns map[uint32]*VirtualConnection
	isIdle   bool

	fp          FrameParser
	done        chan struct{} // closed when underlying connection closed.
	writeBuffer chan []byte
	dropFull    bool
	maxVirConns int

	// udp only
	packetBuffer *packetbuffer.PacketBuffer
	addr         *net.UDPAddr
	packetConn   net.PacketConn // the underlying udp connection.

	// tcp/unix stream only
	conn       net.Conn // the underlying tcp connection.
	connLocker sync.RWMutex
	dialOpts   *connpool.DialOptions
	isStream   bool
	closed     bool
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

// Connections represents a collection of concrete connections.
type Connections struct {
	nodeKey    string
	maxIdle    int
	opts       *PoolOptions
	destructor func()

	// mu protects the concurrent safety of the following fields.
	mu              sync.Mutex
	conns           []*Connection
	currentIdle     int32
	roundRobinIndex int
	expelled        bool
	err             error
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

func (c *Connection) newVirConn(ctx context.Context, virConnID uint32) *VirtualConnection {
	ctx, cancel := context.WithCancel(ctx)
	vc := &VirtualConnection{
		id:         virConnID,
		conn:       c,
		ctx:        ctx,
		cancelFunc: cancel,
		recvQueue:  queue.New[[]byte](ctx.Done()),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// If connection fails to establish or reconnect, close virtual connection directly.
	if c.closed {
		vc.cancel(c.err)
	}
	// Considering the overflow of request id or the repetition of upper-level request id,
	// you need to first read and check the request id for whether it already exists, if it exists,
	// you need to return error to the original virtual connection.
	if prevConn, ok := c.virConns[virConnID]; ok {
		prevConn.cancel(ErrDupRequestID)
	}
	c.virConns[virConnID] = vc
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
	for _, vc := range c.virConns {
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
	for _, vc := range c.virConns {
		vc.cancel(lastErr)
	}
	c.virConns = make(map[uint32]*VirtualConnection)
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

func (c *Connection) reconnect() (success bool) {
	for {
		conn, err := tryConnect(c.dialOpts)
		if err != nil {
			report.MultiplexedTCPReconnectErr.Incr()
			log.Tracef("reconnect fail: %+v", err)
			if !c.doReconnectBackoff() { // If the current number of retries is greater than the maximum number
				// of retries, doReconnectBackoff will return false, so remove the corresponding connection.
				return false // A new request will trigger a reconnection.
			}
			continue
		}
		c.setRawConn(conn)
		c.done = make(chan struct{})
		if !c.isIdle {
			c.isIdle = true
			c.connsAddIdle()
		}
		// Successfully reconnected, remove the closed flag and reset c.err.
		c.err = nil
		c.closed = false
		go c.reading()
		go c.writing()
		return true
	}
}

func (c *Connection) doReconnectBackoff() bool {
	cur := time.Now()
	if !c.lastReconnectTime.IsZero() && c.lastReconnectTime.Add(reconnectCountResetInterval).Before(cur) {
		// Clear reconnect count if reset interval is reached.
		c.reconnectCount = 0
	}
	c.reconnectCount++
	c.lastReconnectTime = cur
	if c.reconnectCount > maxReconnectCount {
		log.Tracef("reconnection reaches its limit: %d", maxReconnectCount)
		return false
	}
	currentBackoff := time.Duration(c.reconnectCount) * initialBackoff
	if currentBackoff > maxBackoff {
		currentBackoff = maxBackoff
	}
	time.Sleep(currentBackoff)
	return true
}

func (c *Connection) remove(virConnID uint32) {
	if needDestroy := c.doRemove(virConnID); needDestroy {
		c.destroy()
	}
}

func (c *Connection) doRemove(virConnID uint32) (needDestroy bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.virConns, virConnID)
	if c.enableIdleRemove {
		return c.idleRemove()
	}
	return false
}

func (c *Connection) idleRemove() (needDestroy bool) {
	// Determine if the current connection is free.
	if len(c.virConns) != 0 {
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

var _ MuxConn = (*VirtualConnection)(nil)

// MuxConn is virtual connection multiplexing on a real connection.
type MuxConn interface {
	// Write writes data to the connection.
	Write([]byte) error

	// Read reads a packet from connection.
	Read() ([]byte, error)

	// LocalAddr returns the local network address, if known.
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address, if known.
	RemoteAddr() net.Addr

	// Close closes the connection.
	// Any blocked Read or Write operations will be unblocked and return errors.
	Close()
}

// VirtualConnection multiplexes virtual connections.
type VirtualConnection struct {
	id        uint32
	conn      *Connection
	recvQueue *queue.Queue[[]byte]

	ctx        context.Context
	cancelFunc context.CancelFunc
	closed     uint32

	err error
	mu  sync.RWMutex
}

// RemoteAddr gets the peer address of the connection.
func (vc *VirtualConnection) RemoteAddr() net.Addr {
	if !vc.conn.isStream {
		return vc.conn.addr
	}
	if vc.conn == nil {
		return nil
	}
	conn := vc.conn.getRawConn()
	if conn == nil {
		return nil
	}
	return conn.RemoteAddr()
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
	return rsp, nil
}

// Close closes the connection.
func (vc *VirtualConnection) Close() {
	if atomic.CompareAndSwapUint32(&vc.closed, 0, 1) {
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

func makeNodeKey(network, address string) string {
	var key strings.Builder
	key.Grow(len(network) + len(address) + 1)
	key.WriteString(network)
	key.WriteString("_")
	key.WriteString(address)
	return key.String()
}

func isStream(network string) (bool, error) {
	switch network {
	case "tcp", "tcp4", "tcp6", "unix":
		return true, nil
	case "udp", "udp4", "udp6":
		return false, nil
	default:
		return false, ErrNetworkNotSupport
	}
}

func filterOutConnection(in []*Connection, exclude *Connection) []*Connection {
	out := in[:0]
	for _, v := range in {
		if v != exclude {
			out = append(out, v)
		}
	}
	// If a connection is successfully removed, empty the last value of the slice to avoid memory leaks.
	for i := len(out); i < len(in); i++ {
		in[i] = nil
	}
	return out
}
