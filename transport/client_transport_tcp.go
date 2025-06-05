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
	"net"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport/internal/dialer"
	ierrs "trpc.group/trpc-go/trpc-go/transport/internal/errs"
	imsg "trpc.group/trpc-go/trpc-go/transport/internal/msg"
)

// tcpRoundTrip sends tcp request. It supports send, sendAndRcv, keepalive and multiplex.
func (c *clientTransport) tcpRoundTrip(ctx context.Context, reqData []byte,
	opts *RoundTripOptions) ([]byte, error) {
	if opts.Pool == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: connection pool empty")
	}

	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: framer builder empty")
	}
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
		Dial:                  connpool.Dial,
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
	// TCP connection is exclusively multiplexed. Close determines whether connection should be put
	// back into the connection pool to be reused.
	defer conn.Close()
	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithLocalAddr(conn.LocalAddr())

	report.TCPClientTransportSendSize.Set(float64(len(reqData)))
	if rpczenable.Enabled {
		_, ender = span.NewChild("SendMessage")
	}
	// Write data to connection.
	err = c.tcpWriteFrame(ctx, conn, reqData)
	if rpczenable.Enabled {
		ender.End()
	}
	if err != nil {
		return nil, err
	}

	if rpczenable.Enabled {
		_, ender = span.NewChild("ReceiveMessage")
	}
	// Read data from connection.
	rspData, err := c.tcpReadFrame(conn, opts)
	if rpczenable.Enabled {
		ender.End()
	}
	return rspData, err
}

// tcpWriteReqData writes the tcp frame.
func (c *clientTransport) tcpWriteFrame(ctx context.Context, conn net.Conn, reqData []byte) error {
	// Send package in a loop.
	sentNum := 0
	num := 0
	var err error
	for sentNum < len(reqData) {
		num, err = conn.Write(reqData[sentNum:])
		if err != nil {
			return ierrs.WrapAsClientTimeoutErrOr(err, errs.RetClientNetErr, "tcp client transport Write")
		}
		sentNum += num
	}
	return nil
}

// tcpReadFrame reads the tcp frame.
func (c *clientTransport) tcpReadFrame(conn net.Conn, opts *RoundTripOptions) ([]byte, error) {
	// Send only.
	if opts.ReqType == SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	var fr codec.Framer
	if opts.DisableConnectionPool {
		// Do not create new Framer for each connection in connection pool.
		fr = opts.FramerBuilder.New(codec.NewReader(conn))
	} else {
		// The Framer is bound to conn in the connection pool.
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

// multiplexed handle multiplexed request.
func (c *clientTransport) multiplexed(ctx context.Context, req []byte, opts *RoundTripOptions) ([]byte, error) {
	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: framer builder empty")
	}
	getOpts := multiplexed.NewGetOptions()
	getOpts.WithMsg(opts.Msg)
	getOpts.WithFramerBuilder(opts.FramerBuilder)
	getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
	getOpts.WithLocalAddr(opts.LocalAddr)
	conn, err := opts.Multiplexed.GetVirtualConn(ctx, opts.Network, opts.Address, getOpts)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithLocalAddr(conn.LocalAddr())

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
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp client multiplexed transport Write: "+err.Error())
	}

	// SendOnly does not need to read response.
	if opts.ReqType == codec.SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	buf, err := conn.Read()
	if err != nil {
		if err == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"tcp client multiplexed transport ReadFrame: "+err.Error())
		}
		if err == context.DeadlineExceeded {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp client multiplexed transport ReadFrame: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp client multiplexed transport ReadFrame: "+err.Error())
	}
	return buf, nil
}
