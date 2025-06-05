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
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	icontext "trpc.group/trpc-go/trpc-go/internal/context"
	igr "trpc.group/trpc-go/trpc-go/internal/graceful"
	"trpc.group/trpc-go/trpc-go/internal/http/fastop"
	inet "trpc.group/trpc-go/trpc-go/internal/net"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	// Server transport (protocol file service).
	transport.RegisterServerTransport(protocol.HTTP, DefaultServerTransport)
	transport.RegisterServerTransport(protocol.HTTPS, DefaultHTTPSServerTransport)
	transport.RegisterServerTransport(protocol.HTTP2, DefaultHTTP2ServerTransport)
	// Server transport (no protocol file service).
	transport.RegisterServerTransport(protocol.HTTPNoProtocol, DefaultServerTransport)
	transport.RegisterServerTransport(protocol.HTTPSNoProtocol, DefaultHTTPSServerTransport)
	transport.RegisterServerTransport(protocol.HTTP2NoProtocol, DefaultHTTP2ServerTransport)
	// Client transport.
	transport.RegisterClientTransport(protocol.HTTP, DefaultClientTransport)
	transport.RegisterClientTransport(protocol.HTTPS, DefaultHTTPSClientTransport)
	transport.RegisterClientTransport(protocol.HTTP2, DefaultHTTP2ClientTransport)
}

// DefaultServerTransport is the default server http transport.
var DefaultServerTransport = NewServerTransport(transport.WithReusePort(true))

// DefaultHTTPSServerTransport is the default server https transport.
var DefaultHTTPSServerTransport = makeServerHTTPSExplicit(NewServerTransport(transport.WithReusePort(true)))

// DefaultHTTP2ServerTransport is the default server http2 transport.
var DefaultHTTP2ServerTransport = NewServerTransport(transport.WithReusePort(true))

// ServerTransport is the http transport layer.
type ServerTransport struct {
	Server        *http.Server // Support external configuration.
	opts          *transport.ServerTransportOptions
	explicitHTTPS bool
}

func makeServerHTTPSExplicit(t transport.ServerTransport) transport.ServerTransport {
	s, ok := t.(*ServerTransport)
	if !ok {
		panic(fmt.Sprintf("makeServerHTTPSExplicit expects %T, got %T", (*ServerTransport)(nil), t))
	}
	s.explicitHTTPS = true
	return s
}

// NewServerTransport creates http transport.
// The default idle time is set 1 min in config.go,
// which can be customized through ServerTransportOption.
func NewServerTransport(opt ...transport.ServerTransportOption) transport.ServerTransport {
	opts := &transport.ServerTransportOptions{}

	// Write func options to field opts.
	for _, o := range opt {
		o(opts)
	}
	s := &ServerTransport{
		opts: opts,
	}
	return s
}

// ListenAndServe handles configuration.
func (t *ServerTransport) ListenAndServe(ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: protocol.TCP,
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
	serveFunc := func(w http.ResponseWriter, r *http.Request) {
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

		var (
			span  rpcz.Span
			ender rpcz.Ender
		)
		if rpczenable.Enabled {
			span, ender, ctx = rpcz.NewSpanContext(ctx, "http-server")
			defer ender.End()
			span.SetAttribute(rpcz.HTTPAttributeURL, r.URL)
			span.SetAttribute(rpcz.HTTPAttributeRequestContentLength, r.ContentLength)
		}

		// Records LocalAddr and RemoteAddr to Context.
		localAddr, ok := h.Request.Context().Value(http.LocalAddrContextKey).(net.Addr)
		if ok {
			msg.WithLocalAddr(localAddr)
		}
		remoteAddr := inet.ResolveAddress(protocol.TCP, h.Request.RemoteAddr)
		msg.WithRemoteAddr(remoteAddr)
		_, err := opts.Handler.Handle(ctx, emptyBuf)
		if err != nil {
			if rpczenable.Enabled {
				span.SetAttribute(rpcz.TRPCAttributeError, err)
			}
			log.Errorf("http server transport handle fail: %v", err)
			if err == ErrEncodeMissingHeader || errors.Is(err, errs.ErrServerNoResponse) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "http server handle error: %+v", err)
			}
			return
		}
	}

	s, err := t.newHTTPServer(serveFunc, opts)
	if err != nil {
		return err
	}

	t.configureHTTPServer(s, opts)

	return t.serve(ctx, s, opts)
}

