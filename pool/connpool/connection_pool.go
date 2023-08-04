package connpool

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
)

const (
	defaultDialTimeout     = 200 * time.Millisecond
	defaultIdleTimeout     = 50 * time.Second
	defaultMaxIdle         = 65536
	defaultCheckInterval   = 3 * time.Second
	defaultPoolIdleTimeout = 2 * defaultIdleTimeout
)

var globalBuffer []byte = make([]byte, 1)

// DefaultConnectionPool is the default connection pool, replaceable.
var DefaultConnectionPool = NewConnectionPool()

// connection pool error message.
var (
	ErrPoolLimit  = errors.New("connection pool limit")  // ErrPoolLimit number of connections exceeds the limit error.
	ErrPoolClosed = errors.New("connection pool closed") // ErrPoolClosed connection pool closed error.
	ErrConnClosed = errors.New("conn closed")            // ErrConnClosed connection closed.
	ErrNoDeadline = errors.New("dial no deadline")       // ErrNoDeadline has no deadline set.
	ErrConnInPool = errors.New("conn already in pool")   // ErrNoDeadline has no deadline set.
)

// HealthChecker idle connection health check function.
// The function supports quick check and comprehensive check.
// Quick check is called when an idle connection is obtained,
// and only checks whether the connection status is abnormal.
// The function returns true to indicate that the connection is available normally.
type HealthChecker func(pc *PoolConn, isFast bool) bool

// NewConnectionPool creates a connection pool.
func NewConnectionPool(opt ...Option) Pool {
	// Default value, tentative, need to debug to determine the specific value.
	opts := &Options{
		MaxIdle:         defaultMaxIdle,
		IdleTimeout:     defaultIdleTimeout,
		DialTimeout:     defaultDialTimeout,
		PoolIdleTimeout: defaultPoolIdleTimeout,
		Dial:            Dial,
	}
	for _, o := range opt {
		o(opts)
	}
	return &pool{
		opts:            opts,
		connectionPools: new(sync.Map),
	}
}

// pool connection pool factory, maintains connection pools corresponding to all addresses,
// and connection pool option information.
type pool struct {
	opts            *Options
	connectionPools *sync.Map
}

type dialFunc = func(ctx context.Context) (net.Conn, error)

func (p *pool) getDialFunc(network string, address string, opts GetOptions) dialFunc {
	dialOpts := &DialOptions{
		Network:       network,
		Address:       address,
		LocalAddr:     opts.LocalAddr,
		CACertFile:    opts.CACertFile,
		TLSCertFile:   opts.TLSCertFile,
		TLSKeyFile:    opts.TLSKeyFile,
		TLSServerName: opts.TLSServerName,
		IdleTimeout:   p.opts.IdleTimeout,
	}

	return func(ctx context.Context) (net.Conn, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		d, ok := ctx.Deadline()
		if !ok {
			return nil, ErrNoDeadline
		}

		opts := *dialOpts
		opts.Timeout = time.Until(d)
		return p.opts.Dial(&opts)
	}
}

