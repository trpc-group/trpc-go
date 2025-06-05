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
	"io"
	"math"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/addrutil"
	icontext "trpc.group/trpc-go/trpc-go/internal/context"
	ierror "trpc.group/trpc-go/trpc-go/internal/error"
	ikeeporder "trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/internal/writev"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	ibufio "trpc.group/trpc-go/trpc-go/transport/internal/bufio"
	"trpc.group/trpc-go/trpc-go/transport/internal/frame"
)

type handleParam struct {
	req   []byte
	c     *tcpconn
	start time.Time
}

func (p *handleParam) reset() {
	p.req = nil
	p.c = nil
	p.start = time.Time{}
}

var handleParamPool = &sync.Pool{
	New: func() interface{} { return new(handleParam) },
}

func createRoutinePool(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pool, err := ants.NewPoolWithFunc(size, func(args interface{}) {
		param, ok := args.(*handleParam)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		report.TCPServerAsyncGoroutineScheduleDelay.Set(float64(time.Since(param.start).Microseconds()))
		if param.c == nil {
			log.Tracef("routine pool tcpconn is nil, shouldn't happen!")
			return
		}
		param.c.handleSync(param.req)
		param.reset()
		handleParamPool.Put(param)
	})
	if err != nil {
		log.Tracef("routine pool create error: %v", err)
		return nil
	}
	return pool
}

func (s *serverTransport) serveTCP(ctx context.Context, ln net.Listener, opts *ListenServeOptions) error {
	opts.ActiveCnt.Add(1)
	defer opts.ActiveCnt.Add(-1)
	// Create a goroutine pool if ServerAsync enabled.
	var pool *ants.PoolWithFunc
	if opts.ServerAsync {
		pool = createRoutinePool(opts.Routines)
	}
	for tempDelay := time.Duration(0); ; {
		rwc, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				tempDelay = nextTempDelay(tempDelay)
				log.Tracef("transport: accept error: %+v, tempDelay: %+v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			select {
			case <-ctx.Done(): // If this error is triggered by the user, such as during a restart,
				return err // it is possible to directly return the error, causing the current listener to exit.
			default:
				// Restricted access to the internal/poll.ErrNetClosing type necessitates comparing a string literal.
				const accept, closeError = "accept", "use of closed network connection"
				const msg = "the server transport, listening on %s, encountered an error: %+v; this error was handled" +
					" gracefully by the framework to prevent abnormal termination, serving as a reference for" +
					" investigating acceptance errors that can't be filtered by the Temporary interface"
				if e, ok := err.(*net.OpError); ok && e.Op == accept && strings.Contains(e.Err.Error(), closeError) {
					log.Infof("listener with address %s is closed", ln.Addr())
					return err
				}
				log.Errorf(msg, ln.Addr(), err)
				continue
			}
		}
		tempDelay = 0
		if tcpConn, ok := rwc.(*net.TCPConn); ok {
			if err := tcpConn.SetKeepAlive(true); err != nil {
				log.Tracef("tcp conn set keepalive error: %v", err)
			}
			if s.opts.KeepAlivePeriod > 0 {
				if err := tcpConn.SetKeepAlivePeriod(s.opts.KeepAlivePeriod); err != nil {
					log.Tracef("tcp conn set keepalive period error: %v", err)
				}
			}
		}

		key, tc := s.newTCPConn(ctx, rwc, pool, opts)
		s.m.Lock()
		s.addrToConn[key] = tc
		s.m.Unlock()

		opts.ActiveCnt.Add(1)
		go func() {
			tc.serve()
			opts.ActiveCnt.Add(-1)
		}()
	}
}

