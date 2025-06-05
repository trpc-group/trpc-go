//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

// Package multiplexed implements a connection pool that supports connection multiplexing.
package multiplex

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/tnet/tls"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/queue"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
)

const (
	defaultDialTimeout          = 200 * time.Millisecond
	defaultConnNumberPerHost    = 2
	defaultMaxPickConnRetries   = 100
	defaultConcurrentDialGroups = 1 // Default to 1 for backward compatibility.
)

var (
	// ErrConnClosed indicates connection is closed.
	ErrConnClosed = errors.New("connection is closed")
	// ErrDuplicateID indicates request ID already exist.
	ErrDuplicateID = errors.New("request ID already exist")
	// ErrInvalid indicates the operation is invalid.
	ErrInvalid = errors.New("it's invalid")
	// ErrExceedMaxRetries indicates the operation exceed the max retries of get a virtual connection
	ErrExceedMaxRetries = errors.New("exceed max retires")

	errTooManyVirtualConns  = errors.New("the number of virtual connections exceeds the limit")
	errTooManyConcreteConns = errors.New("the number of concrete connections exceeds the limit")
	errNoAvailableConn      = errors.New("there is no avilable connection")
)

// PoolOption represents some settings for the multiplexed pool.
type PoolOption struct {
	dialTimeout                      time.Duration
	maxConcurrentVirtualConnsPerConn int
	enableMetrics                    bool
	connectNumberPerHost             int
	concurrentDialGroups             int // Number of singleflight groups per host for concurrent dials.
}

// OptPool is function to modify PoolOption.
type OptPool func(*PoolOption)

// WithDialTimeout returns an OptPool which sets dial timeout.
func WithDialTimeout(timeout time.Duration) OptPool {
	return func(o *PoolOption) {
		o.dialTimeout = timeout
	}
}

// WithMaxConcurrentVirtualConnsPerConn returns an OptPool which sets the number
// of concurrent virtual connections per connection.
func WithMaxConcurrentVirtualConnsPerConn(max int) OptPool {
	return func(o *PoolOption) {
		o.maxConcurrentVirtualConnsPerConn = max
	}
}

// WithEnableMetrics returns an OptPool which enable metrics.
func WithEnableMetrics() OptPool {
	return func(o *PoolOption) {
		o.enableMetrics = true
	}
}

// WithConnectNumber returns an Option which sets the number of connections for each peer in the multiplex pool
// and this Option only takes effect when MaxConcurrentVirtualConnsPerConn is 0.
func WithConnectNumber(number int) OptPool {
	return func(o *PoolOption) {
		o.connectNumberPerHost = number
	}
}

// WithConcurrentDialGroupsPerHost returns an OptPool which sets the number of concurrent dial groups.
// Higher values allow more parallel connections to be established to the same host.
// Default is 3 groups.
func WithConcurrentDialGroupsPerHost(n int) OptPool {
	return func(o *PoolOption) {
		if n > 0 {
			o.concurrentDialGroups = n
		}
	}
}