// Get is used to get the connection from the connection pool.
func (p *pool) Get(network string, address string, opts GetOptions) (net.Conn, error) {
	ctx, cancel := opts.getDialCtx(p.opts.DialTimeout)
	if cancel != nil {
		defer cancel()
	}
	key := getNodeKey(network, address, opts.Protocol)
	if v, ok := p.connectionPools.Load(key); ok {
		return v.(*ConnectionPool).Get(ctx)
	}

	newPool := &ConnectionPool{
		Dial:               p.getDialFunc(network, address, opts),
		MinIdle:            p.opts.MinIdle,
		MaxIdle:            p.opts.MaxIdle,
		MaxActive:          p.opts.MaxActive,
		Wait:               p.opts.Wait,
		MaxConnLifetime:    p.opts.MaxConnLifetime,
		IdleTimeout:        p.opts.IdleTimeout,
		framerBuilder:      opts.FramerBuilder,
		customReader:       opts.CustomReader,
		forceClosed:        p.opts.ForceClose,
		PushIdleConnToTail: p.opts.PushIdleConnToTail,
		onCloseFunc:        func() { p.connectionPools.Delete(key) },
		poolIdleTimeout:    p.opts.PoolIdleTimeout,
	}

	if newPool.MaxActive > 0 {
		newPool.token = make(chan struct{}, p.opts.MaxActive)
	}

	newPool.checker = newPool.defaultChecker
	if p.opts.Checker != nil {
		newPool.checker = p.opts.Checker
	}

	// Avoid the problem of writing concurrently to the pool map during initialization.
	v, ok := p.connectionPools.LoadOrStore(key, newPool)
	if !ok {
		newPool.RegisterChecker(defaultCheckInterval, newPool.checker)
		newPool.keepMinIdles()
		return newPool.Get(ctx)
	}
	return v.(*ConnectionPool).Get(ctx)
}

// ConnectionPool is the connection pool.
type ConnectionPool struct {
	Dial        func(context.Context) (net.Conn, error) // initialize the connection.
	MinIdle     int                                     // Minimum number of idle connections.
	MaxIdle     int                                     // Maximum number of idle connections, 0 means no limit.
	MaxActive   int                                     // Maximum number of active connections, 0 means no limit.
	IdleTimeout time.Duration                           // idle connection timeout.
	// Whether to wait when the maximum number of active connections is reached.
	Wait               bool
	MaxConnLifetime    time.Duration // Maximum lifetime of the connection.
	mu                 sync.Mutex    // Control concurrent locks.
	checker            HealthChecker // Idle connection health check function.
	closed             bool          // Whether the connection pool has been closed.
	token              chan struct{} // control concurrency by applying token.
	idleSize           int           // idle connections size.
	idle               connList      // idle connection list.
	framerBuilder      codec.FramerBuilder
	forceClosed        bool // Force close the connection, suitable for streaming scenarios.
	PushIdleConnToTail bool // connection to ip will be push tail when ConnectionPool.put method is called.
	// customReader creates a reader encapsulating the underlying connection.
	customReader    func(io.Reader) io.Reader
	onCloseFunc     func()        // execute when checker goroutine judge the connection_pool is useless.
	used            int32         // size of connections used by user, atomic.
	lastGetTime     int64         // last get connection millisecond timestamp, atomic.
	poolIdleTimeout time.Duration // pool idle timeout.
}

func (p *ConnectionPool) keepMinIdles() {
	p.mu.Lock()
	count := p.MinIdle - p.idleSize
	if count > 0 {
		p.idleSize += count
	}
	p.mu.Unlock()

	for i := 0; i < count; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
			defer cancel()
			if err := p.addIdleConn(ctx); err != nil {
				p.mu.Lock()
				p.idleSize--
				p.mu.Unlock()
			}
		}()
	}
}

func (p *ConnectionPool) addIdleConn(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrPoolClosed
	}
	p.mu.Unlock()

	c, err := p.dial(ctx)
	if err != nil {
		return err
	}

	// put in idle list
	pc := p.newPoolConn(c)
	p.mu.Lock()
	if p.closed {
		pc.closed = true
		pc.Conn.Close()
	} else {
		pc.t = time.Now()
		if !p.PushIdleConnToTail {
			p.idle.pushHead(pc)
		} else {
			p.idle.pushTail(pc)
		}
	}
	p.mu.Unlock()
	return nil
}

// Get gets the connection from the connection pool.
func (p *ConnectionPool) Get(ctx context.Context) (*PoolConn, error) {
	var (
		pc  *PoolConn
		err error
	)
	if pc, err = p.get(ctx); err != nil {
		report.ConnectionPoolGetConnectionErr.Incr()
		return nil, err
	}
	return pc, nil
}

