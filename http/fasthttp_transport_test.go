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

package http_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/server"
	helloworld "trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
	"trpc.group/trpc-go/trpc-go/transport"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestFastHTTPServerTransport(t *testing.T) {
	ctx := context.Background()
	st := thttp.DefaultFastHTTPServerTransport
	ln := mustListen(t)
	defer ln.Close()

	// Perform a test for normal case.
	require.Nil(t, st.ListenAndServe(ctx,
		transport.WithListener(ln),
		transport.WithHandler(transport.Handler(&h{})),
	))

	// Perform a test for handler not found.
	require.NotNil(t, st.ListenAndServe(ctx,
		transport.WithListener(ln),
	))

	// Perform a test for invalid network.
	require.NotNil(t, st.ListenAndServe(ctx,
		transport.WithListenAddress("127.0.0.2:8888"),
		transport.WithHandler(transport.Handler(&h{})),
		transport.WithListenNetwork("invalid network")),
	)

	t.Run("invalid tls", func(t *testing.T) {
		// Perform a test for CACertFile not found.
		require.NotNil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(serverCert, serverKey, notExistPem)),
		)
		require.NotNil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(serverCert, serverKey,
				strings.Join([]string{caPem, notExistPem}, tlsFileSeparator))),
		)
		// Perform a test for cert or key files not exist.
		require.NotNil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(notExistCert, serverKey, caPem)),
		)
		require.NotNil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(serverCert, notExistKey, caPem)),
		)
		require.NotNil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(notExistCert, notExistKey, caPem)),
		)
		// Perform a test for invalid cert and key files length.
		require.NotNil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(
				strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
				strings.Join([]string{serverKey}, tlsFileSeparator),
				caPem)),
		)
		// Perform a test for cert and key files not exist.
		require.NotNil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(
				strings.Join([]string{serverCert, notExistCert}, tlsFileSeparator),
				strings.Join([]string{serverKey, notExistKey}, tlsFileSeparator),
				caPem)),
		)
	})

	t.Run("valid tls", func(t *testing.T) {
		// Empty CACertFile.
		require.Nil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(serverCert, serverKey, "")),
		)
		// Perform a test for normal single CACertFile.
		require.Nil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(serverCert, serverKey, caPem)),
		)
		// Perform a test for normal multiple CACertFiles.
		require.Nil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(serverCert, serverKey,
				strings.Join([]string{caPem, caPem}, tlsFileSeparator))),
		)
		// Perform a test for single CACertFile and multiple cert and key files.
		require.Nil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(
				strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
				strings.Join([]string{serverKey, serverKey}, tlsFileSeparator),
				caPem)),
		)
		// Perform a test for multiple CACertFiles and multiple cert and key files.
		require.Nil(t, st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(
				strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
				strings.Join([]string{serverKey, serverKey}, tlsFileSeparator),
				strings.Join([]string{caPem, caPem}, tlsFileSeparator))),
		)
	})
}

func TestFastHTTPDisableReusePort(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewFastHTTPServerTransport(transport.WithReusePort(false))
	ln1 := mustListen(t)
	defer ln1.Close()
	option := transport.WithListener(ln1)
	handler := transport.WithHandler(transport.Handler(&h{}))
	require.Nil(t, tp.ListenAndServe(ctx, option, handler), "Failed to new client transport")

	option = transport.WithListenAddress(ln1.Addr().String())
	require.NotNil(t, tp.ListenAndServe(ctx, option, handler, transport.WithListenNetwork("tcp1")))

	ln2 := mustListen(t)
	defer ln2.Close()
	option = transport.WithListener(ln2)
	tls := transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "")
	require.Nil(t, tp.ListenAndServe(ctx, option, handler, tls))

	ln3 := mustListen(t)
	defer ln3.Close()
	option = transport.WithListener(ln3)
	tls = transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "root")
	require.Nil(t, tp.ListenAndServe(ctx, option, handler, tls))

	ln4 := mustListen(t)
	defer ln4.Close()
	option = transport.WithListener(ln4)
	tls = transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "../testdata/ca.key")
	require.NotNil(t, tp.ListenAndServe(ctx, option, handler, tls))
}

