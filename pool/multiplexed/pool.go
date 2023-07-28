package multiplexed

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
	"golang.org/x/sync/singleflight"
	"trpc.group/trpc-go/trpc-go/internal/queue"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
)

// DefaultMultiplexedPool is the default multiplexed implementation.
var DefaultMultiplexedPool = NewPool(NewDialFunc())

const (
	defaultDialTimeout = time.Second
)

var (
	// ErrFrameParserNil indicates that frame parse is nil.
	ErrFrameParserNil = errors.New("frame parser is nil")
	// ErrWriteNotFinished write operation is not completed.
	ErrWriteNotFinished = errors.New("write not finished")
	// ErrConnClosed indicates connection is closed.
	ErrConnClosed = errors.New("connection is closed")
	// ErrDuplicateID indicates request ID already exist.
	ErrDuplicateID = errors.New("request ID already exist")

	errTooManyVirtualConns = errors.New("the number of virtual connections exceeds the limit")
)

// NewPool creates a new multiplexed pool, which uses dialFunc to dial new connections.
func NewPool(dialFunc DialFunc, opt ...OptPool) Pool {
	opts := &PoolOption{
		dialTimeout: defaultDialTimeout,
	}
	for _, o := range opt {
		o(opts)
	}
	m := &pool{
		dialFunc:                         dialFunc,
		dialTimeout:                      opts.dialTimeout,
		maxConcurrentVirtualConnsPerConn: opts.maxConcurrentVirtualConnsPerConn,
		hosts:                            make(map[string]*host),
	}
	if opts.enableMetrics {
		go m.metrics()
	}
	return m
}

var _ Pool = (*pool)(nil)

type pool struct {
	dialFunc                         DialFunc
	dialTimeout                      time.Duration
	maxConcurrentVirtualConnsPerConn int

	hosts map[string]*host // key is network+address
	mu    sync.RWMutex
}

// GetVirtualConn gets a virtual connection to the address on named network.
// Multiple VirtualConns can multiplex on a real connection.
func (p *pool) GetVirtualConn(
	ctx context.Context,
	network string,
	address string,
	opts GetOptions,
) (VirtualConn, error) {
	if opts.FrameParser == nil {
		return nil, ErrFrameParserNil
	}
	host := p.getHost(network, address, opts)

	// Rlock here to make sure that host has not been closed. If host is closed, rLock
	// will return false. And it also avoids reading host.conns while it is being modified.
	if !host.mu.rLock() {
		return nil, ErrConnClosed
	}
	virtualConn, err := newVirtualConn(ctx, host.conns, opts.VID, isClosedOrFull)
	if err != errNoAvailableConn {
		host.mu.rUnlock()
		return virtualConn, err
	}
	host.mu.rUnlock()

	// If all concrete connections have reached their capacity for virtual connections, the
	// subsequent loop will attempt to retry creating a virtual connection on the existing
	// concrete connections or establish a new concrete connection and then construct a
	// virtual connection on it.
	for {
		// Lock here to ensure that the connection being created is not missed when reading host.conns,
		// because singleflightDial will lock host.mu before adding the new connection to host.conns asynchronously.
		if !host.mu.lock() {
			return nil, ErrConnClosed
		}
		virtualConn, err = newVirtualConn(ctx, host.conns, opts.VID, isClosedOrFull)
		if err != errNoAvailableConn {
			host.mu.unlock()
			return virtualConn, err
		}
		// if all connections are closed or can't take more virtual connection, create one.
		dialing := host.singleflightDial()
		host.mu.unlock()

		conn, err := waitDialing(ctx, dialing)
		if err != nil {
			return nil, err
		}
		// create new connection when the number of virtual connections exceeds the limit.
		virtualConn, err = newVirtualConn(ctx, []*connection{conn}, opts.VID, isFull)
		if err != errNoAvailableConn {
			return virtualConn, err
		}
	}
}

