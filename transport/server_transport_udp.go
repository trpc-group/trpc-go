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
	"math"
	"net"
	"time"

	"github.com/panjf2000/ants/v2"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/packetbuffer"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
)

func (s *serverTransport) serveUDP(ctx context.Context, rwc *net.UDPConn, pool *ants.PoolWithFunc,
	opts *ListenServeOptions) error {

	// Sets the size of the operating system's receive buffer associated with the connection.
	if s.opts.RecvUDPRawSocketBufSize > 0 {
		rwc.SetReadBuffer(s.opts.RecvUDPRawSocketBufSize)
	}

	var tempDelay time.Duration
	buf := packetbuffer.New(rwc, s.opts.RecvUDPPacketBufferSize)
	defer buf.Close()
	fr := opts.FramerBuilder.New(buf)
	copyFrame := !codec.IsSafeFramer(fr)

	for {
		select {
		case <-ctx.Done():
			return errors.New("recv server close event")
		default:
		}

		req, err := fr.ReadFrame()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			report.UDPServerTransportReadFail.Incr()
			log.Trace("transport: udpconn serve ReadFrame fail ", err)
			buf.Next()
			continue
		}
		tempDelay = 0

		report.UDPServerTransportReceiveSize.Set(float64(len(req)))

		remoteAddr, ok := buf.CurrentPacketAddr().(*net.UDPAddr)
		if !ok {
			log.Trace("transport: udpconn serve address is not udp address")
			buf.Next()
			continue
		}

		// One packet of udp corresponds to one trpc packet,
		// and after parsing, there should not be any remaining data.
		if err := buf.Next(); err != nil {
			report.UDPServerTransportUnRead.Incr()
			log.Trace("transport: udpconn serve ReadFrame data remaining bytes data, ", err)
			continue
		}

		c := &udpconn{
			conn:       s.newConn(ctx, opts),
			rwc:        rwc,
			remoteAddr: remoteAddr,
		}

		if copyFrame {
			c.req = make([]byte, len(req))
			copy(c.req, req)
		} else {
			c.req = req
		}

		if pool == nil {
			go c.serve()
			continue
		}
		if err := pool.Invoke(c); err != nil {
			report.UDPServerTransportJobQueueFullFail.Incr()
			log.Trace("transport: udpconn serve routine pool put job queue fail ", err)
			go c.serve()
		}
	}
}

// udpconn is the UDP connection which is established when server receives a client connecting
// request.
type udpconn struct {
	*conn
	req        []byte
	rwc        *net.UDPConn
	remoteAddr *net.UDPAddr
}

func (c *udpconn) serve() {
	// Generate a new empty message binding to the ctx.
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)

	// Set local address and remote address to message.
	msg.WithLocalAddr(c.rwc.LocalAddr())
	msg.WithRemoteAddr(c.remoteAddr)

	rsp, err := c.handle(ctx, c.req)
	if err != nil {
		if err != errs.ErrServerNoResponse {
			report.UDPServerTransportHandleFail.Incr()
			log.Tracef("udp handle fail:%v", err)
		}
		return
	}

	report.UDPServerTransportSendSize.Set(float64(len(rsp)))
	if _, err := c.rwc.WriteToUDP(rsp, c.remoteAddr); err != nil {
		report.UDPServerTransportWriteFail.Incr()
		log.Tracef("udp write out fail:%v", err)
		return
	}
}

func createUDPRoutinePool(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pool, err := ants.NewPoolWithFunc(size, func(args interface{}) {
		c, ok := args.(*udpconn)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		c.serve()
	})
	if err != nil {
		log.Tracef("routine pool create error:%v", err)
		return nil
	}
	return pool
}
