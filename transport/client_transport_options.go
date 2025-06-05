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
	"net/http"

	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
)

const (
	defaultClientUDPRecvSize    = 65536
	defaultMaxConcurrentStreams = 1000
	defaultMaxIdleConnsPerHost  = 2
	// align with net/http for fasthttp
	defaultMaxRedirectsCount = 10
)

// ClientTransportOptions is the client transport options.
type ClientTransportOptions struct {
	UDPRecvSize                      int
	TCPRecvQueueSize                 int
	MaxConcurrentStreams             int
	MaxIdleConnsPerHost              int
	DisableHTTPEncodeTransInfoBase64 bool
	StreamMultiplexedPool            multiplexed.Pool

	// thttp
	NewHTTPClientTransport func() *http.Transport

	// fasthttp
	MaxRedirectsCount int
}

// ClientTransportOption modifies the ClientTransportOptions.
type ClientTransportOption func(*ClientTransportOptions)

// WithClientUDPRecvSize returns a ClientTransportOption which sets client UDP receive size.
func WithClientUDPRecvSize(size int) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.UDPRecvSize = size
	}
}

// WithClientTCPRecvQueueSize returns a ClientTransportOption which sets TCP receive queue size.
//
// Deprecated: TCP receive queue size is unlimited now.
func WithClientTCPRecvQueueSize(size int) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.TCPRecvQueueSize = size
	}
}

// WithMaxConcurrentStreams returns a ClientTransportOption which sets the maximum concurrent
// streams in each TCP connection.
// DefaultMaxConcurrentStreams is used by default. Zero means no limit.
func WithMaxConcurrentStreams(n int) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.MaxConcurrentStreams = n
	}
}

// WithMaxIdleConnsPerHost returns a ClientTransportOption which sets the maximum idle connections
// per host.
// DefaultMaxIdleConnsPerHost is used by default. Zero means no limit.
func WithMaxIdleConnsPerHost(n int) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.MaxIdleConnsPerHost = n
	}
}

// WithDisableEncodeTransInfoBase64 returns a ClientTransportOption indicates disable
// encoding the transinfo value by base64 in HTTP.
func WithDisableEncodeTransInfoBase64() ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.DisableHTTPEncodeTransInfoBase64 = true
	}
}

// WithStreamMultiplexedPool returns a ClientTransportOption which sets the stream multiplexed pool.
func WithStreamMultiplexedPool(p multiplexed.Pool) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.StreamMultiplexedPool = p
	}
}

// WithNewHTTPClientTransport returns a ClientTransportOption which allows user to customize std http transport in
// trpc http client.
// The other way is setting thttp.StdHTTPTransport, however, it has global effects, and can not be used to customize a
// single trpc http request.
func WithNewHTTPClientTransport(newTransport func() *http.Transport) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.NewHTTPClientTransport = newTransport
	}
}

// WithMaxRedirectsCount returns a ClientTransportOption which allows user to customize redirectsCount.
func WithMaxRedirectsCount(c int) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.MaxRedirectsCount = c
	}
}

func defaultClientTransportOptions() *ClientTransportOptions {
	return &ClientTransportOptions{
		UDPRecvSize:          defaultClientUDPRecvSize,
		MaxConcurrentStreams: defaultMaxConcurrentStreams,
		MaxIdleConnsPerHost:  defaultMaxIdleConnsPerHost,
		MaxRedirectsCount:    defaultMaxRedirectsCount,
	}
}
