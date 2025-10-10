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

// https certificate file generation method:
// 1. ca certificate:
// openssl genrsa -out ca.key 2048
// openssl req -x509 -new -nodes -key ca.key -subj "/CN=*" -days 5000 -out ca.pem
// 2. server certificate:
// openssl genrsa -out server.key 2048
// openssl req -new -key server.key -subj "/CN=*" -out server.csr
// openssl x509 -req -in server.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out server.crt -days 5000 <(printf "subjectAltName=DNS:localhost")
// 3. client certificate:
// openssl genrsa -out client.key 2048
// openssl req -new -key client.key -subj "/CN=*" -out client.csr
// openssl x509 -req -in client.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out client.crt -days 5000 <(printf "subjectAltName=DNS:localhost")
// 4. show certificate content:
// openssl x509 -text -in server.crt -noout

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/pool/httppool"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
	"trpc.group/trpc-go/trpc-go/transport"
)

const (
	tlsFileSeparator = ":"
	serverCert       = "../testdata/server.crt"
	serverKey        = "../testdata/server.key"
	clientCert       = "../testdata/client.crt"
	clientKey        = "../testdata/client.key"
	notExistCert     = "not_exist.crt"
	notExistKey      = "not_exist.key"
	caPem            = "../testdata/ca.pem"
	notExistPem      = "not_exist.pem"
)

func TestStartServer(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultServerTransport
	ln := mustListen(t)
	defer ln.Close()
	option := transport.WithListener(ln)
	handler := transport.WithHandler(&h{})
	require.Nil(t, tp.ListenAndServe(ctx, option, handler), "Failed to new client transport")
	require.NotNil(t, tp.ListenAndServe(ctx, transport.WithListenAddress("127.0.0.1:8888"), handler, transport.WithListenNetwork("tcp1")))
	t.Run("invalid tls", func(t *testing.T) {
		// CACertFile not exist.
		invalidTLS := transport.WithServeTLS(serverCert, serverKey, notExistPem)
		require.NotNil(t, tp.ListenAndServe(ctx, option, handler, invalidTLS))
		invalidTLS = transport.WithServeTLS(serverCert, serverKey,
			strings.Join([]string{caPem, notExistPem}, tlsFileSeparator))
		require.NotNil(t, tp.ListenAndServe(ctx, option, handler, invalidTLS))
		// Cert or key files not exist.
		invalidTLS = transport.WithServeTLS(notExistCert, serverKey, caPem)
		require.NotNil(t, tp.ListenAndServe(ctx, option, handler, invalidTLS))
		invalidTLS = transport.WithServeTLS(serverCert, notExistKey, caPem)
		require.NotNil(t, tp.ListenAndServe(ctx, option, handler, invalidTLS))
		invalidTLS = transport.WithServeTLS(notExistCert, notExistKey, caPem)
		require.NotNil(t, tp.ListenAndServe(ctx, option, handler, invalidTLS))
		// Invalid cert and key files length.
		invalidTLS = transport.WithServeTLS(
			strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
			strings.Join([]string{serverKey}, tlsFileSeparator),
			caPem)
		require.NotNil(t, tp.ListenAndServe(ctx, option, handler, invalidTLS))
		// Cert and key files not exist.
		invalidTLS = transport.WithServeTLS(
			strings.Join([]string{serverCert, notExistCert}, tlsFileSeparator),
			strings.Join([]string{serverKey, notExistKey}, tlsFileSeparator),
			caPem)
		require.NotNil(t, tp.ListenAndServe(ctx, option, handler, invalidTLS))
	})

	t.Run("valid tls", func(t *testing.T) {
		// Empty CACertFile.
		invalidTLS := transport.WithServeTLS(serverCert, serverKey, "")
		require.Nil(t, tp.ListenAndServe(ctx, option, handler, invalidTLS))
		// Normal single CACertFile.
		validTLS := transport.WithServeTLS(serverCert, serverKey, caPem)
		require.Nil(t, tp.ListenAndServe(ctx, option, handler, validTLS))
		// Normal multiple CACertFiles.
		validTLS = transport.WithServeTLS(serverCert, serverKey,
			strings.Join([]string{caPem, caPem}, tlsFileSeparator))
		require.Nil(t, tp.ListenAndServe(ctx, option, handler, validTLS))
		// Single CACertFile and multiple cert and key files.
		validTLS = transport.WithServeTLS(
			strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
			strings.Join([]string{serverKey, serverKey}, tlsFileSeparator),
			caPem)
		require.Nil(t, tp.ListenAndServe(ctx, option, handler, validTLS))
		// Multiple CACertFiles and multiple cert and key files.
		validTLS = transport.WithServeTLS(
			strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
			strings.Join([]string{serverKey, serverKey}, tlsFileSeparator),
			strings.Join([]string{caPem, caPem}, tlsFileSeparator))
		require.Nil(t, tp.ListenAndServe(ctx, option, handler, validTLS))
	})

}

func TestH2C(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer ln.Close()
	handler := transport.WithHandler(&h{})
	tp := thttp.NewServerTransport(transport.WithReusePort(true), transport.WithEnableH2C(true))
	require.Nil(t, tp.ListenAndServe(ctx, transport.WithListener(ln), handler))
}

