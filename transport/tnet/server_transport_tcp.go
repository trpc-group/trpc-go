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
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/tnet/tls"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/addrutil"
	ikeeporder "trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/internal/reuseport"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	intertls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
	ierrs "trpc.group/trpc-go/trpc-go/transport/internal/errs"
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
	if !ok {
		return s.listen(opts)
	}
	pln, err := transport.GetPassedListener(opts.Network, opts.Address)
	if err != nil {
		if errors.Is(err, ierrs.ErrListenerNotFound) {
			log.Infof("listener %s:%s not found, maybe it is a new service, fallback to create a new listener",
				opts.Network, opts.Address)
			return s.listen(opts)
		}
		return nil, err
	}
	listener, ok := pln.(net.Listener)
	if !ok {
		log.Errorf("invalid net.Listener type: %T for %s:%s, want: net.Listener, fallback to create a new listener",
			pln, opts.Network, opts.Address)
		return s.listen(opts)
	}
	return listener, nil
}

func (s *serverTransport) listen(opts *transport.ListenServeOptions) (net.Listener, error) {
	var listener net.Listener
	if s.opts.ReusePort {
		var err error
		listener, err = reuseport.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("%s reuseport listen %s error: %w", opts.Network, opts.Address, err)
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
		rawConn:                        conn,
		pool:                           pool,
		handler:                        opts.Handler,
		serverAsync:                    opts.ServerAsync,
		framer:                         opts.FramerBuilder.New(conn),
		keepOrderPreDecodeExtractor:    opts.KeepOrderPreDecodeExtractor,
		keepOrderPreUnmarshalExtractor: opts.KeepOrderPreUnmarshalExtractor,
		orderedGroups:                  opts.OrderedGroups,
	}
	// To avoid overwriting packets, check whether we should copy packages by Framer and some other configurations.
	tc.copyFrame = frame.ShouldCopy(opts.CopyFrame, tc.serverAsync, codec.IsSafeFramer(tc.framer))

	s.storeConn(addrutil.AddrToKey(conn.LocalAddr(), conn.RemoteAddr()), tc)
	return tc
}

// onConnClosed is triggered after the connection with the client is closed.
func (s *serverTransport) onConnClosed(conn net.Conn, handler transport.Handler) {
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithLocalAddr(conn.LocalAddr())
	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithServerRspErr(errs.NewFrameError(errs.RetServerSystemErr, "Server connection closed"))
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
	// keepOrderPreDecodeExtractor specifies whether the current connection should
	// keep order for the incoming requests with respect to the extracted key from the decoded information.
	keepOrderPreDecodeExtractor ikeeporder.PreDecodeExtractor
	// keepOrderPreUnmarshalExtractor specifies whether the current connection should
	// keep order for the incoming requests with respect to the extracted key from request struct.
	keepOrderPreUnmarshalExtractor ikeeporder.PreUnmarshalExtractor
	// orderedGroups specifies the groups in which to keep order for incoming requests.
	orderedGroups ikeeporder.OrderedGroups
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

	if tc.keepOrderPreDecodeExtractor != nil {
		if ok := tc.handleKeepOrderPreDecode(req); ok {
			return nil
		}
	}

	if tc.keepOrderPreUnmarshalExtractor != nil {
		if ok := tc.handleKeepOrderPreUnmarshal(req); ok {
			return nil
		}
	}

	if !tc.serverAsync || tc.pool == nil || frame.ContainTRPCStreamHeader(req) {
		tc.handleSync(req)
		return nil
	}

	if err := tc.pool.Invoke(newTask(req, tc.handleSync)); err != nil {
		report.TCPServerTransportJobQueueFullFail.Incr()
		log.Trace("transport: tcpConn serve routine pool put job queue fail ", err)
		tc.handleSyncWithErr(req, errs.ErrServerRoutinePoolBusy)
	}
	return nil
}

func (tc *tcpConn) handleSync(req []byte) {
	tc.handleSyncWithErr(req, nil)
}

func (tc *tcpConn) handleSyncWithErr(req []byte, e error) {
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	tc.handleSyncWithErrAndContext(ctx, msg, req, e)
}

