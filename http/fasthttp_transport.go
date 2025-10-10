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
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	igr "trpc.group/trpc-go/trpc-go/internal/graceful"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	// Server transport (protocol file service).
	transport.RegisterServerTransport(protocol.FastHTTP, DefaultFastHTTPServerTransport)

	// Server transport (no protocol file service).
	transport.RegisterServerTransport(protocol.FastHTTPNoProtocol, DefaultFastHTTPServerTransport)

	// Client transport.
	transport.RegisterClientTransport(protocol.FastHTTP, DefaultFastHTTPClientTransport)
}

// FastHTTPServerTransport is the fasthttp transport layer.
// Users can directly configure the *fasthttp.Server by setting the Server field in FastHTTPServerTransport.
// Alternatively, configuration can also be done through opts.
type FastHTTPServerTransport struct {
	// Support external configuration.
	Server *fasthttp.Server
	opts   *transport.ServerTransportOptions
}

var (
	// DefaultFastHTTPClientTransport is the default fasthttp client transport.
	DefaultFastHTTPClientTransport = NewFastHTTPClientTransport()
	// DefaultFastHTTPServerTransport is the default fasthttp reuseport server transport.
	DefaultFastHTTPServerTransport = NewFastHTTPServerTransport(transport.WithReusePort(true))
)

// NewFastHTTPServerTransport creates fasthttp transport. The default idle time
// is set 1 min in config.go, which can be customized through ServerTransportOption.
// After invoking NewFastHTTPServerTransport(), user can configure the *fasthttp.Server
// by setting the Server field. Manually configuring st.Server.Handler by the user
// may introduce risks, so user MUST configure the st.Server.Handler by ListenServeOption.
func NewFastHTTPServerTransport(opt ...transport.ServerTransportOption) *FastHTTPServerTransport {
	opts := &transport.ServerTransportOptions{}
	for _, o := range opt {
		o(opts)
	}

	return &FastHTTPServerTransport{
		opts: opts,
	}
}

// ListenAndServe handles configuration and provides fasthttp service.
// The default network is tcp, which can be customized through ListenServeOption.
// It implements the transport.ServerTransport interface for FastHTTPServerTransport.
// Manually configuring st.Server.Handler by the user may introduce risks,
// so user MUST configure the st.Server.Handler by ListenServeOption.
func (st *FastHTTPServerTransport) ListenAndServe(
	ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: "tcp",
	}
	for _, o := range opt {
		o(opts)
	}
	// Manually configuring st.Server.Handler by the user may introduce risks,
	// so user MUST configure the st.Server.Handler by ListenServeOption.
	if opts.Handler == nil {
		return errors.New("fasthttp server transport handler empty")
	}
	return st.listenAndServeFastHTTP(ctx, opts)
}

// listenAndServeFastHTTP handles configuration and provides fasthttp service.
func (st *FastHTTPServerTransport) listenAndServeFastHTTP(
	ctx context.Context, opts *transport.ListenServeOptions) error {
	if err := st.configureFastHTTPServer(ctx, opts); err != nil {
		return err
	}
	return st.serve(ctx, opts)
}