func TestFastHTTPServerTransportWithErrHandler(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultFastHTTPServerTransport
	ln := mustListen(t)
	defer ln.Close()
	require.Nil(t, tp.ListenAndServe(ctx,
		transport.WithListener(ln),
		transport.WithHandler(transport.Handler(&errHandler{}))),
	)

	ct := thttp.NewFastHTTPClientTransport()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestFastHTTPServerTransportWithErrHeaderHandler(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultFastHTTPServerTransport
	ln := mustListen(t)
	defer ln.Close()
	require.Nil(t, tp.ListenAndServe(ctx,
		transport.WithListener(ln),
		transport.WithHandler(transport.Handler(&errHeaderHandler{}))),
	)

	ct := thttp.NewFastHTTPClientTransport()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndNoCheckFastHTTPServer(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	// Only enables https server and do not verify client certificate.
	require.Nil(
		t,
		thttp.NewFastHTTPServerTransport(transport.WithReusePort(true)).ListenAndServe(
			ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", ""),
		),
	)

	ct := thttp.NewFastHTTPClientTransport()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})

	rsp, err := ct.RoundTrip(
		ctx,
		[]byte("{\"username\":\"xyz\","+"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// Fully trust the https server and do not verify server certificate,
		// can only be used in test env.
		transport.WithDialTLS("", "", "none", ""),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndCheckFastHTTPServer(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewFastHTTPServerTransport(transport.WithReusePort(true))
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	err := tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&h{})),
		// Only enables https server and do not verify client certificate.
		transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", ""),
		transport.WithListener(ln),
	)
	require.Nil(t, err, "Failed to new client transport")

	ct := thttp.NewFastHTTPClientTransport()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// Uses ca public key to verify server certificate.
		transport.WithDialTLS("", "", "../testdata/ca.pem", "localhost"),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndCheckClientNoCertFastHTTP(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewFastHTTPServerTransport(transport.WithReusePort(true))
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	err := tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&h{})),
		// Enables two-way authentication http server and need to verify client certificate.
		transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "../testdata/ca.pem"),
		transport.WithListener(ln),
	)
	require.Nil(t, err, "Failed to new client transport")

	ct := thttp.NewFastHTTPClientTransport()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})

	_, err = ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// If the client's own certificate is not sent, will return TLS verification failed.
		transport.WithDialTLS("", "", "../testdata/ca.pem", "localhost"),
	)
	require.NotNil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndCheckClientFastHTTP(t *testing.T) {
	ln := mustListen(t)

	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.FastHTTP),
		server.WithListener(ln),
		server.WithTLS(
			"../testdata/server.crt",
			"../testdata/server.key",
			"../testdata/ca.pem",
		),
	)
	pattern := "/" + t.Name()

	thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	defer func() {
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	thttp.FastHTTPHandleFunc(pattern, func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString(t.Name())
	})

	thttp.RegisterNoProtocolService(service)
	s := &server.Server{}
	s.AddService(serviceName, service)

	go s.Serve()
	time.Sleep(1 * time.Second)

	c := thttp.NewFastHTTPClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithTLS("../testdata/client.crt", "../testdata/client.key", "../testdata/ca.pem", "localhost"),
	)
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t, c.Post(context.Background(), pattern, req, rsp,
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
	))
	t.Log(string(rsp.Data))
	require.Equal(t, t.Name(), string(rsp.Data))
}

