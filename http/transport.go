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

// Package http provides support for http protocol by default,
// provides rpc server with http protocol, and provides rpc client
// for calling http protocol.
package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	stdhttp "net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	icontext "trpc.group/trpc-go/trpc-go/internal/context"
	"trpc.group/trpc-go/trpc-go/internal/reuseport"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	st := NewServerTransport(func() *stdhttp.Server { return &stdhttp.Server{} })
	DefaultServerTransport = st
	DefaultHTTP2ServerTransport = st
	// Server transport (protocol file service).
	transport.RegisterServerTransport("http", st)
	transport.RegisterServerTransport("http2", st)
	// Server transport (no protocol file service).
	transport.RegisterServerTransport("http_no_protocol", st)
	transport.RegisterServerTransport("http2_no_protocol", st)
	// Client transport.
	transport.RegisterClientTransport("http", DefaultClientTransport)
	transport.RegisterClientTransport("http2", DefaultHTTP2ClientTransport)
}

// DefaultServerTransport is the default server http transport.
var DefaultServerTransport transport.ServerTransport

// DefaultHTTP2ServerTransport is the default server http2 transport.
var DefaultHTTP2ServerTransport transport.ServerTransport

// ServerTransport is the http transport layer.
type ServerTransport struct {
	newServer func() *stdhttp.Server
	reusePort bool
	enableH2C bool
}

// NewServerTransport creates a new ServerTransport which implement transport.ServerTransport.
// The parameter newStdHttpServer is used to create the underlying stdhttp.Server when ListenAndServe, and that server
// is modified by opts of this function and ListenAndServe.
func NewServerTransport(
	newStdHttpServer func() *stdhttp.Server,
	opts ...OptServerTransport,
) *ServerTransport {
	st := ServerTransport{newServer: newStdHttpServer}
	for _, opt := range opts {
		opt(&st)
	}
	return &st
}

// ListenAndServe handles configuration.
func (t *ServerTransport) ListenAndServe(ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: "tcp",
	}
	for _, o := range opt {
		o(opts)
	}
	if opts.Handler == nil {
		return errors.New("http server transport handler empty")
	}
	return t.listenAndServeHTTP(ctx, opts)
}

var emptyBuf []byte

func (t *ServerTransport) listenAndServeHTTP(ctx context.Context, opts *transport.ListenServeOptions) error {
	// All trpc-go http server transport only register this http.Handler.
	serveFunc := func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h := &Header{Request: r, Response: w}
		ctx := WithHeader(r.Context(), h)

		// Generates new empty general message structure data and save it to ctx.
		ctx, msg := codec.WithNewMessage(ctx)
		defer codec.PutBackMessage(msg)
		// The old request must be replaced to ensure that the context is embedded.
		h.Request = r.WithContext(ctx)
		defer func() {
			// Fix issues/778
			if r.MultipartForm == nil {
				r.MultipartForm = h.Request.MultipartForm
			}
		}()

		span, ender, ctx := rpcz.NewSpanContext(ctx, "http-server")
		defer ender.End()
		span.SetAttribute(rpcz.HTTPAttributeURL, r.URL)
		span.SetAttribute(rpcz.HTTPAttributeRequestContentLength, r.ContentLength)

		// Records LocalAddr and RemoteAddr to Context.
		localAddr, ok := h.Request.Context().Value(stdhttp.LocalAddrContextKey).(net.Addr)
		if ok {
			msg.WithLocalAddr(localAddr)
		}
		raddr, _ := net.ResolveTCPAddr("tcp", h.Request.RemoteAddr)
		msg.WithRemoteAddr(raddr)
		_, err := opts.Handler.Handle(ctx, emptyBuf)
		if err != nil {
			span.SetAttribute(rpcz.TRPCAttributeError, err)
			log.Errorf("http server transport handle fail:%v", err)
			if err == ErrEncodeMissingHeader || errors.Is(err, errs.ErrServerNoResponse) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(fmt.Sprintf("http server handle error: %+v", err)))
			}
			return
		}
	}

	s, err := t.newHTTPServer(serveFunc, opts)
	if err != nil {
		return err
	}

	if err := t.serve(ctx, s, opts); err != nil {
		return err
	}
	return nil
}

