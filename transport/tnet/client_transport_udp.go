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
	"fmt"
	"net"

	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
	"trpc.group/trpc-go/trpc-go/transport/internal/dialer"
	ierrs "trpc.group/trpc-go/trpc-go/transport/internal/errs"
)

func (c *clientTransport) udpRoundTrip(ctx context.Context, reqData []byte,
	opts *transport.RoundTripOptions) ([]byte, error) {
	ln, raddr, err := dialer.DialUDP(ctx, dialer.DialOptions{
		Network:                   opts.Network,
		Address:                   opts.Address,
		LocalAddr:                 opts.LocalAddr,
		DialUDP:                   dialUDP,
		DialTimeout:               opts.DialTimeout,
		ConnectionMode:            opts.ConnectionMode,
		ExactUDPBufferSizeEnabled: c.opts.ExactUDPBufferSizeEnabled,
	})
	if err != nil {
		return nil, err
	}
	defer ln.Close()
	conn, ok := ln.(tnet.PacketConn)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tnet udp client transport: conn is not a tnet.PacketConn")
	}

	msg := codec.Message(ctx)
	msg.WithRemoteAddr(raddr)
	msg.WithLocalAddr(conn.LocalAddr())
	// Send a request.
	report.UDPClientTransportSendSize.Set(float64(len(reqData)))
	if err := udpWriteFrame(ctx, conn, reqData, opts); err != nil {
		return nil, err
	}
	// Receive a response.
	rsp, err := udpReadFrame(ctx, conn, opts)
	if err != nil {
		report.UDPClientTransportReadFail.Incr()
		return nil, err
	}
	report.UDPClientTransportReceiveSize.Set(float64(len(rsp)))
	return rsp, nil
}

func udpWriteFrame(ctx context.Context, conn tnet.PacketConn, reqData []byte, opts *transport.RoundTripOptions) error {
	if rpczenable.Enabled {
		span := rpcz.SpanFromContext(ctx)
		_, ender := span.NewChild("SendMessage")
		defer ender.End()
	}

	// Sending udp request packets
	var num int
	var err error
	if opts.ConnectionMode == transport.Connected {
		num, err = conn.Write(reqData)
	} else {
		num, err = conn.WriteTo(reqData, codec.Message(ctx).RemoteAddr())
	}
	if err != nil {
		return ierrs.WrapAsClientTimeoutErrOr(err, errs.RetClientNetErr, "tnet udp client transport WriteTo failed")
	}
	if num != len(reqData) {
		return errs.NewFrameError(errs.RetClientNetErr, "tnet udp client transport WriteTo: num mismatch")
	}
	return nil
}

func udpReadFrame(ctx context.Context, conn tnet.PacketConn, opts *transport.RoundTripOptions) ([]byte, error) {
	if rpczenable.Enabled {
		span := rpcz.SpanFromContext(ctx)
		_, ender := span.NewChild("ReceiveMessage")
		defer ender.End()
	}

	// If it is SendOnly, returns directly without waiting for the server's response.
	if opts.ReqType == transport.SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	// Receive server's response.
	packet, _, err := conn.ReadPacket()
	if err != nil {
		return nil, ierrs.WrapAsClientTimeoutErrOr(err, errs.RetClientNetErr,
			"tnet udp client transport ReadPacket failed")
	}
	defer packet.Free()
	rawData, err := packet.Data()
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientNetErr, "tnet udp client transport read packet data: "+err.Error())
	}
	buf := bytes.NewBuffer(rawData)
	framer := opts.FramerBuilder.New(buf)
	rsp, err := framer.ReadFrame()
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientReadFrameErr, "tnet udp client transport ReadFrame: "+err.Error())
	}
	return rsp, nil
}

func dialUDP(ctx context.Context, opts dialer.DialOptions) (net.PacketConn, error) {
	if opts.ConnectionMode == transport.NotConnected {
		// Listen on all available IP addresses of the local system by default,
		// and a port number is automatically chosen.
		const defaultLocalAddr = ":"
		localAddr := defaultLocalAddr
		if opts.LocalAddr != "" {
			localAddr = opts.LocalAddr
		}
		lns, err := tnet.ListenPackets(opts.Network, localAddr, false)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr,
				"tnet udp client transport listen packets: "+err.Error())
		}
		svr, err := tnet.NewUDPService(
			lns,
			func(conn tnet.PacketConn) error { return nil },
			tnet.WithExactUDPBufferSizeEnabled(opts.ExactUDPBufferSizeEnabled))
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr, "tnet udp client transport new service: "+err.Error())
		}
		go svr.Serve(ctx)
		return lns[0], nil
	}
	conn, err := tnet.DialUDP(opts.Network, opts.Address, opts.DialTimeout)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			fmt.Sprintf("tnet udp client transport dial udp: %s", err.Error()))
	}
	conn.SetExactUDPBufferSizeEnabled(opts.ExactUDPBufferSizeEnabled)
	return conn, nil
}
