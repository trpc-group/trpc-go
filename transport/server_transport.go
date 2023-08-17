package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/panjf2000/ants/v2"
	"trpc.group/trpc-go/trpc-go/internal/reuseport"

	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
)

const (
	// EnvGraceRestart is the flag of graceful restart.
	EnvGraceRestart = "TRPC_IS_GRACEFUL"

	// EnvGraceFirstFd is the fd of graceful first listener.
	EnvGraceFirstFd = "TRPC_GRACEFUL_1ST_LISTENFD"

	// EnvGraceRestartFdNum is the number of fd for graceful restart.
	EnvGraceRestartFdNum = "TRPC_GRACEFUL_LISTENFD_NUM"

	// EnvGraceRestartPPID is the PPID of graceful restart.
	EnvGraceRestartPPID = "TRPC_GRACEFUL_PPID"
)

var (
	errUnSupportedListenerType = errors.New("not supported listener type")
	errUnSupportedNetworkType  = errors.New("not supported network type")
	errFileIsNotSocket         = errors.New("file is not a socket")
)

// DefaultServerTransport is the default implementation of ServerStreamTransport.
var DefaultServerTransport = NewServerTransport(WithReusePort(true))

// NewServerTransport creates a new ServerTransport.
func NewServerTransport(opt ...ServerTransportOption) ServerTransport {
	r := newServerTransport(opt...)
	return &r
}

// newServerTransport creates a new serverTransport.
func newServerTransport(opt ...ServerTransportOption) serverTransport {
	// this is the default option.
	opts := defaultServerTransportOptions()
	for _, o := range opt {
		o(opts)
	}
	addrToConn := make(map[string]*tcpconn)
	return serverTransport{addrToConn: addrToConn, m: &sync.RWMutex{}, opts: opts}
}

// serverTransport is the implementation details of server transport, may be tcp or udp.
type serverTransport struct {
	addrToConn map[string]*tcpconn
	m          *sync.RWMutex
	opts       *ServerTransportOptions
}

// ListenAndServe starts Listening, returns an error on failure.
func (s *serverTransport) ListenAndServe(ctx context.Context, opts ...ListenServeOption) error {
	lsopts := &ListenServeOptions{}
	for _, opt := range opts {
		opt(lsopts)
	}

	if lsopts.Listener != nil {
		return s.listenAndServeStream(ctx, lsopts)
	}
	// Support simultaneous listening TCP and UDP.
	networks := strings.Split(lsopts.Network, ",")
	for _, network := range networks {
		lsopts.Network = network
		switch lsopts.Network {
		case "tcp", "tcp4", "tcp6", "unix":
			if err := s.listenAndServeStream(ctx, lsopts); err != nil {
				return err
			}
		case "udp", "udp4", "udp6":
			if err := s.listenAndServePacket(ctx, lsopts); err != nil {
				return err
			}
		default:
			return fmt.Errorf("server transport: not support network type %s", lsopts.Network)
		}
	}
	return nil
}

// ---------------------------------stream server-----------------------------------------//

var (
	// listenersMap records the listeners in use in the current process.
	listenersMap = &sync.Map{}
	// inheritedListenersMap record the listeners inherited from the parent process.
	// A key(host:port) may have multiple listener fds.
	inheritedListenersMap = &sync.Map{}
	// once controls fds passed from parent process to construct listeners.
	once sync.Once
)

// GetListenersFds gets listener fds.
func GetListenersFds() []*ListenFd {
	listenersFds := []*ListenFd{}
	listenersMap.Range(func(key, _ interface{}) bool {
		var (
			fd  *ListenFd
			err error
		)

		switch k := key.(type) {
		case net.Listener:
			fd, err = getListenerFd(k)
		case net.PacketConn:
			fd, err = getPacketConnFd(k)
		default:
			log.Errorf("listener type passing not supported, type: %T", key)
			err = fmt.Errorf("not supported listener type: %T", key)
		}
		if err != nil {
			log.Errorf("cannot get the listener fd, err: %v", err)
			return true
		}
		listenersFds = append(listenersFds, fd)
		return true
	})
	return listenersFds
}

// SaveListener saves the listener.
func SaveListener(listener interface{}) error {
	switch listener.(type) {
	case net.Listener, net.PacketConn:
		listenersMap.Store(listener, struct{}{})
	default:
		return fmt.Errorf("not supported listener type: %T", listener)
	}
	return nil
}

// getTCPListener gets the TCP/Unix listener.
func (s *serverTransport) getTCPListener(opts *ListenServeOptions) (listener net.Listener, err error) {
	listener = opts.Listener

	if listener != nil {
		return listener, nil
	}

	v, _ := os.LookupEnv(EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// find the passed listener
		pln, err := getPassedListener(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}

		listener, ok := pln.(net.Listener)
		if !ok {
			return nil, errors.New("invalid net.Listener")
		}
		return listener, nil
	}

	// Reuse port. To speed up IO, the kernel dispatches IO ReadReady events to threads.
	if s.opts.ReusePort && opts.Network != "unix" {
		listener, err = reuseport.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("%s reuseport error:%v", opts.Network, err)
		}
	} else {
		listener, err = net.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}
	}

	return listener, nil
}