func TestStartDisableKeepAlivesFastHTTPServer(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.fasthttp.server.ListenerTest"),
		server.WithProtocol(protocol.FastHTTP),
		server.WithTransport(
			thttp.NewFastHTTPServerTransport(transport.WithReusePort(true)),
		),
		server.WithDisableKeepAlives(true),
	)

	thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	defer func() {
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	thttp.FastHTTPHandleFunc("/keepalive", func(ctx *fasthttp.RequestCtx) {
		// default: Connection: Keep-Alive, not thing we need to do.
	})
	thttp.RegisterDefaultService(service)

	s.AddService("trpc.fasthttp.server.ListenerTest", service)
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()
	defer func() {
		_ = s.Close(nil)
	}()

	time.Sleep(100 * time.Millisecond)

	dialCount := 0
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialCount++
				conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
				return conn, err
			},
		},
	}
	num := 3
	url := fmt.Sprintf("http://%s/keepalive", ln.Addr())
	for i := 0; i < num; i++ {
		resp, err := client.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		_, err = io.Copy(io.Discard, resp.Body)
		require.Nil(t, err)
	}
	// We set server.WithDisableKeepAlives(true) and Connection: Keep-Alive,
	// and the server.WithDisableKeepAlives(true) takes effect,
	// it goes without saying the priority.
	require.Equal(t, num, dialCount)
}

func TestFastHTTPClientTransport(t *testing.T) {
	go fasthttp.ListenAndServe("127.0.0.1:8088", func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Method()) == fasthttp.MethodPut {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.WriteString("unsupported method")
			return
		}
		ctx.Write(ctx.Request.Body())
	})
	time.Sleep(time.Second)

	ct := thttp.NewFastHTTPClientTransport(transport.WithClientUDPRecvSize(65536))
	require.NotNil(t, ct)

	ctx, msg := codec.WithNewMessage(context.Background())

	// Perform a test for reqHeader is nil.
	_, err := ct.RoundTrip(context.Background(), nil)
	require.NotNil(t, err)

	// Perform a test for rspHeader is nil.
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{})
	_, err = ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)

	// Perform a test for HOST is nil.
	ctx, msg = codec.WithNewMessage(context.Background())
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{
		DecorateRequest: func(requestCtx *fasthttp.Request) *fasthttp.Request { return requestCtx },
	})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)

	// FastHTTPClientReqHeader.Host > transport.WithDialAddress.
	ctx, msg = codec.WithNewMessage(context.Background())
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{
		Host:            "a",
		DecorateRequest: func(requestCtx *fasthttp.Request) *fasthttp.Request { return requestCtx },
	})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	_, err = ct.RoundTrip(ctx, nil, transport.WithDialAddress("127.0.0.1:8088"))
	require.NotNil(t, err)

	// FastHTTPClientReqHeader.Host > transport.WithDialAddress.
	ctx, msg = codec.WithNewMessage(context.Background())
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{
		Host:            "127.0.0.1:8088",
		DecorateRequest: func(requestCtx *fasthttp.Request) *fasthttp.Request { return requestCtx },
	})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	_, err = ct.RoundTrip(ctx, nil, transport.WithDialAddress("a"))
	require.Nil(t, err)

	// Perform a test for for setTransInfo.
	ctx, msg = codec.WithNewMessage(context.Background())
	msg.WithClientMetaData(codec.MetaData{"testK": []byte("testV")})
	reqHead := &thttp.FastHTTPClientReqHeader{
		Host:            "127.0.0.1:8088",
		DecorateRequest: func(requestCtx *fasthttp.Request) *fasthttp.Request { return requestCtx },
	}
	msg.WithClientReqHead(reqHead)
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	msg.WithDyeing(true)
	msg.WithEnvTransfer("test")
	_, err = ct.RoundTrip(ctx, nil)
	require.Nil(t, err)
	require.NotNil(t, reqHead.Request.Header.Peek("Trpc-Trans-Info"))
}

