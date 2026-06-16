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

package http_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestFastHTTPRegistration(t *testing.T) {
	require.NotNil(t, codec.GetServer("fasthttp"))
	require.NotNil(t, codec.GetClient("fasthttp"))
	require.NotNil(t, codec.GetServer("fasthttp_no_protocol"))
	require.NotNil(t, transport.GetServerTransport("fasthttp"))
	require.NotNil(t, transport.GetServerTransport("fasthttp_no_protocol"))
	require.NotNil(t, transport.GetClientTransport("fasthttp"))
}

func TestFastHTTPServerTransportServe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := thttp.NewFastHTTPServerTransport(transport.WithReusePort(false))
	err = st.ListenAndServe(ctx,
		transport.WithListener(ln),
		transport.WithHandler(transportHandlerFunc(func(ctx context.Context, _ []byte) ([]byte, error) {
			requestCtx := thttp.RequestCtx(ctx)
			if requestCtx == nil {
				return nil, errors.New("missing fasthttp request context")
			}
			requestCtx.SetStatusCode(fasthttp.StatusAccepted)
			requestCtx.SetBodyString("ok")
			return nil, nil
		})),
	)
	require.NoError(t, err)

	status, body, err := fasthttp.Get(nil, "http://"+ln.Addr().String()+"/ping")
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusAccepted, status)
	require.Equal(t, "ok", string(body))
}

func TestFastHTTPClientTransportRoundTrip(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	server := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			if string(ctx.Method()) != fasthttp.MethodGet || string(ctx.Path()) != "/hello" {
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return
			}
			ctx.SetStatusCode(fasthttp.StatusCreated)
			ctx.Response.Header.SetContentType("application/json")
			ctx.SetBodyString(`{"msg":"ok"}`)
		},
	}
	go func() {
		_ = server.Serve(ln)
	}()
	defer server.Shutdown()

	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName("/hello")
	msg.WithCallerServiceName("trpc.test.client")
	msg.WithCalleeServiceName("trpc.test.server")
	msg.WithSerializationType(codec.SerializationTypeJSON)

	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(rsp)
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodGet})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})

	ct := thttp.NewFastHTTPClientTransport()
	_, err = ct.RoundTrip(ctx, nil, transport.WithDialAddress(ln.Addr().String()))
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusCreated, rsp.StatusCode())
	require.Equal(t, `{"msg":"ok"}`, string(rsp.Body()))
}

func TestFastHTTPClientTransportRejectsNilDecoratedRequest(t *testing.T) {
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName("/hello")
	msg.WithCallerServiceName("trpc.test.client")
	msg.WithCalleeServiceName("trpc.test.server")
	msg.WithClientReqHead(&thttp.FastHTTPClientReqHeader{
		Method: fasthttp.MethodGet,
		DecorateRequest: func(*fasthttp.Request) *fasthttp.Request {
			return nil
		},
	})
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{})

	ct := thttp.NewFastHTTPClientTransport()
	_, err := ct.RoundTrip(ctx, nil, transport.WithDialAddress("127.0.0.1:1"))
	require.ErrorContains(t, err, "DecorateRequest returned nil")
}

type transportHandlerFunc func(context.Context, []byte) ([]byte, error)

func (f transportHandlerFunc) Handle(ctx context.Context, req []byte) ([]byte, error) {
	return f(ctx, req)
}