func TestDisableReusePort(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(transport.WithReusePort(false))
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
	tls := transport.WithServeTLS(serverCert, serverKey, "")
	require.Nil(t, tp.ListenAndServe(ctx, option, handler, tls))

	ln3 := mustListen(t)
	defer ln3.Close()
	option = transport.WithListener(ln3)
	tls = transport.WithServeTLS(serverCert, serverKey, "root")
	require.Nil(t, tp.ListenAndServe(ctx, option, handler, tls))

	ln4 := mustListen(t)
	defer ln4.Close()
	option = transport.WithListener(ln4)
	tls = transport.WithServeTLS(serverCert, serverKey, "../testdata/ca.key")
	require.NotNil(t, tp.ListenAndServe(ctx, option, handler, tls))
}

func TestStartServerWithNoHandler(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultServerTransport
	ln := mustListen(t)
	defer ln.Close()
	option := transport.WithListener(ln)
	require.NotNil(t, tp.ListenAndServe(ctx, option), "http server transport handler empty")
}

func TestErrHandler(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultServerTransport
	ln := mustListen(t)
	defer ln.Close()
	option := transport.WithListener(ln)
	h := transport.WithHandler(transport.Handler(&errHandler{}))
	require.Nil(t, tp.ListenAndServe(ctx, option, h))

	ct := thttp.NewClientTransport(true)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestErrHeaderHandler(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultServerTransport
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	err := tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&errHeaderHandler{})),
		transport.WithListener(ln),
	)
	require.Nil(t, err)

	ct := thttp.NewClientTransport(true)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestListenAndServeFailedDueToBadCertificationFile(t *testing.T) {
	ctx := context.Background()
	oldLogger := log.DefaultLogger
	defer func() {
		log.DefaultLogger = oldLogger
	}()
	errorCh := make(chan error)
	log.DefaultLogger = &testLog{Logger: oldLogger, errorCh: errorCh}

	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	require.NotNil(
		t,
		thttp.DefaultServerTransport.ListenAndServe(
			ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(notExistCert, serverKey, ""),
		),
		"failed to new client transport",
	)
}

func TestStartTLSServerAndNoCheckServer(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	// Only enables https server and do not verify client certificate.
	require.Nil(
		t,
		thttp.DefaultServerTransport.ListenAndServe(
			ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(serverCert, serverKey, ""),
		),
		"Failed to new client transport",
	)

	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

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

func TestServerWithListenerOption(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer ln.Close()
	service := server.New(
		server.WithServiceName("trpc.http.server.ListenerTest"),
		server.WithProtocol(protocol.HTTP),
		server.WithListener(ln),
	)
	thttp.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Printf("Protocol: %s\n", r.Proto)
		w.Write([]byte(r.Proto))
		return nil
	})
	thttp.RegisterDefaultService(service)
	s := &server.Server{}
	s.AddService("trpc.http.server.ListenerTest", service)
	go func() {
		require.Nil(t, s.Serve())
	}()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%v/index", ln.Addr()))
	require.Nil(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	require.Equal(t, []byte("HTTP/1.1"), body)

	const invalidAddr = "localhost:910439"
	resp, err = http.Get(fmt.Sprintf("http://%s/index", invalidAddr))
	require.NotNil(t, err)
	require.Nil(t, resp)
}

func TestStartDisableKeepAlivesServer(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer ln.Close()
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.http.server.ListenerTest"),
		server.WithProtocol(protocol.HTTP),
		server.WithTransport(
			thttp.NewServerTransport(transport.WithReusePort(true)),
		),
		server.WithDisableKeepAlives(true),
	)
	thttp.HandleFunc("/disable-keepalives", func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set(thttp.Connection, "keep-alive")
		return nil
	})
	thttp.RegisterDefaultService(service)
	s.AddService("trpc.http.server.ListenerTest", service)
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
	url := fmt.Sprintf("http://%s/disable-keepalives", ln.Addr())
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

func TestStartH2cServer(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer ln.Close()
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.h2c.server.Greeter"),
		server.WithProtocol(protocol.HTTP2),
		server.WithTransport(thttp.NewServerTransport(transport.WithReusePort(true),
			transport.WithEnableH2C(true))),
	)
	thttp.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Printf("Protocol: %s\n", r.Proto)
		w.Write([]byte(r.Proto))
		return nil
	})
	thttp.HandleFunc("/main", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Printf("Protocol: %s\n", r.Proto)
		w.Write([]byte(r.Proto))
		return nil
	})
	thttp.RegisterDefaultService(service)
	s.AddService("trpc.h2c.server.Greeter", service)

	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// h2c client
	h2cClient := http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
	url := fmt.Sprintf("http://%s/", ln.Addr())
	resp, err := h2cClient.Get(url + "main")
	require.Nil(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	require.Equal(t, []byte("HTTP/2.0"), body)

	// http1 client
	resp2, err := http.Get(url)
	require.Nil(t, err)
	defer resp2.Body.Close()
	body, err = io.ReadAll(resp2.Body)
	require.Nil(t, err)
	require.Equal(t, []byte("HTTP/1.1"), body)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
}

