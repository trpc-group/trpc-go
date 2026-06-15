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

package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"trpc.group/trpc-go/trpc-go/internal/reuseport"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/transport"
)

var (
	// DefaultRESTServerTransport is the default RESTful ServerTransport.
	DefaultRESTServerTransport = NewRESTServerTransport(false, transport.WithReusePort(true))

	// DefaultRESTHeaderMatcher is the default REST HeaderMatcher.
	DefaultRESTHeaderMatcher = func(ctx context.Context,
		_ http.ResponseWriter,
		r *http.Request,
		serviceName, methodName string,
	) (context.Context, error) {
		return putRESTMsgInCtx(ctx, r.Header.Get, serviceName, methodName)
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
		return putRESTMsgInCtx(ctx, headerGetter, serviceName, methodName)
	}

	errReplaceRouter = errors.New("not allow to replace router when is based on fasthttp")
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
	service, method string,
) (context.Context, error) {
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithCalleeServiceName(service)
	msg.WithServerRPCName(method)
	msg.WithSerializationType(codec.SerializationTypePB)
	if v := headerGetter(TrpcTimeout); v != "" {
		i, _ := strconv.Atoi(v)
		msg.WithRequestTimeout(time.Millisecond * time.Duration(i))
	}
	if v := headerGetter(TrpcCaller); v != "" {
		msg.WithCallerServiceName(v)
	}
	if v := headerGetter(TrpcMessageType); v != "" {
		i, _ := strconv.Atoi(v)
		msg.WithDyeing((int32(i) & int32(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)
	}
	if v := headerGetter(TrpcTransInfo); v != "" {
		if _, err := unmarshalTransInfo(msg, v); err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

// RESTServerTransport is the RESTful ServerTransport.
type RESTServerTransport struct {
	basedOnFastHTTP bool
	opts            *transport.ServerTransportOptions
}

// NewRESTServerTransport creates a RESTful ServerTransport.
func NewRESTServerTransport(basedOnFastHTTP bool, opt ...transport.ServerTransportOption) transport.ServerTransport {
	opts := &transport.ServerTransportOptions{
		IdleTimeout: time.Minute,
	}

	for _, o := range opt {
		o(opts)
	}

	return &RESTServerTransport{
		basedOnFastHTTP: basedOnFastHTTP,
		opts:            opts,
	}
}

// ListenAndServe implements interface of transport.ServerTransport.
func (st *RESTServerTransport) ListenAndServe(ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: "tcp",
	}
	for _, o := range opt {
		o(opts)
	}
	if !st.basedOnFastHTTP && st.opts.EnableH2C && (len(opts.TLSKeyFile) != 0 || len(opts.TLSCertFile) != 0) {
		return errors.New("restful server transport h2c and tls cannot be enabled at the same time")
	}
	// Get listener.
	ln := opts.Listener
	if ln == nil {
		var err error
		ln, err = st.getListener(opts)
		if err != nil {
			return fmt.Errorf("restfull server transport get listener err: %w", err)
		}
	}
	// Save listener.
	if err := transport.SaveListener(ln); err != nil {
		return fmt.Errorf("save restful listener error: %w", err)
	}
	// Convert to tcpKeepAliveListener.
	if tcpln, ok := ln.(*net.TCPListener); ok {
		ln = tcpKeepAliveListener{tcpln}
	}
	// Config tls.
	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		tlsConf, err := generateTLSConfig(opts)
		if err != nil {
			return err
		}
		ln = tls.NewListener(ln, tlsConf)
	}

	go func() {
		<-opts.StopListening
		ln.Close()
	}()

	return st.serve(ctx, ln, opts)
}

// serve starts service.
func (st *RESTServerTransport) serve(
	ctx context.Context,
	ln net.Listener,
	opts *transport.ListenServeOptions,
) error {
	// Get router.
	router := restful.GetRouter(opts.ServiceName)
	if router == nil {
		if st.basedOnFastHTTP {
			router = restful.NewRouter()
		} else {
			router = http.NewServeMux()
		}
	}

	if st.basedOnFastHTTP { // Based on fasthttp.
		r, ok := router.(*restful.Router)
		if !ok {
			return errReplaceRouter
		}
		server := &fasthttp.Server{Handler: r.HandleRequestCtx}
		go func() {
			_ = server.Serve(ln)
		}()
		if st.opts.ReusePort {
			go func() {
				<-ctx.Done()
				_ = server.Shutdown()
			}()
		}
		return nil
	}
	// Based on net/http.
	server := &http.Server{Addr: opts.Address, Handler: router}
	if st.opts.EnableH2C && (len(opts.TLSKeyFile) != 0 || len(opts.TLSCertFile) != 0) {
		return errors.New("restful server transport h2c and tls cannot be enabled at the same time")
	}
	if st.opts.EnableH2C {
		server.Handler = h2c.NewHandler(router, newHTTP2Server(st.opts.HTTP2Config))
	}
	if st.opts.HTTP2Config != nil {
		if err := http2.ConfigureServer(server, newHTTP2Server(st.opts.HTTP2Config)); err != nil {
			return fmt.Errorf("restful server configure http2 error:%w", err)
		}
	}
	go func() {
		_ = server.Serve(ln)
	}()
	if st.opts.ReusePort {
		go func() {
			<-ctx.Done()
			_ = server.Shutdown(context.TODO())
		}()
	}
	return nil
}

// getListener gets listener.
func (st *RESTServerTransport) getListener(opts *transport.ListenServeOptions) (net.Listener, error) {
	var err error
	var ln net.Listener

	v, _ := os.LookupEnv(transport.EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// Find the passed listener.
		pln, err := transport.GetPassedListener(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}

		ln, ok = pln.(net.Listener)
		if !ok {
			return nil, errors.New("invalid net.Listener")
		}

		return ln, nil
	}

	if st.opts.ReusePort {
		ln, err = reuseport.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("restful reuseport listen error: %w", err)
		}
	} else {
		ln, err = net.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("restful listen error: %w", err)
		}
	}

	return ln, nil
}

// generateTLSConfig generates config of tls.
func generateTLSConfig(opts *transport.ListenServeOptions) (*tls.Config, error) {
	return itls.GetServerConfig(opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile, opts.TLSCertProvider)
}
