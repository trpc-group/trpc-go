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
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/server"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestFastHTTPRegisterDefaultService(t *testing.T) {
	defer func() {
		err := recover()
		require.New(t).Contains(err, "duplicate method name")
		thttp.DefaultServerCodec.AutoReadBody = true
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	s := server.New()
	thttp.FastHTTPHandleFunc("/test/path", func(ctx *fasthttp.RequestCtx) {})
	thttp.FastHTTPHandleFunc("/test/path", func(ctx *fasthttp.RequestCtx) {})
	thttp.RegisterDefaultService(s)
}

func TestFastHTTPHandler(t *testing.T) {
	thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	defer func() {
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	serviceName := "trpc.fasthttp.test.no_protocol"
	ln, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer ln.Close()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol(protocol.FastHTTPNoProtocol),
		server.WithListener(ln),
	)
	thttp.FastHTTPHandleFunc("/index", func(ctx *fasthttp.RequestCtx) {
		ctx.Write(ctx.Request.Header.Protocol())
	})
	s := &server.Server{}
	s.AddService(serviceName, service)
	thttp.RegisterNoProtocolService(s.Service(serviceName))
	go func() {
		require.Nil(t, s.Serve())
	}()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	// Perform a test for stdhttp Client.
	resp, err := http.Get(fmt.Sprintf("http://%v/index", ln.Addr()))
	require.Nil(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	require.Equal(t, []byte("HTTP/1.1"), body)

	// Perform a test for FastHTTPClient.
	cli := thttp.NewFastHTTPClient(serviceName)
	statusCode, rspBody, err := cli.Get(nil, fmt.Sprintf("http://%v/index", ln.Addr()))
	require.Equal(t, fasthttp.StatusOK, statusCode)
	require.Equal(t, "HTTP/1.1", string(rspBody))
	require.Nil(t, err)

	// Perform a test for FastHTTPClientProxy.
	target := "ip://" + ln.Addr().String()
	b := &codec.Body{}
	fastProxy := thttp.NewFastHTTPClientProxy(serviceName,
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget(target),
		client.WithProtocol(protocol.FastHTTPNoProtocol))
	err = fastProxy.Get(context.Background(), "/index", b)
	require.Nil(t, err)
	require.Equal(t, "HTTP/1.1", string(b.Data))

	const invalidAddr = "localhost:910439"
	resp, err = http.Get(fmt.Sprintf("http://%s/index", invalidAddr))
	require.NotNil(t, err)
	require.Nil(t, resp)
}