func (t *ServerTransport) serve(ctx context.Context, s *stdhttp.Server, opts *transport.ListenServeOptions) error {
	ln := opts.Listener
	if ln == nil {
		var err error
		ln, err = t.getListener(opts.Network, s.Addr)
		if err != nil {
			return fmt.Errorf("http server transport get listener err: %w", err)
		}
	}

	if err := transport.SaveListener(ln); err != nil {
		return fmt.Errorf("save http listener error: %w", err)
	}

	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		go func() {
			if err := s.ServeTLS(
				tcpKeepAliveListener{ln.(*net.TCPListener)},
				opts.TLSCertFile,
				opts.TLSKeyFile,
			); err != stdhttp.ErrServerClosed {
				log.Errorf("serve TLS failed: %w", err)
			}
		}()
	} else {
		go func() {
			_ = s.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
		}()
	}

	// Reuse ports: Kernel distributes IO ReadReady events to multiple cores and threads to accelerate IO efficiency.
	if t.reusePort {
		go func() {
			<-ctx.Done()
			_ = s.Shutdown(context.TODO())
		}()
	}
	return nil
}

func (t *ServerTransport) getListener(network, addr string) (net.Listener, error) {
	var ln net.Listener
	v, _ := os.LookupEnv(transport.EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// Find the passed listener.
		pln, err := transport.GetPassedListener(network, addr)
		if err != nil {
			return nil, err
		}
		ln, ok = pln.(net.Listener)
		if !ok {
			return nil, fmt.Errorf("invalid listener type, want net.Listener, got %T", pln)
		}
		return ln, nil
	}

	if t.reusePort {
		ln, err := reuseport.Listen(network, addr)
		if err != nil {
			return nil, fmt.Errorf("http reuseport listen error:%v", err)
		}
		return ln, nil
	}

	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("http listen error:%v", err)
	}
	return ln, nil
}

// newHTTPServer creates http server.
func (t *ServerTransport) newHTTPServer(
	serveFunc func(w stdhttp.ResponseWriter, r *stdhttp.Request),
	opts *transport.ListenServeOptions,
) (*stdhttp.Server, error) {
	s := t.newServer()
	s.Addr = opts.Address
	s.Handler = stdhttp.HandlerFunc(serveFunc)
	if t.enableH2C {
		h2s := &http2.Server{}
		s.Handler = h2c.NewHandler(stdhttp.HandlerFunc(serveFunc), h2s)
		return s, nil
	}
	if len(opts.CACertFile) != 0 { // Enable two-way authentication to verify client certificate.
		s.TLSConfig = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
		certPool, err := itls.GetCertPool(opts.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("http server get ca cert file error:%v", err)
		}
		s.TLSConfig.ClientCAs = certPool
	}
	if opts.DisableKeepAlives {
		s.SetKeepAlivesEnabled(false)
	}
	if opts.IdleTimeout > 0 {
		s.IdleTimeout = opts.IdleTimeout
	}
	return s, nil
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept accepts new request.
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// ClientTransport client side http transport.
type ClientTransport struct {
	stdhttp.Client // http client, exposed variables, allow user to customize settings.
	opts           *transport.ClientTransportOptions
	tlsClients     map[string]*stdhttp.Client // Different certificate file use different TLS client.
	tlsLock        sync.RWMutex
	http2Only      bool
}

// DefaultClientTransport default client http transport.
var DefaultClientTransport = NewClientTransport(false)

// DefaultHTTP2ClientTransport default client http2 transport.
var DefaultHTTP2ClientTransport = NewClientTransport(true)

// NewClientTransport creates http transport.
func NewClientTransport(http2Only bool, opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}

	// Write func options to field opts.
	for _, o := range opt {
		o(opts)
	}
	return &ClientTransport{
		opts: opts,
		Client: stdhttp.Client{
			Transport: NewRoundTripper(StdHTTPTransport),
		},
		tlsClients: make(map[string]*stdhttp.Client),
		http2Only:  http2Only,
	}
}