// configureFastHTTPServer configures the fasthttp server
// based on the provided options or default values.
func (st *FastHTTPServerTransport) configureFastHTTPServer(
	ctx context.Context,
	opts *transport.ListenServeOptions,
) error {
	if st.Server == nil {
		st.Server = &fasthttp.Server{}
	}

	// Wrap opts.Handler for st.Server.Handler.
	st.Server.Handler = func(requestCtx *fasthttp.RequestCtx) {
		// User should avoid holding references to incoming RequestCtx and/or
		// its members after the Handler return.
		ctx := WithRequestCtx(ctx, requestCtx)
		// Generates new empty general message structure data and save it to ctx.
		ctx, msg := codec.WithNewMessage(ctx)
		defer codec.PutBackMessage(msg)

		var (
			span  rpcz.Span
			ender rpcz.Ender
		)

		if rpczenable.Enabled {
			span, ender, ctx = rpcz.NewSpanContext(ctx, "fasthttp-server")
			defer ender.End()
			span.SetAttribute(rpcz.HTTPAttributeURL, requestCtx.URI().String())
			span.SetAttribute(rpcz.HTTPAttributeRequestContentLength, requestCtx.Request.Header.ContentLength())
		}

		msg.WithLocalAddr(requestCtx.LocalAddr())
		msg.WithRemoteAddr(requestCtx.RemoteAddr())

		_, err := opts.Handler.Handle(ctx, nil)
		if err != nil {
			if rpczenable.Enabled {
				span.SetAttribute(rpcz.TRPCAttributeError, err)
			}
			log.Errorf("fasthttp server transport handle fail: %w", err)
			if errors.Is(err, ErrEncodeMissingRequestCtx) || errors.Is(err, errs.ErrServerNoResponse) {
				requestCtx.SetStatusCode(fasthttp.StatusInternalServerError)
				fmt.Fprintf(requestCtx, "fasthttp server handle error: %+v", err)
			}
			return
		}
	}

	if opts.DisableKeepAlives {
		st.Server.DisableKeepalive = true
	}

	// Configure the st.Server.TLSConfig for https.
	// Enable fasthttp server to verify client certificate.
	if len(opts.CACertFile) != 0 {
		st.Server.TLSConfig = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
		certPool, err := itls.GetCertPool(opts.CACertFile)
		if err != nil {
			return fmt.Errorf("fasthttp server get ca cert file error: %w", err)
		}
		st.Server.TLSConfig.ClientCAs = certPool
	}

	// The priority of options is strange but align with thttp.
	// Now ServerTransportOptions prioritized over the priority of ListenServeOptions,
	// Although Server these two should be at the same level (because LAS will only be performed once),
	// but if we compare it to Client, it would be equivalent to
	// ClientTransportOptions prioritized over RoundTripOptions.
	idleTimeout := opts.IdleTimeout
	if st.opts.IdleTimeout > 0 {
		idleTimeout = st.opts.IdleTimeout
	}
	st.Server.IdleTimeout = idleTimeout
	return nil
}

// serve uses the fasthttp server to provide services.
func (st *FastHTTPServerTransport) serve(ctx context.Context, opts *transport.ListenServeOptions) error {
	ln, err := getListener(opts, st.opts.ReusePort)
	if err != nil {
		return fmt.Errorf("fasthttp server transport get listener err: %w", err)
	}
	if err := transport.SaveListener(ln); err != nil {
		return fmt.Errorf("save listener error: %w", err)
	}
	ln = igr.UnwrapListener(ln)

	// ServeTLS will only be invoked if TLSKeyFile and TLSCertFile are configured.
	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		// We have already initialized the TLSConfig and created a cert pool for ClientCAs.
		// Therefore, we only need to load the TLS key pairs here.
		certs, err := itls.LoadTLSKeyPairs(opts.TLSCertFile, opts.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("load tls key pairs err: %w", err)
		}
		// If opts.CACertFile is empty, TLSConfig will be nil. Check it first.
		if st.Server.TLSConfig == nil {
			st.Server.TLSConfig = &tls.Config{}
		}
		st.Server.TLSConfig.Certificates = certs

		go func() {
			// The TLSConfig has been initialized, including ClientCAs and Certificates.
			// Therefore, it is only necessary to pass empty cert and key files to ServeTLS.
			if err := st.Server.ServeTLS(tcpKeepAliveListener{TCPListener: ln.(*net.TCPListener)},
				"", ""); err != nil {
				log.Errorf("serve TLS failed: %v", err)
			}
		}()
	} else {
		go func() {
			if err := st.Server.Serve(tcpKeepAliveListener{TCPListener: ln.(*net.TCPListener)}); err != nil {
				log.Errorf("serve err: %w", err)
			}
		}()
	}

	opts.ActiveCnt.Add(1)
	go func() {
		<-ctx.Done()
		if err := st.Server.Shutdown(); err != nil {
			log.Infof("shutdown err: %w", err)
		}
		opts.ActiveCnt.Add(-1)
	}()

	return nil
}

// FastHTTPClientTransport client side http transport.
// Users can directly configure the *fasthttp.Client by setting the Client field in FastHTTPClientTransport.
// Alternatively, configuration can also be done through opts.
type FastHTTPClientTransport struct {
	// Fasthttp client, exposed variables, allows user to customize settings.
	Client *fasthttp.Client
	opts   *transport.ClientTransportOptions
}

// NewFastHTTPClientTransport creates fasthttp transport.
func NewFastHTTPClientTransport(ctOpt ...transport.ClientTransportOption) *FastHTTPClientTransport {
	ctOpts := &transport.ClientTransportOptions{}
	for _, o := range ctOpt {
		o(ctOpts)
	}

	return &FastHTTPClientTransport{
		Client: &fasthttp.Client{},
		opts:   ctOpts,
	}
}