func (t *ServerTransport) serve(ctx context.Context, s *http.Server, opts *transport.ListenServeOptions) error {
	ln, err := getListener(opts, t.opts.ReusePort)
	if err != nil {
		return fmt.Errorf("http server transport get listener err: %w", err)
	}

	if err := transport.SaveListener(ln); err != nil {
		return fmt.Errorf("save http listener error: %w", err)
	}
	ln = igr.UnwrapListener(ln)
	if t.explicitHTTPS &&
		(len(opts.TLSCertFile) == 0 || len(opts.TLSKeyFile) == 0) {
		return errors.New("server uses 'https' protocol, but some of the cert/key files are not provided, " +
			"please consider either setting the protocol to 'http' or providing all the necessary files for 'https'")
	}
	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		// We have already initialized the TLSConfig and created a cert pool for ClientCAs.
		// Therefore, we only need to load the TLS key pairs here.
		certs, err := itls.LoadTLSKeyPairs(opts.TLSCertFile, opts.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("load tls key pairs err: %w", err)
		}
		// If opts.CACertFile is empty, TLSConfig will be nil. Check it first.
		if s.TLSConfig == nil {
			s.TLSConfig = &tls.Config{}
		}
		s.TLSConfig.Certificates = certs

		go func() {
			// The TLSConfig has been initialized, including ClientCAs and Certificates.
			// Therefore, it is only necessary to pass empty cert and key files to ServeTLS.
			if err := s.ServeTLS(tcpKeepAliveListener{ln.(*net.TCPListener)},
				"", ""); err != http.ErrServerClosed {
				log.Errorf("serve TLS failed: %v", err)
			}
		}()
	} else {
		go func() {
			_ = s.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
		}()
	}

	opts.ActiveCnt.Add(1)
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
		opts.ActiveCnt.Add(-1)
	}()

	return nil
}

func getListener(opts *transport.ListenServeOptions, reusePort bool) (net.Listener, error) {
	if opts.Listener != nil {
		return opts.Listener, nil
	}

	return igr.Listen(opts.Network, opts.Address, reusePort)
}

// configureHTTPServer sets properties of http server.
func (t *ServerTransport) configureHTTPServer(svr *http.Server, opts *transport.ListenServeOptions) {
	if t.Server != nil {
		svr.ReadTimeout = t.Server.ReadTimeout
		svr.ReadHeaderTimeout = t.Server.ReadHeaderTimeout
		svr.WriteTimeout = t.Server.WriteTimeout
		svr.MaxHeaderBytes = t.Server.MaxHeaderBytes
		svr.IdleTimeout = t.Server.IdleTimeout
		svr.ConnState = t.Server.ConnState
		svr.ErrorLog = t.Server.ErrorLog
		svr.ConnContext = t.Server.ConnContext
	}

	idleTimeout := opts.IdleTimeout
	if t.opts.IdleTimeout > 0 {
		idleTimeout = t.opts.IdleTimeout
	}
	svr.IdleTimeout = idleTimeout
}