// listenAndServeStream starts listening, returns an error on failure.
func (s *serverTransport) listenAndServeStream(ctx context.Context, opts *ListenServeOptions) error {
	if opts.FramerBuilder == nil {
		return errors.New("tcp transport FramerBuilder empty")
	}
	ln, err := s.getTCPListener(opts)
	if err != nil {
		return fmt.Errorf("get tcp listener err: %w", err)
	}
	// We MUST save the raw TCP listener (instead of (*tls.listener) if TLS is enabled)
	// to guarantee the underlying fd can be successfully retrieved for hot restart.
	listenersMap.Store(ln, struct{}{})
	ln, err = mayLiftToTLSListener(ln, opts)
	if err != nil {
		return fmt.Errorf("may lift to tls listener err: %w", err)
	}
	go s.serveStream(ctx, ln, opts)
	return nil
}

func mayLiftToTLSListener(ln net.Listener, opts *ListenServeOptions) (net.Listener, error) {
	if !(len(opts.TLSCertFile) > 0 && len(opts.TLSKeyFile) > 0) {
		return ln, nil
	}
	// Enable TLS.
	tlsConf, err := itls.GetServerConfig(opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("tls get server config err: %w", err)
	}
	return tls.NewListener(ln, tlsConf), nil
}

func (s *serverTransport) serveStream(ctx context.Context, ln net.Listener, opts *ListenServeOptions) error {
	return s.serveTCP(ctx, ln, opts)
}

// ---------------------------------packet server-----------------------------------------//

// listenAndServePacket starts listening, returns an error on failure.
func (s *serverTransport) listenAndServePacket(ctx context.Context, opts *ListenServeOptions) error {
	pool := createUDPRoutinePool(opts.Routines)
	// Reuse port. To speed up IO, the kernel dispatches IO ReadReady events to threads.
	if s.opts.ReusePort {
		reuseport.ListenerBacklogMaxSize = 4096
		cores := runtime.NumCPU()
		for i := 0; i < cores; i++ {
			udpconn, err := s.getUDPListener(opts)
			if err != nil {
				return err
			}
			listenersMap.Store(udpconn, struct{}{})

			go s.servePacket(ctx, udpconn, pool, opts)
		}
	} else {
		udpconn, err := s.getUDPListener(opts)
		if err != nil {
			return err
		}
		listenersMap.Store(udpconn, struct{}{})

		go s.servePacket(ctx, udpconn, pool, opts)
	}
	return nil
}

// getUDPListener gets UDP listener.
func (s *serverTransport) getUDPListener(opts *ListenServeOptions) (udpConn net.PacketConn, err error) {
	v, _ := os.LookupEnv(EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// Find the passed listener.
		ln, err := getPassedListener(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}
		listener, ok := ln.(net.PacketConn)
		if !ok {
			return nil, errors.New("invalid net.PacketConn")
		}
		return listener, nil
	}

	if s.opts.ReusePort {
		udpConn, err = reuseport.ListenPacket(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("udp reuseport error:%v", err)
		}
	} else {
		udpConn, err = net.ListenPacket(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("udp listen error:%v", err)
		}
	}

	return udpConn, nil
}

func (s *serverTransport) servePacket(ctx context.Context, rwc net.PacketConn, pool *ants.PoolWithFunc,
	opts *ListenServeOptions) error {
	switch rwc := rwc.(type) {
	case *net.UDPConn:
		return s.serveUDP(ctx, rwc, pool, opts)
	default:
		return errors.New("transport not support PacketConn impl")
	}
}

// ------------------------ tcp/udp connection structures ----------------------------//

func (s *serverTransport) newConn(ctx context.Context, opts *ListenServeOptions) *conn {
	idleTimeout := opts.IdleTimeout
	if s.opts.IdleTimeout > 0 {
		idleTimeout = s.opts.IdleTimeout
	}
	return &conn{
		ctx:         ctx,
		handler:     opts.Handler,
		idleTimeout: idleTimeout,
	}
}

// conn is the struct of connection which is established when server receive a client connecting
// request.
type conn struct {
	ctx         context.Context
	cancelCtx   context.CancelFunc
	idleTimeout time.Duration
	lastVisited time.Time
	handler     Handler
}

func (c *conn) handle(ctx context.Context, req []byte) ([]byte, error) {
	return c.handler.Handle(ctx, req)
}

func (c *conn) handleClose(ctx context.Context) error {
	if closeHandler, ok := c.handler.(CloseHandler); ok {
		return closeHandler.HandleClose(ctx)
	}
	return nil
}

var errNotFound = errors.New("listener not found")

// GetPassedListener gets the inherited listener from parent process by network and address.
func GetPassedListener(network, address string) (interface{}, error) {
	return getPassedListener(network, address)
}

