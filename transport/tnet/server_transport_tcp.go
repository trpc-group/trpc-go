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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/tnet/tls"
	"trpc.group/trpc-go/trpc-go/internal/reuseport"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/addrutil"
	"trpc.group/trpc-go/trpc-go/internal/report"
	intertls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
	"trpc.group/trpc-go/trpc-go/transport/internal/frame"
)

type task struct {
	req    []byte
	handle handler
	start  time.Time
}

type handler = func(req []byte)

func (t *task) reset() {
	t.req = nil
	t.handle = nil
	t.start = time.Time{}
}

var taskPool = &sync.Pool{
	New: func() interface{} { return new(task) },
}

func newTask(req []byte, handle handler) *task {
	t := taskPool.Get().(*task)
	t.req = req
	t.handle = handle
	t.start = time.Now()
	return t
}

// createRoutinePool creates a goroutines pool to avoid the performance overhead caused
// by frequent creation and destruction of goroutines. It also helps to control the number
// of concurrent goroutines, which can prevent sudden spikes in traffic by implementing
// throttling mechanisms.
func createRoutinePool(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pf := func(args interface{}) {
		t, ok := args.(*task)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		report.TCPServerAsyncGoroutineScheduleDelay.Set(float64(time.Since(t.start).Microseconds()))
		t.handle(t.req)
		t.reset()
		taskPool.Put(t)
	}
	pool, err := ants.NewPoolWithFunc(size, pf)
	if err != nil {
		log.Tracef("routine pool create error: %v", err)
		return nil
	}
	return pool
}

func (s *serverTransport) getTCPListener(opts *transport.ListenServeOptions) (net.Listener, error) {
	if opts.Listener != nil {
		return opts.Listener, nil
	}

	// During graceful restart, the relevant information has
	// already been stored in environment variables.
	v, _ := os.LookupEnv(transport.EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		pln, err := transport.GetPassedListener(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}
		listener, ok := pln.(net.Listener)
		if !ok {
			return nil, errors.New("invalid net.Listener")
		}
		return listener, nil
	}
	var listener net.Listener
	if s.opts.ReusePort {
		var err error
		listener, err = reuseport.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("%s reuseport error: %w", opts.Network, err)
		}
		return listener, nil
	}
	return tnet.Listen(opts.Network, opts.Address)
}

func (s *serverTransport) listenAndServeTCP(ctx context.Context, opts *transport.ListenServeOptions) error {
	// Create a goroutine pool if ServerAsync enabled.
	var pool *ants.PoolWithFunc
	if opts.ServerAsync {
		pool = createRoutinePool(opts.Routines)
	}

	listener, err := s.getTCPListener(opts)
	if err != nil {
		return fmt.Errorf("trpc-tnet-transport get TCP listener fail, %w", err)
	}
	if err := transport.SaveListener(listener); err != nil {
		return fmt.Errorf("save tnet listener failed: %w", err)
	}

	if opts.TLSCertFile != "" && opts.TLSKeyFile != "" {
		return s.startTLSService(ctx, listener, pool, opts)
	}
	return s.startService(ctx, listener, pool, opts)
}

func (s *serverTransport) startService(
	ctx context.Context,
	listener net.Listener,
	pool *ants.PoolWithFunc,
	opts *transport.ListenServeOptions,
) error {
	go func() {
		<-opts.StopListening
		listener.Close()
	}()
	tnetOpts := []tnet.Option{
		tnet.WithOnTCPOpened(func(conn tnet.Conn) error {
			tc := s.onConnOpened(conn, pool, opts)
			conn.SetMetaData(tc)
			return nil
		}),
		tnet.WithOnTCPClosed(func(conn tnet.Conn) error {
			s.onConnClosed(conn, opts.Handler)
			return nil
		}),
		tnet.WithTCPIdleTimeout(opts.IdleTimeout),
		tnet.WithTCPKeepAlive(s.opts.KeepAlivePeriod),
	}
	svr, err := tnet.NewTCPService(
		listener,
		func(conn tnet.Conn) error {
			m := conn.GetMetaData()
			return handleTCP(m)
		},
		tnetOpts...)
	if err != nil {
		return fmt.Errorf("trpc-tnet-transport NewTCPService fail, %w", err)
	}
	go svr.Serve(ctx)
	return nil
}

func (s *serverTransport) startTLSService(
	ctx context.Context,
	listener net.Listener,
	pool *ants.PoolWithFunc,
	opts *transport.ListenServeOptions,
) error {
	conf, err := intertls.GetServerConfig(opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("get tls config fail: %w", err)
	}

	tlsOpts := []tls.ServerOption{
		tls.WithOnOpened(func(conn tls.Conn) error {
			tc := s.onConnOpened(conn, pool, opts)
			conn.SetMetaData(tc)
			return nil
		}),
		tls.WithOnClosed(func(conn tls.Conn) error {
			s.onConnClosed(conn, opts.Handler)
			return nil
		}),
		tls.WithServerTLSConfig(conf),
		tls.WithServerIdleTimeout(opts.IdleTimeout),
		tls.WithTCPKeepAlive(s.opts.KeepAlivePeriod),
	}
	svr, err := tls.NewService(
		listener,
		func(conn tls.Conn) error {
			m := conn.GetMetaData()
			return handleTCP(m)
		},
		tlsOpts...)
	if err != nil {
		return fmt.Errorf("trpc-tnet-transport TLS NewService fail, %w", err)
	}
	go svr.Serve(ctx)
	return nil
}

