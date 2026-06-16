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
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestFastHTTPServiceDesc(t *testing.T) {
	defer func() { ServiceDesc.Methods = ServiceDesc.Methods[:0] }()

	requestCtx := &fasthttp.RequestCtx{}
	baseCtx := context.Background()
	requestCtx.SetUserValue(CtxKey{}, baseCtx)
	gotCtx, ok := GetContext(requestCtx)
	require.True(t, ok)
	require.Equal(t, baseCtx, gotCtx)

	requestCtx.SetUserValue(CtxKey{}, "invalid")
	gotCtx, ok = GetContext(requestCtx)
	require.False(t, ok)
	require.Nil(t, gotCtx)

	called := false
	FastHTTPHandleFunc("/fast", func(ctx *fasthttp.RequestCtx) {
		called = true
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	})
	require.Len(t, ServiceDesc.Methods, 1)

	ctx := WithRequestCtx(context.Background(), requestCtx)
	_, err := ServiceDesc.Methods[0].Func(nil, ctx, func(interface{}) (filter.ServerChain, error) {
		return filter.ServerChain{}, nil
	})
	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, fasthttp.StatusNoContent, requestCtx.Response.StatusCode())

	_, err = ServiceDesc.Methods[0].Func(nil, context.Background(), func(interface{}) (filter.ServerChain, error) {
		return filter.ServerChain{}, nil
	})
	require.ErrorContains(t, err, "missing requestCtx")

	_, err = ServiceDesc.Methods[0].Func(nil, ctx, func(interface{}) (filter.ServerChain, error) {
		return nil, errors.New("filter error")
	})
	require.ErrorContains(t, err, "filter error")
}

func TestFastHTTPTransportPrivateHelpers(t *testing.T) {
	require.Equal(t, "https://example.com/path", buildRequestURI("https", "example.com", "/path"))
	require.Equal(t, "custom", inferScheme("custom", ""))
	require.Equal(t, "https", inferScheme("", "ca.pem"))
	require.Equal(t, "http", inferScheme("", ""))
	require.Equal(t, "value", encodeBytes([]byte("value"), true))
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("value")), encodeBytes([]byte("value"), false))
	require.Equal(t, "value", encodeString("value", true))
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("value")), encodeString("value", false))
}

func TestFastHTTPCheckRequest(t *testing.T) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.Header.SetMethod(fasthttp.MethodGet)
	require.ErrorContains(t, checkRequest(req), "URL host cannot be empty")

	req.SetRequestURI("ftp://example.com/path")
	require.ErrorContains(t, checkRequest(req), "unsupported URL scheme")

	req.SetRequestURI("http:///path")
	require.ErrorContains(t, checkRequest(req), "URL host cannot be empty")

	req.SetRequestURI("http://example.com/path")
	require.NoError(t, checkRequest(req))
}

func TestFastHTTPGetRequestSetsHeaders(t *testing.T) {
	ct := NewFastHTTPClientTransport()
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName("/hello")
	msg.WithCallerServiceName("trpc.test.client")
	msg.WithCalleeServiceName("trpc.test.server")
	msg.WithRequestTimeout(2 * time.Second)
	msg.WithSerializationType(codec.SerializationTypeJSON)
	msg.WithClientMetaData(codec.MetaData{"meta-key": []byte("meta-value")})
	msg.WithDyeing(true)
	msg.WithDyeingKey("dye-key")
	msg.WithEnvTransfer("production")
	_ = ctx

	reqHeader := &FastHTTPClientReqHeader{Method: fasthttp.MethodPost}
	opts := &transport.RoundTripOptions{Address: "example.com:80"}
	require.NoError(t, ct.getRequest(reqHeader, []byte(`{"name":"trpc"}`), msg, opts))
	defer fasthttp.ReleaseRequest(reqHeader.Request)

	req := reqHeader.Request
	require.Equal(t, fasthttp.MethodPost, string(req.Header.Method()))
	require.Equal(t, "http://example.com:80/hello", req.URI().String())
	require.Equal(t, "example.com:80", opts.TLSServerName)
	require.Equal(t, "trpc.test.client", string(req.Header.Peek(TrpcCaller)))
	require.Equal(t, "trpc.test.server", string(req.Header.Peek(TrpcCallee)))
	require.Equal(t, "2000", string(req.Header.Peek(TrpcTimeout)))
	require.Equal(t, serializationTypeContentType[codec.SerializationTypeJSON], string(req.Header.Peek(fasthttp.HeaderContentType)))
	require.Equal(t, []byte(`{"name":"trpc"}`), req.Body())
	require.Equal(t,
		strconv.Itoa(int(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)),
		string(req.Header.Peek(canonicalTrpcMessageType)),
	)

	var transInfo map[string]string
	require.NoError(t, json.Unmarshal(req.Header.Peek(TrpcTransInfo), &transInfo))
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("meta-value")), transInfo["meta-key"])
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("dye-key")), transInfo[TrpcDyeingKey])
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("production")), transInfo[TrpcEnv])
}