func (ct *ClientTransport) getRequest(reqHeader *ClientReqHeader,
	reqBody []byte, msg codec.Msg, opts *transport.RoundTripOptions) (*stdhttp.Request, error) {
	req, err := ct.newRequest(reqHeader, reqBody, msg, opts)
	if err != nil {
		return nil, err
	}

	if reqHeader.Header != nil {
		req.Header = make(stdhttp.Header)
		for h, val := range reqHeader.Header {
			req.Header[h] = val
		}
	}
	if len(reqHeader.Host) != 0 {
		req.Host = reqHeader.Host
	}
	req.Header.Set(TrpcCaller, msg.CallerServiceName())
	req.Header.Set(TrpcCallee, msg.CalleeServiceName())
	req.Header.Set(TrpcTimeout, strconv.Itoa(int(msg.RequestTimeout()/time.Millisecond)))
	if opts.DisableConnectionPool {
		req.Header.Set(Connection, "close")
		req.Close = true
	}
	if t := msg.CompressType(); icodec.IsValidCompressType(t) && t != codec.CompressTypeNoop {
		req.Header.Set("Content-Encoding", compressTypeContentEncoding[t])
	}
	if msg.SerializationType() != codec.SerializationTypeNoop {
		if len(req.Header.Get("Content-Type")) == 0 {
			req.Header.Set("Content-Type",
				serializationTypeContentType[msg.SerializationType()])
		}
	}
	if err := ct.setTransInfo(msg, req); err != nil {
		return nil, err
	}
	if len(opts.TLSServerName) == 0 {
		opts.TLSServerName = req.Host
	}
	return req, nil
}

func (ct *ClientTransport) setTransInfo(msg codec.Msg, req *stdhttp.Request) error {
	var m map[string]string
	if md := msg.ClientMetaData(); len(md) > 0 {
		m = make(map[string]string, len(md))
		for k, v := range md {
			m[k] = ct.encodeBytes(v)
		}
	}

	// Set dyeing information.
	if msg.Dyeing() {
		if m == nil {
			m = make(map[string]string)
		}
		m[TrpcDyeingKey] = ct.encodeString(msg.DyeingKey())
		req.Header.Set(TrpcMessageType, strconv.Itoa(int(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)))
	}

	if msg.EnvTransfer() != "" {
		if m == nil {
			m = make(map[string]string)
		}
		m[TrpcEnv] = ct.encodeString(msg.EnvTransfer())
	} else {
		// If msg.EnvTransfer() empty, transmitted env info in req.TransInfo should be cleared
		if _, ok := m[TrpcEnv]; ok {
			m[TrpcEnv] = ""
		}
	}

	if len(m) > 0 {
		val, err := codec.Marshal(codec.SerializationTypeJSON, m)
		if err != nil {
			return errs.NewFrameError(errs.RetClientValidateFail, "http client json marshal metadata fail: "+err.Error())
		}
		req.Header.Set(TrpcTransInfo, string(val))
	}

	return nil
}

func (ct *ClientTransport) newRequest(reqHeader *ClientReqHeader,
	reqBody []byte, msg codec.Msg, opts *transport.RoundTripOptions) (*stdhttp.Request, error) {
	if reqHeader.Request != nil {
		return reqHeader.Request, nil
	}
	scheme := reqHeader.Schema
	if scheme == "" {
		if len(opts.CACertFile) > 0 || strings.HasSuffix(opts.Address, ":443") {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	body := reqHeader.ReqBody
	if body == nil {
		body = bytes.NewReader(reqBody)
	}

	request, err := stdhttp.NewRequest(
		reqHeader.Method,
		fmt.Sprintf("%s://%s%s", scheme, opts.Address, msg.ClientRPCName()),
		body)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"http client transport NewRequest: "+err.Error())
	}
	return request, nil
}

func (ct *ClientTransport) encodeBytes(in []byte) string {
	if ct.opts.DisableHTTPEncodeTransInfoBase64 {
		return string(in)
	}
	return base64.StdEncoding.EncodeToString(in)
}

func (ct *ClientTransport) encodeString(in string) string {
	if ct.opts.DisableHTTPEncodeTransInfoBase64 {
		return in
	}
	return base64.StdEncoding.EncodeToString([]byte(in))
}