// onConnOpened is triggered after a successful connection is established with the client.
func (s *serverTransport) onConnOpened(conn net.Conn, pool *ants.PoolWithFunc,
	opts *transport.ListenServeOptions) *tcpConn {
	tc := &tcpConn{
		rawConn:     conn,
		pool:        pool,
		handler:     opts.Handler,
		serverAsync: opts.ServerAsync,
		framer:      opts.FramerBuilder.New(conn),
	}
	// To avoid overwriting packets, check whether we should copy packages by Framer and some other configurations.
	tc.copyFrame = frame.ShouldCopy(opts.CopyFrame, tc.serverAsync, codec.IsSafeFramer(tc.framer))

	s.storeConn(addrutil.AddrToKey(conn.LocalAddr(), conn.RemoteAddr()), tc)
	return tc
}

// onConnClosed is triggered after the connection with the client is closed.
func (s *serverTransport) onConnClosed(conn net.Conn, handler transport.Handler) {
	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithLocalAddr(conn.LocalAddr())
	msg.WithRemoteAddr(conn.RemoteAddr())
	e := &errs.Error{
		Type: errs.ErrorTypeFramework,
		Code: errs.RetServerSystemErr,
		Desc: "trpc",
		Msg:  "Server connection closed",
	}
	msg.WithServerRspErr(e)
	if closeHandler, ok := handler.(transport.CloseHandler); ok {
		if err := closeHandler.HandleClose(ctx); err != nil {
			log.Trace("transport: notify connection close failed", err)
		}
	}

	// Release the connection resources stored on the transport.
	s.deleteConn(addrutil.AddrToKey(conn.LocalAddr(), conn.RemoteAddr()))
}

func handleTCP(conn interface{}) error {
	tc, ok := conn.(*tcpConn)
	if !ok {
		return errors.New("bug: tcpConn type assert fail")
	}
	return tc.onRequest()
}

type tcpConn struct {
	rawConn     net.Conn
	framer      transport.Framer
	pool        *ants.PoolWithFunc
	handler     transport.Handler
	serverAsync bool
	copyFrame   bool
}

// onRequest is triggered when there is incoming data on the connection with the client.
func (tc *tcpConn) onRequest() error {
	req, err := tc.framer.ReadFrame()
	if err != nil {
		if err == tnet.ErrConnClosed {
			report.TCPServerTransportReadEOF.Incr()
			return err
		}
		report.TCPServerTransportReadFail.Incr()
		log.Trace("transport: tcpConn onRequest ReadFrame fail ", err)
		return err
	}
	if tc.copyFrame {
		reqCopy := make([]byte, len(req))
		copy(reqCopy, req)
		req = reqCopy
	}
	report.TCPServerTransportReceiveSize.Set(float64(len(req)))

	if !tc.serverAsync || tc.pool == nil {
		tc.handleSync(req)
		return nil
	}

	if err := tc.pool.Invoke(newTask(req, tc.handleSync)); err != nil {
		report.TCPServerTransportJobQueueFullFail.Incr()
		log.Trace("transport: tcpConn serve routine pool put job queue fail ", err)
		tc.handleWithErr(req, errs.ErrServerRoutinePoolBusy)
	}
	return nil
}

func (tc *tcpConn) handleSync(req []byte) {
	tc.handleWithErr(req, nil)
}

func (tc *tcpConn) handleWithErr(req []byte, e error) {
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithServerRspErr(e)
	msg.WithLocalAddr(tc.rawConn.LocalAddr())
	msg.WithRemoteAddr(tc.rawConn.RemoteAddr())

	rsp, err := tc.handle(ctx, req)
	if err != nil {
		if err != errs.ErrServerNoResponse {
			report.TCPServerTransportHandleFail.Incr()
			log.Trace("transport: tcpConn serve handle fail ", err)
			tc.close()
			return
		}
		return
	}
	report.TCPServerTransportSendSize.Set(float64(len(rsp)))
	if _, err = tc.rawConn.Write(rsp); err != nil {
		report.TCPServerTransportWriteFail.Incr()
		log.Trace("transport: tcpConn write fail ", err)
		tc.close()
		return
	}
}

func (tc *tcpConn) handle(ctx context.Context, req []byte) ([]byte, error) {
	return tc.handler.Handle(ctx, req)
}

func (tc *tcpConn) close() {
	if err := tc.rawConn.Close(); err != nil {
		log.Tracef("transport: tcpConn close fail %v", err)
	}
}
