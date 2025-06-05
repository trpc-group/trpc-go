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
	"net"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
	"trpc.group/trpc-go/trpc-go/transport/internal/dialer"
	ierrs "trpc.group/trpc-go/trpc-go/transport/internal/errs"
	imsg "trpc.group/trpc-go/trpc-go/transport/internal/msg"
)

func (c *clientTransport) tcpRoundTrip(ctx context.Context, reqData []byte,
	opts *transport.RoundTripOptions) ([]byte, error) {
	var (
		span  rpcz.Span
		ender rpcz.Ender
	)
	if rpczenable.Enabled {
		span = rpcz.SpanFromContext(ctx)
		_, ender = span.NewChild("DialTCP")
	}
	conn, err := dialer.DialTCP(ctx, dialer.DialOptions{
		Network:               opts.Network,
		Address:               opts.Address,
		LocalAddr:             opts.LocalAddr,
		Dial:                  Dial,
		DialTimeout:           opts.DialTimeout,
		Pool:                  opts.Pool,
		FramerBuilder:         opts.FramerBuilder,
		DisableConnectionPool: opts.DisableConnectionPool,
		Protocol:              opts.Protocol,
		CACertFile:            opts.CACertFile,
		TLSCertFile:           opts.TLSCertFile,
		TLSKeyFile:            opts.TLSKeyFile,
		TLSServerName:         opts.TLSServerName,
	})
	if rpczenable.Enabled {
		ender.End()
	}
	msg := codec.Message(ctx)
	if err != nil {
		msg = imsg.WithLocalAddr(msg, opts.Network, opts.LocalAddr)
		return nil, err
	}
	defer conn.Close()
	if !validateTnetConn(conn) && !validateTnetTLSConn(conn) {
		msg = imsg.WithLocalAddr(msg, opts.Network, opts.LocalAddr)
		return nil, errs.NewFrameError(errs.RetClientConnectFail, "tnet transport doesn't support non tnet.Conn")
	}

	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithLocalAddr(conn.LocalAddr())

	report.TCPClientTransportSendSize.Set(float64(len(reqData)))
	// Send a request.
	if rpczenable.Enabled {
		_, ender = span.NewChild("SendMessage")
	}
	err = tcpWriteFrame(conn, reqData)
	if rpczenable.Enabled {
		ender.End()
	}
	if err != nil {
		return nil, err
	}
	// Receive a response.
	if rpczenable.Enabled {
		_, ender = span.NewChild("ReceiveMessage")
	}
	rspData, err := tcpReadFrame(conn, opts)
	if rpczenable.Enabled {
		ender.End()
	}
	return rspData, err
}

func tcpWriteFrame(conn net.Conn, reqData []byte) error {
	// When writing data on a tnet connection, there will be no partial write success,
	// only complete success or complete failure.
	_, err := conn.Write(reqData)
	if err != nil {
		return ierrs.WrapAsClientTimeoutErrOr(err, errs.RetClientNetErr, "tcp client tnet transport Write")
	}
	return nil
}

func tcpReadFrame(conn net.Conn, opts *transport.RoundTripOptions) ([]byte, error) {
	if opts.ReqType == transport.SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	var fr codec.Framer
	// The connection retrieved from the connection pool has already implemented the Framer interface.
	if opts.DisableConnectionPool {
		fr = opts.FramerBuilder.New(codec.NewReader(conn))
	} else {
		var ok bool
		fr, ok = conn.(codec.Framer)
		if !ok {
			return nil, errs.NewFrameError(errs.RetClientConnectFail,
				"tcp client transport: framer not implemented")
		}
	}

	rspData, err := fr.ReadFrame()
	if err != nil {
		return nil, ierrs.WrapAsClientTimeoutErrOr(err, errs.RetClientReadFrameErr, "tcp client transport ReadFrame")
	}
	report.TCPClientTransportReceiveSize.Set(float64(len(rspData)))
	return rspData, nil
}

func (c *clientTransport) multiplexed(
	ctx context.Context, req []byte, opts *transport.RoundTripOptions,
) ([]byte, error) {
	getOpts := multiplexed.NewGetOptions()
	getOpts.WithFramerBuilder(opts.FramerBuilder)
	getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
	getOpts.WithLocalAddr(opts.LocalAddr)
	getOpts.WithMsg(opts.Msg)
	conn, err := opts.Multiplexed.GetVirtualConn(ctx, opts.Network, opts.Address, getOpts)
	if err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientNetErr, "tcp client get multiplexed connection failed")
	}
	defer conn.Close()
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(conn.RemoteAddr())

	err = conn.Write(req)
	info, ok := keeporder.ClientInfoFromContext(ctx)
	if ok && info != nil {
		select {
		// Notify the keep-order client who is waiting for the
		// request sending procedure to be finished.
		case info.SendError <- err:
		default:
		}
	}
	if err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientNetErr, "tcp client multiplexed write failed")
	}

	// no need to receive response when request type is SendOnly.
	if opts.ReqType == codec.SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	buf, err := conn.Read()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"tcp tnet multiplexed ReadFrame: "+err.Error())
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp tnet multiplexed ReadFrame: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp tnet multiplexed ReadFrame: "+err.Error())
	}
	return buf, nil
}