func TestHttp2StartTLSServerAndNoCheckServer(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	// Only enables https server and do not verify client certificate.
	require.Nil(
		t,
		thttp.DefaultServerTransport.ListenAndServe(
			ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(serverCert, serverKey, ""),
		),
		"Failed to new client transport",
	)

	ct := thttp.NewClientTransport(true)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

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

func TestStartTLSServerAndCheckServer(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultServerTransport
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	err := tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&h{})),
		// Only enables https server and do not verify client certificate.
		transport.WithServeTLS(serverCert, serverKey, ""),
		transport.WithListener(ln),
	)
	require.Nil(t, err, "Failed to new client transport")

	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// Uses ca public key to verify server certificate.
		transport.WithDialTLS("", "", caPem, "localhost"),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndCheckClientNoCert(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultServerTransport
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	err := tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&h{})),
		// Enables two-way authentication http server and need to verify client certificate.
		transport.WithServeTLS(serverCert, serverKey, caPem),
		transport.WithListener(ln),
	)
	require.Nil(t, err, "Failed to new client transport")

	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	_, err = ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// If the client's own certificate is not sent, will return TLS verification failed.
		transport.WithDialTLS("", "", caPem, "localhost"),
	)
	require.NotNil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndCheckClient(t *testing.T) {
	ctx := context.Background()
	tp := thttp.DefaultServerTransport
	ln := mustListen(t)
	defer func() { require.Nil(t, ln.Close()) }()
	// Enables two-way authentication http server and need to verify client certificate.
	err := tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&h{})),
		// Only enables https server and do not verify client certificate.
		transport.WithServeTLS(serverCert, serverKey, caPem),
		transport.WithListener(ln),
	)
	require.Nil(t, err, "Failed to new client transport")

	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// Need to send the client's own certificate to server.
		transport.WithDialTLS(serverCert, serverKey, caPem, "localhost"),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestNewClientTransport(t *testing.T) {
	ct := thttp.NewClientTransport(false)
	require.NotNil(t, ct, "Failed to new client transport")

	ct2 := thttp.NewClientTransport(true)
	require.NotNil(t, ct2, "Failed to new http2 client transport")
}

func TestNewClientTransportWithOption(t *testing.T) {
	opt := transport.WithClientUDPRecvSize(65536)
	ct := thttp.NewClientTransport(false, opt)
	require.NotNil(t, ct, "client transport option not empty")
}

func TestClientRoundTrip(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})
	ln := mustListen(t)
	defer ln.Close()
	go http.Serve(ln, nil)
	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()))
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestClientRoundTripWithNoHead(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress("127.0.0.1:18080"))
	require.Nil(t, rsp, "no head roundtrip rsp not empty")
	require.NotNil(t, err, "no head roundtrip err nil")

}

func TestClientWithSelectorNode(t *testing.T) {
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
		tp := thttp.NewServerTransport(transport.WithReusePort(false))
		option := transport.WithListener(tt.listener)
		handler := transport.WithHandler(transport.Handler(&h{}))
		err := tp.ListenAndServe(ctx, option, handler)
		require.Nil(t, err, "Failed to new client transport")

		proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
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

func TestClient(t *testing.T) {
	ctx := context.Background()
	old := codec.GetSerializer(codec.SerializationTypeJSON)
	defer func() { codec.RegisterSerializer(codec.SerializationTypeJSON, old) }()
	codec.RegisterSerializer(codec.SerializationTypeJSON, &codec.JSONPBSerialization{})
	tp := thttp.NewServerTransport(transport.WithReusePort(false))
	ln := mustListen(t)
	defer ln.Close()
	option := transport.WithListener(ln)
	handler := transport.WithHandler(transport.Handler(&h{}))
	require.Nil(t, tp.ListenAndServe(ctx, option, handler), "Failed to new client transport")

	header := &thttp.ClientReqHeader{}
	header.AddHeader("ContentType", "application/json")

	proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithReqHead(header),
		client.WithMetaData("k1", []byte("v1")),
	)
	reqBody := &codec.Body{
		Data: []byte("{\"username\":\"xyz\"," +
			"\"password\":\"xyz\",\"from\":\"xyz\"}"),
	}
	rspBody := &codec.Body{}

	require.Nil(t, proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody), "Failed to post")
	require.Nil(t, proxy.Put(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody), "Failed to put")
	require.Nil(t, proxy.Delete(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody), "Failed to delete")
	require.Nil(t, proxy.Get(ctx, "/trpc.test.helloworld.Greeter/SayHello", rspBody), "Failed to get")
	require.Nil(t, proxy.Patch(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody), "Failed to patch")

	// Test client with options.
	proxy = thttp.NewClientProxy("trpc.test.helloworld.Greeter")
	reqBody = &codec.Body{
		Data: []byte("{\"username\":\"xyz\"," +
			"\"password\":\"xyz\",\"from\":\"xyz\"}"),
	}
	rspBody = &codec.Body{}
	require.Nil(t,
		proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody,
			client.WithTarget("ip://"+ln.Addr().String()),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithReqHead(header),
			client.WithMetaData("k1", []byte("v1")),
		), "Failed to post")

	require.NotNil(t,
		proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody,
			client.WithTarget("ip://127.0.0.1:180"),
		), "Failed to post")
}

func TestReqHeader(t *testing.T) {
	ctx := context.Background()
	// Invalid url.
	header := &thttp.ClientReqHeader{}
	header.AddHeader("Content-Type", "application/json")
	proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://127.0.0.1:18080:www.baidu.com//"),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithReqHead(header),
	)
	reqBody := &codec.Body{}
	rspBody := &codec.Body{}
	err := proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody)
	require.NotNil(t, err)
}

