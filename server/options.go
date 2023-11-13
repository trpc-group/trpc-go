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

package server

import (
	"context"
	"net"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/transport"
)

// Options are server side options.
type Options struct {
	container string // container name

	Namespace   string // namespace like "Production", "Development" etc.
	EnvName     string // environment name
	SetName     string // "Set" name
	ServiceName string // service name

	Address                  string        // listen address, ip:port
	Timeout                  time.Duration // timeout for handling a request
	DisableRequestTimeout    bool          // whether to disable request timeout that inherits from upstream
	DisableKeepAlives        bool          // disables keep-alives
	CurrentSerializationType int
	CurrentCompressType      int

	protocol   string // protocol like "trpc", "http" etc.
	network    string // network like "tcp", "udp" etc.
	handlerSet bool   // whether that custom handler is set

	ServeOptions []transport.ListenServeOption
	Transport    transport.ServerTransport

	Registry registry.Registry
	Codec    codec.Codec

	Filters          filter.ServerChain              // filter chain
	FilterNames      []string                        // the name of filters
	StreamHandle     StreamHandle                    // server stream processing
	StreamTransport  transport.ServerStreamTransport // server stream transport plugin
	MaxWindowSize    uint32                          // max window size for server stream
	CloseWaitTime    time.Duration                   // min waiting time when closing server for wait deregister finish
	MaxCloseWaitTime time.Duration                   // max waiting time when closing server for wait requests finish

	RESTOptions   []restful.Option // RESTful router options
	StreamFilters StreamFilterChain
}

// StreamHandle is the interface that defines server stream processing.
type StreamHandle interface {
	// StreamHandleFunc does server stream processing.
	StreamHandleFunc(ctx context.Context, sh StreamHandler, si *StreamServerInfo, req []byte) ([]byte, error)
	// Init does the initialization, mainly passing and saving Options.
	Init(opts *Options) error
}

// Option sets server options.
type Option func(*Options)

// WithNamespace returns an Option that sets namespace for server.
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

// WithStreamTransport returns an Option that sets transport.ServerStreamTransport for server.
func WithStreamTransport(st transport.ServerStreamTransport) Option {
	return func(o *Options) {
		o.StreamTransport = st
	}
}

// WithEnvName returns an Option that sets environment name.
func WithEnvName(envName string) Option {
	return func(o *Options) {
		o.EnvName = envName
	}
}

// WithContainer returns an Option that sets container name.
func WithContainer(container string) Option {
	return func(o *Options) {
		o.container = container
	}
}

// WithSetName returns an Option that sets "Set" name.
func WithSetName(setName string) Option {
	return func(o *Options) {
		o.SetName = setName
	}
}

// WithServiceName returns an Option that sets service name.
func WithServiceName(s string) Option {
	return func(o *Options) {
		o.ServiceName = s
		o.ServeOptions = append(o.ServeOptions, transport.WithServiceName(s))
	}
}

// WithFilter returns an Option that adds a filter.Filter (pre or post).
func WithFilter(f filter.ServerFilter) Option {
	return func(o *Options) {
		const filterName = "server.WithFilter"
		o.Filters = append(o.Filters, f)
		o.FilterNames = append(o.FilterNames, filterName)
	}
}

// WithNamedFilter returns an Option that adds named filter
func WithNamedFilter(name string, f filter.ServerFilter) Option {
	return func(o *Options) {
		o.Filters = append(o.Filters, f)
		o.FilterNames = append(o.FilterNames, name)
	}
}

// WithFilters returns an Option that adds a filter chain.
func WithFilters(fs []filter.ServerFilter) Option {
	return func(o *Options) {
		for _, f := range fs {
			o.Filters = append(o.Filters, f)
			o.FilterNames = append(o.FilterNames, "server.WithFilters")
		}
	}
}

// WithStreamFilter returns an Option that adds a stream filter (pre or post).
func WithStreamFilter(sf StreamFilter) Option {
	return func(o *Options) {
		o.StreamFilters = append(o.StreamFilters, sf)
	}
}

// WithStreamFilters returns an Option that adds a stream filter chain.
func WithStreamFilters(sfs ...StreamFilter) Option {
	return func(o *Options) {
		o.StreamFilters = append(o.StreamFilters, sfs...)
	}
}

// WithAddress returns an Option that sets address (ip:port or :port).
func WithAddress(s string) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithListenAddress(s))
		o.Address = s
	}
}

// WithTLS returns an Option that sets TLS certificate files' path.
// The input param certFile represents server certificate.
// The input param keyFile represents server private key.
// The input param caFile represents CA certificate, which is used for client-to-server authentication(mTLS).
// If cafile is empty, no client validation.
// Also, caFile="root" means local ca file would be used to validate client.
// All certificates must be X.509 certificates.
func WithTLS(certFile, keyFile, caFile string) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithServeTLS(certFile, keyFile, caFile))
	}
}