func getPassedListener(network, address string) (interface{}, error) {
	once.Do(inheritListeners)

	key := network + ":" + address
	v, ok := inheritedListenersMap.Load(key)
	if !ok {
		return nil, errNotFound
	}

	listeners := v.([]interface{})
	if len(listeners) == 0 {
		return nil, errNotFound
	}

	ln := listeners[0]
	listeners = listeners[1:]
	if len(listeners) == 0 {
		inheritedListenersMap.Delete(key)
	} else {
		inheritedListenersMap.Store(key, listeners)
	}

	return ln, nil
}

// ListenFd is the listener fd.
type ListenFd struct {
	OriginalListenCloser io.Closer
	Fd                   uintptr
	Name                 string
	Network              string
	Address              string
}

// inheritListeners stores the listener according to start listenfd and number of listenfd passed
// by environment variables.
func inheritListeners() {
	firstListenFd, err := strconv.ParseUint(os.Getenv(EnvGraceFirstFd), 10, 32)
	if err != nil {
		log.Errorf("invalid %s, error: %v", EnvGraceFirstFd, err)
	}

	num, err := strconv.ParseUint(os.Getenv(EnvGraceRestartFdNum), 10, 32)
	if err != nil {
		log.Errorf("invalid %s, error: %v", EnvGraceRestartFdNum, err)
	}

	for fd := firstListenFd; fd < firstListenFd+num; fd++ {
		file := os.NewFile(uintptr(fd), "")
		listener, addr, err := fileListener(file)
		file.Close()
		if err != nil {
			log.Errorf("get file listener error: %v", err)
			continue
		}

		key := addr.Network() + ":" + addr.String()
		v, ok := inheritedListenersMap.LoadOrStore(key, []interface{}{listener})
		if ok {
			listeners := v.([]interface{})
			listeners = append(listeners, listener)
			inheritedListenersMap.Store(key, listeners)
		}
	}
}

func fileListener(file *os.File) (interface{}, net.Addr, error) {
	// Check file status.
	fin, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	// Is this a socket fd.
	if fin.Mode()&os.ModeSocket == 0 {
		return nil, nil, errFileIsNotSocket
	}

	// tcp, tcp4 or tcp6.
	if listener, err := net.FileListener(file); err == nil {
		return listener, listener.Addr(), nil
	}

	// udp, udp4 or udp6.
	if packetConn, err := net.FilePacketConn(file); err == nil {
		return packetConn, packetConn.LocalAddr(), nil
	}

	return nil, nil, errUnSupportedNetworkType
}

func getPacketConnFd(c net.PacketConn) (*ListenFd, error) {
	sc, ok := c.(syscall.Conn)
	if !ok {
		return nil, fmt.Errorf("getPacketConnFd err: %w", errUnSupportedListenerType)
	}
	lnFd, err := getRawFd(sc)
	if err != nil {
		return nil, fmt.Errorf("getPacketConnFd getRawFd err: %w", err)
	}
	return &ListenFd{
		OriginalListenCloser: c,
		Fd:                   lnFd,
		Name:                 "a udp listener fd",
		Network:              c.LocalAddr().Network(),
		Address:              c.LocalAddr().String(),
	}, nil
}

func getListenerFd(ln net.Listener) (*ListenFd, error) {
	sc, ok := ln.(syscall.Conn)
	if !ok {
		return nil, fmt.Errorf("getListenerFd err: %w", errUnSupportedListenerType)
	}
	fd, err := getRawFd(sc)
	if err != nil {
		return nil, fmt.Errorf("getListenerFd getRawFd err: %w", err)
	}
	return &ListenFd{
		OriginalListenCloser: ln,
		Fd:                   fd,
		Name:                 "a tcp listener fd",
		Network:              ln.Addr().Network(),
		Address:              ln.Addr().String(),
	}, nil
}

// getRawFd acts like:
//
//	func (ln *net.TCPListener) (uintptr, error) {
//		f, err := ln.File()
//		if err != nil {
//			return 0, err
//		}
//		fd, err := f.Fd()
//		if err != nil {
//			return 0, err
//		}
//	}
//
// But it differs in an important way:
//
//	The method (*os.File).Fd() will set the original file descriptor to blocking mode as a side effect of fcntl(),
//	which will lead to indefinite hangs of Close/Read/Write, etc.
//
// References:
//   - https://github.com/golang/go/issues/29277
//   - https://github.com/golang/go/issues/29277#issuecomment-447526159
//   - https://github.com/golang/go/issues/29277#issuecomment-448117332
//   - https://github.com/golang/go/issues/43894
func getRawFd(sc syscall.Conn) (uintptr, error) {
	c, err := sc.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("sc.SyscallConn err: %w", err)
	}
	var lnFd uintptr
	if err := c.Control(func(fd uintptr) {
		lnFd = fd
	}); err != nil {
		return 0, fmt.Errorf("c.Control err: %w", err)
	}
	return lnFd, nil
}
