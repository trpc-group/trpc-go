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
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	igr "trpc.group/trpc-go/trpc-go/internal/graceful"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
	ierrs "trpc.group/trpc-go/trpc-go/transport/internal/errs"
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
var DefaultServerTransport = NewServerStreamTransport(WithReusePort(true))

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
	lsopts.fixKeepOrder()

	if lsopts.Listener != nil {
		return s.listenAndServeStream(ctx, lsopts)
	}
	// Support simultaneous listening TCP and UDP.
	networks := strings.Split(lsopts.Network, ",")
	for _, network := range networks {
		lsopts.Network = network
		switch lsopts.Network {
		case protocol.TCP, protocol.TCP4, protocol.TCP6, protocol.UNIX:
			if err := s.listenAndServeStream(ctx, lsopts); err != nil {
				return err
			}
		case protocol.UDP, protocol.UDP4, protocol.UDP6:
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
	// inheritedListenersMap records the listeners inherited from the parent process.
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
func (s *serverTransport) getTCPListener(opts *ListenServeOptions) (net.Listener, error) {
	if opts.Listener != nil {
		return opts.Listener, nil
	}
	listener, err := igr.Listen(opts.Network, opts.Address, s.opts.ReusePort)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to graceful restart listen %s: %s: %w", opts.Network, opts.Address, err)
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
	ln, err = itls.MayLiftToTLSListener(ln, opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("may lift to tls listener failed, CACertFile(%s), TLSCertFile(%s), TLSKeyFile(%s): %w",
			opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile, err)
	}
	go func() {
		if err := s.serveStream(ctx, ln, opts); err != nil {
			log.Infof("serve stream exited: %v", err)
		}
	}()
	return nil
}

func (s *serverTransport) serveStream(ctx context.Context, ln net.Listener, opts *ListenServeOptions) error {
	var once sync.Once
	closeListener := func() { ln.Close() }
	defer once.Do(closeListener)
	// Create a goroutine to watch ctx.Done() channel.
	// Once Server.Close(), TCP listener should be closed immediately and won't accept any new connection.
	go func() {
		select {
		case <-ctx.Done():
		// ctx.Done will perform the following two actions:
		// 1. Stop listening.
		// 2. Cancel all currently established connections.
		// Whereas opts.StopListening will only stop listening.
		case <-opts.StopListening:
		}
		log.Tracef("recv server close event")
		once.Do(closeListener)
	}()
	return s.serveTCP(ctx, ln, opts)
}

// ---------------------------------packet server-----------------------------------------//

// listenAndServePacket starts listening, returns an error on failure.
func (s *serverTransport) listenAndServePacket(ctx context.Context, opts *ListenServeOptions) error {
	pool := createUDPRoutinePool(opts.Routines)
	listenerNum := 1
	// Reuse port. To speed up IO, the kernel dispatches IO ReadReady events to threads.
	if s.opts.ReusePort {
		// reuseport.ListenerBacklogMaxSize = 4096
		// Use runtime.GOMAXPROCS(0) to get the actual number of available CPUs instead of runtime.NumCPU().
		// This helps avoid creating too many listeners in containerized environments.
		listenerNum = runtime.GOMAXPROCS(0)
	}
	for i := 0; i < listenerNum; i++ {
		udpconn, err := s.getUDPListener(opts)
		if err != nil {
			return err
		}
		go func() {
			if err := s.serveUDP(ctx, udpconn, pool, opts); err != nil {
				log.Infof("serve packet failed: %v", err)
			}
		}()
	}
	return nil
}

// getUDPListener gets UDP listener.
func (s *serverTransport) getUDPListener(opts *ListenServeOptions) (net.PacketConn, error) {
	udpConn := opts.UDPListener
	if udpConn != nil {
		return udpConn, nil
	}
	udpConn, err := igr.ListenPacket(opts.Network, opts.Address, s.opts.ReusePort)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to graceful restart listen packet %s:%s: %w", opts.Network, opts.Address, err)
	}
	return udpConn, nil
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
		readTimeout: opts.ReadTimeout,
	}
}

// conn is the struct of connection which is established when server receive a client connecting
// request.
type conn struct {
	ctx         context.Context
	idleTimeout time.Duration
	readTimeout time.Duration
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

// GetPassedListener gets the inherited listener from parent process by network and address.
func GetPassedListener(network, address string) (interface{}, error) {
	return getPassedListener(network, address)
}

func getPassedListener(network, address string) (interface{}, error) {
	once.Do(inheritListeners)

	key := network + ":" + address
	v, ok := inheritedListenersMap.Load(key)
	if !ok {
		return nil, ierrs.ErrListenerNotFound
	}

	listeners := v.([]interface{})
	if len(listeners) == 0 {
		return nil, ierrs.ErrListenerNotFound
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
	// Deprecated: File field is no longer usable.
	File    *os.File
	Fd      uintptr
	Name    string
	Network string
	Address string
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
		Fd:      lnFd,
		Name:    "a udp listener fd",
		Network: c.LocalAddr().Network(),
		Address: c.LocalAddr().String(),
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
		Fd:      fd,
		Name:    "a tcp listener fd",
		Network: ln.Addr().Network(),
		Address: ln.Addr().String(),
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