func (s *serverTransport) newTCPConn(
	ctx context.Context,
	rwc net.Conn,
	pool *ants.PoolWithFunc,
	opts *ListenServeOptions,
) (string, *tcpconn) {
	br := ibufio.NewReader(rwc, codec.GetReaderSize())
	tc := &tcpconn{
		conn:                           s.newConn(ctx, opts),
		rwc:                            rwc,
		bufReader:                      br,
		fr:                             opts.FramerBuilder.New(br),
		remoteAddr:                     rwc.RemoteAddr(),
		localAddr:                      rwc.LocalAddr(),
		serverAsync:                    opts.ServerAsync,
		writev:                         opts.Writev,
		keepOrderPreDecodeExtractor:    opts.KeepOrderPreDecodeExtractor,
		keepOrderPreUnmarshalExtractor: opts.KeepOrderPreUnmarshalExtractor,
		orderedGroups:                  opts.OrderedGroups,
		st:                             s,
		pool:                           pool,
		serviceActiveCnt:               opts.ActiveCnt,
	}
	// Start goroutine sending with writev.
	if tc.writev {
		tc.buffer = writev.NewBuffer()
		tc.closeNotify = make(chan struct{}, 1)
		tc.buffer.Start(tc.rwc, tc.closeNotify)
	}
	// To avoid over writing packages, checks whether should we copy packages by Framer and
	// some other configurations.
	tc.copyFrame = frame.ShouldCopy(opts.CopyFrame, tc.serverAsync, codec.IsSafeFramer(tc.fr))
	return addrutil.AddrToKey(tc.localAddr, tc.remoteAddr), tc
}

func nextTempDelay(tempDelay time.Duration) time.Duration {
	if tempDelay == 0 {
		tempDelay = 5 * time.Millisecond
	} else {
		tempDelay *= 2
	}
	if max := 1 * time.Second; tempDelay > max {
		tempDelay = max
	}
	return tempDelay
}

// tcpconn is the connection which is established when server accept a client connecting request.
type tcpconn struct {
	*conn
	rwc         net.Conn
	bufReader   *ibufio.Reader
	fr          codec.Framer
	localAddr   net.Addr
	remoteAddr  net.Addr
	serverAsync bool
	writev      bool
	copyFrame   bool
	closeOnce   sync.Once
	st          *serverTransport
	pool        *ants.PoolWithFunc
	buffer      *writev.Buffer
	closeNotify chan struct{}

	serveDone chan struct{}

	// keepOrderPreDecodeExtractor specifies whether the current connection should
	// keep order for the incoming requests with respect to the extracted key from the decoded information.
	keepOrderPreDecodeExtractor ikeeporder.PreDecodeExtractor
	// keepOrderPreUnmarshalExtractor specifies whether the current connection should
	// keep order for the incoming requests with respect to the extracted key from request struct.
	keepOrderPreUnmarshalExtractor ikeeporder.PreUnmarshalExtractor
	// orderedGroups specifies the groups in which to keep order for incoming requests.
	orderedGroups ikeeporder.OrderedGroups

	// serviceActiveCnt comes from the service for which tcpconn is serving.
	// It's tcpconn's responsibility to +/- serviceActiveCnt.
	// activeCnt-1 represents remaining requests within tcpconn for which responses have not yet been sent.
	// The one comes from tcp connection reading loop.
	// It works as if a reference cnt for tcpconn.close.
	serviceActiveCnt activeCnt
	activeCnt        int32
}

// close closes socket and cleans up.
func (c *tcpconn) close() {
	c.closeOnce.Do(func() {
		// Send error msg to handler.
		ctx, msg := codec.WithNewMessage(context.Background())
		defer codec.PutBackMessage(msg)
		msg.WithLocalAddr(c.localAddr)
		msg.WithRemoteAddr(c.remoteAddr)
		msg.WithServerRspErr(errs.NewFrameError(errs.RetServerSystemErr, "Server connection closed"))
		// The connection closing message is handed over to handler.
		if err := c.conn.handleClose(ctx); err != nil {
			log.Trace("transport: notify connection close failed", err)
		}
		// Notify to stop writev sending goroutine.
		if c.writev {
			close(c.closeNotify)
		}

		// Remove cache in server stream transport.
		key := addrutil.AddrToKey(c.localAddr, c.remoteAddr)
		c.st.m.Lock()
		delete(c.st.addrToConn, key)
		c.st.m.Unlock()

		// Finally, close the socket connection.
		c.rwc.Close()
	})
}