// Close releases the connection.
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.idle.count = 0
	p.idleSize = 0
	pc := p.idle.head
	p.idle.head, p.idle.tail = nil, nil
	p.mu.Unlock()
	for ; pc != nil; pc = pc.next {
		pc.Conn.Close()
		pc.closed = true
	}
	return nil
}

// get gets the connection from the connection pool.
func (p *ConnectionPool) get(ctx context.Context) (*PoolConn, error) {
	if err := p.getToken(ctx); err != nil {
		return nil, err
	}

	atomic.StoreInt64(&p.lastGetTime, time.Now().UnixMilli())
	atomic.AddInt32(&p.used, 1)

	// try to get an idle connection.
	if pc := p.getIdleConn(); pc != nil {
		return pc, nil
	}

	// get new connection.
	pc, err := p.getNewConn(ctx)
	if err != nil {
		p.freeToken()
		return nil, err
	}
	return pc, nil
}

// if p.Wait is True, return err when timeout.
// if p.Wait is False, return err when token empty immediately.
func (p *ConnectionPool) getToken(ctx context.Context) error {
	if p.MaxActive <= 0 {
		return nil
	}

	if p.Wait {
		select {
		case p.token <- struct{}{}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		select {
		case p.token <- struct{}{}:
			return nil
		default:
			return ErrPoolLimit
		}
	}
}

func (p *ConnectionPool) freeToken() {
	if p.MaxActive <= 0 {
		return
	}
	<-p.token
}

func (p *ConnectionPool) getIdleConn() *PoolConn {
	p.mu.Lock()
	for p.idle.head != nil {
		pc := p.idle.head
		p.idle.popHead()
		p.idleSize--
		p.mu.Unlock()
		if p.checker(pc, true) {
			return pc
		}
		pc.Conn.Close()
		pc.closed = true
		p.mu.Lock()
	}
	p.mu.Unlock()
	return nil
}

func (p *ConnectionPool) getNewConn(ctx context.Context) (*PoolConn, error) {
	// If the connection pool has been closed, return an error directly.
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}
	p.mu.Unlock()

	c, err := p.dial(ctx)
	if err != nil {
		return nil, err
	}

	report.ConnectionPoolGetNewConnection.Incr()
	return p.newPoolConn(c), nil
}

func (p *ConnectionPool) newPoolConn(c net.Conn) *PoolConn {
	pc := &PoolConn{
		Conn:       c,
		created:    time.Now(),
		pool:       p,
		forceClose: p.forceClosed,
		inPool:     false,
	}
	if p.framerBuilder != nil {
		pc.fr = p.framerBuilder.New(p.customReader(pc))
		pc.copyFrame = !codec.IsSafeFramer(pc.fr)
	}
	return pc
}

func (p *ConnectionPool) checkHealthOnce() {
	p.mu.Lock()
	n := p.idle.count
	for i := 0; i < n && p.idle.head != nil; i++ {
		pc := p.idle.head
		p.idle.popHead()
		p.idleSize--
		p.mu.Unlock()
		if p.checker(pc, false) {
			p.mu.Lock()
			p.idleSize++
			p.idle.pushTail(pc)
		} else {
			pc.Conn.Close()
			pc.closed = true
			p.mu.Lock()
		}
	}
	p.mu.Unlock()
}

func (p *ConnectionPool) checkRoutine(interval time.Duration) {
	for {
		time.Sleep(interval)
		p.mu.Lock()
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return
		}
		p.checkHealthOnce()

		if p.checkPoolIdleTimeout() {
			return
		}

		// Check if the minimum number of idle connections is met.
		p.checkMinIdle()
	}
}

func (p *ConnectionPool) checkMinIdle() {
	if p.MinIdle <= 0 {
		return
	}
	p.keepMinIdles()
}