func TestReqHeaderWithContentType(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer ln.Close()
	option := transport.WithListener(ln)
	handler := transport.WithHandler(transport.Handler(&h{}))
	tp := thttp.NewServerTransport(transport.WithReusePort(false))
	require.Nil(t, tp.ListenAndServe(ctx, option, handler), "Failed to new client transport")
	var tests = []struct {
		expected string
	}{
		{"application/json"},
		{"application/jsonp"},
		{"application/jsonp123"},
		{"application/text123"},
	}
	for _, tt := range tests {
		header := &thttp.ClientReqHeader{}
		header.AddHeader("Content-Type", tt.expected)
		proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
			client.WithTarget("ip://"+ln.Addr().String()),
			client.WithSerializationType(codec.SerializationTypeForm),
			client.WithReqHead(header),
		)
		reqBody := &codec.Body{}
		rspBody := &codec.Body{}
		err := proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody)
		require.Nil(t, err)
	}
}

func TestHandler(t *testing.T) {
	var (
		handler     = func(w http.ResponseWriter, r *http.Request) {}
		handlerFunc = func(w http.ResponseWriter, r *http.Request) error {
			return nil
		}
		service = server.New(server.WithProtocol(protocol.HTTP))
	)

	thttp.Handle("*", http.HandlerFunc(handler))
	thttp.HandleFunc("/path/do/not/equal/to/*", handlerFunc)
	thttp.RegisterDefaultService(service)

	for _, method := range thttp.ServiceDesc.Methods {
		method.Func(nil, context.TODO(), func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), nil
		})

		method.Func(nil, context.TODO(), func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), errors.New("invalid filter")
		})

		header := &thttp.Header{
			Request:  &http.Request{},
			Response: &httptest.ResponseRecorder{},
		}
		ctx := thttp.WithHeader(context.TODO(), header)
		_, err := method.Func(nil, ctx, func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), nil
		})
		require.Nil(t, err)
	}
}

func TestMux(t *testing.T) {
	var handler = func(w http.ResponseWriter, r *http.Request) {}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	var service = &mockService{}
	thttp.RegisterServiceMux(service, mux)
	desc, _ := service.desc.(*server.ServiceDesc)
	for _, method := range desc.Methods {
		method.Func(nil, context.TODO(), func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), nil
		})

		method.Func(nil, context.TODO(), func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), errors.New("invalid filter")
		})

		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		header := &thttp.Header{
			Request:  req,
			Response: &httptest.ResponseRecorder{},
		}
		ctx := thttp.WithHeader(context.TODO(), header)
		_, err := method.Func(nil, ctx, func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), nil
		})
		require.Nil(t, err)
	}
}

// TestCheckRedirect tests set CheckRedirect
func TestCheckRedirect(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer ln.Close()
	// server
	go func() {
		// real backend
		h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("real"))
		})
		http.Handle("/real", h)

		// redirect a
		rha := http.RedirectHandler("/b", http.StatusMovedPermanently)
		http.Handle("/a", rha)

		// redirect b
		rhb := http.RedirectHandler("/real", http.StatusMovedPermanently)
		http.Handle("/b", rhb)

		http.Serve(ln, nil)
	}()
	time.Sleep(200 * time.Millisecond)

	// sets CheckRedirect
	checkRedirect := func(_ *http.Request, via []*http.Request) error {
		if len(via) > 1 {
			return errors.New("more than once")
		}
		return nil
	}
	thttp.DefaultClientTransport.(*thttp.ClientTransport).CheckRedirect = checkRedirect
	defer func() {
		thttp.DefaultClientTransport.(*thttp.ClientTransport).CheckRedirect = nil
	}()
	proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithSerializationType(codec.SerializationTypeNoop),
	)
	reqBody := &codec.Body{}
	rspBody := &codec.Body{}
	// only redirect once form /b
	require.Nil(t, proxy.Post(ctx, "/b", reqBody, rspBody))
	// redirect twice from /a
	err := proxy.Post(ctx, "/a", reqBody, rspBody)
	require.NotNil(t, err)
	require.True(t, strings.Contains(err.Error(), "more than once"))
}

func TestTransportError(t *testing.T) {
	http.HandleFunc("/timeout", func(http.ResponseWriter, *http.Request) {
		time.Sleep(time.Second)
	})
	http.HandleFunc("/cancel", func(http.ResponseWriter, *http.Request) {})
	ln := mustListen(t)
	defer ln.Close()
	go func() { http.Serve(ln, nil) }()
	time.Sleep(200 * time.Millisecond)

	proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithTimeout(time.Millisecond*500),
	)
	rspBody := &codec.Body{}

	err := proxy.Get(context.Background(), "/timeout", rspBody)
	terr, ok := err.(*errs.Error)
	require.True(t, ok)
	require.Equal(t, terr.Code, int32(errs.RetClientTimeout))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = proxy.Get(ctx, "/cancel", rspBody)
	terr, ok = err.(*errs.Error)
	require.True(t, ok)
	require.Equal(t, terr.Code, int32(errs.RetClientCanceled))
}

func TestClientRoundDyeing(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithDyeing(true)
	dyeingKey := "dyeingKey"
	msg.WithDyeingKey(dyeingKey)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	req := &http.Request{
		Header: http.Header{},
	}
	reqHeader := &thttp.ClientReqHeader{
		Request: req,
	}
	msg.WithClientReqHead(reqHeader)
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	meta := codec.MetaData{
		thttp.TrpcDyeingKey: []byte(dyeingKey),
	}
	msg.WithClientMetaData(meta)
	_, err := ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Equal(t,
		strconv.Itoa(int(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)), req.Header.Get(thttp.TrpcMessageType))
}

