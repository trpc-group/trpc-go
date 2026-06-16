//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package transport

import (
	"net"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/keeporder/actor"
)

// ListenServeOptions is the server options on start.
type ListenServeOptions struct {
	ServiceName   string
	Address       string
	Network       string
	Handler       Handler
	FramerBuilder codec.FramerBuilder
	Listener      net.Listener

	CACertFile      string        // ca certification file
	TLSCertFile     string        // server certification file
	TLSKeyFile      string        // server key file
	TLSCertProvider string        // provider used to load TLS certificate files
	Routines        int           // size of goroutine pool
	ServerAsync     bool          // whether enable server async
	Writev          bool          // whether enable writev in server
	CopyFrame       bool          // whether copy frame
	IdleTimeout     time.Duration // idle timeout of connection

	// KeepOrderPreDecodeExtractor specifies the pre-decoding extractor to use for keeping order.
	KeepOrderPreDecodeExtractor KeepOrderPreDecodeExtractor
	// KeepOrderPreUnmarshalExtractor specifies the pre-unmarshalling extractor to use for keeping order.
	KeepOrderPreUnmarshalExtractor KeepOrderPreUnmarshalExtractor
	// OrderedGroups specifies the groups used to keep request order.
	OrderedGroups OrderedGroups

	// DisableKeepAlives, if true, disables keep-alives and only use the
	// connection for a single request.
	// This used for rpc transport layer like http, it's unrelated to
	// the TCP keep-alives.
	DisableKeepAlives bool

	// StopListening is used to instruct the server transport to stop listening.
	StopListening <-chan struct{}
}

func (o *ListenServeOptions) fixKeepOrder() {
	if o.OrderedGroups == nil {
		o.OrderedGroups = actor.Default
	}
}

// ListenServeOption modifies the ListenServeOptions.
type ListenServeOption func(*ListenServeOptions)

// WithServiceName returns a ListenServeOption which sets the service name.
func WithServiceName(name string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.ServiceName = name
	}
}

// WithServerFramerBuilder returns a ListenServeOption which sets server frame builder.
func WithServerFramerBuilder(fb codec.FramerBuilder) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.FramerBuilder = fb
	}
}

// WithListenAddress returns a ListenServerOption which sets listening address.
func WithListenAddress(address string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Address = address
	}
}

// WithListenNetwork returns a ListenServeOption which sets listen network.
func WithListenNetwork(network string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Network = network
	}
}

// WithListener returns a ListenServeOption which allows users to use their customized listener for
// specific accept/read/write logics.
func WithListener(lis net.Listener) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Listener = lis
	}
}

// WithHandler returns a ListenServeOption which sets business Handler.
func WithHandler(handler Handler) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Handler = handler
	}
}

// WithServeTLS returns a ListenServeOption which sets TLS relatives.
func WithServeTLS(certFile, keyFile, caFile string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.TLSCertFile = certFile
		opts.TLSKeyFile = keyFile
		opts.CACertFile = caFile
	}
}

// WithServeCertProvider returns a ListenServeOption which sets the TLS certificate provider.
func WithServeCertProvider(providerName string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.TLSCertProvider = providerName
	}
}

// WithServerAsync returns a ListenServeOption which enables server async.
// When another frameworks call trpc, they may use long connections. tRPC server can not handle
// them concurrently, thus timeout.
// This option takes effect for each TCP connections.
func WithServerAsync(serverAsync bool) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.ServerAsync = serverAsync
	}
}

// WithKeepOrderPreDecodeExtractor returns a ListenServeOption which enables keep-order processing
// by extracting the order key from the decoded request body.
func WithKeepOrderPreDecodeExtractor(preDecodeExtractor KeepOrderPreDecodeExtractor) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.KeepOrderPreDecodeExtractor = preDecodeExtractor
	}
}

// WithKeepOrderPreUnmarshalExtractor returns a ListenServeOption which enables keep-order processing
// by extracting the order key from the unmarshaled request body.
func WithKeepOrderPreUnmarshalExtractor(preUnmarshalExtractor KeepOrderPreUnmarshalExtractor) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.KeepOrderPreUnmarshalExtractor = preUnmarshalExtractor
	}
}

// WithOrderedGroups returns a ListenServeOption which specifies the groups used to keep request order.
func WithOrderedGroups(groups OrderedGroups) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.OrderedGroups = groups
	}
}

// WithWritev returns a ListenServeOption which enables writev.
func WithWritev(writev bool) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Writev = writev
	}
}

// WithMaxRoutines returns a ListenServeOption which sets the max number of async goroutines.
// It's recommended to reserve twice of expected goroutines, but no less than MAXPROCS. The default
// value is (1<<31 - 1).
// This option takes effect only when async mod is enabled. It's ignored on sync mod.
func WithMaxRoutines(routines int) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Routines = routines
	}
}

// WithCopyFrame returns a ListenServeOption which sets whether copy frames.
// In stream RPC, even server use sync mod, stream is asynchronous, we need to copy frame to avoid
// over writing.
func WithCopyFrame(copyFrame bool) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.CopyFrame = copyFrame
	}
}

// WithDisableKeepAlives returns a ListenServeOption which disables keep-alives.
func WithDisableKeepAlives(disable bool) ListenServeOption {
	return func(options *ListenServeOptions) {
		options.DisableKeepAlives = disable
	}
}

// WithServerIdleTimeout returns a ListenServeOption which sets the server idle timeout.
func WithServerIdleTimeout(timeout time.Duration) ListenServeOption {
	return func(options *ListenServeOptions) {
		options.IdleTimeout = timeout
	}
}

// WithStopListening returns a ListenServeOption which notifies the transport to stop listening.
func WithStopListening(ch <-chan struct{}) ListenServeOption {
	return func(options *ListenServeOptions) {
		options.StopListening = ch
	}
}
