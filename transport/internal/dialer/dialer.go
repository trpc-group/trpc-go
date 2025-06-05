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

// Package dialer provides common function for transport to dial.
package dialer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/rpcz"
)

// DialOptions is the options for dialing.
type DialOptions struct {
	Network                   string
	Address                   string
	LocalAddr                 string
	Dial                      connpool.DialFunc
	DialUDP                   DialUDPFunc
	DialTimeout               time.Duration
	Pool                      connpool.Pool
	FramerBuilder             codec.FramerBuilder
	DisableConnectionPool     bool
	Protocol                  string
	ConnectionMode            ConnectionMode
	ExactUDPBufferSizeEnabled bool

	CACertFile    string
	TLSCertFile   string
	TLSKeyFile    string
	TLSServerName string
}

// DialUDPFunc connects to a udp endpoint with the informations in options.
type DialUDPFunc func(ctx context.Context, opts DialOptions) (net.PacketConn, error)

// DialTCP establishes a TCP connection based on the DialOptions.
func DialTCP(ctx context.Context, opts DialOptions) (net.Conn, error) {
	// If ctx has canceled or timeout, just return.
	if err := validateContext(ctx, "before tcp dial"); err != nil {
		return nil, err
	}
	var (
		ctxTimeout    time.Duration
		ctxDeadline   time.Time
		isSetDeadline bool
	)
	ctxDeadline, isSetDeadline = ctx.Deadline()
	if isSetDeadline {
		ctxTimeout = time.Until(ctxDeadline)
	}
	opts.DialTimeout = fixDialTimeout(opts.DialTimeout, ctxTimeout)

	var (
		conn net.Conn
		err  error
	)
	conn, err = dial(ctx, opts)
	if err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientConnectFail, "tcp client transport dial")
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	if isSetDeadline {
		if err := conn.SetDeadline(ctxDeadline); err != nil {
			return nil, errs.WrapFrameError(err, errs.RetClientConnectFail, "set deadline for tcp connection")
		}
	}
	if err := validateContext(ctx, "after tcp dial"); err != nil {
		return nil, err
	}
	return conn, nil
}

// ConnectionMode is the connection mode, either Connected or NotConnected.
type ConnectionMode bool

// ConnectionMode of UDP.
const (
	Connected    = false // UDP which isolates packets from non-same path.
	NotConnected = true  // UDP which allows returning packets from non-same path.
)

// DialUDP establishes an UDP connection based on the DialOptions.
func DialUDP(ctx context.Context, opts DialOptions) (net.PacketConn, *net.UDPAddr, error) {
	if rpczenable.Enabled {
		span := rpcz.SpanFromContext(ctx)
		_, ender := span.NewChild("DialUDP")
		defer ender.End()
	}

	addr, err := net.ResolveUDPAddr(opts.Network, opts.Address)
	if err != nil {
		return nil, nil, errs.NewFrameError(errs.RetClientNetErr,
			"udp client transport ResolveUDPAddr: "+err.Error())
	}
	var (
		ctxTimeout    time.Duration
		ctxDeadline   time.Time
		isSetDeadline bool
	)
	ctxDeadline, isSetDeadline = ctx.Deadline()
	if isSetDeadline {
		ctxTimeout = time.Until(ctxDeadline)
	}
	opts.DialTimeout = fixDialTimeout(opts.DialTimeout, ctxTimeout)
	conn, err := opts.DialUDP(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	if isSetDeadline {
		if err := conn.SetDeadline(ctxDeadline); err != nil {
			return nil, nil, errs.WrapFrameError(err, errs.RetClientConnectFail, "set deadline for udp connection")
		}
	}
	return conn, addr, nil
}

// fixDialTimeout fix the dial timeout based on the old dial timeout and context timeout.
func fixDialTimeout(oldDialTimeout time.Duration, ctxTimeout time.Duration) time.Duration {
	// The connection is established using the minimum of context timeout and dialing timeout.
	dialTimeout := oldDialTimeout
	if ctxTimeout > 0 {
		if ctxTimeout < dialTimeout || dialTimeout == 0 {
			dialTimeout = ctxTimeout
		}
	}
	return dialTimeout
}

func dial(ctx context.Context, opts DialOptions) (net.Conn, error) {
	// Short connection mode, directly dial a connection.
	if opts.DisableConnectionPool {
		return opts.Dial(&connpool.DialOptions{
			Network:       opts.Network,
			Address:       opts.Address,
			LocalAddr:     opts.LocalAddr,
			Timeout:       opts.DialTimeout,
			CACertFile:    opts.CACertFile,
			TLSCertFile:   opts.TLSCertFile,
			TLSKeyFile:    opts.TLSKeyFile,
			TLSServerName: opts.TLSServerName,
		})
	}
	// Connection pool mode, get connection from pool.
	if pool, ok := opts.Pool.(connpool.PoolWithOptions); ok {
		getOpts := connpool.NewGetOptions()
		getOpts.WithContext(ctx)
		getOpts.WithFramerBuilder(opts.FramerBuilder)
		getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
		getOpts.WithLocalAddr(opts.LocalAddr)
		getOpts.WithDialTimeout(opts.DialTimeout)
		getOpts.WithProtocol(opts.Protocol)
		return pool.GetWithOptions(opts.Network, opts.Address, getOpts)
	}
	return opts.Pool.Get(opts.Network, opts.Address, opts.DialTimeout,
		connpool.WithContext(ctx),
		connpool.WithFramerBuilder(opts.FramerBuilder),
		connpool.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName))
}

// DefaultDialUDP creates a default UDP connection based on the DialOptions provided by UDP.
func DefaultDialUDP(ctx context.Context, opts DialOptions) (net.PacketConn, error) {
	if opts.ConnectionMode == NotConnected {
		// Listen on all available IP addresses of the local system by default,
		// and a port number is automatically chosen.
		const defaultLocalAddr = ":"
		localAddr := defaultLocalAddr
		if opts.LocalAddr != "" {
			localAddr = opts.LocalAddr
		}
		conn, err := net.ListenPacket(opts.Network, localAddr)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr, "udp client transport Dial: "+err.Error())
		}
		return conn, nil
	}

	var (
		localAddr net.Addr
		err       error
	)
	if opts.LocalAddr != "" {
		localAddr, err = net.ResolveUDPAddr(opts.Network, opts.LocalAddr)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr,
				"udp client transport LocalAddr ResolveUDPAddr: "+err.Error())
		}
	}
	dialer := net.Dialer{
		LocalAddr: localAddr,
		Timeout:   opts.DialTimeout,
	}
	var udpConn net.Conn
	udpConn, err = dialer.Dial(opts.Network, opts.Address)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			fmt.Sprintf("dial udp fail: %s", err.Error()))
	}

	conn, ok := udpConn.(net.PacketConn)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"udp conn not implement net.PacketConn")
	}
	return conn, err
}

// validateContext check if the context is valid. If it's not, return an error.
func validateContext(ctx context.Context, errMsg string) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return errs.WrapFrameError(ctx.Err(), errs.RetClientCanceled, errMsg+" client canceled")
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errs.WrapFrameError(ctx.Err(), errs.RetClientTimeout, errMsg+" client timeout")
	}
	return nil
}
