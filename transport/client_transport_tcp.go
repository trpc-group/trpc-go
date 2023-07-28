// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package transport

import (
	"context"
	"errors"
	"net"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/rpcz"
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

	conn, err := c.dialTCP(ctx, opts)
	if err != nil {
		return nil, err
	}
	// TCP connection is exclusively multiplexed. Close determines whether connection should be put
	// back into the connection pool to be reused.
	defer conn.Close()
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithLocalAddr(conn.LocalAddr())

	if ctx.Err() == context.Canceled {
		return nil, errs.NewFrameError(errs.RetClientCanceled,
			"tcp client transport canceled before Write: "+ctx.Err().Error())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errs.NewFrameError(errs.RetClientTimeout,
			"tcp client transport timeout before Write: "+ctx.Err().Error())
	}

	report.TCPClientTransportSendSize.Set(float64(len(reqData)))
	span := rpcz.SpanFromContext(ctx)
	_, end := span.NewChild("SendMessage")
	err = c.tcpWriteFrame(ctx, conn, reqData)
	end.End()
	if err != nil {
		return nil, err
	}

	_, end = span.NewChild("ReceiveMessage")
	rspData, err := c.tcpReadFrame(conn, opts)
	end.End()
	return rspData, err
}

// dialTCP establishes a TCP connection.
func (c *clientTransport) dialTCP(ctx context.Context, opts *RoundTripOptions) (net.Conn, error) {
	// If ctx has canceled or timeout, just return.
	if ctx.Err() == context.Canceled {
		return nil, errs.NewFrameError(errs.RetClientCanceled,
			"client canceled before tcp dial: "+ctx.Err().Error())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errs.NewFrameError(errs.RetClientTimeout,
			"client timeout before tcp dial: "+ctx.Err().Error())
	}
	var timeout time.Duration
	d, ok := ctx.Deadline()
	if ok {
		timeout = time.Until(d)
	}

	var conn net.Conn
	var err error
	// Short connection mode, directly dial a connection.
	if opts.DisableConnectionPool {
		// The connection is established using the minimum of ctx timeout and connecting timeout.
		if opts.DialTimeout > 0 && opts.DialTimeout < timeout {
			timeout = opts.DialTimeout
		}
		conn, err = connpool.Dial(&connpool.DialOptions{
			Network:       opts.Network,
			Address:       opts.Address,
			LocalAddr:     opts.LocalAddr,
			Timeout:       timeout,
			CACertFile:    opts.CACertFile,
			TLSCertFile:   opts.TLSCertFile,
			TLSKeyFile:    opts.TLSKeyFile,
			TLSServerName: opts.TLSServerName,
		})
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientConnectFail,
				"tcp client transport dial: "+err.Error())
		}
		if ok {
			conn.SetDeadline(d)
		}
		return conn, nil
	}

	// Connection pool mode, get connection from pool.
	getOpts := connpool.NewGetOptions()
	getOpts.WithContext(ctx)
	getOpts.WithFramerBuilder(opts.FramerBuilder)
	getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
	getOpts.WithLocalAddr(opts.LocalAddr)
	getOpts.WithDialTimeout(opts.DialTimeout)
	getOpts.WithProtocol(opts.Protocol)
	conn, err = opts.Pool.Get(opts.Network, opts.Address, getOpts)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport connection pool: "+err.Error())
	}
	if ok {
		conn.SetDeadline(d)
	}
	return conn, nil
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
			if e, ok := err.(net.Error); ok && e.Timeout() {
				return errs.NewFrameError(errs.RetClientTimeout,
					"tcp client transport Write: "+err.Error())
			}
			return errs.NewFrameError(errs.RetClientNetErr,
				"tcp client transport Write: "+err.Error())
		}
		sentNum += num
	}
	return nil
}

// tcpReadFrame reads the tcp frame.
func (c *clientTransport) tcpReadFrame(conn net.Conn, opts *RoundTripOptions) ([]byte, error) {
	// send only.
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
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp client transport ReadFrame: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientReadFrameErr,
			"tcp client transport ReadFrame: "+err.Error())
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
	getOpts.WithVID(opts.Msg.RequestID())
	fp, ok := opts.FramerBuilder.(multiplexed.FrameParser)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"frame builder does not implement multiplexed.FrameParser")
	}
	getOpts.WithFrameParser(fp)
	getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
	getOpts.WithLocalAddr(opts.LocalAddr)
	conn, err := opts.Multiplexed.GetVirtualConn(ctx, opts.Network, opts.Address, getOpts)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(conn.RemoteAddr())

	if err := conn.Write(req); err != nil {
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp client multiplexed transport Write: "+err.Error())
	}

	// SendOnly does not need to read response.
	if opts.ReqType == codec.SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	buf, err := conn.Read()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"tcp client multiplexed transport ReadFrame: "+err.Error())
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp client multiplexed transport ReadFrame: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp client multiplexed transport ReadFrame: "+err.Error())
	}
	return buf, nil
}