// newHTTPServer creates http server.
func (t *ServerTransport) newHTTPServer(serveFunc func(w http.ResponseWriter, r *http.Request),
	opts *transport.ListenServeOptions) (*http.Server, error) {
	s := &http.Server{
		Addr:    opts.Address,
		Handler: http.HandlerFunc(serveFunc),
	}
	if opts.DisableKeepAlives {
		s.SetKeepAlivesEnabled(false)
	}
	// Enable h2c without tls.
	if t.opts.EnableH2C {
		h2s := &http2.Server{}
		s.Handler = h2c.NewHandler(http.HandlerFunc(serveFunc), h2s)
		return s, nil
	}
	if len(opts.CACertFile) != 0 { // Enable two-way authentication to verify client certificate.
		s.TLSConfig = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
		certPool, err := itls.GetCertPool(opts.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("http server get ca cert file error: %v", err)
		}
		s.TLSConfig.ClientCAs = certPool
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
	http.Client   // http client, exposed variables, allow user to customize settings.
	opts          *transport.ClientTransportOptions
	tlsClients    map[string]*http.Client // Different certificate file use different TLS client.
	tlsLock       sync.RWMutex
	http2Only     bool
	explicitHTTPS bool
}

func makeClientHTTPSExplicit(t transport.ClientTransport) transport.ClientTransport {
	s, ok := t.(*ClientTransport)
	if !ok {
		panic(fmt.Sprintf("makeClientHTTPSExplicit expects %T, got %T", (*ClientTransport)(nil), t))
	}
	s.explicitHTTPS = true
	return s
}

// DefaultClientTransport default client http transport.
var DefaultClientTransport = NewClientTransport(false)

// DefaultHTTPSClientTransport is the default client https transport.
var DefaultHTTPSClientTransport = makeClientHTTPSExplicit(NewClientTransport(false))

// DefaultHTTP2ClientTransport default client http2 transport.
var DefaultHTTP2ClientTransport = NewClientTransport(true)

// NewClientTransport creates http transport.
func NewClientTransport(http2Only bool, opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}

	// Write func options to field opts.
	for _, o := range opt {
		o(opts)
	}

	var tr http.RoundTripper
	if opts.NewHTTPClientTransport != nil {
		tr = NewRoundTripper(opts.NewHTTPClientTransport())
	} else {
		tr = NewRoundTripper(StdHTTPTransport)
	}

	return &ClientTransport{
		opts: opts,
		Client: http.Client{
			Transport: tr,
		},
		tlsClients: make(map[string]*http.Client),
		http2Only:  http2Only,
	}
}

