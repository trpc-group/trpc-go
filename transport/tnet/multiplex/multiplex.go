// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

// Package multiplex implements a connection pool that supports connection multiplexing.
package multiplex

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
	"golang.org/x/sync/singleflight"
	"trpc.group/trpc-go/tnet"

	"trpc.group/trpc-go/trpc-go/internal/queue"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
)

/*
	Pool, host, connection all have lock.
	The process of acquiring a lock during connection creation:
		host.mu.Lock ----> connection.mu.Lock ----> connection.mu.Unlock ----> host.mu.Unlock
	The process of acquiring a lock during connection closure:
		host.mu.Lock ----> 	host.mu.Unlock ----> connection.mu.Lock ----> connection.mu.Unlock
*/

const (
	defaultDialTimeout = 200 * time.Millisecond
)

var (
	// ErrConnClosed indicates connection is closed.
	ErrConnClosed = errors.New("connection is closed")
	// ErrDuplicateID indicates request ID already exist.
	ErrDuplicateID = errors.New("request ID already exist")
	// ErrInvalid indicates the operation is invalid.
	ErrInvalid = errors.New("it's invalid")

	errTooManyVirConns = errors.New("the number of virtual connections exceeds the limit")
)

// PoolOption represents some settings for the multiplex pool.
type PoolOption struct {
	dialTimeout                  time.Duration
	maxConcurrentVirConnsPerConn int
	enableMetrics                bool
}

// OptPool is function to modify PoolOption.
type OptPool func(*PoolOption)

// WithDialTimeout returns an OptPool which sets dial timeout.
func WithDialTimeout(timeout time.Duration) OptPool {
	return func(o *PoolOption) {
		o.dialTimeout = timeout
	}
}

// WithMaxConcurrentVirConnsPerConn returns an OptPool which sets the number
// of concurrent virtual connections per connection.
func WithMaxConcurrentVirConnsPerConn(max int) OptPool {
	return func(o *PoolOption) {
		o.maxConcurrentVirConnsPerConn = max
	}
}

// WithEnableMetrics returns an OptPool which enable metrics.
func WithEnableMetrics() OptPool {
	return func(o *PoolOption) {
		o.enableMetrics = true
	}
}

// NewPool creates a new multiplex pool, which uses dialFunc to dial new connections.
func NewPool(dialFunc connpool.DialFunc, opt ...OptPool) multiplexed.Pool {
	opts := &PoolOption{
		dialTimeout: defaultDialTimeout,
	}
	for _, o := range opt {
		o(opts)
	}
	m := &pool{
		dialFunc:                     dialFunc,
		dialTimeout:                  opts.dialTimeout,
		maxConcurrentVirConnsPerConn: opts.maxConcurrentVirConnsPerConn,
		hosts:                        make(map[string]*host),
	}
	if opts.enableMetrics {
		go m.metrics()
	}
	return m
}

var _ multiplexed.Pool = (*pool)(nil)

type pool struct {
	dialFunc                     connpool.DialFunc
	dialTimeout                  time.Duration
	maxConcurrentVirConnsPerConn int
	hosts                        map[string]*host // key is network+address
	mu                           sync.RWMutex
}

// GetMuxConn gets a multiplexing connection to the address on named network.
// Multiple MuxConns can multiplex on a real connection.
func (p *pool) GetMuxConn(
	ctx context.Context,
	network string,
	address string,
	opts multiplexed.GetOptions,
) (multiplexed.MuxConn, error) {
	if opts.FP == nil {
		return nil, errors.New("frame parser is not provided")
	}
	host := p.getHost(network, address, opts)

	// Rlock here to make sure that host has not been closed. If host is closed, rLock
	// will return false. And it also avoids reading host.conns while it is being modified.
	if !host.mu.rLock() {
		return nil, ErrConnClosed
	}
	virConn, err := newVirConn(ctx, host.conns, opts.VID, isClosedOrFull)
	if virConn != nil || err != nil {
		host.mu.rUnlock()
		return virConn, err
	}
	host.mu.rUnlock()

	for {
		// Lock here to ensure that the connection being created is not missed when reading host.conns,
		// because singleflightDial will lock host.mu before adding the new connection to host.conns asynchronously.
		if !host.mu.lock() {
			return nil, ErrConnClosed
		}
		virConn, err = newVirConn(ctx, host.conns, opts.VID, isClosedOrFull)
		if virConn != nil || err != nil {
			host.mu.unlock()
			return virConn, err
		}
		// if all connections are closed or can't take more virtual connection, create one.
		dialing := host.singleflightDial()
		host.mu.unlock()

		conn, err := waitDialing(ctx, dialing)
		if err != nil {
			return nil, err
		}
		// create new connection when the number of virtual connections exceeds the limit.
		virConn, err = newVirConn(ctx, []*connection{conn}, opts.VID, isFull)
		if virConn != nil || err != nil {
			return virConn, err
		}
	}
}

