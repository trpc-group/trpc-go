// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

func (c *clientTransport) tcpRoundTrip(ctx context.Context, reqData []byte,
	opts *transport.RoundTripOptions) ([]byte, error) {
	// Dial a TCP connection
	conn, err := dialTCP(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithLocalAddr(conn.LocalAddr())

	if err := checkContextErr(ctx); err != nil {
		return nil, fmt.Errorf("before Write: %w", err)
	}

	report.TCPClientTransportSendSize.Set(float64(len(reqData)))
	// Send a request.
	if err := tcpWriteFrame(conn, reqData); err != nil {
		return nil, err
	}
	// Receive a response.
	return tcpReadFrame(conn, opts)
}

func dialTCP(ctx context.Context, opts *transport.RoundTripOptions) (net.Conn, error) {
	if err := checkContextErr(ctx); err != nil {
		return nil, fmt.Errorf("before tcp dial, %w", err)
	}
	var timeout time.Duration
	d, isSetDeadline := ctx.Deadline()
	if isSetDeadline {
		timeout = time.Until(d)
	}

	var conn net.Conn
	var err error
	// Short connection mode, directly dial a connection.
	if opts.DisableConnectionPool {
		if opts.DialTimeout > 0 && opts.DialTimeout < timeout {
			timeout = opts.DialTimeout
		}
		conn, err = Dial(&connpool.DialOptions{
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
			return nil, errs.WrapFrameError(err, errs.RetClientConnectFail, "tcp client transport dial")
		}
		// Set a deadline for subsequent reading on the connection.
		if isSetDeadline {
			if err := conn.SetReadDeadline(d); err != nil {
				log.Tracef("client SetReadDeadline failed %v", err)
			}
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
		return nil, errs.WrapFrameError(err, errs.RetClientConnectFail, "tcp client transport connection pool")
	}
	// The created connection must be a tnet connection.
	if !validateTnetConn(conn) && !validateTnetTLSConn(conn) {
		return nil, errs.NewFrameError(errs.RetClientConnectFail, "tnet transport doesn't support non tnet.Conn")
	}
	if err := conn.SetReadDeadline(d); err != nil {
		log.Tracef("client SetReadDeadline failed %v", err)
	}
	return conn, nil
}

func tcpWriteFrame(conn net.Conn, reqData []byte) error {
	// When writing data on a tnet connection, there will be no partial write success,
	// only complete success or complete failure.
	_, err := conn.Write(reqData)
	if err != nil {
		return wrapNetError("tcp client tnet transport Write", err)
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
		return nil, wrapNetError("tcp client transport ReadFrame", err)
	}
	report.TCPClientTransportReceiveSize.Set(float64(len(rspData)))
	return rspData, nil
}

func wrapNetError(msg string, err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return errs.WrapFrameError(err, errs.RetClientTimeout, msg)
	}
	return errs.WrapFrameError(err, errs.RetClientNetErr, msg)
}

func checkContextErr(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return errs.WrapFrameError(ctx.Err(), errs.RetClientCanceled, "client canceled")
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errs.WrapFrameError(ctx.Err(), errs.RetClientTimeout, "client timeout")
	}
	return nil
}
func (c *clientTransport) multiplex(ctx context.Context, req []byte, opts *transport.RoundTripOptions) ([]byte, error) {
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
	conn, err := opts.Multiplexed.GetMuxConn(ctx, opts.Network, opts.Address, getOpts)
	if err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientNetErr, "tcp client get multiplex connection failed")
	}
	defer conn.Close()
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(conn.RemoteAddr())

	if err := conn.Write(req); err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientNetErr, "tcp client multiplex write failed")
	}

	// no need to receive response when request type is SendOnly.
	if opts.ReqType == codec.SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	buf, err := conn.Read()
	if err != nil {
		if err == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"tcp tnet multiplexed ReadFrame: "+err.Error())
		}
		if err == context.DeadlineExceeded {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp tnet multiplexed ReadFrame: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp tnet multiplexed ReadFrame: "+err.Error())
	}
	return buf, nil
}