// checkPoolIdleTimeout check whether the connection_pool is useless
func (p *ConnectionPool) checkPoolIdleTimeout() bool {
	p.mu.Lock()
	lastGetTime := atomic.LoadInt64(&p.lastGetTime)
	if lastGetTime == 0 || p.poolIdleTimeout == 0 {
		p.mu.Unlock()
		return false
	}
	if time.Now().UnixMilli()-lastGetTime > p.poolIdleTimeout.Milliseconds() &&
		p.onCloseFunc != nil && atomic.LoadInt32(&p.used) == 0 {
		p.mu.Unlock()
		p.onCloseFunc()
		if err := p.Close(); err != nil {
			log.Errorf("failed to close ConnectionPool, error: %v", err)
		}
		return true
	}
	p.mu.Unlock()
	return false
}

// RegisterChecker registers the idle connection check method.
func (p *ConnectionPool) RegisterChecker(interval time.Duration, checker HealthChecker) {
	if interval <= 0 || checker == nil {
		return
	}
	p.mu.Lock()
	p.checker = checker
	p.mu.Unlock()
	go p.checkRoutine(interval)
}

// defaultChecker is the default idle connection check method,
// returning true means the connection is available normally.
func (p *ConnectionPool) defaultChecker(pc *PoolConn, isFast bool) bool {
	// Check whether the connection status is abnormal:
	// closed, network exception or sticky packet processing exception.
	if pc.isRemoteError(isFast) {
		return false
	}
	// Based on performance considerations, the quick check only does the RemoteErr check.
	if isFast {
		return true
	}
	// Check if the connection has exceeded the maximum idle time, if so close the connection.
	if p.IdleTimeout > 0 && pc.t.Add(p.IdleTimeout).Before(time.Now()) {
		report.ConnectionPoolIdleTimeout.Incr()
		return false
	}
	// Check if the connection is still alive.
	if p.MaxConnLifetime > 0 && pc.created.Add(p.MaxConnLifetime).Before(time.Now()) {
		report.ConnectionPoolLifetimeExceed.Incr()
		return false
	}
	return true
}

// dial establishes a connection.
func (p *ConnectionPool) dial(ctx context.Context) (net.Conn, error) {
	if p.Dial != nil {
		return p.Dial(ctx)
	}
	return nil, errors.New("must pass Dial to pool")
}

// put tries to release the connection to the connection pool.
// forceClose depends on GetOptions.ForceClose and will be true
// if the connection fails to read or write.
func (p *ConnectionPool) put(pc *PoolConn, forceClose bool) error {
	if pc.closed {
		return nil
	}
	p.mu.Lock()
	if !p.closed && !forceClose {
		pc.t = time.Now()
		if !p.PushIdleConnToTail {
			p.idle.pushHead(pc)
		} else {
			p.idle.pushTail(pc)
		}
		if p.idleSize >= p.MaxIdle {
			pc = p.idle.tail
			p.idle.popTail()
		} else {
			p.idleSize++
			pc = nil
		}
	}
	p.mu.Unlock()
	if pc != nil {
		pc.closed = true
		pc.Conn.Close()
	}
	p.freeToken()
	atomic.AddInt32(&p.used, -1)
	return nil
}

// PoolConn is the connection in the connection pool.
type PoolConn struct {
	net.Conn
	fr         codec.Framer
	t          time.Time
	created    time.Time
	next, prev *PoolConn
	pool       *ConnectionPool
	closed     bool
	forceClose bool
	copyFrame  bool
	inPool     bool
}

// ReadFrame reads the frame.
func (pc *PoolConn) ReadFrame() ([]byte, error) {
	if pc.closed {
		return nil, ErrConnClosed
	}
	if pc.fr == nil {
		pc.pool.put(pc, true)
		return nil, errors.New("framer not set")
	}
	data, err := pc.fr.ReadFrame()
	if err != nil {
		// ReadFrame failure may be socket Read interface timeout failure
		// or the unpacking fails, in both cases the connection should be closed.
		pc.pool.put(pc, true)
		return nil, err
	}

	// Framer does not support concurrent read safety, copy the data.
	if pc.copyFrame {
		buf := make([]byte, len(data))
		copy(buf, data)
		return buf, err
	}
	return data, err
}