func TestFastHTTPClientWithSelectorNode(t *testing.T) {
	ctx := context.Background()
	type testCase struct {
		target   string
		address  string
		listener net.Listener
	}
	var tests []testCase
	for i := 0; i < 2; i++ {
		ln := mustListen(t)
		defer ln.Close()
		addr := ln.Addr().String()
		tests = append(tests, testCase{"ip://" + addr, addr, ln})
	}
	for _, tt := range tests {
		tp := thttp.NewFastHTTPServerTransport(transport.WithReusePort(false))
		err := tp.ListenAndServe(ctx,
			transport.WithListener(tt.listener),
			transport.WithHandler(transport.Handler(&h{})))
		require.Nil(t, err, "Failed to new client transport")

		proxy := thttp.NewFastHTTPClientProxy("trpc.test.helloworld.Greeter",
			client.WithTarget(tt.target),
			client.WithSerializationType(codec.SerializationTypeNoop),
		)

		reqBody := &codec.Body{
			Data: []byte("{\"username\":\"xyz\"," +
				"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		}
		rspBody := &codec.Body{}
		n := &registry.Node{}
		require.Nil(t,
			proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody, client.WithSelectorNode(n)),
			"Failed to post")
		require.Equal(t, tt.address, n.Address)
	}
}

func TestFastHTTPReqHeaderWithContentType(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer ln.Close()
	tp := thttp.NewFastHTTPServerTransport()
	require.Nil(t, tp.ListenAndServe(ctx,
		transport.WithListener(ln),
		transport.WithHandler(transport.Handler(&h{}))),
	)
	var tests = []struct {
		expected string
	}{
		{"application/json"},
		{"application/jsonp"},
		{"application/jsonp123"},
		{"application/text123"},
	}

	rh := &thttp.FastHTTPClientReqHeader{}

	for _, tt := range tests {
		rh.DecorateRequest = func(r *fasthttp.Request) *fasthttp.Request {
			r.Header.SetContentType(tt.expected)
			return r
		}
		fcp := thttp.NewFastHTTPClientProxy(
			"trpc.test.helloworld.Greeter",
			client.WithTarget("ip://"+ln.Addr().String()),
			client.WithSerializationType(codec.SerializationTypeForm),
			client.WithReqHead(rh),
		)
		reqBody := &codec.Body{}
		rspBody := &codec.Body{}
		err := fcp.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody)
		require.Nil(t, err)
		t.Log(reqBody, rspBody)
	}
}

func TestFastHTTPCheckRedirect(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer ln.Close()
	// Start server for redirect.
	go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/real":
			ctx.WriteString("real")
		case "/a":
			ctx.Redirect("/b", fasthttp.StatusMovedPermanently)
		case "/b":
			ctx.Redirect("/real", fasthttp.StatusMovedPermanently)
		}
	})

	time.Sleep(200 * time.Millisecond)

	ct := thttp.NewFastHTTPClientTransport(transport.WithMaxRedirectsCount(1))
	fcp := thttp.NewFastHTTPClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithTransport(ct),
	)
	reqBody := &codec.Body{}
	rspBody := &codec.Body{}
	// Only redirect once form /b.
	require.Nil(t, fcp.Post(ctx, "/b", reqBody, rspBody))
	t.Log(string(rspBody.Data))
	// Redirect twice from /a.
	err := fcp.Post(ctx, "/a", reqBody, rspBody)
	require.NotNil(t, err)
	require.True(t, strings.Contains(err.Error(), "too many redirects detected when doing the request"))
}

func TestFastHTTPClientTransportError(t *testing.T) {
	http.HandleFunc("/fasthttp_timeout", func(http.ResponseWriter, *http.Request) {
		time.Sleep(time.Second)
	})
	http.HandleFunc("/fasthttp_cancel", func(http.ResponseWriter, *http.Request) {})
	ln := mustListen(t)
	defer ln.Close()
	go func() { http.Serve(ln, nil) }()
	time.Sleep(200 * time.Millisecond)

	fcp := thttp.NewFastHTTPClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithTimeout(500*time.Millisecond),
	)

	rspBody := &codec.Body{}
	err := fcp.Get(context.Background(), "/fasthttp_timeout", rspBody)
	terr, ok := err.(*errs.Error)
	require.True(t, ok)
	require.Equal(t, terr.Code, int32(errs.RetClientTimeout))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = fcp.Get(ctx, "/fasthttp_cancel", rspBody)
	terr, ok = err.(*errs.Error)
	require.True(t, ok)
	require.Equal(t, terr.Code, int32(errs.RetClientCanceled))
}