// RoundTrip implements the transport.ClientTransport interface for FastHTTPClientTransport.
// RoundTrip sends and receives fasthttp packets, put fasthttp response into ctx,
// and no need to return rspBuf here.
// TODO: trace
func (ct *FastHTTPClientTransport) RoundTrip(
	ctx context.Context,
	reqBody []byte,
	opt ...transport.RoundTripOption,
) ([]byte, error) {
	msg := codec.Message(ctx)
	reqHeader, ok := msg.ClientReqHead().(*FastHTTPClientReqHeader)
	if !ok {
		errMsg := fmt.Sprintf(
			"fasthttp client transport: ClientReqHead should be type of *FastHTTPClientReqHeader, current type: %T",
			reqHeader,
		)
		return nil, errs.NewFrameError(errs.RetClientEncodeFail, errMsg)
	}
	rspHeader, ok := msg.ClientRspHead().(*FastHTTPClientRspHeader)
	if !ok {
		errMsg := fmt.Sprintf(
			"fasthttp client transport: ClientReqHead should be type of *FastHTTPClientRspHeader, current type: %T",
			rspHeader,
		)
		return nil, errs.NewFrameError(errs.RetClientEncodeFail, errMsg)
	}

	opts := &transport.RoundTripOptions{}
	for _, o := range opt {
		o(opts)
	}

	if err := ct.getRequest(reqHeader, reqBody, msg, opts); err != nil {
		return nil, err
	}

	if rspHeader.Response == nil {
		rspHeader.Response = fasthttp.AcquireResponse()
	}

	// tfasthttp does NOT have explicitHTTPS, it won't change opts.CACertFile == "" to InsecureSkipVerify.
	// opts.CACertFile == "" means http,
	// opts.CACertFile == "none" means https + InsecureSkipVerify,
	// opts.CACertFile == "xxx" means https + Verify.
	if len(opts.CACertFile) != 0 {
		conf, err := itls.GetClientConfig(opts.TLSServerName, opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
		if err != nil {
			return nil, errs.WrapFrameError(err, errs.RetClientConnectFail, "fail to get client config for tls")
		}
		ct.Client.TLSConfig = conf
	}

	// Use DecorateRequest to make the final modifications to the request before sending it out.
	if reqHeader.DecorateRequest != nil {
		reqHeader.Request = reqHeader.DecorateRequest(reqHeader.Request)
	}

	// Handle timeout and redirect.
	if t, ok := ctx.Deadline(); ok {
		reqHeader.Request.SetTimeout(time.Until(t))
	}

	if err := ct.Client.DoRedirects(reqHeader.Request, rspHeader.Response, ct.opts.MaxRedirectsCount); err != nil {
		if err == fasthttp.ErrTimeout {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"fasthttp client transport RoundTrip timeout: "+err.Error())
		}
		if ctx.Err() == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"fasthttp client transport RoundTrip canceled: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"fasthttp client transport RoundTrip: "+err.Error())
	}

	return nil, nil
}

// 1. Obtain a fasthttp.Request for reqHeader, usually for FastHTTPClientProxy invocation.
// 2. Set the relevant fields from msg into the request headers.
func (ct *FastHTTPClientTransport) getRequest(
	reqHeader *FastHTTPClientReqHeader,
	reqBody []byte,
	msg codec.Msg,
	opts *transport.RoundTripOptions,
) error {
	if reqHeader.Request == nil {
		req, err := ct.acquireRequest(reqHeader, reqBody, msg, opts)
		if err != nil {
			return err
		}
		reqHeader.Request = req
	}

	req := reqHeader.Request
	req.Header.Set(canonicalTrpcCaller, msg.CallerServiceName())
	req.Header.Set(canonicalTrpcCallee, msg.CalleeServiceName())
	req.Header.Set(canonicalTrpcTimeout, strconv.FormatInt(msg.RequestTimeout().Milliseconds(), 10))

	if opts.DisableConnectionPool {
		req.SetConnectionClose()
	}

	if t := msg.CompressType(); icodec.IsValidCompressType(t) && t != codec.CompressTypeNoop {
		req.Header.Set(canonicalContentEncoding, compressTypeContentEncoding[t])
	}

	if v := msg.SerializationType(); v != codec.SerializationTypeNoop &&
		len(req.Header.Peek(canonicalContentType)) == 0 {
		req.Header.Set(canonicalContentType, serializationTypeContentType[v])
	}

	if err := ct.setTransInfo(msg, req); err != nil {
		return err
	}

	if len(opts.TLSServerName) == 0 {
		opts.TLSServerName = string(req.URI().Host())
	}

	return nil
}

