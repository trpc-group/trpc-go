// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package transport

import (
	"context"
	"fmt"
	"net"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/packetbuffer"
	"trpc.group/trpc-go/trpc-go/internal/report"
)

const defaultUDPRecvBufSize = 64 * 1024

// udpRoundTrip sends UDP requests.
func (c *clientTransport) udpRoundTrip(ctx context.Context, reqData []byte,
	opts *RoundTripOptions) ([]byte, error) {
	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"udp client transport: framer builder empty")
	}

	conn, addr, err := c.dialUDP(ctx, opts)
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

	buf := packetbuffer.New(conn, defaultUDPRecvBufSize)
	defer buf.Close()
	fr := opts.FramerBuilder.New(buf)
	req, err := fr.ReadFrame()
	if err != nil {
		report.UDPClientTransportReadFail.Incr()
		if e, ok := err.(net.Error); ok {
			if e.Timeout() {
				return nil, errs.NewFrameError(errs.RetClientTimeout,
					"udp client transport ReadFrame: "+err.Error())
			}
			return nil, errs.NewFrameError(errs.RetClientNetErr,
				"udp client transport ReadFrom: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientReadFrameErr,
			"udp client transport ReadFrame: "+err.Error())
	}
	// One packet of udp corresponds to one trpc packet,
	// and after parsing, there should not be any remaining data
	if err := buf.Next(); err != nil {
		report.UDPClientTransportUnRead.Incr()
		return nil, errs.NewFrameError(errs.RetClientReadFrameErr,
			fmt.Sprintf("udp client transport ReadFrame: %s", err))
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
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return errs.NewFrameError(errs.RetClientTimeout, "udp client transport WriteTo: "+err.Error())
		}
		return errs.NewFrameError(errs.RetClientNetErr, "udp client transport WriteTo: "+err.Error())
	}
	if num != len(reqData) {
		return errs.NewFrameError(errs.RetClientNetErr, "udp client transport WriteTo: num mismatch")
	}
	return nil
}

// dialUDP establishes an UDP connection.
func (c *clientTransport) dialUDP(ctx context.Context, opts *RoundTripOptions) (net.PacketConn, *net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr(opts.Network, opts.Address)
	if err != nil {
		return nil, nil, errs.NewFrameError(errs.RetClientNetErr,
			"udp client transport ResolveUDPAddr: "+err.Error())
	}

	var conn net.PacketConn
	if opts.ConnectionMode == Connected {
		var localAddr net.Addr
		if opts.LocalAddr != "" {
			localAddr, err = net.ResolveUDPAddr(opts.Network, opts.LocalAddr)
			if err != nil {
				return nil, nil, errs.NewFrameError(errs.RetClientNetErr,
					"udp client transport LocalAddr ResolveUDPAddr: "+err.Error())
			}
		}
		dialer := net.Dialer{
			LocalAddr: localAddr,
		}
		var udpConn net.Conn
		udpConn, err = dialer.Dial(opts.Network, opts.Address)
		if err != nil {
			return nil, nil, errs.NewFrameError(errs.RetClientConnectFail,
				fmt.Sprintf("dial udp fail: %s", err.Error()))
		}

		var ok bool
		conn, ok = udpConn.(net.PacketConn)
		if !ok {
			return nil, nil, errs.NewFrameError(errs.RetClientConnectFail,
				"udp conn not implement net.PacketConn")
		}
	} else {
		// Listen on all available IP addresses of the local system by default,
		// and a port number is automatically chosen.
		const defaultLocalAddr = ":"
		localAddr := defaultLocalAddr
		if opts.LocalAddr != "" {
			localAddr = opts.LocalAddr
		}
		conn, err = net.ListenPacket(opts.Network, localAddr)
	}
	if err != nil {
		return nil, nil, errs.NewFrameError(errs.RetClientNetErr, "udp client transport Dial: "+err.Error())
	}
	d, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(d)
	}
	return conn, addr, nil
}