func TestFastHTTPClientRoundDyeing(t *testing.T) {
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithDyeing(true)
	const dyeingKey = "dyeingKey"
	msg.WithDyeingKey(dyeingKey)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{Request: req})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	msg.WithClientMetaData(codec.MetaData{
		thttp.TrpcDyeingKey: []byte(dyeingKey),
	})
	_, err := thttp.DefaultFastHTTPClientTransport.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Equal(t, string(req.Header.Peek(thttp.TrpcMessageType)),
		strconv.Itoa(int(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)))
}

func TestFastHTTPClientRoundEnvTransfer(t *testing.T) {
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithEnvTransfer("feat,master")
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{Request: req})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	_, err := thttp.DefaultFastHTTPClientTransport.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Contains(t, string(req.Header.Peek(thttp.TrpcTransInfo)), thttp.TrpcEnv)
}

func TestFastHTTPDisableBase64EncodeTransInfo(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewFastHTTPClientTransport(transport.WithDisableEncodeTransInfoBase64())
	ctx, msg := codec.WithNewMessage(ctx)
	const (
		envTrans  = "feat,master"
		metaVal   = "value"
		dyeingKey = "dyeingKey"
	)
	msg.WithEnvTransfer(envTrans)
	msg.WithClientMetaData(codec.MetaData{"key": []byte(metaVal)})
	msg.WithDyeing(true)
	msg.WithDyeingKey(dyeingKey)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{Request: req})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	// err != nil but req.Header contains infos.
	_, err := ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Contains(t, string(req.Header.Peek(thttp.TrpcTransInfo)), envTrans)
	require.Contains(t, string(req.Header.Peek(thttp.TrpcTransInfo)), metaVal)
	require.Contains(t, string(req.Header.Peek(thttp.TrpcTransInfo)), dyeingKey)
}

func TestFastHTTPDisableServiceRouterTransInfo(t *testing.T) {
	ctx := context.Background()
	a := require.New(t)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientMetaData(codec.MetaData{thttp.TrpcEnv: []byte("orienv")}) // this emulate decode trpc protocol client request
	msg.WithEnvTransfer("feat,master")
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{Request: req})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})
	_, err := thttp.DefaultFastHTTPClientTransport.RoundTrip(ctx, nil)
	a.NotNil(err)
	info, err := thttp.UnmarshalTransInfo(msg, string(req.Header.Peek(thttp.TrpcTransInfo)))
	a.NoError(err)
	a.Equal(string(info[thttp.TrpcEnv]), "feat,master")

	msg.WithEnvTransfer("") // DisableServiceRouter would clear EnvTransfer
	_, err = thttp.DefaultFastHTTPClientTransport.RoundTrip(ctx, nil)
	a.NotNil(err)
	info, err = thttp.UnmarshalTransInfo(msg, string(req.Header.Peek(thttp.TrpcTransInfo)))
	a.NoError(err)
	a.Equal(string(info[thttp.TrpcEnv]), "")
}

func TestFastHTTPSUseClientVerify(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.FastHTTPNoProtocol),
		server.WithListener(ln),
		server.WithTLS(
			"../testdata/server.crt",
			"../testdata/server.key",
			"../testdata/ca.pem",
		),
	)
	pattern := "/" + t.Name()
	thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	defer func() {
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	thttp.FastHTTPHandleFunc(pattern, func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString(t.Name())
	})

	thttp.RegisterNoProtocolService(service)

	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewFastHTTPClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	// Perform a test for normal case.
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithTLS(
				"../testdata/client.crt",
				"../testdata/client.key",
				"../testdata/ca.pem",
				"localhost",
			),
		))
	require.Equal(t, []byte(t.Name()), rsp.Data)

	// Perform a test for bad cert file.
	req = &codec.Body{}
	rsp = &codec.Body{}
	err := c.Post(context.Background(), pattern, req, rsp,
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithTLS(
			"bad cert file",
			"../testdata/server.key",
			"../testdata/ca.pem",
			"localhost",
		),
	)
	require.Equal(t, errs.RetClientConnectFail, errs.Code(err))
	require.Contains(t, errs.Msg(err), "fail to get client config for tls")
}