func TestClientRoundEnvTransfer(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithEnvTransfer("feat,master")
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	req := &http.Request{
		Header: http.Header{},
	}
	reqHeader := &thttp.ClientReqHeader{
		Request: req,
	}
	msg.WithClientReqHead(reqHeader)
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	_, err := ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Contains(t, req.Header.Get(thttp.TrpcTransInfo), thttp.TrpcEnv)
}

func TestDisableBase64EncodeTransInfo(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false, transport.WithDisableEncodeTransInfoBase64())
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
	req := &http.Request{
		Header: http.Header{},
	}
	reqHeader := &thttp.ClientReqHeader{
		Request: req,
	}
	msg.WithClientReqHead(reqHeader)
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	_, err := ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Contains(t, req.Header.Get(thttp.TrpcTransInfo), envTrans)
	require.Contains(t, req.Header.Get(thttp.TrpcTransInfo), metaVal)
	require.Contains(t, req.Header.Get(thttp.TrpcTransInfo), dyeingKey)
}

func TestDisableServiceRouterTransInfo(t *testing.T) {
	ctx := context.Background()
	a := require.New(t)
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientMetaData(codec.MetaData{thttp.TrpcEnv: []byte("orienv")}) // this emulate decode trpc protocol client request
	msg.WithEnvTransfer("feat,master")
	req := &http.Request{
		Header: http.Header{},
	}
	reqHeader := &thttp.ClientReqHeader{
		Request: req,
	}
	msg.WithClientReqHead(reqHeader)
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	_, err := ct.RoundTrip(ctx, nil)
	a.NotNil(err)
	info, err := thttp.UnmarshalTransInfo(msg, req.Header.Get(thttp.TrpcTransInfo))
	a.NoError(err)
	a.Equal(string(info[thttp.TrpcEnv]), "feat,master")

	msg.WithEnvTransfer("") // DisableServiceRouter would clear EnvTransfer
	_, err = ct.RoundTrip(ctx, nil)
	a.NotNil(err)
	info, err = thttp.UnmarshalTransInfo(msg, req.Header.Get(thttp.TrpcTransInfo))
	a.NoError(err)
	a.Equal(string(info[thttp.TrpcEnv]), "")
}

func TestHTTPSUseClientVerify(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.HTTPNoProtocol),
		server.WithListener(ln),
		server.WithTLS(
			serverCert,
			serverKey,
			caPem,
		),
	)
	pattern := "/" + t.Name()
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(t.Name()))
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
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
				clientCert,
				clientKey,
				caPem,
				"localhost",
			),
		))
	require.Equal(t, []byte(t.Name()), rsp.Data)
}

func TestHTTPSProtocolUseClientVerify(t *testing.T) {
	ln := mustListen(t)
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Log(err)
		}
	})
	serviceName := "trpc.app.server.Service" + t.Name()
	s := mustServe(t, serviceName,
		server.WithTransport(transport.NewServerTransport(transport.WithReusePort(false))),
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.HTTP),
		server.WithListener(ln),
		server.WithTLS(serverCert, serverKey, caPem),
	)
	pattern := "/" + t.Name()
	thttp.HandleFunc(pattern, func(w http.ResponseWriter, _ *http.Request) error {
		_, err := w.Write([]byte(t.Name()))
		return err
	})
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	t.Run("bad cert file", func(t *testing.T) {
		c := thttp.NewClientProxy(
			serviceName,
			client.WithTarget("ip://"+ln.Addr().String()),
		)
		req := &codec.Body{}
		rsp := &codec.Body{}
		err := c.Post(context.Background(), pattern, req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithTLS(notExistCert, serverKey, caPem, "localhost"),
		)
		require.Equal(t, errs.RetClientConnectFail, errs.Code(err))
		require.Contains(t, errs.Msg(err), "getting standard http client failed")
	})
}

func TestHTTPHeaderStamp(t *testing.T) {
	ln := mustListen(t)
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Log(err)
		}
	})
	serviceName := "trpc.app.server.Service" + t.Name()
	s := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.HTTPNoProtocol),
		server.WithListener(ln),
	)
	pattern := "/" + t.Name()
	const (
		key = "key"
		val = "val"
	)
	thttp.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) error {
		if v := r.Header.Get(key); v != val {
			return fmt.Errorf("want '%s', got '%s'", val, v)
		}
		_, err := w.Write([]byte(t.Name()))
		return err
	})
	thttp.RegisterNoProtocolService(s)
	go s.Serve()
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	reqHeader := &thttp.ClientReqHeader{}
	r, err := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+pattern, bytes.NewBuffer([]byte("")))
	require.Nil(t, err)
	r.Header.Add(key, val)
	reqHeader.Request = r
	reqHeader.AddHeader("a", "b") // This header should not overwrite the "key: val" set in the reqHeader.Request.
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t, c.Post(context.Background(), pattern, req, rsp,
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithReqHead(reqHeader),
	))
}

