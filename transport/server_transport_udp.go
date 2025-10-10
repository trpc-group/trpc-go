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
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	icontext "trpc.group/trpc-go/trpc-go/internal/context"
	ierror "trpc.group/trpc-go/trpc-go/internal/error"
	"trpc.group/trpc-go/trpc-go/internal/packetbuffer"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
)

type handleUDPParam struct {
	req        []byte
	remoteAddr net.Addr
	uc         *udpconn
}

func (p *handleUDPParam) reset() {
	p.req = nil
	p.uc = nil
	p.remoteAddr = nil
}

var handleUDPParamPool = &sync.Pool{
	New: func() interface{} { return new(handleUDPParam) },
}

func createUDPRoutinePool(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pool, err := ants.NewPoolWithFunc(size, func(args interface{}) {
		param, ok := args.(*handleUDPParam)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		if param.uc == nil {
			log.Tracef("routine pool udpconn is nil, shouldn't happen!")
			return
		}
		param.uc.handleSync(param.req, param.remoteAddr)
		param.reset()
		handleUDPParamPool.Put(param)
	})
	if err != nil {
		log.Tracef("routine pool create error:%v", err)
		return nil
	}
	return pool
}

func (s *serverTransport) serveUDP(ctx context.Context, rwc net.PacketConn, pool *ants.PoolWithFunc,
	opts *ListenServeOptions) error {
	uc := s.newUDPConn(ctx, rwc, pool, opts)
	uc.incrActiveCnt()
	defer func() {
		uc.decrActiveCnt()
	}()

	// Sets the size of the operating system's receive buffer associated with the connection.
	type readBufferSetter interface {
		// Sourced from *net.UDPConn
		SetReadBuffer(bytes int) error
	}
	if rwc, ok := rwc.(readBufferSetter); ok && s.opts.RecvUDPRawSocketBufSize > 0 {
		rwc.SetReadBuffer(s.opts.RecvUDPRawSocketBufSize)
	}

	var tempDelay time.Duration
	buf := packetbuffer.New(make([]byte, s.opts.RecvUDPPacketBufferSize))
	fr := opts.FramerBuilder.New(buf)
	copyFrame := !codec.IsSafeFramer(fr)
	for {
		select {
		case <-opts.StopListening:
			return errors.New("recv server close event: stop listening")
		case <-ctx.Done():
			return fmt.Errorf("recv server close event: %w", ctx.Err())
		default:
		}
		// Clean up buffer before reading new data.
		buf.Reset()
		num, raddr, err := rwc.ReadFrom(buf.Bytes())
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				tempDelay = nextTempDelay(tempDelay)
				log.Tracef("transport: udpconn serve ReadFrom error: %+v, tempDelay: %+v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		// Update the buffer according to the actual length of the received data.
		buf.Advance(num)
		req, err := fr.ReadFrame()
		if err != nil {
			report.UDPServerTransportReadFail.Incr()
			log.Trace("transport: udpconn serve ReadFrame fail ", err)
			continue
		}
		report.UDPServerTransportReceiveSize.Set(float64(len(req)))
		if buf.UnRead() > 0 {
			report.UDPServerTransportUnRead.Incr()
			log.Trace("transport: udpconn serve ReadFrame data remaining %d bytes data", buf.UnRead())
			continue
		}

		uc.incrActiveCnt()
		select {
		case <-ctx.Done():
			if !errors.Is(icontext.Cause(ctx), ierror.GracefulRestart) {
				uc.decrActiveCnt()
				return fmt.Errorf("recv server close event: %w", ctx.Err())
			}
		default:
		}

		if copyFrame {
			reqCopy := make([]byte, len(req))
			copy(reqCopy, req)
			req = reqCopy
		}
		uc.handle(req, raddr)
	}
}

func (s *serverTransport) newUDPConn(
	ctx context.Context,
	rwc net.PacketConn,
	pool *ants.PoolWithFunc,
	opts *ListenServeOptions,
) *udpconn {
	uc := &udpconn{
		conn:             s.newConn(ctx, opts),
		rwc:              rwc,
		pool:             pool,
		serviceActiveCnt: opts.ActiveCnt,
	}
	return uc
}

// udpconn is the UDP connection which is established when server receives a client connecting
// request.
type udpconn struct {
	*conn
	rwc       net.PacketConn
	pool      *ants.PoolWithFunc
	closeOnce sync.Once
	// serviceActiveCnt is provided by the service that udpconn is serving.
	// It is udpconn's responsibility to increase or decrease serviceActiveCnt.
	serviceActiveCnt activeCnt
	// activeCnt is the reference count for udpconn.
	// When activeCnt reaches 0, udpconn.close is called.
	activeCnt int64
}

// close closes socket and cleans up.
func (c *udpconn) close() {
	c.closeOnce.Do(func() {
		// Send error msg to handler.
		ctx, msg := codec.WithNewMessage(context.Background())
		defer codec.PutBackMessage(msg)
		msg.WithLocalAddr(c.rwc.LocalAddr())
		msg.WithServerRspErr(errs.NewFrameError(errs.RetServerSystemErr, "Server connection closed"))
		// The connection closing message is handed over to handler.
		if err := c.conn.handleClose(ctx); err != nil {
			log.Trace("transport: notify connection close failed", err)
		}

		// Finally, close the socket connection.
		c.rwc.Close()
	})
}

// write encapsulates udp conn write.
func (c *udpconn) writeTo(p []byte, addr net.Addr) (int, error) {
	return c.rwc.WriteTo(p, addr)
}

func (c *udpconn) handle(req []byte, remoteAddr net.Addr) {
	args := handleUDPParamPool.Get().(*handleUDPParam)
	args.req = req
	args.remoteAddr = remoteAddr
	args.uc = c
	if err := c.pool.Invoke(args); err != nil {
		report.UDPServerTransportJobQueueFullFail.Incr()
		log.Trace("transport: udpconn serve routine pool put job queue fail ", err)
		c.handleSyncWithErr(req, remoteAddr, errs.ErrServerRoutinePoolBusy)
	}
}

func (c *udpconn) handleSync(req []byte, remoteAddr net.Addr) {
	c.handleSyncWithErr(req, remoteAddr, nil)
}

func (c *udpconn) handleSyncWithErr(req []byte, remoteAddr net.Addr, e error) {
	defer c.decrActiveCnt()

	// Generate a new empty message binding to the ctx.
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)

	// Set local address and remote address to message.
	msg.WithLocalAddr(c.rwc.LocalAddr())
	msg.WithRemoteAddr(remoteAddr)
	msg.WithServerRspErr(e)

	rsp, err := c.conn.handle(ctx, req)
	if err != nil {
		if err != errs.ErrServerNoResponse {
			report.UDPServerTransportHandleFail.Incr()
			log.Tracef("udp handle fail: %v", err)
		}
		return
	}

	report.UDPServerTransportSendSize.Set(float64(len(rsp)))
	if _, err := c.writeTo(rsp, remoteAddr); err != nil {
		report.UDPServerTransportWriteFail.Incr()
		log.Tracef("udp write to fail:%v", err)
		return
	}
}

func (c *udpconn) incrActiveCnt() {
	atomic.AddInt64(&c.activeCnt, 1)
	c.serviceActiveCnt.Add(1)
}

func (c *udpconn) decrActiveCnt() {
	if atomic.AddInt64(&c.activeCnt, -1) == 0 {
		c.close()
	}
	c.serviceActiveCnt.Add(-1)
}
