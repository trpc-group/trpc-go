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
	"bytes"
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
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
	ierrs "trpc.group/trpc-go/trpc-go/transport/internal/errs"
)

type udpTask struct {
	req        []byte
	remoteAddr net.Addr
	handle     udpHandler
	start      time.Time
}

type udpHandler = func(req []byte, remoteAddr net.Addr)

func (t *udpTask) reset() {
	t.req = nil
	t.remoteAddr = nil
	t.handle = nil
	t.start = time.Time{}
}

var udpTaskPool = &sync.Pool{
	New: func() interface{} { return new(udpTask) },
}

func newUDPTask(req []byte, remoteAddr net.Addr, handle udpHandler) *udpTask {
	t := udpTaskPool.Get().(*udpTask)
	t.req = req
	t.remoteAddr = remoteAddr
	t.handle = handle
	t.start = time.Now()
	return t
}

// createRoutinePool creates a goroutines pool to avoid the performance overhead caused
// by frequent creation and destruction of goroutines. It also helps to control the number
// of concurrent goroutines, which can prevent sudden spikes in traffic by implementing
// throttling mechanisms.
func createUDPRoutinePool(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pf := func(args interface{}) {
		t, ok := args.(*udpTask)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		t.handle(t.req, t.remoteAddr)
		t.reset()
		udpTaskPool.Put(t)
	}
	pool, err := ants.NewPoolWithFunc(size, pf)
	if err != nil {
		log.Tracef("routine pool create error: %v", err)
		return nil
	}
	return pool
}