func TestExplicitHTTPSProtocolUseClientVerify(t *testing.T) {
	ln := mustListen(t)
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Log(err)
		}
	})
	serviceName := "trpc.app.server.Service" + t.Name()
	s := server.New(
		server.WithTransport(transport.NewServerTransport(transport.WithReusePort(false))),
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.HTTPSNoProtocol), // Explicit https.
		server.WithListener(ln),
		server.WithTLS(serverCert, serverKey, ""),
	)
	pattern := "/" + t.Name()
	thttp.HandleFunc(pattern, func(w http.ResponseWriter, _ *http.Request) error {
		_, err := w.Write([]byte(t.Name()))
		return err
	})
	thttp.RegisterNoProtocolService(s)
	go s.Serve()
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithProtocol(protocol.HTTPS), // Explicit https.
	)
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t, c.Post(context.Background(), pattern, req, rsp,
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
	))
	require.Equal(t, t.Name(), string(rsp.Data))
}

func TestHTTPSSkipClientVerify(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.HTTPNoProtocol),
		server.WithListener(ln),
		server.WithTLS(serverCert, serverKey, ""),
	)
	pattern := "/" + t.Name()
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(t.Name()))
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
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
			client.WithTLS("", "", "none", ""),
		))
	require.Equal(t, []byte(t.Name()), rsp.Data)
}

func TestListenAndServeHTTPHead(t *testing.T) {
	ctx := context.Background()
	ln := mustListen(t)
	defer ln.Close()
	st := thttp.NewServerTransport()
	require.Nil(t, st.ListenAndServe(ctx,
		transport.WithHandler(&httpHeadHandler{
			func(ctx context.Context, _ []byte) (rsp []byte, err error) {
				head := thttp.Head(ctx)
				head.Response.WriteHeader(http.StatusOK)
				head.Response.Write([]byte(fmt.Sprintf("%+v", thttp.Head(head.Request.Context()) != nil)))
				return
			}}),
		transport.WithListener(ln),
	))
	time.Sleep(200 * time.Millisecond)
	rsp, err := http.Get("http://" + ln.Addr().String())
	require.Nil(t, err)
	bs, err := io.ReadAll(rsp.Body)
	require.Nil(t, err)
	require.Equal(t, fmt.Sprintf("%+v", true), string(bs))
}

type httpHeadHandler struct {
	handle func(ctx context.Context, req []byte) (rsp []byte, err error)
}

func (h *httpHeadHandler) Handle(ctx context.Context, req []byte) (rsp []byte, err error) {
	return h.handle(ctx, req)
}

func TestHTTPSendFormData(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	type response struct {
		Message string `json:"message"`
	}
	s := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bs, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			t.Logf("server read: %q\n", bs)
			rsp := &response{Message: string(bs)}
			bs, err = json.Marshal(rsp)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(bs)
		}),
	}
	go s.Serve(ln)

	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := make(url.Values)
	req.Add("key", "value")

	// Use manual read to read response (requires trpc-go >= v0.13.0)
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true, // Requires trpc-go >= v0.13.0.
	}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithSerializationType(codec.SerializationTypeForm),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)
	body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
	defer body.Close()            // Do remember to close the body.
	bs, err := io.ReadAll(body)
	require.Nil(t, err)
	require.NotNil(t, bs)

	// Or predefine the response struct to avoid manual read.
	rsp1 := &response{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp1,
			client.WithSerializationType(codec.SerializationTypeForm),
		))
	require.NotNil(t, rsp1.Message)
	t.Logf("receive: %s\n", rsp1.Message)
}

func TestHTTPStreamFileUpload(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	go http.Serve(ln, &fileHandler{})
	// Start client.
	c := thttp.NewClientProxy(
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
	reqHeader := &thttp.ClientReqHeader{
		Method:  http.MethodPost,
		Header:  header,
		ReqBody: body, // Stream send.
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	// Upload file.
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHeader),
		))
	require.Equal(t, []byte(fileName), rsp.Data)
}

type fileHandler struct{}

func (*fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, h, err := r.FormFile("field_name")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	// Write back file name.
	w.Write([]byte(h.Filename))
}

func TestHTTPStreamRead(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	go http.Serve(ln, &fileServer{})

	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)
	body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
	defer body.Close()            // Do remember to close the body.
	bs, err := io.ReadAll(body)
	require.Nil(t, err)
	require.NotNil(t, bs)
}

func TestHTTPSendReceiveChunk(t *testing.T) {
	// HTTP chunked example:
	//   1. Client sends chunks: Add "chunked" transfer encoding header, and use io.Reader as body.
	//   2. Client reads chunks: The Go/net/http automatically handles the chunked reading.
	//                           Users can simply read resp.Body in a loop until io.EOF.
	//   3. Server reads chunks: Similar to client reads chunks.
	//   4. Server sends chunks: Assert http.ResponseWriter as http.Flusher, call flusher.Flush() after
	//         writing a part of data, it will automatically trigger "chunked" encoding to send a chunk.

	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	go http.Serve(ln, &chunkedServer{})

	// Start client.
	c := thttp.NewClientProxy(
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

	// 1. Client sends chunks.

	// Add request headers.
	header := http.Header{}
	header.Add("Content-Type", "text/plain")
	// Add chunked transfer encoding header.
	header.Add("Transfer-Encoding", "chunked")
	reqHead := &thttp.ClientReqHeader{
		Method:  http.MethodPost,
		Header:  header,
		ReqBody: file, // Stream send (for chunks).
	}

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHead),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)

	// 2. Client reads chunks.

	// Do stream reads directly from rspHead.Response.Body.
	body := rspHead.Response.Body
	defer body.Close() // Do remember to close the body.
	buf := make([]byte, 4096)
	var idx int
	for {
		n, err := body.Read(buf)
		if err == io.EOF {
			t.Logf("reached io.EOF\n")
			break
		}
		t.Logf("read chunk %d of length %d: %q\n", idx, n, buf[:n])
		idx++
	}
}