func (p *pool) getHost(network string, address string, opts multiplexed.GetOptions) *host {
	hostName := strings.Join([]string{network, address}, "_")
	p.mu.RLock()
	if h, ok := p.hosts[hostName]; ok {
		p.mu.RUnlock()
		return h
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if h, ok := p.hosts[hostName]; ok {
		return h
	}
	h := &host{
		network:  network,
		address:  address,
		hostName: hostName,
		dialOpts: dialOption{
			fp:            opts.FP,
			localAddr:     opts.LocalAddr,
			caCertFile:    opts.CACertFile,
			tlsCertFile:   opts.TLSCertFile,
			tlsKeyFile:    opts.TLSKeyFile,
			tlsServerName: opts.TLSServerName,
			dialTimeout:   p.dialTimeout,
		},
		dialFunc:                     p.dialFunc,
		maxConcurrentVirConnsPerConn: p.maxConcurrentVirConnsPerConn,
	}
	h.deleteHostFromPool = func() {
		p.deleteHost(h)
	}
	p.hosts[hostName] = h
	return h
}

func (p *pool) deleteHost(h *host) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.hosts, h.hostName)
}

func (p *pool) metrics() {
	for {
		p.mu.RLock()
		hostCopied := make([]*host, 0, len(p.hosts))
		for _, host := range p.hosts {
			hostCopied = append(hostCopied, host)
		}
		p.mu.RUnlock()
		for _, host := range hostCopied {
			host.metrics()
		}
		time.Sleep(3 * time.Second)
	}
}

type dialOption struct {
	fp            multiplexed.FrameParser
	localAddr     string
	dialTimeout   time.Duration
	caCertFile    string
	tlsCertFile   string
	tlsKeyFile    string
	tlsServerName string
}

// host manages all connections to the same network and address.
type host struct {
	network                      string
	address                      string
	hostName                     string
	dialOpts                     dialOption
	dialFunc                     connpool.DialFunc
	sfg                          singleflight.Group
	deleteHostFromPool           func()
	mu                           stateRWMutex
	conns                        []*connection
	maxConcurrentVirConnsPerConn int
}

func (h *host) singleflightDial() <-chan singleflight.Result {
	ch := h.sfg.DoChan(h.hostName, func() (connection interface{}, err error) {
		rawConn, err := h.dialFunc(&connpool.DialOptions{
			Network:       h.network,
			Address:       h.address,
			Timeout:       h.dialOpts.dialTimeout,
			LocalAddr:     h.dialOpts.localAddr,
			CACertFile:    h.dialOpts.caCertFile,
			TLSCertFile:   h.dialOpts.tlsCertFile,
			TLSKeyFile:    h.dialOpts.tlsKeyFile,
			TLSServerName: h.dialOpts.tlsServerName,
		})
		if err != nil {
			return nil, err
		}
		defer func() {
			if err != nil {
				rawConn.Close()
			}
		}()
		conn, err := h.wrapRawConn(rawConn, h.dialOpts.fp)
		if err != nil {
			return nil, err
		}
		// storeConn will call h.mu.Lock
		if err := h.storeConn(conn); err != nil {
			return nil, fmt.Errorf("store connection failed, %w", err)
		}
		return conn, nil
	})
	return ch
}