// acquireRequest is often used by FastHTTPClientProxy, and it sets
// the relevant request Method, URI, reqBody, and Host.
// Request is acquired and released in fasthttp.
func (ct *FastHTTPClientTransport) acquireRequest(
	reqHeader *FastHTTPClientReqHeader,
	reqBody []byte,
	msg codec.Msg,
	rtOpts *transport.RoundTripOptions,
) (*fasthttp.Request, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(reqHeader.Method)

	req.SetRequestURI(
		fmt.Sprintf("%s://%s%s",
			ct.inferScheme(reqHeader.Scheme, rtOpts),
			rtOpts.Address,
			msg.ClientRPCName(),
		),
	)

	req.SetBody(reqBody)
	// After SetRequestURI.
	if len(reqHeader.Host) != 0 {
		req.SetHost(reqHeader.Host)
	}

	// Align With req, err := net/http.NewRequest(method, url, body).
	if err := checkRequest(req); err != nil {
		// Remember to releaseRequest.
		fasthttp.ReleaseRequest(req)
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"fasthttp client transport acquireRequest: "+err.Error())
	}
	return req, nil
}

// checkRequest checks fasthttp request with the logic of net/http.NewRequest.
func checkRequest(req *fasthttp.Request) error {
	if len(req.Header.Method()) == 0 {
		return errors.New("method cannot be empty")
	}

	uri := req.URI()
	if req.URI() == nil {
		return errors.New("URI cannot be nil")
	}

	scheme := string(uri.Scheme())
	if scheme == "" {
		return errors.New("URL scheme cannot be empty")
	}

	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %s", scheme)
	}

	if len(uri.Host()) == 0 {
		return errors.New("URL host cannot be empty")
	}
	return nil
}

// setTransInfo add the TransInfo in the msg to fasthttp.Request.Header.
func (ct *FastHTTPClientTransport) setTransInfo(msg codec.Msg, req *fasthttp.Request) error {
	// Delay the allocation of a map to avoid unnecessary memory allocation.
	// When adding new branches to the subsequent code, please remember to
	// check if the map is nil and construct it promptly.
	var m map[string]string

	if md := msg.ClientMetaData(); len(md) > 0 {
		m = make(map[string]string, len(md))
		for k, v := range md {
			m[k] = encodeBytes(v, ct.opts.DisableHTTPEncodeTransInfoBase64)
		}
	}

	if msg.Dyeing() {
		if m == nil {
			m = make(map[string]string)
		}
		m[TrpcDyeingKey] = encodeString(msg.DyeingKey(), ct.opts.DisableHTTPEncodeTransInfoBase64)
		req.Header.Set(canonicalTrpcMessageType,
			strconv.Itoa(int(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)))
	}

	if msg.EnvTransfer() != "" {
		if m == nil {
			m = make(map[string]string)
		}
		m[TrpcEnv] = encodeString(msg.EnvTransfer(), ct.opts.DisableHTTPEncodeTransInfoBase64)
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
			return errs.NewFrameError(
				errs.RetClientValidateFail, "fasthttp client json marshal metadata fail: "+err.Error(),
			)
		}
		req.Header.Set(canonicalTrpcTransInfo, string(val))
	}

	return nil
}

// inferScheme just by scheme and TLS config in tfasthttp without explicitHTTPS.
func (ct *FastHTTPClientTransport) inferScheme(scheme string, rtOpts *transport.RoundTripOptions) string {
	if scheme != "" {
		return scheme
	}
	if len(rtOpts.CACertFile) > 0 {
		return protocol.HTTPS
	}
	return protocol.HTTP
}

func encodeBytes(in []byte, disableHTTPEncodeTransInfoBase64 bool) string {
	if disableHTTPEncodeTransInfoBase64 {
		return string(in)
	}
	return base64.StdEncoding.EncodeToString(in)
}

func encodeString(in string, disableHTTPEncodeTransInfoBase64 bool) string {
	if disableHTTPEncodeTransInfoBase64 {
		return in
	}
	return base64.StdEncoding.EncodeToString([]byte(in))
}