// NewPool creates a new multiplexed pool, which uses dialFunc to dial new connections.
func NewPool(dialFunc connpool.DialFunc, opt ...OptPool) multiplexed.Pool {
	opts := &PoolOption{
		dialTimeout:          defaultDialTimeout,
		connectNumberPerHost: defaultConnNumberPerHost,
		concurrentDialGroups: defaultConcurrentDialGroups, // Initialize with default.
	}
	for _, o := range opt {
		o(opts)
	}
	m := &pool{
		dialFunc:                     dialFunc,
		dialTimeout:                  opts.dialTimeout,
		maxConcurrentVirConnsPerConn: opts.maxConcurrentVirtualConnsPerConn,
		connectNumberPerHost:         opts.connectNumberPerHost,
		hosts:                        make(map[string]*host),
		concurrentDialGroups:         opts.concurrentDialGroups, // Store the option.
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
	connectNumberPerHost         int
	hosts                        map[string]*host // key is network+address
	mu                           sync.RWMutex
	concurrentDialGroups         int // Number of singleflight groups per host.
}

// GetVirtualConn gets a virtual connection to the address on named network.
// Multiple VirtualConns can multiplex on a real connection.
func (p *pool) GetVirtualConn(
	ctx context.Context,
	network string,
	address string,
	opts multiplexed.GetOptions,
) (multiplexed.VirtualConn, error) {
	if opts.FramerBuilder == nil {
		return nil, errors.New("framer builder is not provided")
	}
	host, err := p.getHost(ctx, network, address, opts)
	if err != nil {
		return nil, err
	}

	// Rlock here to make sure that host has not been closed. If host is closed, rLock
	// will return false. And it also avoids reading host.conns while it is being modified.
	if !host.mu.rLock() {
		return nil, ErrConnClosed
	}
	// Try to pick single concrete conn with read lock
	conn, err := host.tryPickConn()
	// If error occurred, retry below
	if err == nil {
		vc, err := conn.newVirtualConn(ctx, opts.Msg)
		if err == nil {
			host.mu.rUnlock()
			return vc, nil
		}
		if !isClosedOrFull(err) {
			// Possible request id is duplicated, return directly
			host.mu.rUnlock()
			return nil, err
		}
		// Connection closed or exceed maxVirtualConnsPerConn, retry below
	}
	host.mu.rUnlock()

	// If all concrete connections have reached their capacity for virtual connections, the
	// subsequent loop will attempt to retry creating a virtual connection on the existing
	// concrete connections or establish a new concrete connection and then construct a
	// virtual connection on it.
	for i := 0; i < defaultMaxPickConnRetries; i++ {
		if !host.mu.lock() {
			// All concrete connection closed
			return nil, ErrConnClosed
		}
		// Must single flight dial here to avoid concurrent dial
		isNewConn, dialing := host.pickConn()
		host.mu.unlock()

		// Waiting dial result from single flight dial or old conn
		conn, err := waitConcreteConn(ctx, dialing)
		if err != nil {
			return nil, err
		}

		vc, err := conn.newVirtualConn(ctx, opts.Msg)
		if err == nil {
			return vc, nil
		}
		if isClosed(err) && isNewConn {
			// New connection but it's closed, possible dial failed.
			return nil, err
		}
		if isFull(err) {
			// Connection exceed maxVirtualConnsPerConn, retry
			continue
		}
		return nil, err
	}
	return nil, ErrExceedMaxRetries
}

func (p *pool) getHost(ctx context.Context, network string, address string, opts multiplexed.GetOptions) (*host, error) {
	hostName := strings.Join([]string{network, address}, "_")
	p.mu.RLock()
	if h, ok := p.hosts[hostName]; ok {
		p.mu.RUnlock()
		return h, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if h, ok := p.hosts[hostName]; ok {
		return h, nil
	}
	h := &host{
		network:  network,
		address:  address,
		hostName: hostName,
		dialOpts: dialOption{
			framerBuilder: opts.FramerBuilder,
			localAddr:     opts.LocalAddr,
			caCertFile:    opts.CACertFile,
			tlsCertFile:   opts.TLSCertFile,
			tlsKeyFile:    opts.TLSKeyFile,
			tlsServerName: opts.TLSServerName,
			dialTimeout:   p.dialTimeout,
		},
		dialFunc:                         p.dialFunc,
		maxConcurrentVirtualConnsPerConn: p.maxConcurrentVirConnsPerConn,
		connectNumberPerHost:             p.connectNumberPerHost,
		conns:                            make([]*connection, 0, p.connectNumberPerHost),
		dialGroups:                       make([]*singleflight.Group, p.concurrentDialGroups),
	}

	// Initialize separate singleflight groups.
	for i := 0; i < p.concurrentDialGroups; i++ {
		h.dialGroups[i] = &singleflight.Group{}
	}

	if h.maxConcurrentVirtualConnsPerConn == 0 {
		h.pickConn = h.pickConnFixedConcrete
		h.tryPickConn = h.tryPickConnFixedConcrete
	} else {
		h.pickConn = h.pickConnUnlimited
		h.tryPickConn = h.tryPickConnUnlimited
	}
	h.deleteHostFromPool = func() {
		p.deleteHost(h)
	}
	if err := h.initialize(ctx); err != nil {
		return nil, err
	}
	p.hosts[hostName] = h
	return h, nil
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
	framerBuilder codec.FramerBuilder
	localAddr     string
	dialTimeout   time.Duration
	caCertFile    string
	tlsCertFile   string
	tlsKeyFile    string
	tlsServerName string
}

// host manages all connections to the same network and address.
type host struct {
	network                          string
	address                          string
	hostName                         string
	dialOpts                         dialOption
	dialFunc                         connpool.DialFunc
	dialGroups                       []*singleflight.Group // Multiple singleflight groups for concurrent dials.
	dialGroupIndex                   atomic.Uint32         // For round-robin selection of dial groups.
	deleteHostFromPool               func()
	maxConcurrentVirtualConnsPerConn int
	connectNumberPerHost             int
	pickConn                         func() (bool, <-chan singleflight.Result)
	tryPickConn                      func() (*connection, error)
	// mu not only ensures the concurrency safety of conns but also guarantees
	// the closure safety of host, which means when the host is triggered to close,
	// it ensures that there are no ongoing additions of connections, and further
	// additions of connections are not allowed.
	mu              stateRWMutex
	conns           []*connection
	roundRobinIndex atomic.Uint32
}

func (h *host) singleflightDial() <-chan singleflight.Result {
	// Round-robin select a singleflight group.
	idx := h.dialGroupIndex.Inc() % uint32(len(h.dialGroups))
	group := h.dialGroups[idx]

	// Use the selected group for this dial operation.
	ch := group.DoChan(h.hostName, func() (connection interface{}, err error) {
		return h.dial()
	})
	return ch
}

func (h *host) dial() (*connection, error) {
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
	conn, err := h.wrapRawConn(rawConn, h.dialOpts.framerBuilder)
	if err != nil {
		return nil, err
	}
	if err := h.storeConn(conn); err != nil {
		return nil, fmt.Errorf("store connection failed, %w", err)
	}
	return conn, nil
}

func waitConcreteConn(ctx context.Context, dialing <-chan singleflight.Result) (*connection, error) {
	select {
	case result := <-dialing:
		return expandSFResult(result)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (h *host) wrapRawConn(rawConn net.Conn, builder codec.FramerBuilder) (*connection, error) {
	framer := builder.New(rawConn)
	decoder, ok := framer.(codec.Decoder)
	if !ok {
		return nil, errors.New("framer must implements codec.Decoder")
	}
	conn := &connection{
		decoder:                   decoder,
		copyFrame:                 !codec.IsSafeFramer(framer),
		idToVirtualConn:           newShardMap(defaultShardSize),
		maxConcurrentVirtualConns: h.maxConcurrentVirtualConnsPerConn,
	}
	conn.deleteConnFromHost = func() {
		if isLastConn := h.deleteConn(conn); isLastConn {
			h.deleteHostFromPool()
		}
	}
	switch c := rawConn.(type) {
	case tnet.Conn:
		conn.rawConn = c
		c.SetOnRequest(func(tnet.Conn) error {
			return conn.onRequest()
		})
		c.SetOnClosed(func(tnet.Conn) error {
			conn.close(ErrConnClosed)
			return nil
		})
	case tls.Conn:
		conn.rawConn = c
		c.SetOnRequest(func(tls.Conn) error {
			return conn.onRequest()
		})
		c.SetOnClosed(func(tls.Conn) error {
			conn.close(ErrConnClosed)
			return nil
		})
	default:
		return nil, fmt.Errorf("dialed connection type %T does't implements tnet.Conn or tnet/tls.Conn", c)
	}
	return conn, nil
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
	var virtualConnNum uint32
	for _, conn := range conns {
		virtualConnNum += conn.idToVirtualConn.length()
	}
	metrics.Gauge(strings.Join([]string{"trpc.MuxConcurrentConnections", h.network, h.address}, ".")).
		Set(float64(len(conns)))
	metrics.Gauge(strings.Join([]string{"trpc.MuxConcurrentVirConns", h.network, h.address}, ".")).
		Set(float64(virtualConnNum))
	log.Debugf("tnet multiplexed status: network: %s, address: %s, connections number: %d,"+
		"concurrent virtual connection number: %d\n", h.network, h.address, len(conns), virtualConnNum)
}

func expandSFResult(result singleflight.Result) (*connection, error) {
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Val.(*connection), nil
}

func (h *host) initialize(ctx context.Context) error {
	// Waiting for connection dialing to avoid concurrent execution with GetVirtualConnection and initialize
	waitCh := make(chan error, 1)
	eg := errgroup.Group{}
	for i := 0; i < h.connectNumberPerHost; i++ {
		eg.Go(func() error {
			_, err := h.dial()
			return err
		})
	}
	go func() {
		waitCh <- eg.Wait()
		close(waitCh)
	}()
	select {
	case err := <-waitCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *host) pickConnFixedConcrete() (bool, <-chan singleflight.Result) {
	index := h.roundRobinIndex.Inc() % uint32(h.connectNumberPerHost)
	if index < uint32(len(h.conns)) {
		ch := make(chan singleflight.Result, 1)
		ch <- singleflight.Result{Val: h.conns[index], Err: nil}
		return false, ch
	}
	return true, h.singleflightDial()
}

func (h *host) pickConnUnlimited() (bool, <-chan singleflight.Result) {
	for _, c := range h.conns {
		// Executed with rwlock of host, it is not very necessary to lock conn.
		// If the state of conn changed, we can retry above.
		if c.rawConn.IsActive() && c.canTakeNewVirtualConn() {
			ch := make(chan singleflight.Result, 1)
			ch <- singleflight.Result{Val: c, Err: nil}
			return false, ch
		}
	}
	return true, h.singleflightDial()
}

func (h *host) tryPickConnFixedConcrete() (*connection, error) {
	// Executed with rlock of host
	index := h.roundRobinIndex.Inc() % uint32(h.connectNumberPerHost)
	if index < uint32(len(h.conns)) {
		return h.conns[index], nil
	}
	return nil, errNoAvailableConn
}

func (h *host) tryPickConnUnlimited() (*connection, error) {
	for _, c := range h.conns {
		// Executed with rlock of host, it is not very necessary to lock conn.
		// If the state of conn changed, we can retry above.
		if c.rawConn.IsActive() && c.canTakeNewVirtualConn() {
			return c, nil
		}
	}
	return nil, errNoAvailableConn
}

type stateConn interface {
	net.Conn
	IsActive() bool
}

// connection wraps the underlying tnet.Conn, and manages many virtualConnections.
type connection struct {
	rawConn                   stateConn
	deleteConnFromHost        func()
	decoder                   codec.Decoder
	copyFrame                 bool
	isClosed                  atomic.Bool
	maxConcurrentVirtualConns int

	// mu not only ensures the concurrency safety of idToVirtualConn but
	// also guarantees the closure safety of connection, which means when
	// the connection is triggered to close, it ensures that there are no
	// ongoing additions of virtual connections, and further additions of
	// virtual connections are not allowed.
	mu              stateRWMutex
	idToVirtualConn *shardMap
}

func (c *connection) onRequest() error {
	rsp, err := c.decoder.Decode()
	if err != nil {
		c.close(err)
		return err
	}
	vc, ok := c.idToVirtualConn.load(rsp.GetRequestID())
	// If the virtualConn corresponding to the id cannot be found,
	// the virtualConn has been closed and the current response is discarded.
	if !ok {
		return nil
	}
	c.dispatch(rsp, vc)
	return nil
}

func (c *connection) canTakeNewVirtualConn() bool {
	return c.maxConcurrentVirtualConns == 0 || c.idToVirtualConn.length() < uint32(c.maxConcurrentVirtualConns)
}

func (c *connection) dispatch(rsp codec.TransportResponseFrame, vc *virtualConnection) {
	if err := c.decoder.UpdateMsg(rsp, vc.msg); err != nil {
		vc.close(err)
		return
	}
	rspBuf := rsp.GetResponseBuf()
	if c.copyFrame {
		copyBuf := make([]byte, len(rspBuf))
		copy(copyBuf, rspBuf)
		rspBuf = copyBuf
	}
	vc.recvQueue.Put(rspBuf)
}

func (c *connection) close(cause error) {
	if !c.isClosed.CAS(false, true) {
		return
	}
	c.deleteConnFromHost()
	c.deleteAllVirtualConn(cause)
	c.rawConn.Close()
}

func (c *connection) deleteAllVirtualConn(cause error) {
	if !c.mu.lock() {
		return
	}
	defer c.mu.unlock()
	c.mu.closeLocked()
	for _, vc := range c.idToVirtualConn.loadAll() {
		vc.notifyRead(cause)
	}
	c.idToVirtualConn.reset()
}

func (c *connection) newVirtualConn(ctx context.Context, msg codec.Msg) (*virtualConnection, error) {
	if !c.mu.rLock() {
		return nil, ErrConnClosed
	}
	defer c.mu.rUnlock()
	if !c.rawConn.IsActive() {
		return nil, ErrConnClosed
	}
	// CanTakeNewVirtualConn and loadOrStore are not atomic, which may cause
	// the actual concurrent virtualConn numbers to exceed the limit max value.
	// Implementing atomic functions requires higher lock granularity,
	// which affects performance.
	if !c.canTakeNewVirtualConn() {
		return nil, errTooManyVirtualConns
	}
	id := msg.RequestID()
	ctx, cancel := context.WithCancel(ctx)
	vc := &virtualConnection{
		ctx:        ctx,
		msg:        msg,
		id:         id,
		cancelFunc: cancel,
		recvQueue:  queue.New[[]byte](ctx.Done()),
		write:      c.rawConn.Write,
		localAddr:  c.rawConn.LocalAddr(),
		remoteAddr: c.rawConn.RemoteAddr(),
		deleteVirtualConnFromConn: func() {
			c.deleteVirtualConn(id)
		},
	}
	_, loaded := c.idToVirtualConn.loadOrStore(vc.id, vc)
	if loaded {
		cancel()
		return nil, ErrDuplicateID
	}
	return vc, nil
}

func (c *connection) deleteVirtualConn(id uint32) {
	c.idToVirtualConn.delete(id)
}

var (
	_ multiplexed.VirtualConn = (*virtualConnection)(nil)
)

type virtualConnection struct {
	write                     func(b []byte) (int, error)
	deleteVirtualConnFromConn func()
	recvQueue                 *queue.Queue[[]byte]
	msg                       codec.Msg
	err                       atomic.Error
	ctx                       context.Context
	cancelFunc                context.CancelFunc
	id                        uint32
	isClosed                  atomic.Bool
	localAddr                 net.Addr
	remoteAddr                net.Addr
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
	bts, ok := vc.recvQueue.Get()
	if !ok {
		return nil, vc.wrapError(errors.New("received data failed"))
	}
	return bts, nil
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
	vc.deleteVirtualConnFromConn()
}

func (vc *virtualConnection) wrapError(err error) error {
	if loaded := vc.err.Load(); loaded != nil {
		return multierror.Append(err, loaded).ErrorOrNil()
	}
	if ctxErr := vc.ctx.Err(); ctxErr != nil {
		return multierror.Append(err, ctxErr).ErrorOrNil()
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

func isClosedOrFull(err error) bool {
	if err == ErrConnClosed || err == errTooManyVirtualConns {
		return true
	}
	return false
}

func isClosed(err error) bool {
	return err == ErrConnClosed
}

func isFull(err error) bool {
	return err == errTooManyVirtualConns
}
