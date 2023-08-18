// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

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
	defer func() {
		// opts.Ctx is only used to pass ctx parameters, ctx is not recommended to be held by data structures.
		o.Ctx = nil
	}()

	for {
		// If the RPC request does not set ctx, create a new ctx.
		if ctx == nil {
			break
		}
		// If the RPC request does not set the ctx timeout, create a new ctx.
		deadline, ok := ctx.Deadline()
		if !ok {
			break
		}
		// If the RPC request timeout is greater than the set timeout, create a new ctx.
		d := time.Until(deadline)
		if o.DialTimeout > 0 && o.DialTimeout < d {
			break
		}
		return ctx, nil
	}

	if o.DialTimeout > 0 {
		dialTimeout = o.DialTimeout
	}
	if dialTimeout == 0 {
		dialTimeout = defaultDialTimeout
	}
	return context.WithTimeout(context.Background(), dialTimeout)
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

// Pool is the interface that specifies client connection pool options.
// Compared with Pool, Pool directly uses the GetOptions data structure for function input parameters.
// Compared with function option input parameter mode, it can reduce memory escape and improve calling performance.
type Pool interface {
	Get(network string, address string, opt GetOptions) (net.Conn, error)
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
