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

// Package connpool provides the connection pool.
package connpool

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	intertls "trpc.group/trpc-go/trpc-go/internal/tls"
)

// GetOptions is the get conn configuration.
type GetOptions struct {
	FramerBuilder codec.FramerBuilder
	CustomReader  func(io.Reader) io.Reader
	Ctx           context.Context

	CACertFile    string // ca certificate.
	TLSCertFile   string // client certificate.
	TLSKeyFile    string // client secret key.
	TLSServerName string // The client verifies the server's service name,
	// if not filled in, it defaults to the http hostname.

	LocalAddr   string        // The local address when establishing a connection, which is randomly selected by default.
	DialTimeout time.Duration // Connection establishment timeout.
	Protocol    string        // protocol type.
}

func (o *GetOptions) getDialCtx(dialTimeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := o.Ctx
	// opts.Ctx is only used to pass ctx parameters, ctx is not recommended to be held by data structures.
	defer func() { o.Ctx = nil }()

	if o.contextHasValidDialTimeout(ctx) {
		return ctx, nil
	}

	// If the RPC request does not set ctx or the ctx timeout is invalid, create a new ctx.
	if o.DialTimeout > 0 {
		dialTimeout = o.DialTimeout
	}
	if dialTimeout == 0 {
		dialTimeout = defaultDialTimeout
	}
	return context.WithTimeout(context.Background(), dialTimeout)
}

func (o *GetOptions) contextHasValidDialTimeout(ctx context.Context) bool {
	// If the RPC request does not set ctx, return invalid.
	if ctx == nil {
		return false
	}
	// If the RPC request does not set the ctx timeout, return invalid.
	deadline, ok := ctx.Deadline()
	if !ok {
		return false
	}
	// If the RPC request timeout is greater than the set timeout, return invalid.
	d := time.Until(deadline)
	if o.DialTimeout > 0 && o.DialTimeout < d {
		return false
	}
	// Otherwise, the timeout is valid.
	return true
}

// NewGetOptions creates and initializes GetOptions.
func NewGetOptions() GetOptions {
	return GetOptions{
		CustomReader: codec.NewReader,
	}
}

// WithFramerBuilder returns an Option which sets the FramerBuilder.
func (o *GetOptions) WithFramerBuilder(fb codec.FramerBuilder) {
	o.FramerBuilder = fb
}

// WithDialTLS returns an Option which sets the client to support TLS.
func (o *GetOptions) WithDialTLS(certFile, keyFile, caFile, serverName string) {
	o.TLSCertFile = certFile
	o.TLSKeyFile = keyFile
	o.CACertFile = caFile
	o.TLSServerName = serverName
}

// WithContext returns an Option which sets the requested ctx.
func (o *GetOptions) WithContext(ctx context.Context) {
	o.Ctx = ctx
}

// WithLocalAddr returns an Option which sets the local address when establishing a connection,
// and it is randomly selected by default when there are multiple network cards.
func (o *GetOptions) WithLocalAddr(addr string) {
	o.LocalAddr = addr
}

// WithDialTimeout returns an Option which sets the connection timeout.
func (o *GetOptions) WithDialTimeout(dur time.Duration) {
	o.DialTimeout = dur
}

// WithProtocol returns an Option which sets the backend service protocol name.
func (o *GetOptions) WithProtocol(s string) {
	o.Protocol = s
}

// WithCustomReader returns an option which sets a customReader. Connection pool will uses this customReader
// to create a reader encapsulating the underlying connection, which is usually used to create a buffer.
func (o *GetOptions) WithCustomReader(customReader func(io.Reader) io.Reader) {
	o.CustomReader = customReader
}

// GetOption Options helper.
// Deprecated: please use PoolWithOptions instead.
type GetOption func(*GetOptions)

// WithFramerBuilder returns an Option which sets the FramerBuilder.
// Deprecated: please use PoolWithOptions instead.
func WithFramerBuilder(fb codec.FramerBuilder) GetOption {
	return func(opts *GetOptions) {
		opts.FramerBuilder = fb
	}
}

// WithDialTLS returns an Option which sets the client to support TLS.
// Deprecated: please use PoolWithOptions instead.
func WithDialTLS(certFile, keyFile, caFile, serverName string) GetOption {
	return func(opts *GetOptions) {
		opts.TLSCertFile = certFile
		opts.TLSKeyFile = keyFile
		opts.CACertFile = caFile
		opts.TLSServerName = serverName
	}
}

// WithContext returns an Option which sets the requested ctx.
// Deprecated: please use PoolWithOptions instead.
func WithContext(ctx context.Context) GetOption {
	return func(opts *GetOptions) {
		opts.Ctx = ctx
	}
}

// Pool is the interface that specifies client connection pool.
type Pool interface {
	Get(network string, address string, timeout time.Duration, opt ...GetOption) (net.Conn, error)
}

// PoolWithOptions is the interface that specifies client connection pool options.
// Compared with Pool, PoolWithOptions directly uses the GetOptions data structure for function input parameters.
// Compared with function option input parameter mode, it can reduce memory escape and improve calling performance.
type PoolWithOptions interface {
	GetWithOptions(network string, address string, opt GetOptions) (net.Conn, error)
}

// DialFunc connects to an endpoint with the information in options.
type DialFunc func(opts *DialOptions) (net.Conn, error)

// DialOptions request parameters.
type DialOptions struct {
	Network       string
	Address       string
	LocalAddr     string
	Timeout       time.Duration
	CACertFile    string // ca certificate.
	TLSCertFile   string // client certificate.
	TLSKeyFile    string // client secret key.
	TLSServerName string // The client verifies the server's service name,
	// if not filled in, it defaults to the http hostname.
	IdleTimeout time.Duration
}

// Dial initiates the request.
func Dial(opts *DialOptions) (net.Conn, error) {
	var localAddr net.Addr
	if opts.LocalAddr != "" {
		var err error
		localAddr, err = net.ResolveTCPAddr(opts.Network, opts.LocalAddr)
		if err != nil {
			return nil, err
		}
	}
	dialer := &net.Dialer{
		Timeout:   opts.Timeout,
		LocalAddr: localAddr,
	}
	if opts.CACertFile == "" {
		return dialer.Dial(opts.Network, opts.Address)
	}

	if opts.TLSServerName == "" {
		opts.TLSServerName = opts.Address
	}

	tlsConf, err := intertls.GetClientConfig(opts.TLSServerName, opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientDecodeFail, "client dial tls fail: "+err.Error())
	}
	return tls.DialWithDialer(dialer, opts.Network, opts.Address, tlsConf)
}