func waitDialing(ctx context.Context, dialing <-chan singleflight.Result) (*connection, error) {
	select {
	case result := <-dialing:
		return expandSFResult(result)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (h *host) wrapRawConn(rawConn net.Conn, fp multiplexed.FrameParser) (*connection, error) {
	// TODO: support tls
	tc, ok := rawConn.(tnet.Conn)
	if !ok {
		return nil, errors.New("dialed connection must implements tnet.Conn")
	}

	c := &connection{
		rawConn:               tc,
		fp:                    fp,
		idToVirConn:           newShardMap(defaultShardSize),
		maxConcurrentVirConns: h.maxConcurrentVirConnsPerConn,
	}
	c.deleteConnFromHost = func() {
		if isLastConn := h.deleteConn(c); isLastConn {
			h.deleteHostFromPool()
		}
	}
	// TODO: support closing idle connections
	c.rawConn.SetOnRequest(c.onRequest)
	c.rawConn.SetOnClosed(func(tnet.Conn) error {
		c.close(ErrConnClosed)
		return nil
	})
	return c, nil
}

func (h *host) loadAllConns() ([]*connection, error) {
	if !h.mu.rLock() {
		return nil, ErrConnClosed
	}
	defer h.mu.rUnlock()
	conns := make([]*connection, len(h.conns))
	copy(conns, h.conns)
	return conns, nil
}

func (h *host) storeConn(conn *connection) error {
	if !h.mu.lock() {
		return ErrConnClosed
	}
	defer h.mu.unlock()
	h.conns = append(h.conns, conn)
	return nil
}

func (h *host) deleteConn(conn *connection) (isLastConn bool) {
	if !h.mu.lock() {
		return false
	}
	defer h.mu.unlock()
	h.conns = filterOutConn(h.conns, conn)
	// close host if the last conn is deleted
	if len(h.conns) == 0 {
		h.mu.closeLocked()
		return true
	}
	return false
}

func (h *host) metrics() {
	conns, err := h.loadAllConns()
	if err != nil {
		return
	}
	var virConnNum uint32
	for _, conn := range conns {
		virConnNum += conn.idToVirConn.length()
	}
	metrics.Gauge(strings.Join([]string{"trpc.MuxConcurrentConnections", h.network, h.address}, ".")).
		Set(float64(len(conns)))
	metrics.Gauge(strings.Join([]string{"trpc.MuxConcurrentVirConns", h.network, h.address}, ".")).
		Set(float64(virConnNum))
	log.Debugf("tnet multiplex status: network: %s, address: %s, connections number: %d,"+
		"concurrent virtual connection number: %d\n", h.network, h.address, len(conns), virConnNum)
}

func expandSFResult(result singleflight.Result) (*connection, error) {
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Val.(*connection), nil
}

// connection wraps the underlying tnet.Conn, and manages many virtualConnections.
type connection struct {
	rawConn               tnet.Conn
	deleteConnFromHost    func()
	fp                    multiplexed.FrameParser
	isClosed              atomic.Bool
	mu                    stateRWMutex
	idToVirConn           *shardMap
	maxConcurrentVirConns int
}

func (c *connection) onRequest(conn tnet.Conn) error {
	vid, buf, err := c.fp.Parse(conn)
	if err != nil {
		c.close(err)
		return err
	}
	vc, ok := c.idToVirConn.load(vid)
	// If the virConn corresponding to the id cannot be found,
	// the virConn has been closed and the current response is discarded.
	if !ok {
		return nil
	}
	vc.recvQueue.Put(buf)
	return nil
}

func (c *connection) canTakeNewVirConn() bool {
	return c.maxConcurrentVirConns == 0 || c.idToVirConn.length() < uint32(c.maxConcurrentVirConns)
}

func (c *connection) close(cause error) {
	if !c.isClosed.CAS(false, true) {
		return
	}
	c.deleteConnFromHost()
	c.deleteAllVirConn(cause)
	c.rawConn.Close()
}

func (c *connection) deleteAllVirConn(cause error) {
	if !c.mu.lock() {
		return
	}
	defer c.mu.unlock()
	c.mu.closeLocked()
	for _, vc := range c.idToVirConn.loadAll() {
		vc.notifyRead(cause)
	}
	c.idToVirConn.reset()
}

func (c *connection) newVirConn(ctx context.Context, vid uint32) (*virtualConnection, error) {
	if !c.mu.rLock() {
		return nil, ErrConnClosed
	}
	defer c.mu.rUnlock()
	if !c.rawConn.IsActive() {
		return nil, ErrConnClosed
	}
	// CanTakeNewVirConn and loadOrStore are not atomic, which may cause
	// the actual concurrent virConn numbers to exceed the limit max value.
	// Implementing atomic functions requires higher lock granularity,
	// which affects performance.
	if !c.canTakeNewVirConn() {
		return nil, errTooManyVirConns
	}
	ctx, cancel := context.WithCancel(ctx)
	vc := &virtualConnection{
		ctx:        ctx,
		id:         vid,
		cancelFunc: cancel,
		recvQueue:  queue.New[[]byte](ctx.Done()),
		write:      c.rawConn.Write,
		localAddr:  c.rawConn.LocalAddr(),
		remoteAddr: c.rawConn.RemoteAddr(),
		deleteVirConnFromConn: func() {
			c.deleteVirConn(vid)
		},
	}
	_, loaded := c.idToVirConn.loadOrStore(vc.id, vc)
	if loaded {
		cancel()
		return nil, ErrDuplicateID
	}
	return vc, nil
}

func (c *connection) deleteVirConn(id uint32) {
	c.idToVirConn.delete(id)
}

var (
	_ multiplexed.MuxConn = (*virtualConnection)(nil)
)

type virtualConnection struct {
	write                 func(b []byte) (int, error)
	deleteVirConnFromConn func()
	recvQueue             *queue.Queue[[]byte]
	err                   atomic.Error
	ctx                   context.Context
	cancelFunc            context.CancelFunc
	id                    uint32
	isClosed              atomic.Bool
	localAddr             net.Addr
	remoteAddr            net.Addr
}

// Write writes data to the connection.
// Write and ReadFrame can be concurrent, multiple Write can be concurrent.
func (vc *virtualConnection) Write(b []byte) error {
	if vc.isClosed.Load() {
		return vc.wrapError(ErrConnClosed)
	}
	_, err := vc.write(b)
	return err
}

// Read reads a packet from connection.
// Write and Read can be concurrent, multiple Read can't be concurrent.
func (vc *virtualConnection) Read() ([]byte, error) {
	if vc.isClosed.Load() {
		return nil, vc.wrapError(ErrConnClosed)
	}
	rsp, ok := vc.recvQueue.Get()
	if !ok {
		return nil, vc.wrapError(errors.New("received data failed"))
	}
	return rsp, nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (vc *virtualConnection) Close() {
	vc.close(nil)
}

// LocalAddr returns the local network address, if known.
func (vc *virtualConnection) LocalAddr() net.Addr {
	return vc.localAddr
}

// RemoteAddr returns the remote network address, if known.
func (vc *virtualConnection) RemoteAddr() net.Addr {
	return vc.remoteAddr
}

func (vc *virtualConnection) notifyRead(cause error) {
	if !vc.isClosed.CAS(false, true) {
		return
	}
	vc.err.Store(cause)
	vc.cancelFunc()
}

func (vc *virtualConnection) close(cause error) {
	vc.notifyRead(cause)
	vc.deleteVirConnFromConn()
}

func (vc *virtualConnection) wrapError(err error) error {
	if loaded := vc.err.Load(); loaded != nil {
		return fmt.Errorf("%w, %s", err, loaded.Error())
	}
	if ctxErr := vc.ctx.Err(); ctxErr != nil {
		return fmt.Errorf("%w, %s", err, ctxErr.Error())
	}
	return err
}

func filterOutConn(in []*connection, exclude *connection) []*connection {
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

func newVirConn(
	ctx context.Context,
	conns []*connection,
	vid uint32,
	isTolerable func(error) bool,
) (*virtualConnection, error) {
	for _, conn := range conns {
		virConn, err := conn.newVirConn(ctx, vid)
		if isTolerable(err) {
			continue
		}
		return virConn, err
	}
	return nil, nil
}

func isClosedOrFull(err error) bool {
	if err == ErrConnClosed || err == errTooManyVirConns {
		return true
	}
	return false
}

func isFull(err error) bool {
	return err == errTooManyVirConns
}
