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

// Package main is the client main package for FastHTTP demo.
package main

import (
	"context"

	"github.com/valyala/fasthttp"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	callWithFastHTTPClientProxy()
	callWithFastHTTPClient()
}

func callWithFastHTTPClientProxy() {
	proxy := thttp.NewFastHTTPClientProxy(
		"trpc.app.server.fasthttp",
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:8080"),
	)

	reqHead := &thttp.FastHTTPClientReqHeader{
		Method: fasthttp.MethodPost,
		DecorateRequest: func(req *fasthttp.Request) *fasthttp.Request {
			req.Header.Set("hello", "proxy")
			return req
		},
	}
	rspHead := &thttp.FastHTTPClientRspHeader{}
	req := &codec.Body{Data: []byte("Hello, FastHTTP proxy!")}
	rsp := &codec.Body{}

	if err := proxy.Post(context.Background(), "/v1/hello", req, rsp,
		client.WithReqHead(reqHead),
		client.WithRspHead(rspHead),
	); err != nil {
		log.Warnf("FastHTTPClientProxy request failed: %v", err)
		return
	}
	log.Infof("FastHTTPClientProxy response: %q, reply header: %q",
		rsp.Data, rspHead.Response.Header.Peek("reply"))
}

func callWithFastHTTPClient() {
	fc := thttp.NewFastHTTPClient("trpc.app.server.fasthttp")

	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("hello", "client")
	req.SetRequestURI("http://127.0.0.1:8080/v1/hello")

	if err := fc.Do(req, rsp); err != nil {
		log.Warnf("FastHTTPClient request failed: %v", err)
		return
	}
	log.Infof("FastHTTPClient response: %q, reply header: %q",
		rsp.Body(), rsp.Header.Peek("reply"))
}