func TestFastHTTPSSkipClientVerify(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.FastHTTPNoProtocol),
		server.WithListener(ln),
		server.WithTransport(thttp.NewFastHTTPServerTransport(transport.WithReusePort(true))),
		server.WithTLS(
			"../testdata/server.crt",
			"../testdata/server.key",
			"",
		),
	)
	pattern := "/" + t.Name()
	thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	defer func() {
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	thttp.FastHTTPHandleFunc(pattern, func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString(t.Name())
	})
	thttp.RegisterNoProtocolService(service)
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewFastHTTPClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithTLS(
				"", "", "none", "",
			),
		))
	require.Equal(t, []byte(t.Name()), rsp.Data)
}

func TestFastHTTPSendFormData(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	type response struct {
		Message string `json:"message"`
	}
	go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
		bs := ctx.Request.Body()
		rsp := &response{Message: string(bs)}
		bs, err := json.Marshal(rsp)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return
		}
		ctx.Response.Header.SetContentType("application/json")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Write(bs)
	})

	// Start client.
	fcp := thttp.NewFastHTTPClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := make(url.Values)
	req.Add("key", "value")

	rspHead := &thttp.FastHTTPClientRspHeader{
		ManualReadBody: true,
	}
	rsp := &codec.Body{}
	require.Nil(t,
		fcp.Post(context.Background(), "/", req, rsp,
			client.WithSerializationType(codec.SerializationTypeForm),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)
	require.NotNil(t, rspHead.Response.Body())
	require.Equal(t, "{\"message\":\"key=value\"}", string(rspHead.Response.Body()))

	// Or predefine the response struct to avoid manual read.
	rsp1 := &response{}
	require.Nil(t,
		fcp.Post(context.Background(), "/", req, rsp1,
			client.WithSerializationType(codec.SerializationTypeForm),
		))
	require.NotNil(t, rsp1.Message)
}

func TestFastHTTPStreamFileUpload(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
		h, err := ctx.FormFile("field_name")
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
		}
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Write([]byte(h.Filename))
	})
	// Start client.
	fcp := thttp.NewFastHTTPClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	// Open and read file.
	fileDir, err := os.Getwd()
	require.Nil(t, err)
	fileName := "README.md"
	filePath := path.Join(fileDir, fileName)
	file, err := os.Open(filePath)
	require.Nil(t, err)
	defer file.Close()

	// Construct multipart form file.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("field_name", filepath.Base(file.Name()))
	require.Nil(t, err)
	io.Copy(part, file)
	require.Nil(t, writer.Close())

	// Add multipart form data header.
	header := http.Header{}
	header.Add("Content-Type", writer.FormDataContentType())
	reqHeader := &thttp.FastHTTPClientReqHeader{
		Method: fasthttp.MethodPost,
		// set by DecorateRequest
		DecorateRequest: func(r *fasthttp.Request) *fasthttp.Request {
			r.Header.SetContentType(writer.FormDataContentType())
			r.SetBodyStream(body, -1)
			return r
		},
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	// Upload file.
	require.Nil(t,
		fcp.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHeader),
		))
	require.Equal(t, []byte(fileName), rsp.Data)
}

func TestFastHTTPStreamRead(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
		fasthttp.ServeFile(ctx, "./README.md")
	})

	fcp := thttp.NewFastHTTPClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	rspHead := &thttp.FastHTTPClientRspHeader{ManualReadBody: true}
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		fcp.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithRspHead(rspHead),
		),
	)
	require.Nil(t, rsp.Data)
	require.NotNil(t, rspHead.Response.Body())
}