// write encapsulates tcp conn write.
func (c *tcpconn) write(p []byte) (int, error) {
	if c.writev {
		return c.buffer.Write(p)
	}
	return c.rwc.Write(p)
}

func (c *tcpconn) serve() {
	atomic.AddInt32(&c.activeCnt, 1)
	c.serveDone = make(chan struct{})
	defer close(c.serveDone)
	addConnection(c)

	var drainBuffer bool
	var readDeadline time.Time
	lastVisited := time.Now()
	// The updateInterval is the minimum of 5s and c.readTimeout/2.
	updateInterval := minDuration(5*time.Second, c.readTimeout/2)
	for {
		now := time.Now()
		if c.idleTimeout > 0 && now.Sub(lastVisited) > c.idleTimeout {
			report.TCPServerTransportIdleTimeout.Incr()
			return
		}
		if c.readTimeout > 0 && readDeadline.Sub(now) < c.readTimeout-updateInterval {
			readDeadline = now.Add(c.readTimeout)
			if err := c.rwc.SetReadDeadline(readDeadline); err != nil {
				log.Trace("transport: tcpconn SetReadDeadline fail ", err)
				return
			}
		}
		req, err := c.fr.ReadFrame()
		if err != nil {
			if err == io.EOF {
				report.TCPServerTransportReadEOF.Incr() // client has closed the connections.
				return
			}
			// Server closes the connection if client sends no package in last idle timeout.
			if e, ok := err.(net.Error); ok && e.Timeout() {
				if errors.Is(icontext.Cause(c.ctx), ierror.GracefulRestart) {
					return
				}
				continue
			}
			report.TCPServerTransportReadFail.Incr()
			log.Trace("transport: tcpconn serve ReadFrame fail ", err)
			return
		}

		lastVisited = now
		c.serviceActiveCnt.Add(1)
		atomic.AddInt32(&c.activeCnt, 1)
		if !drainBuffer {
			select {
			case <-c.ctx.Done():
				if !errors.Is(icontext.Cause(c.ctx), ierror.GracefulRestart) {
					c.decrActiveCnt()
					return
				}
				drainBuffer = true
				c.bufReader.Unbuffer()
			default:
			}
		}

		report.TCPServerTransportReceiveSize.Set(float64(len(req)))
		// if framer is not concurrent safe, copy the data to avoid over writing.
		if c.copyFrame {
			reqCopy := make([]byte, len(req))
			copy(reqCopy, req)
			req = reqCopy
		}
		c.handle(req)
		if drainBuffer && c.bufReader.Buffered() == 0 {
			return
		}
	}
}

func minDuration(d1, d2 time.Duration) time.Duration {
	if d2 < d1 {
		return d2
	}
	return d1
}

func (c *tcpconn) handle(req []byte) {
	if c.keepOrderPreDecodeExtractor != nil {
		if ok := c.handleKeepOrderPreDecode(req); ok {
			return
		}
		// If not ok, the request will be processed as a normal non-keep-order request.
	}
	if c.keepOrderPreUnmarshalExtractor != nil {
		if ok := c.handleKeepOrderPreUnmarshal(req); ok {
			return
		}
		// If not ok, the request will be processed as a normal non-keep-order request.
	}

	if !c.serverAsync || c.pool == nil || frame.ContainTRPCStreamHeader(req) {
		c.handleSync(req)
		return
	}

	// Using sync.pool to dispatch package processing goroutine parameters can reduce a memory
	// allocation and slightly promote performance.
	args := handleParamPool.Get().(*handleParam)
	args.req = req
	args.c = c
	args.start = time.Now()
	if err := c.pool.Invoke(args); err != nil {
		report.TCPServerTransportJobQueueFullFail.Incr()
		log.Trace("transport: tcpconn serve routine pool put job queue fail ", err)
		c.handleSyncWithErr(req, errs.ErrServerRoutinePoolBusy)
	}
}

