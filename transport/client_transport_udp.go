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
	"fmt"
	"net"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/packetbuffer"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/pool/objectpool"
	"trpc.group/trpc-go/trpc-go/transport/internal/dialer"
	ierrs "trpc.group/trpc-go/trpc-go/transport/internal/errs"
)

const defaultUDPRecvBufSize = 64 * 1024

var udpBufPool = objectpool.NewBytesPool(defaultUDPRecvBufSize)

// udpRoundTrip sends UDP requests.
func (c *clientTransport) udpRoundTrip(ctx context.Context, reqData []byte,
	opts *RoundTripOptions) ([]byte, error) {
	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"udp client transport: framer builder empty")
	}

	conn, addr, err := dialer.DialUDP(ctx, dialer.DialOptions{
		Network:        opts.Network,
		Address:        opts.Address,
		LocalAddr:      opts.LocalAddr,
		DialUDP:        dialer.DefaultDialUDP,
		DialTimeout:    opts.DialTimeout,
		ConnectionMode: opts.ConnectionMode,
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(addr)
	msg.WithLocalAddr(conn.LocalAddr())

	if ctx.Err() == context.Canceled {
		return nil, errs.NewFrameError(errs.RetClientCanceled,
			"udp client transport canceled before Write: "+ctx.Err().Error())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errs.NewFrameError(errs.RetClientTimeout,
			"udp client transport timeout before Write: "+ctx.Err().Error())
	}

	report.UDPClientTransportSendSize.Set(float64(len(reqData)))
	if err := c.udpWriteFrame(conn, reqData, addr, opts); err != nil {
		return nil, err
	}
	return c.udpReadFrame(ctx, conn, opts)
}

// udpReadFrame reads UDP frame.
func (c *clientTransport) udpReadFrame(
	ctx context.Context, conn net.PacketConn, opts *RoundTripOptions) ([]byte, error) {
	// If it is SendOnly, returns directly without waiting for the server's response.
	if opts.ReqType == SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	select {
	case <-ctx.Done():
		return nil, errs.NewFrameError(errs.RetClientTimeout, "udp client transport select after Write: "+ctx.Err().Error())
	default:
	}

	recvData := udpBufPool.Get()
	defer udpBufPool.Put(recvData)
	buf := packetbuffer.New(recvData)
	fr := opts.FramerBuilder.New(buf)
	// Receive server's response.
	num, _, err := conn.ReadFrom(buf.Bytes())
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return nil, errs.NewFrameError(errs.RetClientTimeout, "udp client transport ReadFrom: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr, "udp client transport ReadFrom: "+err.Error())
	}
	if num == 0 {
		return nil, errs.NewFrameError(errs.RetClientNetErr, "udp client transport ReadFrom: num empty")
	}
	// Update the buffer according to the actual length of the received data.
	buf.Advance(num)
	req, err := fr.ReadFrame()
	if err != nil {
		report.UDPClientTransportReadFail.Incr()
		return nil, errs.NewFrameError(errs.RetClientReadFrameErr,
			"udp client transport ReadFrame: "+err.Error())
	}
	// One packet of udp corresponds to one trpc packet,
	// and after parsing, there should not be any remaining data
	if buf.UnRead() > 0 {
		report.UDPClientTransportUnRead.Incr()
		return nil, errs.NewFrameError(errs.RetClientReadFrameErr,
			fmt.Sprintf("udp client transport ReadFrame: remaining %d bytes data", buf.UnRead()))
	}
	report.UDPClientTransportReceiveSize.Set(float64(len(req)))
	// Framer is used for every request so there is no need to copy memory.
	return req, nil
}

// udpWriteReqData write UDP frame.
func (c *clientTransport) udpWriteFrame(conn net.PacketConn,
	reqData []byte, addr *net.UDPAddr, opts *RoundTripOptions) error {
	// Sending udp request packets
	var num int
	var err error
	if opts.ConnectionMode == Connected {
		udpconn := conn.(*net.UDPConn)
		num, err = udpconn.Write(reqData)
	} else {
		num, err = conn.WriteTo(reqData, addr)
	}
	if err != nil {
		return ierrs.WrapAsClientTimeoutErrOr(err, errs.RetClientNetErr, "udp client transport WriteTo failed")
	}
	if num != len(reqData) {
		return errs.NewFrameError(errs.RetClientNetErr, "udp client transport WriteTo: num mismatch")
	}
	return nil
}