func TestFastHTTPSendReceiveChunk(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
		b := make([]byte, len(ctx.Request.Body()))
		copy(b, ctx.Request.Body())
		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			// 3. Server reads chunks.
			// io.ReadAll will read until io.EOF.
			// fasthttp will automatically handle chunked body reads.
			w.Write(b)

			// 4. Server sends chunks.
			for i := 0; i < 10; i++ {
				fmt.Fprintf(w, "this is a rsp number %d\n", i)
				time.Sleep(100 * time.Millisecond)
			}
			// Do not forget flushing streamed data.
			if err := w.Flush(); err != nil {
				return
			}
		})
	})

	// Start client.
	fcp := thttp.NewFastHTTPClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	// 1. Client sends chunks.
	reqHead := &thttp.FastHTTPClientReqHeader{
		Method: fasthttp.MethodPost,
		DecorateRequest: func(r *fasthttp.Request) *fasthttp.Request {
			r.Header.SetContentType("text/plain")
			r.SetBodyStreamWriter(func(w *bufio.Writer) {
				for i := 0; i < 10; i++ {
					fmt.Fprintf(w, "this is a req number %d\n", i)
					time.Sleep(100 * time.Millisecond)
				}
				// Do not forget flushing streamed data.
				if err := w.Flush(); err != nil {
					return
				}
			})
			return r
		},
	}
	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.FastHTTPClientRspHeader{
		ManualReadBody: true,
	}

	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		fcp.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHead),
			client.WithRspHead(rspHead),
		),
	)
	require.Nil(t, rsp.Data)
	// 2. Client reads chunks.
	t.Log(string(rspHead.Response.Body()))
	require.Equal(t, "chunked", string(reqHead.Request.Header.Peek("Transfer-Encoding")))
	require.Equal(t, "chunked", string(rspHead.Response.Header.Peek("Transfer-Encoding")))
}

func TestFastHTTPTimeoutHandler(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	s := server.New(
		server.WithServiceName("trpc.app.server.Service_http"),
		server.WithListener(ln),
		server.WithProtocol(protocol.FastHTTPNoProtocol))
	defer s.Close(nil)
	const timeout = 50 * time.Millisecond
	path := "/" + t.Name()
	thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	defer func() {
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	thttp.FastHTTPHandleFunc(path, fasthttp.TimeoutHandler(func(ctx *fasthttp.RequestCtx) {
		time.Sleep(time.Second)
	}, timeout, "timeout"))
	thttp.RegisterNoProtocolService(s)
	go s.Serve()

	// Start client.
	c := thttp.NewFastHTTPClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	req := &codec.Body{}
	rsp := &codec.Body{}
	err := c.Post(context.Background(), path, req, rsp,
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
	)
	require.NotNil(t, err)
	require.Contains(t, fmt.Sprint(err), "timeout", "expect err is timeout err, got: %s", err)
}

func TestFastHTTPClientReqRspDifferentContentType(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.FastHTTPNoProtocol),
		server.WithListener(ln),
	)
	const (
		hello = "hello "
		key   = "key"
	)
	pattern := "/" + t.Name()
	thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	defer func() {
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	thttp.FastHTTPHandleFunc(pattern, func(ctx *fasthttp.RequestCtx) {
		req, err := url.ParseQuery(string(ctx.Request.Body()))
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}
		rsp := &helloworld.HelloReply{Message: hello + req.Get(key)}
		bs, err := codec.Marshal(codec.SerializationTypePB, rsp)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return
		}
		ctx.Response.Header.SetContentType("application/protobuf")
		ctx.Write(bs)
	})

	thttp.RegisterNoProtocolService(service)
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	fcp := thttp.NewFastHTTPClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := make(url.Values)
	req.Add(key, t.Name())
	rsp := &helloworld.HelloReply{}
	require.Nil(t,
		fcp.Post(context.Background(), pattern, req, rsp,
			client.WithSerializationType(codec.SerializationTypeForm),
		))
	require.Equal(t, hello+t.Name(), rsp.Message)
}