func (p *pool) getHost(network string, address string, opts GetOptions) *host {
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
			frameParser:   opts.FrameParser,
			localAddr:     opts.LocalAddr,
			caCertFile:    opts.CACertFile,
			tlsCertFile:   opts.TLSCertFile,
			tlsKeyFile:    opts.TLSKeyFile,
			tlsServerName: opts.TLSServerName,
			dialTimeout:   p.dialTimeout,
		},
		dialFunc:                         p.dialFunc,
		maxConcurrentVirtualConnsPerConn: p.maxConcurrentVirtualConnsPerConn,
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

// host manages all connections to the same network and address.
type host struct {
	network                          string
	address                          string
	hostName                         string
	dialOpts                         dialOption
	dialFunc                         DialFunc
	sfg                              singleflight.Group
	deleteHostFromPool               func()
	maxConcurrentVirtualConnsPerConn int

	// mu not only ensures the concurrency safety of conns but also guarantees
	// the closure safety of host, which means when the host is triggered to close,
	// it ensures that there are no ongoing additions of connections, and further
	// additions of connections are not allowed.
	mu    stateRWMutex
	conns []*connection
}

func (h *host) singleflightDial() <-chan singleflight.Result {
	ch := h.sfg.DoChan(h.hostName, func() (connection interface{}, err error) {
		rawConn, err := h.dialFunc(h.dialOpts.frameParser, &connpool.DialOptions{
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
		conn, err := h.wrapRawConn(rawConn, h.dialOpts.frameParser)
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

func (h *host) wrapRawConn(rawConn Conn, fp FrameParser) (*connection, error) {
	c := &connection{
		rawConn:                   rawConn,
		frameParser:               fp,
		idToVirtualConn:           newShardMap(defaultShardSize),
		maxConcurrentVirtualConns: h.maxConcurrentVirtualConnsPerConn,
	}
	c.deleteConnFromHost = func() {
		if isLastConn := h.deleteConn(c); isLastConn {
			h.deleteHostFromPool()
		}
	}

	if err := rawConn.Start(c); err != nil {
		return nil, err
	}
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

func waitDialing(ctx context.Context, dialing <-chan singleflight.Result) (*connection, error) {
	select {
	case result := <-dialing:
		return expandSFResult(result)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func expandSFResult(result singleflight.Result) (*connection, error) {
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Val.(*connection), nil
}

var _ Notifier = (*connection)(nil)

// connection wraps the underlying Conn, and manages many virtualConns.
type connection struct {
	rawConn                   Conn
	deleteConnFromHost        func()
	frameParser               FrameParser
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

// Dispatch dispatches buffer to appropriate virtual connection based on the specified vid.
func (c *connection) Dispatch(vid uint32, buf []byte) {
	vc, ok := c.idToVirtualConn.load(vid)
	// If the virtualConn corresponding to the id cannot be found,
	// the virtualConn has been closed and the current response is discarded.
	if ok {
		vc.recvQueue.Put(buf)
	}
}

// Close closes the connection.
// All virtual connections related to this connection will be closed.
func (c *connection) Close(err error) {
	c.close(err)
}

func (c *connection) canTakeNewVirtualConn() bool {
	return c.maxConcurrentVirtualConns == 0 || c.idToVirtualConn.length() < uint32(c.maxConcurrentVirtualConns)
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

func (c *connection) newVirtualConn(ctx context.Context, vid uint32) (*virtualConn, error) {
	if !c.mu.rLock() {
		return nil, ErrConnClosed
	}
	defer c.mu.rUnlock()
	if !c.rawConn.IsActive() {
		return nil, ErrConnClosed
	}
	// CanTakeNewVirtualConn and loadOrStore are not atomic, which may cause
	// the actual concurrent virtualConn numbers to exceed the limit max
	// value. Implementing atomic functions requires higher lock granularity,
	// which affects performance.
	if !c.canTakeNewVirtualConn() {
		return nil, errTooManyVirtualConns
	}
	ctx, cancel := context.WithCancel(ctx)
	vc := &virtualConn{
		ctx:        ctx,
		id:         vid,
		cancelFunc: cancel,
		recvQueue:  queue.New[[]byte](ctx.Done()),
		write:      c.rawConn.Write,
		localAddr:  c.rawConn.LocalAddr(),
		remoteAddr: c.rawConn.RemoteAddr(),
		deleteVirtualConnFromConn: func() {
			c.deleteVirtualConn(vid)
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
	_ VirtualConn = (*virtualConn)(nil)
)

// virtualConn implements VirtualConn.
type virtualConn struct {
	write                     func(b []byte) (int, error)
	deleteVirtualConnFromConn func()
	recvQueue                 *queue.Queue[[]byte]
	err                       atomic.Error
	ctx                       context.Context
	cancelFunc                context.CancelFunc
	id                        uint32
	isClosed                  atomic.Bool
	localAddr                 net.Addr
	remoteAddr                net.Addr
}

// Write writes data to the virtual connection.
// Write and ReadFrame can be concurrent, multiple Write can be concurrent.
func (vc *virtualConn) Write(b []byte) error {
	if vc.isClosed.Load() {
		return vc.wrapError(ErrConnClosed)
	}
	_, err := vc.write(b)
	return err
}

// Read reads a packet from the virtual connection.
// Write and Read can be concurrent, multiple Read can't be concurrent.
func (vc *virtualConn) Read() ([]byte, error) {
	if vc.isClosed.Load() {
		return nil, vc.wrapError(ErrConnClosed)
	}
	rsp, ok := vc.recvQueue.Get()
	if !ok {
		return nil, vc.wrapError(errors.New("received data failed"))
	}
	return rsp, nil
}

// Close closes the virtual connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (vc *virtualConn) Close() {
	vc.close(nil)
}

// LocalAddr returns the local network address, if known.
func (vc *virtualConn) LocalAddr() net.Addr {
	return vc.localAddr
}

// RemoteAddr returns the remote network address, if known.
func (vc *virtualConn) RemoteAddr() net.Addr {
	return vc.remoteAddr
}

func (vc *virtualConn) notifyRead(cause error) {
	if !vc.isClosed.CAS(false, true) {
		return
	}
	vc.err.Store(cause)
	vc.cancelFunc()
}

func (vc *virtualConn) close(cause error) {
	vc.notifyRead(cause)
	vc.deleteVirtualConnFromConn()
}

func (vc *virtualConn) wrapError(err error) error {
	return multierror.Append(err, vc.err.Load(), vc.ctx.Err()).ErrorOrNil()
}

type dialOption struct {
	frameParser   FrameParser
	localAddr     string
	dialTimeout   time.Duration
	caCertFile    string
	tlsCertFile   string
	tlsKeyFile    string
	tlsServerName string
}

var errNoAvailableConn = errors.New("there is no avilable connection")

func newVirtualConn(
	ctx context.Context,
	conns []*connection,
	vid uint32,
	isTolerable func(error) bool,
) (*virtualConn, error) {
	for _, conn := range conns {
		virtualConn, err := conn.newVirtualConn(ctx, vid)
		if isTolerable(err) {
			continue
		}
		return virtualConn, err
	}
	return nil, errNoAvailableConn
}

func filterOutConn(in []*connection, exclude *connection) []*connection {
	out := in[:0]
	for _, v := range in {
		if v != exclude {
			out = append(out, v)
		}
	}
	// If a connection is successfully removed, empty the last value of the slice to avoid memory leaks.
	if len(in) != len(out) {
		in[len(in)-1] = nil
	}
	return out
}

func isClosedOrFull(err error) bool {
	if err == ErrConnClosed || err == errTooManyVirtualConns {
		return true
	}
	return false
}

func isFull(err error) bool {
	return err == errTooManyVirtualConns
}