// WithNetwork returns an Option that sets network, tcp by default.
func WithNetwork(s string) Option {
	return func(o *Options) {
		o.network = s
		o.ServeOptions = append(o.ServeOptions, transport.WithListenNetwork(s))
	}
}

// WithListener returns an Option that sets net.Listener for accept, read/write op customization.
func WithListener(lis net.Listener) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithListener(lis))
	}
}

// WithServerAsync returns an Option that sets whether to enable server asynchronous or not.
// When enable it, the server can cyclically receive packets and process request and response
// packets concurrently for the same connection.
func WithServerAsync(serverAsync bool) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithServerAsync(serverAsync))
	}
}

// WithWritev returns an Option that sets whether to enable writev or not.
func WithWritev(writev bool) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithWritev(writev))
	}
}

// WithMaxRoutines returns an Option that sets max number of goroutines.
// It only works for server async mode.
// MaxRoutines should be set to twice as expected number of routines (can be calculated by expected QPS),
// and larger than MAXPROCS.
// If MaxRoutines is not set or set to 0, it will be set to (1<<31 - 1).
// Requests exceeding MaxRoutines will be queued. Prolonged overages may lead to OOM!
// MaxRoutines is not the solution to alleviate server overloading.
func WithMaxRoutines(routines int) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithMaxRoutines(routines))
	}
}

// WithTimeout returns an Option that sets timeout for handling a request.
func WithTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// WithDisableRequestTimeout returns an Option that disables timeout for handling requests.
func WithDisableRequestTimeout(disable bool) Option {
	return func(o *Options) {
		o.DisableRequestTimeout = disable
	}
}

// WithRegistry returns an Option that sets registry.Registry.
// One service, one registry.Registry.
func WithRegistry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// WithTransport returns an Option that sets transport.ServerTransport.
func WithTransport(t transport.ServerTransport) Option {
	return func(o *Options) {
		if t != nil {
			o.Transport = t
		}
	}
}

// WithProtocol returns an Option that sets protocol of service.
// This Option also sets framerbuilder and codec plugin.
func WithProtocol(s string) Option {
	return func(o *Options) {
		o.protocol = s
		o.Codec = codec.GetServer(s)
		fb := transport.GetFramerBuilder(s)
		if fb != nil {
			o.ServeOptions = append(o.ServeOptions, transport.WithServerFramerBuilder(fb))
		}
		trans := transport.GetServerTransport(s)
		if trans != nil {
			o.Transport = trans
		}
	}
}

// WithHandler returns an Option that sets transport.Handler (service itself by default).
func WithHandler(h transport.Handler) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithHandler(h))
		o.handlerSet = true
	}
}

// WithCurrentSerializationType returns an Option that sets current serialization type.
// It's often used for transparent proxy without serialization.
// If current serialization type is not set, serialization type will be determined by
// serialization field of request protocol.
func WithCurrentSerializationType(t int) Option {
	return func(o *Options) {
		o.CurrentSerializationType = t
	}
}

// WithCurrentCompressType returns an Option that sets current compress type.
func WithCurrentCompressType(t int) Option {
	return func(o *Options) {
		o.CurrentCompressType = t
	}
}

// WithMaxWindowSize returns an Option that sets max window size for server stream.
func WithMaxWindowSize(w uint32) Option {
	return func(o *Options) {
		o.MaxWindowSize = w
	}
}

// WithCloseWaitTime returns an Option that sets min waiting time when close service.
// It's used for service's graceful restart.
// Default: 0ms, max: 10s.
func WithCloseWaitTime(t time.Duration) Option {
	return func(o *Options) {
		o.CloseWaitTime = t
	}
}

// WithMaxCloseWaitTime returns an Option that sets max waiting time when close service.
// It's used for wait requests finish.
// Default: 0ms.
func WithMaxCloseWaitTime(t time.Duration) Option {
	return func(o *Options) {
		o.MaxCloseWaitTime = t
	}
}

// WithRESTOptions returns an Option that sets RESTful router options.
func WithRESTOptions(opts ...restful.Option) Option {
	return func(o *Options) {
		o.RESTOptions = append(o.RESTOptions, opts...)
	}
}

// WithIdleTimeout returns an Option that sets idle connection timeout.
// Notice: it doesn't work for server streaming.
func WithIdleTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithServerIdleTimeout(t))
	}
}

// WithDisableKeepAlives returns an Option that disables keep-alives.
func WithDisableKeepAlives(disable bool) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithDisableKeepAlives(disable))
	}
}