func TestFastHTTPProxy(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()

	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.FastHTTPNoProtocol),
		server.WithListener(ln),
	)
	pattern := "/" + t.Name()
	thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	defer func() {
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	thttp.FastHTTPHandleFunc(pattern, func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.SetContentType("application/json")
		ctx.Write(ctx.Request.Body())
	})
	thttp.RegisterNoProtocolService(service)
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(1000 * time.Millisecond)

	// Start client.
	c := thttp.NewFastHTTPClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	type request struct {
		Message string `json:"message"`
	}
	data := "hello"
	bs, err := json.Marshal(&request{Message: data})
	require.Nil(t, err)
	req := &codec.Body{Data: bs}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeJSON),
		))
	require.Equal(t, bs, rsp.Data)

	// Example of client-side streaming reads for proxy.

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.FastHTTPClientRspHeader{
		ManualReadBody: true,
	}
	req = &codec.Body{Data: bs}
	rsp = &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)
	require.Equal(t, bs, rspHead.Response.Body())
}

type mockFastHTTPClientTransport struct {
}

func (ct *mockFastHTTPClientTransport) RoundTrip(ctx context.Context, req []byte, opts ...transport.RoundTripOption) (rsp []byte, err error) {
	msg := codec.Message(ctx)
	fasthttpRsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(fasthttpRsp)
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: fasthttpRsp})
	rAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	if err != nil {
		return nil, err
	}
	msg.WithRemoteAddr(rAddr)
	return []byte("mock fasthttp client transport"), nil
}

func TestFastHTTPGotConnectionRemoteAddr(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		fcp := thttp.NewFastHTTPClientProxy(t.Name(),
			client.WithTarget("dns://new.qq.com/"),
			client.WithTransport(&mockFastHTTPClientTransport{}),
		)
		rsp := &codec.Body{}
		require.Nil(t, fcp.Get(ctx, "/", rsp,
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithFilter(
				func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
					err := next(ctx, req, rsp)
					msg := codec.Message(ctx)
					addr := msg.RemoteAddr()
					require.NotNil(t, addr, "expect to get remote addr from msg in connection reuse case")
					t.Logf("addr = %+v\n", addr)
					return err
				},
			),
		))
	}
}

func TestPOSTOnlyForFastHTTPRPC(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	defer func() {
		thttp.DefaultFastHTTPServerCodec.POSTOnly = false
	}()
	thttp.DefaultFastHTTPServerCodec.POSTOnly = true
	s := server.New(
		server.WithProtocol(protocol.FastHTTP),
		server.WithListener(ln),
	)
	helloworld.RegisterGreeterService(s, &greeterServerImpl{})
	go s.Serve()
	defer s.Close(nil)

	url := fmt.Sprintf("http://%s%s", ln.Addr(), "/trpc.examples.restful.helloworld.Greeter/SayHello")
	// Perform a test for stdhttp.
	rsp, err := http.Get(url)
	require.Nil(t, err)
	require.Equal(t, fasthttp.StatusBadRequest, rsp.StatusCode)

	// Perform a test for fasthttp.
	fasthttpReq := fasthttp.AcquireRequest()
	fasthttpRsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fasthttpReq)
	defer fasthttp.ReleaseResponse(fasthttpRsp)

	fasthttpReq.SetRequestURI(url)
	fasthttpReq.Header.SetMethod(fasthttp.MethodGet)
	err = fasthttp.Do(fasthttpReq, fasthttpRsp)
	require.Nil(t, err)
	require.Equal(t, fasthttp.StatusBadRequest, fasthttpRsp.StatusCode())
	require.Contains(t,
		string(fasthttpRsp.Header.Peek("trpc-error-msg")),
		"server codec only allows POST method request, the current method is GET")
}