// RoundTrip sends and receives http packets, put http response into ctx,
// no need to return rspBuf here.
func (ct *ClientTransport) RoundTrip(
	ctx context.Context,
	reqBody []byte,
	callOpts ...transport.RoundTripOption,
) (rspBody []byte, err error) {
	msg := codec.Message(ctx)
	reqHeader, ok := msg.ClientReqHead().(*ClientReqHeader)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"http client transport: ReqHead should be type of *http.ClientReqHeader")
	}
	rspHeader, ok := msg.ClientRspHead().(*ClientRspHeader)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"http client transport: RspHead should be type of *http.ClientRspHeader")
	}

	var opts transport.RoundTripOptions
	for _, o := range callOpts {
		o(&opts)
	}

	// Sets reqHeader.
	req, err := ct.getRequest(reqHeader, reqBody, msg, &opts)
	if err != nil {
		return nil, err
	}
	trace := &httptrace.ClientTrace{
		ConnectStart: func(network, addr string) {
			tcpAddr, _ := net.ResolveTCPAddr(network, addr)
			msg.WithRemoteAddr(tcpAddr)
		},
	}
	reqCtx := ctx
	cancel := context.CancelFunc(func() {})
	if rspHeader.ManualReadBody {
		// In the scenario of Manual Read body, the lifecycle of rsp body is different
		// from that of invoke ctx, and is independently controlled by body.Close().
		// Therefore, the timeout/cancel function in the original context needs to be replaced.
		controlCtx := context.Background()
		if deadline, ok := ctx.Deadline(); ok {
			controlCtx, cancel = context.WithDeadline(context.Background(), deadline)
		}
		reqCtx = icontext.NewContextWithValues(controlCtx, ctx)
	}
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	request := req.WithContext(httptrace.WithClientTrace(reqCtx, trace))

	client, err := ct.getStdHTTPClient(opts.CACertFile, opts.TLSCertFile,
		opts.TLSKeyFile, opts.TLSServerName)
	if err != nil {
		return nil, err
	}

	rspHeader.Response, err = client.Do(request)
	if err != nil {
		if e, ok := err.(*url.Error); ok {
			if e.Timeout() {
				return nil, errs.NewFrameError(errs.RetClientTimeout,
					"http client transport RoundTrip timeout: "+err.Error())
			}
		}
		if ctx.Err() == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"http client transport RoundTrip canceled: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"http client transport RoundTrip: "+err.Error())
	}
	rspHeader.Response.Body = &responseBodyWithCancel{body: rspHeader.Response.Body, cancel: cancel}
	return emptyBuf, nil
}

// responseBodyWithCancel implements io.ReadCloser.
// It wraps response body and cancel function.
type responseBodyWithCancel struct {
	body   io.ReadCloser
	cancel context.CancelFunc
}

func (b *responseBodyWithCancel) Read(p []byte) (int, error) {
	return b.body.Read(p)
}

func (b *responseBodyWithCancel) Close() error {
	b.cancel()
	return b.body.Close()
}

func (ct *ClientTransport) getStdHTTPClient(caFile, certFile,
	keyFile, serverName string) (*stdhttp.Client, error) {
	if len(caFile) == 0 { // HTTP requests share one client.
		return &ct.Client, nil
	}

	cacheKey := fmt.Sprintf("%s-%s-%s", caFile, certFile, serverName)
	ct.tlsLock.RLock()
	cli, ok := ct.tlsClients[cacheKey]
	ct.tlsLock.RUnlock()
	if ok {
		return cli, nil
	}

	ct.tlsLock.Lock()
	defer ct.tlsLock.Unlock()
	cli, ok = ct.tlsClients[cacheKey]
	if ok {
		return cli, nil
	}

	conf, err := itls.GetClientConfig(serverName, caFile, certFile, keyFile)
	if err != nil {
		return nil, err
	}
	client := &stdhttp.Client{
		CheckRedirect: ct.Client.CheckRedirect,
		Timeout:       ct.Client.Timeout,
	}
	if ct.http2Only {
		client.Transport = &http2.Transport{
			TLSClientConfig: conf,
		}
	} else {
		tr := StdHTTPTransport.Clone()
		tr.TLSClientConfig = conf
		client.Transport = NewRoundTripper(tr)
	}
	ct.tlsClients[cacheKey] = client
	return client, nil
}

// StdHTTPTransport all RoundTripper object used by http and https.
var StdHTTPTransport = &stdhttp.Transport{
	Proxy: stdhttp.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	IdleConnTimeout:       50 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	MaxIdleConnsPerHost:   100,
	DisableCompression:    true,
	ExpectContinueTimeout: time.Second,
}

// NewRoundTripper creates new NewRoundTripper and can be replaced.
var NewRoundTripper = newValueDetachedTransport