type chunkedServer struct{}

func (*chunkedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 3. Server reads chunks.

	// io.ReadAll will read until io.EOF.
	// Go/net/http will automatically handle chunked body reads.
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("io.ReadAll err: %+v", err)))
		return
	}

	// 4. Server sends chunks.

	// Send HTTP chunks using http.Flusher.
	// Reference: https://stackoverflow.com/questions/26769626/send-a-chunked-http-response-from-a-go-server.
	// The "Transfer-Encoding" header will be handled by the writer implicitly, so no need to set it.
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("expected http.ResponseWriter to be an http.Flusher"))
		return
	}
	chunks := 10
	chunkSize := (len(bs) + chunks - 1) / chunks
	for i := 0; i < chunks; i++ {
		start := i * chunkSize
		end := (i + 1) * chunkSize
		if end > len(bs) {
			end = len(bs)
		}
		w.Write(bs[start:end])
		flusher.Flush() // Trigger "chunked" encoding and send a chunk.
		time.Sleep(500 * time.Millisecond)
	}
}

func TestHTTPTimeoutHandler(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	s := server.New(
		server.WithServiceName("trpc.app.server.Service_http"),
		server.WithListener(ln),
		server.WithProtocol(protocol.HTTPNoProtocol),
	)
	defer s.Close(nil)
	const timeout = 50 * time.Millisecond
	path := "/" + t.Name()
	thttp.Handle(path, http.TimeoutHandler(&fileServer{sleep: 2 * timeout}, timeout, "timeout"))
	thttp.RegisterNoProtocolService(s)
	go s.Serve()

	// Start client.
	c := thttp.NewClientProxy(
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

type fileServer struct {
	sleep time.Duration
}

func (s *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(s.sleep)
	http.ServeFile(w, r, "./README.md")
}

func TestHTTPClientReqRspDifferentContentType(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.HTTPNoProtocol),
		server.WithListener(ln),
	)
	const (
		hello = "hello "
		key   = "key"
	)
	pattern := "/" + t.Name()
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		req, err := url.ParseQuery(string(bs))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rsp := &helloworld.HelloReply{Message: hello + req.Get(key)}
		bs, err = codec.Marshal(codec.SerializationTypePB, rsp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/protobuf")
		w.Write(bs)
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := make(url.Values)
	req.Add(key, t.Name())
	rsp := &helloworld.HelloReply{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithSerializationType(codec.SerializationTypeForm),
		))
	require.Equal(t, hello+t.Name(), rsp.Message)
}

func TestHTTPProxy(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.HTTPNoProtocol),
		server.WithListener(ln),
	)
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(bs)
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
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
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeJSON),
		))
	require.Equal(t, bs, rsp.Data)

	// Example of client-side streaming reads for proxy.

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req = &codec.Body{Data: bs}
	rsp = &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)
	body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
	defer body.Close()            // Do remember to close the body.
	result, err := io.ReadAll(body)
	require.Nil(t, err)
	require.Equal(t, bs, result)
}

func TestHTTPGotConnectionRemoteAddr(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		proxy := thttp.NewClientProxy(t.Name(),
			client.WithTarget("dns://new.qq.com/"),
			client.WithTransport(&mockTransport{}))
		rsp := &codec.Body{}
		require.Nil(t, proxy.Get(ctx, "/", rsp,
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithFilter(
				func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
					err := next(ctx, req, rsp)
					msg := codec.Message(ctx)
					addr := msg.RemoteAddr()
					require.NotNil(t, addr, "expect to get remote addr from msg in connection reuse case")
					t.Logf("addr = %+v\n", addr)
					return err
				})))
	}
}

func TestCustomizeHTTPClientTransport(t *testing.T) {
	transportMustFailErr := fmt.Errorf("%s must fail", t.Name())
	tr := thttp.NewClientTransport(false,
		transport.WithNewHTTPClientTransport(func() *http.Transport {
			return &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return nil, transportMustFailErr
				},
			}
		}))

	require.Contains(t, thttp.NewClientProxy(t.Name(), client.WithTransport(tr)).
		Get(context.Background(), "/", nil).
		Error(), transportMustFailErr.Error())

	require.Contains(t,
		thttp.NewClientProxy(
			t.Name(),
			client.WithTransport(tr),
			client.WithTLS("", "", "none", t.Name()),
		).Get(
			context.Background(), "/", nil,
		).Error(), transportMustFailErr.Error())
}

func TestPOSTOnlyForHTTPRPC(t *testing.T) {
	ln := mustListen(t)
	defer ln.Close()
	defer func() {
		thttp.DefaultServerCodec.POSTOnly = false
	}()
	thttp.DefaultServerCodec.AutoReadBody = true
	thttp.DefaultServerCodec.POSTOnly = true
	s := server.New(
		server.WithProtocol(protocol.HTTP),
		server.WithListener(ln),
	)
	helloworld.RegisterGreeterService(s, &greeterServerImpl{})
	go s.Serve()
	defer s.Close(nil)
	rsp, err := http.Get(fmt.Sprintf("http://%s%s", ln.Addr(), "/trpc.examples.restful.helloworld.Greeter/SayHello"))
	require.Nil(t, err)
	require.Equal(t, http.StatusBadRequest, rsp.StatusCode)
	require.Equal(t,
		"service codec Decode: server codec only allows POST method request, the current method is GET",
		rsp.Header.Get("trpc-error-msg"),
	)
}