func TestFastHTTPGetRequestWithProvidedRequest(t *testing.T) {
	ct := NewFastHTTPClientTransport(transport.WithDisableEncodeTransInfoBase64())
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName("/provided")
	msg.WithCallerServiceName("trpc.test.client")
	msg.WithCalleeServiceName("trpc.test.server")
	msg.WithSerializationType(codec.SerializationTypeJSON)
	msg.WithClientMetaData(codec.MetaData{"meta-key": []byte("meta-value")})
	_ = ctx

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod(fasthttp.MethodGet)
	reqHeader := &FastHTTPClientReqHeader{Request: req}
	opts := &transport.RoundTripOptions{Address: "example.com:80", DisableConnectionPool: true}
	require.NoError(t, ct.getRequest(reqHeader, nil, msg, opts))
	require.Equal(t, "http://example.com:80/provided", req.URI().String())
	require.True(t, req.ConnectionClose())

	var transInfo map[string]string
	require.NoError(t, json.Unmarshal(req.Header.Peek(TrpcTransInfo), &transInfo))
	require.Equal(t, "meta-value", transInfo["meta-key"])
}

func TestFastHTTPCopyServerConfig(t *testing.T) {
	template := &fasthttp.Server{
		Name:                  "template",
		IdleTimeout:           time.Second,
		ReadTimeout:           2 * time.Second,
		WriteTimeout:          3 * time.Second,
		MaxConnsPerIP:         4,
		DisableKeepalive:      true,
		NoDefaultServerHeader: true,
		TLSConfig:             &tls.Config{ServerName: "example.com", MinVersion: tls.VersionTLS12},
	}
	target := &fasthttp.Server{}
	copyServerConfig(target, template)

	require.Equal(t, template.Name, target.Name)
	require.Equal(t, template.IdleTimeout, target.IdleTimeout)
	require.Equal(t, template.ReadTimeout, target.ReadTimeout)
	require.Equal(t, template.WriteTimeout, target.WriteTimeout)
	require.Equal(t, template.MaxConnsPerIP, target.MaxConnsPerIP)
	require.Equal(t, template.DisableKeepalive, target.DisableKeepalive)
	require.Equal(t, template.NoDefaultServerHeader, target.NoDefaultServerHeader)
	require.Equal(t, "example.com", target.TLSConfig.ServerName)
	require.NotSame(t, template.TLSConfig, target.TLSConfig)
}

func TestFastHTTPConfigureServerAndListener(t *testing.T) {
	st := NewFastHTTPServerTransport(transport.WithReusePort(false), transport.WithIdleTimeout(2*time.Second))
	st.Server = &fasthttp.Server{Name: "template"}
	server, err := st.configureFastHTTPServer(context.Background(), &transport.ListenServeOptions{
		DisableKeepAlives: true,
		IdleTimeout:       time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, "template", server.Name)
	require.True(t, server.DisableKeepalive)
	require.Equal(t, 2*time.Second, server.IdleTimeout)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	got, err := getFastHTTPListener(&transport.ListenServeOptions{Listener: ln}, true)
	require.NoError(t, err)
	require.Equal(t, ln, got)

	got, err = getFastHTTPListener(&transport.ListenServeOptions{
		Network: "tcp",
		Address: "127.0.0.1:0",
	}, false)
	require.NoError(t, err)
	require.NoError(t, got.Close())
}
