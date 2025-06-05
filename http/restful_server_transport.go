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

package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/http/fastop"
	inet "trpc.group/trpc-go/trpc-go/internal/net"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/transport"
)

var (
	// DefaultRESTServerTransport is the default RESTful ServerTransport.
	DefaultRESTServerTransport transport.ServerTransport = NewRESTServerTransportBasedOnStdHTTP(func() *http.Server {
		return &http.Server{}
	}, WithReusePort())
	// DefaultRESTHeaderMatcher is the default REST HeaderMatcher.
	DefaultRESTHeaderMatcher = func(ctx context.Context,
		_ http.ResponseWriter,
		r *http.Request,
		serviceName, methodName string,
	) (context.Context, error) {
		return putRESTMsgInCtx(ctx, func(key string) string {
			return fastop.CanonicalHeaderGet(r.Header, key)
		}, inet.ResolveAddress(protocol.TCP, r.RemoteAddr), serviceName, methodName)
	}

	// DefaultRESTFastHTTPHeaderMatcher is the default REST FastHTTPHeaderMatcher.
	DefaultRESTFastHTTPHeaderMatcher = func(
		ctx context.Context,
		requestCtx *fasthttp.RequestCtx,
		serviceName, methodName string,
	) (context.Context, error) {
		headerGetter := func(k string) string {
			return string(requestCtx.Request.Header.Peek(k))
		}
		return putRESTMsgInCtx(ctx, headerGetter, requestCtx.RemoteAddr(), serviceName, methodName)
	}
)

func init() {
	// Compatible with thttp.
	restful.SetCtxForCompatibility(func(ctx context.Context, w http.ResponseWriter,
		r *http.Request) context.Context {
		return WithHeader(ctx, &Header{Response: w, Request: r})
	})
	restful.DefaultHeaderMatcher = DefaultRESTHeaderMatcher
	restful.DefaultFastHTTPHeaderMatcher = DefaultRESTFastHTTPHeaderMatcher
	transport.RegisterServerTransport("restful", DefaultRESTServerTransport)
}

