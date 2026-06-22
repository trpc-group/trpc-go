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

// Package main is the server main package for FastHTTP demo.
package main

import (
	"fmt"

	"github.com/valyala/fasthttp"

	trpc "trpc.group/trpc-go/trpc-go"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func main() {
	s := trpc.NewServer()

	thttp.FastHTTPHandleFunc("/v1/hello", func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.SetContentType("text/plain")
		ctx.Response.Header.Set("reply", "response head")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.WriteString(string(ctx.Path()) + ", " + string(ctx.Request.Header.Peek("hello")))
		if string(ctx.Method()) == fasthttp.MethodPost {
			ctx.WriteString("[POST]")
		}
	})

	thttp.RegisterNoProtocolService(s.Service("trpc.app.server.fasthttp"))

	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
