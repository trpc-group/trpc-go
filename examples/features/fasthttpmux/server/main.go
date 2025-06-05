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

// Package main is the server main package for http demo.
package main

import (
	"fmt"

	"trpc.group/trpc-go/trpc-go"
	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"

	thttp "trpc.group/trpc-go/trpc-go/http"
)

func main() {
	// Init server.
	s := trpc.NewServer()

	router := routing.New()
	router.Get("/v1/hello", func(ctx *routing.Context) error {
		ctx.Response.Header.SetContentType("application/text")
		ctx.Response.Header.Set("reply", "response head")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString("/v1/hello, " + string(ctx.Request.Header.Peek("hello")))
		return nil
	})

	router.Get("/v2/hello", func(ctx *routing.Context) error {
		ctx.Response.Header.SetContentType("application/text")
		ctx.Response.Header.Set("reply", "response head")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString("/v2/hello, " + string(ctx.Request.Header.Peek("hello")))
		return nil
	})

	router.Post("/v1/hello", func(ctx *routing.Context) error {
		ctx.Response.Header.SetContentType("application/text")
		ctx.Response.Header.Set("reply", "response head")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString("/v1/hello, " + string(ctx.Request.Header.Peek("hello")))
		ctx.WriteString("[POST]")
		return nil
	})

	router.Post("/v2/hello", func(ctx *routing.Context) error {
		ctx.Response.Header.SetContentType("application/text")
		ctx.Response.Header.Set("reply", "response head")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString("/v2/hello, " + string(ctx.Request.Header.Peek("hello")))
		ctx.WriteString("[POST]")
		return nil
	})

	thttp.FastHTTPHandleFunc("*", router.HandleRequest)
	thttp.FastHTTPHandleFunc("/123", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("no routing")
	})
	thttp.RegisterNoProtocolService(s.Service("trpc.app.server.fasthttp"))

	// Start serving and listening.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