func (tc *tcpConn) handleSyncWithErrAndContext(ctx context.Context, msg codec.Msg, req []byte, e error) {
	msg.WithServerRspErr(e)
	msg.WithLocalAddr(tc.rawConn.LocalAddr())
	msg.WithRemoteAddr(tc.rawConn.RemoteAddr())

	var (
		span        rpcz.Span
		serverEnder rpcz.Ender
	)
	if rpczenable.Enabled {
		span, serverEnder, ctx = rpcz.NewSpanContext(ctx, "server")
		span.SetAttribute(rpcz.TRPCAttributeRequestSize, len(req))
	}

	rsp, err := tc.handle(ctx, req)
	if rpczenable.Enabled {
		defer func(serverEnder rpcz.Ender) {
			span.SetAttribute(rpcz.TRPCAttributeRPCName, msg.ServerRPCName())
			if err == nil {
				span.SetAttribute(rpcz.TRPCAttributeError, msg.ServerRspErr())
			} else {
				span.SetAttribute(rpcz.TRPCAttributeError, err)
			}
			serverEnder.End()
		}(serverEnder)
	}
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
	var sendMessageEnder rpcz.Ender
	if rpczenable.Enabled {
		span.SetAttribute(rpcz.TRPCAttributeResponseSize, len(rsp))
		_, sendMessageEnder = span.NewChild("SendMessage")
	}
	_, err = tc.rawConn.Write(rsp)
	if rpczenable.Enabled {
		sendMessageEnder.End()
	}
	if err != nil {
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

func (tc *tcpConn) handleKeepOrderPreDecode(req []byte) bool {
	pdh, ok := tc.handler.(ikeeporder.PreDecodeHandler)
	if !ok {
		panic("bug: handler must implement pre-decode interface for keep-order requests")
	}
	ctx, msg := codec.WithNewMessage(context.Background())
	reqBody, err := pdh.PreDecode(ctx, req)
	if err != nil {
		log.Warnf("pre-decode error: %+v, fallback to non-keep-order scenario", err)
		codec.PutBackMessage(msg)
		return false
	}
	keepOrderKey, ok := tc.keepOrderPreDecodeExtractor(ctx, reqBody)
	if !ok {
		codec.PutBackMessage(msg)
		return false
	}
	ctx = ikeeporder.NewContextWithPreDecode(ctx, &ikeeporder.PreDecodeInfo{ReqBodyBuf: reqBody})
	tc.orderedGroups.Add(keepOrderKey, func() {
		defer func() {
			codec.PutBackMessage(msg)
			if err := recover(); err != nil {
				log.ErrorContextf(ctx, "[PANIC]%v\n%s\n", err, debug.Stack())
				report.PanicNum.Incr()
			}
		}()
		tc.handleSyncWithErrAndContext(ctx, msg, req, nil)
	})
	return true
}

func (tc *tcpConn) handleKeepOrderPreUnmarshal(req []byte) bool {
	puh, ok := tc.handler.(ikeeporder.PreUnmarshalHandler)
	if !ok {
		panic("bug: handler must implement pre-unmarshal interface for keep-order requests")
	}
	ctx, msg := codec.WithNewMessage(context.Background())
	info := &ikeeporder.PreUnmarshalInfo{}
	ctx = ikeeporder.NewContextWithPreUnmarshal(ctx, info)
	reqBody, err := puh.PreUnmarshal(ctx, req)
	if err != nil {
		log.Warnf("pre-unmarshal error: %+v, fallback to non-keep-order scenario", err)
		codec.PutBackMessage(msg)
		return false
	}
	keepOrderKey, ok := tc.keepOrderPreUnmarshalExtractor(ctx, reqBody)
	if !ok {
		codec.PutBackMessage(msg)
		return false
	}
	tc.orderedGroups.Add(keepOrderKey, func() {
		defer func() {
			codec.PutBackMessage(msg)
			if err := recover(); err != nil {
				log.ErrorContextf(ctx, "[PANIC]%v\n%s\n", err, debug.Stack())
				report.PanicNum.Incr()
			}
		}()
		tc.handleSyncWithErrAndContext(ctx, msg, req, nil)
	})
	return true
}