func (c *tcpconn) handleKeepOrderPreDecode(req []byte) bool {
	pdh, ok := c.handler.(ikeeporder.PreDecodeHandler)
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
	keepOrderKey, ok := c.keepOrderPreDecodeExtractor(ctx, reqBody)
	if !ok {
		// Do not keep order.
		codec.PutBackMessage(msg)
		return false
	}
	ctx = ikeeporder.NewContextWithPreDecode(ctx, &ikeeporder.PreDecodeInfo{ReqBodyBuf: reqBody})
	c.orderedGroups.Add(keepOrderKey, func() {
		defer func() {
			codec.PutBackMessage(msg)
			c.decrActiveCnt()
			if err := recover(); err != nil {
				log.ErrorContextf(ctx, "[PANIC]%v\n%s\n", err, debug.Stack())
				report.PanicNum.Incr()
			}
		}()
		c.handleSyncWithErrAndContext(ctx, msg, req, nil)
	})
	return true
}

func (c *tcpconn) handleKeepOrderPreUnmarshal(req []byte) bool {
	puh, ok := c.handler.(ikeeporder.PreUnmarshalHandler)
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
	keepOrderKey, ok := c.keepOrderPreUnmarshalExtractor(ctx, reqBody)
	if !ok {
		// Do not keep order.
		codec.PutBackMessage(msg)
		return false
	}
	c.orderedGroups.Add(keepOrderKey, func() {
		defer func() {
			codec.PutBackMessage(msg)
			c.decrActiveCnt()
			if err := recover(); err != nil {
				log.ErrorContextf(ctx, "[PANIC]%v\n%s\n", err, debug.Stack())
				report.PanicNum.Incr()
			}
		}()
		c.handleSyncWithErrAndContext(ctx, msg, req, nil)
	})
	return true
}

func (c *tcpconn) handleSync(req []byte) {
	c.handleSyncWithErr(req, nil)
}

func (c *tcpconn) handleSyncWithErr(req []byte, e error) {
	defer c.decrActiveCnt()

	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	c.handleSyncWithErrAndContext(ctx, msg, req, e)
}

func (c *tcpconn) handleSyncWithErrAndContext(ctx context.Context, msg codec.Msg, req []byte, e error) {
	msg.WithServerRspErr(e)
	// Record local addr and remote addr to context.
	msg.WithLocalAddr(c.localAddr)
	msg.WithRemoteAddr(c.remoteAddr)

	var (
		span             rpcz.Span
		ender            rpcz.Ender
		sendMessageEnder rpcz.Ender
	)
	if rpczenable.Enabled {
		span, ender, ctx = rpcz.NewSpanContext(ctx, "server")
		span.SetAttribute(rpcz.TRPCAttributeRequestSize, len(req))
	}

	rsp, err := c.conn.handle(ctx, req)

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
			report.TCPServerTransportHandleFail.Incr()
			log.Trace("transport: tcpconn serve handle fail ", err)
			c.close()
			return
		}
		// On stream RPC, server does not need to write rsp, just returns.
		return
	}
	report.TCPServerTransportSendSize.Set(float64(len(rsp)))
	if rpczenable.Enabled {
		span.SetAttribute(rpcz.TRPCAttributeResponseSize, len(rsp))
		_, sendMessageEnder = span.NewChild("SendMessage")
	}
	// common RPC write rsp.
	_, err = c.write(rsp)
	if rpczenable.Enabled {
		sendMessageEnder.End()
	}

	if err != nil {
		report.TCPServerTransportWriteFail.Incr()
		log.Trace("transport: tcpconn write fail ", err)
		c.close()
	}
}

func (c *tcpconn) decrActiveCnt() {
	if atomic.AddInt32(&c.activeCnt, -1) == 0 {
		c.close()
	}
	c.serviceActiveCnt.Add(-1)
}