// putRESTMsgInCtx puts a new codec.Msg, service name and method name in ctx.
// Metadata will be extracted from the request header if the header value exists.
func putRESTMsgInCtx(
	ctx context.Context,
	headerGetter func(string) string,
	remoteAddr net.Addr,
	service, method string,
) (context.Context, error) {
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithCalleeServiceName(service)
	msg.WithServerRPCName(method)
	msg.WithSerializationType(codec.SerializationTypePB)
	msg.WithRemoteAddr(remoteAddr)
	if v := headerGetter(canonicalTrpcTimeout); v != "" {
		i, _ := strconv.Atoi(v)
		msg.WithRequestTimeout(time.Millisecond * time.Duration(i))
	}

	if v := headerGetter(canonicalTrpcCaller); v != "" {
		msg.WithCallerServiceName(v)
	}
	if v := headerGetter(canonicalTrpcCallerMethod); v != "" {
		msg.WithCallerMethod(v)
	}
	if v := headerGetter(canonicalTrpcMessageType); v != "" {
		i, _ := strconv.Atoi(v)
		msg.WithDyeing((int32(i) & int32(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)
	}
	if v := headerGetter(canonicalTrpcTransInfo); v != "" {
		if _, err := unmarshalTransInfo(msg, v); err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

// RESTServerTransport is the RESTful Server Transport based on standard http.
type RESTServerTransport struct {
	newStdHTTPServer func() *http.Server
	reusePort        bool
}

// NewRESTServerTransportBasedOnStdHTTP return *RESTServerTransport based on standard http.
func NewRESTServerTransportBasedOnStdHTTP(newStdHTTPServer func() *http.Server, opts ...RESTServerTransportOption,
) *RESTServerTransport {
	var options restServerTransportOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &RESTServerTransport{
		newStdHTTPServer: newStdHTTPServer,
		reusePort:        options.reusePort,
	}
}

// ListenAndServe implements interface of transport.ServerTransport.
func (t *RESTServerTransport) ListenAndServe(ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: protocol.TCP,
	}
	for _, o := range opt {
		o(opts)
	}
	ln, err := listen(t.reusePort, opts)
	if err != nil {
		return fmt.Errorf("listening: %w", err)
	}
	return t.serve(ctx, ln, opts)
}

func (t *RESTServerTransport) serve(ctx context.Context, ln net.Listener, opts *transport.ListenServeOptions) error {
	router := restful.GetRouter(opts.ServiceName)
	if router == nil {
		return fmt.Errorf("getting service %s router failed: empty router, "+
			"the corresponding router has not been registered", opts.ServiceName)
	}
	server := t.newStdHTTPServer()
	server.Handler = router
	server.Addr = opts.Address
	if opts.IdleTimeout > 0 {
		server.IdleTimeout = opts.IdleTimeout
	}
	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		config, err := itls.GetServerConfig(opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("rest server transport serve get tls config err: %w", err)
		}
		server.TLSConfig = config
	}
	server.SetKeepAlivesEnabled(!opts.DisableKeepAlives)
	go func() {
		_ = server.Serve(ln)
	}()
	if t.reusePort {
		go func() {
			<-ctx.Done()
			_ = server.Shutdown(context.TODO())
		}()
	}
	return nil
}

// NewRestServerFastHTTPTransport return *RESTServerTransport based on fast http.
func NewRestServerFastHTTPTransport(
	newFastHTTPServer func() *fasthttp.Server,
	opts ...RESTServerTransportOption,
) *RestServerTransportBaseOnFastHTTP {
	var options restServerTransportOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &RestServerTransportBaseOnFastHTTP{
		newFastHTTPServer: newFastHTTPServer,
		reusePort:         options.reusePort,
	}
}

// RestServerTransportBaseOnFastHTTP is the RESTful Server Transport based on fasthttp.
type RestServerTransportBaseOnFastHTTP struct {
	newFastHTTPServer func() *fasthttp.Server
	reusePort         bool
}

// ListenAndServe implements interface of transport.ServerTransport.
func (t *RestServerTransportBaseOnFastHTTP) ListenAndServe(ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: protocol.TCP,
	}
	for _, o := range opt {
		o(opts)
	}
	ln, err := listen(t.reusePort, opts)
	if err != nil {
		return fmt.Errorf("listening, reusePort(%v): %w", t.reusePort, err)
	}
	return t.serve(ctx, ln, opts)
}

func (t *RestServerTransportBaseOnFastHTTP) serve(ctx context.Context, ln net.Listener, opts *transport.ListenServeOptions,
) error {
	s := t.newFastHTTPServer()
	s.Handler = restful.GetFasthttpRouter(opts.ServiceName)
	if opts.IdleTimeout > 0 {
		s.IdleTimeout = opts.IdleTimeout
	}
	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		config, err := itls.GetServerConfig(opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("rest server transport serve get tls config err: %w", err)
		}
		s.TLSConfig = config
	}
	s.DisableKeepalive = opts.DisableKeepAlives
	go func() {
		_ = s.Serve(ln)
	}()
	if t.reusePort {
		go func() {
			<-ctx.Done()
			_ = s.Shutdown()
		}()
	}
	return nil
}

func listen(reusePort bool, opts *transport.ListenServeOptions) (net.Listener, error) {
	ln, err := getListener(opts, reusePort)
	if err != nil {
		return nil, fmt.Errorf("getting listener, reusePort(%v): %w", reusePort, err)
	}

	if err := transport.SaveListener(ln); err != nil {
		return nil, fmt.Errorf("saving restful listener: %w", err)
	}

	ln = mayLiftToTCPKeepAliveListener(ln)

	ln, err = itls.MayLiftToTLSListener(ln, opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("may lift to tls listener failed, CACertFile(%s), TLSCertFile(%s), TLSKeyFile(%s): %w",
			opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile, err)
	}

	// Close listener on stop signal.
	go func() {
		<-opts.StopListening
		ln.Close()
	}()
	return ln, nil
}

func mayLiftToTCPKeepAliveListener(ln net.Listener) net.Listener {
	if tcpln, ok := ln.(*net.TCPListener); ok {
		return tcpKeepAliveListener{tcpln}
	}
	return ln
}

// NewRESTServerTransport creates a RESTful ServerTransport.
// Deprecated: Use NewRestServerFastHTTPTransport, or NewRESTServerTransportBasedOnStdHTTP instead.
func NewRESTServerTransport(basedOnFastHTTP bool, opt ...transport.ServerTransportOption) transport.ServerTransport {
	opts := &transport.ServerTransportOptions{}
	for _, o := range opt {
		o(opts)
	}

	var tOptions []RESTServerTransportOption
	if opts.ReusePort {
		tOptions = append(tOptions, WithReusePort())
	}

	if basedOnFastHTTP {
		return NewRestServerFastHTTPTransport(func() *fasthttp.Server { return &fasthttp.Server{} }, tOptions...)
	}
	return NewRESTServerTransportBasedOnStdHTTP(func() *http.Server {
		return &http.Server{}
	}, tOptions...)

}