// getUDPListener gets UDP listener.
func (s *serverTransport) getUDPListeners(opts *transport.ListenServeOptions) ([]tnet.PacketConn, error) {
	if opts.UDPListener != nil {
		listener, err := tnet.NewPacketConn(opts.UDPListener)
		if err != nil {
			return nil, fmt.Errorf("tnet new packet conn: %w", err)
		}
		return []tnet.PacketConn{listener}, nil
	}

	// During graceful restart, the relevant information has
	// already been stored in environment variables.
	v, _ := os.LookupEnv(transport.EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if !ok {
		return tnet.ListenPackets(opts.Network, opts.Address, s.opts.ReusePort)
	}
	pln, err := transport.GetPassedListener(opts.Network, opts.Address)
	if err != nil {
		if errors.Is(err, ierrs.ErrListenerNotFound) {
			log.Infof("listener %s:%s not found, maybe it is a new service, fallback to create a new listener",
				opts.Network, opts.Address)
			return tnet.ListenPackets(opts.Network, opts.Address, s.opts.ReusePort)
		}
		return nil, err
	}
	ln, ok := pln.(net.PacketConn)
	if !ok {
		log.Errorf("invalid net.PacketConn type: %T for %s:%s, want: net.PacketConn, fallback to create a new listener",
			pln, opts.Network, opts.Address)
		return tnet.ListenPackets(opts.Network, opts.Address, s.opts.ReusePort)
	}
	listener, err := tnet.NewPacketConn(ln)
	if err != nil {
		return nil, fmt.Errorf("tnet new packet conn %s:%s error: %w", opts.Network, opts.Address, err)
	}
	return []tnet.PacketConn{listener}, nil
}

func (s *serverTransport) listenAndServeUDP(ctx context.Context, opts *transport.ListenServeOptions) error {
	pool := createUDPRoutinePool(opts.Routines)

	listeners, err := s.getUDPListeners(opts)
	if err != nil {
		return fmt.Errorf("get UDP listeners: %w", err)
	}
	for _, listener := range listeners {
		if err := transport.SaveListener(listener); err != nil {
			return fmt.Errorf("save listener failed: %w", err)
		}
	}

	return s.startUDPService(ctx, listeners, pool, opts)
}

func (s *serverTransport) startUDPService(
	ctx context.Context,
	listeners []tnet.PacketConn,
	pool *ants.PoolWithFunc,
	opts *transport.ListenServeOptions,
) error {
	go func() {
		<-opts.StopListening
		for _, listener := range listeners {
			listener.Close()
		}
	}()
	for _, listener := range listeners {
		listener.SetMetaData(s.newUDPConn(listener, pool, opts))
	}
	tnetOpts := []tnet.Option{
		tnet.WithOnUDPClosed(func(conn tnet.PacketConn) error {
			s.onUDPConnClosed(conn, opts.Handler)
			return nil
		}),
		tnet.WithExactUDPBufferSizeEnabled(s.opts.ExactUDPBufferSizeEnabled),
	}
	if s.opts.MaxUDPPacketSize > 0 {
		tnetOpts = append(tnetOpts, tnet.WithMaxUDPPacketSize(s.opts.MaxUDPPacketSize))
	}
	svr, err := tnet.NewUDPService(
		listeners,
		func(conn tnet.PacketConn) error {
			m := conn.GetMetaData()
			return handleUDP(m)
		},
		tnetOpts...)
	if err != nil {
		return fmt.Errorf("tnet new UDP service: %w", err)
	}
	go svr.Serve(ctx)
	return nil
}

func (s *serverTransport) newUDPConn(conn tnet.PacketConn, pool *ants.PoolWithFunc,
	opts *transport.ListenServeOptions) *udpConn {
	return &udpConn{
		rawConn:       conn,
		pool:          pool,
		handler:       opts.Handler,
		framerBuilder: opts.FramerBuilder,
	}
}

// onConnClosed is triggered after the connection with the client is closed.
func (s *serverTransport) onUDPConnClosed(conn tnet.PacketConn, handler transport.Handler) {
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithLocalAddr(conn.LocalAddr())
	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithServerRspErr(errs.NewFrameError(errs.RetServerSystemErr, "Server connection closed"))
	if closeHandler, ok := handler.(transport.CloseHandler); ok {
		if err := closeHandler.HandleClose(ctx); err != nil {
			log.Trace("transport tnet: notify connection close failed", err)
		}
	}
}

func handleUDP(conn interface{}) error {
	uc, ok := conn.(*udpConn)
	if !ok {
		return errors.New("transport tnet: handler udp: conn should be a udpConn")
	}
	return uc.onRequest()
}

type udpConn struct {
	rawConn       tnet.PacketConn
	framerBuilder codec.FramerBuilder
	pool          *ants.PoolWithFunc
	handler       transport.Handler
}

// onRequest is triggered when there is incoming data on the connection with the client.
func (uc *udpConn) onRequest() error {
	packet, remoteAddr, err := uc.rawConn.ReadPacket()
	if err != nil {
		report.UDPServerTransportReadFail.Incr()
		log.Trace("transport tnet: udpConn onRequest ReadPacket fail ", err)
		return err
	}
	defer packet.Free()

	rawData, err := packet.Data()
	if err != nil {
		report.UDPServerTransportReadFail.Incr()
		log.Trace("transport tnet: udpConn onRequest GetData fail ", err)
		return err
	}
	buf := bytes.NewBuffer(rawData)
	framer := uc.framerBuilder.New(buf)
	req, err := framer.ReadFrame()
	if err != nil {
		report.UDPServerTransportReadFail.Incr()
		log.Trace("transport tnet: udpConn onRequest ReadFrame fail ", err)
		return err
	}
	report.UDPServerTransportReceiveSize.Set(float64(len(req)))
	if err := uc.pool.Invoke(newUDPTask(req, remoteAddr, uc.handleSync)); err != nil {
		report.UDPServerTransportJobQueueFullFail.Incr()
		log.Trace("transport tnet: udpConn serve routine pool put job queue fail ", err)
		uc.handleWithErr(req, remoteAddr, errs.ErrServerRoutinePoolBusy)
	}
	return nil
}

func (uc *udpConn) handleSync(req []byte, remoteAddr net.Addr) {
	uc.handleWithErr(req, remoteAddr, nil)
}

func (uc *udpConn) handleWithErr(req []byte, remoteAddr net.Addr, e error) {
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithServerRspErr(e)
	msg.WithLocalAddr(uc.rawConn.LocalAddr())
	msg.WithRemoteAddr(remoteAddr)

	var (
		span  rpcz.Span
		ender rpcz.Ender
	)
	if rpczenable.Enabled {
		span, ender, ctx = rpcz.NewSpanContext(ctx, "server")
		span.SetAttribute(rpcz.TRPCAttributeRequestSize, len(req))
	}

	rsp, err := uc.handle(ctx, req)
	if rpczenable.Enabled {
		defer func() {
			span.SetAttribute(rpcz.TRPCAttributeRPCName, msg.ServerRPCName())
			if err == nil {
				span.SetAttribute(rpcz.TRPCAttributeError, msg.ServerRspErr())
			} else {
				span.SetAttribute(rpcz.TRPCAttributeError, err)
			}
			ender.End()
		}()
	}
	if err != nil {
		if err != errs.ErrServerNoResponse {
			report.UDPServerTransportHandleFail.Incr()
			log.Trace("transport tnet: udpConn serve handle fail ", err)
			uc.close()
			return
		}
		return
	}
	if _, err = uc.writeTo(ctx, rsp, remoteAddr); err != nil {
		report.UDPServerTransportWriteFail.Incr()
		log.Trace("transport tnet: udpConn write fail ", err)
		uc.close()
		return
	}
}

func (uc *udpConn) handle(ctx context.Context, req []byte) ([]byte, error) {
	return uc.handler.Handle(ctx, req)
}

func (uc *udpConn) close() {
	if err := uc.rawConn.Close(); err != nil {
		log.Tracef("transport tnet: udpConn close fail %v", err)
	}
}

func (uc *udpConn) writeTo(ctx context.Context, rsp []byte, addr net.Addr) (n int, err error) {
	report.UDPServerTransportSendSize.Set(float64(len(rsp)))
	if rpczenable.Enabled {
		span := rpcz.SpanFromContext(ctx)
		span.SetAttribute(rpcz.TRPCAttributeResponseSize, len(rsp))
		_, ender := span.NewChild("SendMessage")
		defer ender.End()
	}
	return uc.rawConn.WriteTo(rsp, addr)
}