func TestDecorateRequest(t *testing.T) {
	// Start server.
	ln := mustListen(t)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithProtocol(protocol.HTTPNoProtocol),
		server.WithListener(ln),
	)
	transferEncodings := make(chan []string, 1)
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		transferEncodings <- r.TransferEncoding
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Write(bs)
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	data := []byte("hello")
	reader := bytes.NewBuffer(data)
	// The first try: use a custom io.Reader and do not provide a reqHeader.DecorateRequest.
	reqHeader := &thttp.ClientReqHeader{
		ReqBody: io.LimitReader(reader, int64(len(data))),
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithReqHead(reqHeader),
		))
	require.Equal(t, data, rsp.Data)
	encoding := <-transferEncodings
	// If reqHeader.DecorateRequest is not used to modify the content length,
	// the request will be sent with chunked encoding.
	require.Contains(t, encoding, "chunked")

	// The second try: still use a custom io.Reader, but provide a reqHeader.DecorateRequest to
	// set the content length.
	reader = bytes.NewBuffer(data)
	reqHeader = &thttp.ClientReqHeader{
		ReqBody: io.LimitReader(reader, int64(len(data))),
		DecorateRequest: func(r *http.Request) *http.Request {
			r.ContentLength = int64(len(data))
			return r
		},
	}
	req = &codec.Body{}
	rsp = &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithReqHead(reqHeader),
		))
	require.Equal(t, data, rsp.Data)
	encoding = <-transferEncodings
	// If reqHeader.DecorateRequest is used to modify the content length,
	// the request will not be sent with chunked encoding.
	require.NotContains(t, encoding, "chunked")
}

func TestClientHTTPPool(t *testing.T) {
	defaultNewRoundTripper := thttp.NewRoundTripper
	thttp.NewRoundTripper = func(r http.RoundTripper) http.RoundTripper {
		return r
	}
	defer func() {
		thttp.NewRoundTripper = defaultNewRoundTripper
	}()

	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})
	ln := mustListen(t)
	defer ln.Close()
	go http.Serve(ln, nil)

	httpOpts := transport.HTTPRoundTripOptions{
		Pool: httppool.Options{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     20,
			IdleConnTimeout:     time.Second,
		},
	}
	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		transport.WithHTTPRoundTripOptions(httpOpts))
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")

	clientTransport, ok := ct.(*thttp.ClientTransport)
	require.True(t, ok)
	httpTransport, ok := clientTransport.Client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, 100, httpTransport.MaxIdleConns)
	require.Equal(t, 10, httpTransport.MaxIdleConnsPerHost)
	require.Equal(t, 20, httpTransport.MaxConnsPerHost)
	require.Equal(t, time.Second, httpTransport.IdleConnTimeout)
}

func TestHTTPClientMisMatchHead(t *testing.T) {
	ctx, msg := codec.WithNewMessage(context.Background())
	ct := thttp.NewClientTransport(false)
	msg.WithClientReqHead(&thttp.ClientRspHeader{})
	_, err := ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "ReqHead should be type of *http.ClientReqHeader")

	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientReqHeader{})
	_, err = ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "RspHead should be type of *http.ClientRspHeader")
}

type h struct{}

func (*h) Handle(ctx context.Context, reqBuf []byte) (rsp []byte, err error) {
	fmt.Println("recv http req")
	return nil, nil
}

type testLog struct {
	log.Logger
	errorCh chan error
}

func (ln *testLog) Errorf(format string, args ...interface{}) {
	ln.errorCh <- fmt.Errorf(format, args...)
}

// mockService is a mock service.
type mockService struct {
	desc interface{}
}

// Register registers route information.
func (m *mockService) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	m.desc = serviceDesc
	return nil
}

// Serve runs service.
func (m *mockService) Serve() error {
	return nil
}

// Close closes service.
func (m *mockService) Close(chan struct{}) error {
	return nil
}

type errHandler struct{}

func (*errHandler) Handle(ctx context.Context, reqBuf []byte) (rsp []byte, err error) {
	return nil, errors.New("mock error")
}

type errHeaderHandler struct{}

func (*errHeaderHandler) Handle(ctx context.Context, reqBuf []byte) (rsp []byte, err error) {
	return nil, thttp.ErrEncodeMissingHeader
}

type mockTransport struct {
}

func (t *mockTransport) RoundTrip(ctx context.Context, req []byte, opts ...transport.RoundTripOption) (rsp []byte, err error) {
	msg := codec.Message(ctx)
	msg.WithClientRspHead(&thttp.ClientRspHeader{
		Response: &http.Response{},
	})
	rAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	if err != nil {
		return nil, err
	}
	msg.WithRemoteAddr(rAddr)
	return []byte("mock transport"), nil
}

func randomListener() (net.Listener, error) {
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	return net.Listen("tcp", "127.0.0.1:0")
}

func mustServe(t *testing.T, serviceName string, option ...server.Option) *server.Server {
	t.Helper()
	s := &server.Server{}
	s.AddService(serviceName, server.New(option...))
	go s.Serve()
	time.Sleep(100 * time.Millisecond)
	return s
}

func mustListen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := randomListener()
	if err != nil {
		t.Fatal(err)
	}
	return ln
}
