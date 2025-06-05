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
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	transport.RegisterClientStreamTransport(transportName, DefaultClientStreamTransport)
}

// DefaultClientStreamTransport is the default implementation of tnet client stream transport.
var DefaultClientStreamTransport = NewClientStreamTransport()

// ClientTransportStreamOption is the client stream transport options.
type ClientTransportStreamOption func(*clientStreamTransportOption)

// WithStreamMultiplexedPool returns a ClientTransportStreamOption which sets the stream multiplexed pool,
func WithStreamMultiplexedPool(p multiplexed.Pool) ClientTransportStreamOption {
	return func(opts *clientStreamTransportOption) {
		opts.multiplexedPool = p
	}
}

type clientStreamTransportOption struct {
	multiplexedPool multiplexed.Pool
}

// NewClientStreamTransport creates a tnet client stream transport.
func NewClientStreamTransport(opts ...ClientTransportStreamOption) transport.ClientStreamTransport {
	options := &clientStreamTransportOption{
		multiplexedPool: DefaultMultiplexedPool,
	}
	for _, opt := range opts {
		opt(options)
	}
	return transport.NewClientStreamTransport(transport.WithStreamMultiplexedPool(options.multiplexedPool))
}