func (ct *ClientTransport) getRequest(reqHeader *ClientReqHeader,
	reqBody []byte, msg codec.Msg, opts *transport.RoundTripOptions) (*http.Request, error) {
	req, err := ct.newRequest(reqHeader, reqBody, msg, opts)
	if err != nil {
		return nil, err
	}

	if reqHeader.Header != nil {
		if req.Header == nil { // 🤔 This check is rarely true, as http.NewRequest always makes the Header beforehand.
			// Create the header just in time to prevent any potential trampling.
			req.Header = make(http.Header)
		}
		for h, val := range reqHeader.Header {
			req.Header[h] = val
		}
	}
	if len(reqHeader.Host) != 0 {
		req.Host = reqHeader.Host
	}
	fastop.CanonicalHeaderSet(req.Header, canonicalTrpcCaller, msg.CallerServiceName())
	fastop.CanonicalHeaderSet(req.Header, canonicalTrpcCallerMethod, msg.CallerMethod())
	fastop.CanonicalHeaderSet(req.Header, canonicalTrpcCallee, msg.CalleeServiceName())
	fastop.CanonicalHeaderSet(req.Header, canonicalTrpcTimeout, strconv.FormatInt(msg.RequestTimeout().Milliseconds(), 10))
	if opts.DisableConnectionPool {
		fastop.CanonicalHeaderSet(req.Header, Connection, "close")
		req.Close = true
	}
	if t := msg.CompressType(); icodec.IsValidCompressType(t) && t != codec.CompressTypeNoop {
		fastop.CanonicalHeaderSet(req.Header, canonicalContentEncoding, compressTypeContentEncoding[t])
	}
	if msg.SerializationType() != codec.SerializationTypeNoop {
		if len(fastop.CanonicalHeaderGet(req.Header, canonicalContentType)) == 0 {
			fastop.CanonicalHeaderSet(req.Header, canonicalContentType,
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

func (ct *ClientTransport) setTransInfo(msg codec.Msg, req *http.Request) error {
	// Delay the allocation of a map to avoid unnecessary memory allocation.
	// When adding new branches to the subsequent code, please remember to
	// check if the map is nil and construct it promptly.
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
		fastop.CanonicalHeaderSet(req.Header, canonicalTrpcMessageType,
			strconv.Itoa(int(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)))
	}

	if msg.EnvTransfer() != "" {
		if m == nil {
			m = make(map[string]string)
		}
		m[TrpcEnv] = ct.encodeString(msg.EnvTransfer())
	} else {
		// If msg.EnvTransfer() empty, transmitted env info in req.TransInfo should be cleared.
		// The map needs to be constructed only when assigning values to it.
		// It is valid to check existence of an element in a nil map.
		if _, ok := m[TrpcEnv]; ok {
			m[TrpcEnv] = ""
		}
	}

	if len(m) > 0 {
		val, err := codec.Marshal(codec.SerializationTypeJSON, m)
		if err != nil {
			return errs.NewFrameError(errs.RetClientValidateFail, "http client json marshal metadata fail: "+err.Error())
		}
		fastop.CanonicalHeaderSet(req.Header, canonicalTrpcTransInfo, string(val))
	}

	return nil
}

func (ct *ClientTransport) newRequest(reqHeader *ClientReqHeader,
	reqBody []byte, msg codec.Msg, opts *transport.RoundTripOptions) (*http.Request, error) {
	if reqHeader.Request != nil {
		return reqHeader.Request, nil
	}

	body := reqHeader.ReqBody
	if body == nil && reqHeader.Method != http.MethodGet { // Body can still be nil if method is GET.
		body = bytes.NewReader(reqBody)
	}

	request, err := http.NewRequest(
		reqHeader.Method,
		fmt.Sprintf("%s://%s%s", ct.inferScheme(reqHeader.Schema, opts), opts.Address, msg.ClientRPCName()),
		body)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"http client transport NewRequest: "+err.Error())
	}
	return request, nil
}

func (ct *ClientTransport) inferScheme(scheme string, opts *transport.RoundTripOptions) string {
	if ct.explicitHTTPS {
		return protocol.HTTPS // This is the raison d'être of the "explicitHTTPS" flag.
	}
	// The following logic is retained for backward compatibility 🤔.
	if scheme == "" {
		if len(opts.CACertFile) > 0 || strings.HasSuffix(opts.Address, ":443") {
			scheme = protocol.HTTPS
		} else {
			scheme = protocol.HTTP
		}
	}
	return scheme
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

// RoundTrip sends and receives http packets, puts http response into ctx,
// no need to return rspBuf here.
func (ct *ClientTransport) RoundTrip(
	ctx context.Context,
	reqBody []byte,
	callOpts ...transport.RoundTripOption,
) (rspBody []byte, err error) {
	msg := codec.Message(ctx)
	reqHeader, rspHeader, err := ct.validateHeaders(msg)
	if err != nil {
		return nil, err
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
		GotConn: func(info httptrace.GotConnInfo) {
			msg.WithRemoteAddr(info.Conn.RemoteAddr())
			msg.WithLocalAddr(info.Conn.LocalAddr())
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

	client, err := ct.getStdHTTPClient(opts)
	if err != nil {
		return nil, err
	}

	// Use DecorateRequest to make the final modifications to the request before sending it out.
	if reqHeader.DecorateRequest != nil {
		request = reqHeader.DecorateRequest(request)
	}

	rspHeader.Response, err = client.Do(request)
	if err != nil {
		return nil, ct.handleRoundTripError(err, ctx.Err())
	}

	if rspHeader.ManualReadBody {
		// Only need to decorate with cancel when it is in manual read body mode.
		decorateWithCancel(rspHeader, cancel)
	}
	return emptyBuf, nil
}

// validateHeaders validates request and response headers.
func (ct *ClientTransport) validateHeaders(msg codec.Msg) (*ClientReqHeader, *ClientRspHeader, error) {
	reqHeader, ok := msg.ClientReqHead().(*ClientReqHeader)
	if !ok {
		return nil, nil, errs.NewFrameError(errs.RetClientEncodeFail,
			fmt.Sprintf("http client transport: ReqHead should be type of *http.ClientReqHeader, current type: %T", reqHeader))
	}

	rspHeader, ok := msg.ClientRspHead().(*ClientRspHeader)
	if !ok {
		return nil, nil, errs.NewFrameError(errs.RetClientEncodeFail,
			fmt.Sprintf("http client transport: RspHead should be type of *http.ClientRspHeader, current type: %T", rspHeader))
	}

	return reqHeader, rspHeader, nil
}

// handleRoundTripError handles errors during RoundTrip.
func (ct *ClientTransport) handleRoundTripError(err error, ctxErr error) error {
	if e, ok := err.(*url.Error); ok && e.Timeout() {
		return errs.NewFrameError(errs.RetClientTimeout,
			"http client transport RoundTrip timeout: "+err.Error())
	}
	if ctxErr == context.Canceled {
		return errs.NewFrameError(errs.RetClientCanceled,
			"http client transport RoundTrip canceled: "+err.Error())
	}
	return errs.NewFrameError(errs.RetClientNetErr,
		"http client transport RoundTrip: "+err.Error())
}

func decorateWithCancel(rspHeader *ClientRspHeader, cancel context.CancelFunc) {
	// Quoted from: https://github.com/golang/go/blob/go1.21.4/src/net/http/response.go#L69
	//
	// "As of Go 1.12, the Body will also implement io.Writer on a successful "101 Switching Protocols" response,
	// as used by WebSockets and HTTP/2's "h2c" mode."
	//
	// Therefore, we require an extra check to ensure io.Writer's conformity,
	// which will then expose the corresponding method.
	//
	// It's important to note that an embedded body may not be capable of exposing all the attached interfaces.
	// Consequently, we perform an explicit interface assertion here.
	if body, ok := rspHeader.Response.Body.(io.ReadWriteCloser); ok {
		rspHeader.Response.Body = &writableResponseBodyWithCancel{ReadWriteCloser: body, cancel: cancel}
	} else {
		rspHeader.Response.Body = &responseBodyWithCancel{ReadCloser: rspHeader.Response.Body, cancel: cancel}
	}
}

// writableResponseBodyWithCancel implements io.ReadWriteCloser.
// It wraps response body and cancel function.
type writableResponseBodyWithCancel struct {
	io.ReadWriteCloser
	cancel context.CancelFunc
}

func (b *writableResponseBodyWithCancel) Close() error {
	b.cancel()
	return b.ReadWriteCloser.Close()
}

// responseBodyWithCancel implements io.ReadCloser.
// It wraps response body and cancel function.
type responseBodyWithCancel struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *responseBodyWithCancel) Close() error {
	b.cancel()
	return b.ReadCloser.Close()
}

func (ct *ClientTransport) getStdHTTPClient(opts transport.RoundTripOptions) (*http.Client, error) {
	// HTTP requests share one client.
	if len(opts.CACertFile) == 0 && !ct.explicitHTTPS {
		// Update transport, like connection pool configurations.
		ct.Client.Transport = roundTripperWithOptions(ct.Client.Transport, opts)
		return &ct.Client, nil
	}
	if opts.CACertFile == "" { // For explicit HTTPS, caFile must not be empty.
		opts.CACertFile = "none" // If it is, set it to "none" to use tlsConf.InsecureSkipVerify=true.
	}

	cacheKey := fmt.Sprintf("%s-%s-%s", opts.CACertFile, opts.TLSCertFile, opts.TLSServerName)
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

	conf, err := itls.GetClientConfig(opts.TLSServerName, opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientConnectFail, "getting standard http client failed")
	}
	client := &http.Client{
		CheckRedirect: ct.Client.CheckRedirect,
		Timeout:       ct.Client.Timeout,
	}
	if ct.http2Only {
		client.Transport = &http2.Transport{
			TLSClientConfig: conf,
		}
	} else {
		var tr *http.Transport
		if ct.opts.NewHTTPClientTransport != nil {
			tr = ct.opts.NewHTTPClientTransport()
		} else {
			tr = StdHTTPTransport.Clone()
		}
		tr.TLSClientConfig = conf
		client.Transport = NewRoundTripper(tr)
		client.Transport = roundTripperWithOptions(client.Transport, opts)
	}
	ct.tlsClients[cacheKey] = client
	return client, nil
}

// StdHTTPTransport all RoundTripper object used by http and https.
var StdHTTPTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
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