// isRemoteError tries to receive a byte to detect whether the peer has actively closed the connection.
// If the peer returns an io.EOF error, it is indicated that the peer has been closed.
// Idle connections should not read data, if the data is read, it means the upper layer's
// sticky packet processing is not done, the connection should also be discarded.
// return true if there is an error in the connection.
func (pc *PoolConn) isRemoteError(isFast bool) bool {
	var err error
	if isFast {
		err = checkConnErrUnblock(pc.Conn, globalBuffer)
	} else {
		err = checkConnErr(pc.Conn, globalBuffer)
	}
	if err != nil {
		report.ConnectionPoolRemoteErr.Incr()
		return true
	}
	return false
}

// reset resets the connection state.
func (pc *PoolConn) reset() {
	if pc == nil {
		return
	}
	pc.Conn.SetDeadline(time.Time{})
}

// Write sends data on the connection.
func (pc *PoolConn) Write(b []byte) (int, error) {
	if pc.closed {
		return 0, ErrConnClosed
	}
	n, err := pc.Conn.Write(b)
	if err != nil {
		pc.pool.put(pc, true)
	}
	return n, err
}

// Read reads data on the connection.
func (pc *PoolConn) Read(b []byte) (int, error) {
	if pc.closed {
		return 0, ErrConnClosed
	}
	n, err := pc.Conn.Read(b)
	if err != nil {
		pc.pool.put(pc, true)
	}
	return n, err
}

// Close overrides the Close method of net.Conn and puts it back into the connection pool.
func (pc *PoolConn) Close() error {
	if pc.closed {
		return ErrConnClosed
	}
	if pc.inPool {
		return ErrConnInPool
	}
	pc.reset()
	return pc.pool.put(pc, pc.forceClose)
}

// GetRawConn gets raw connection in PoolConn.
func (pc *PoolConn) GetRawConn() net.Conn {
	return pc.Conn
}

// connList maintains idle connections and uses stacks to maintain connections.
//
// The stack method has an advantage over the queue. When the request volume is relatively small but the request
// distribution is still relatively uniform, the queue method will cause the occupied connection to be delayed.
type connList struct {
	count      int
	head, tail *PoolConn
}

func (l *connList) pushHead(pc *PoolConn) {
	pc.inPool = true
	pc.next = l.head
	pc.prev = nil
	if l.count == 0 {
		l.tail = pc
	} else {
		l.head.prev = pc
	}
	l.count++
	l.head = pc
}

func (l *connList) popHead() {
	pc := l.head
	l.count--
	if l.count == 0 {
		l.head, l.tail = nil, nil
	} else {
		pc.next.prev = nil
		l.head = pc.next
	}
	pc.next, pc.prev = nil, nil
	pc.inPool = false
}

func (l *connList) pushTail(pc *PoolConn) {
	pc.inPool = true
	pc.next = nil
	pc.prev = l.tail
	if l.count == 0 {
		l.head = pc
	} else {
		l.tail.next = pc
	}
	l.count++
	l.tail = pc
}

func (l *connList) popTail() {
	pc := l.tail
	l.count--
	if l.count == 0 {
		l.head, l.tail = nil, nil
	} else {
		pc.prev.next = nil
		l.tail = pc.prev
	}
	pc.next, pc.prev = nil, nil
	pc.inPool = false
}

func getNodeKey(network, address, protocol string) string {
	const underline = "_"
	var key strings.Builder
	key.Grow(len(network) + len(address) + len(protocol) + 2)
	key.WriteString(network)
	key.WriteString(underline)
	key.WriteString(address)
	key.WriteString(underline)
	key.WriteString(protocol)
	return key.String()
}
