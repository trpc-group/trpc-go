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

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register the handle function for the "/v1/hello" endpoint.
	thttp.FastHTTPHandleFunc("/v1/hello", func(requestCtx *fasthttp.RequestCtx) {
		requestCtx.Response.Header.SetContentType("application/text")
		requestCtx.Response.Header.Set("reply", "response head")
		requestCtx.SetStatusCode(fasthttp.StatusOK)
		requestCtx.WriteString("Hello, " + string(requestCtx.Request.Header.Peek("hello")))
		if string(requestCtx.Method()) == fasthttp.MethodPost {
			requestCtx.WriteString("[POST]")
		}
	})

	// When registering the NoProtocolService, the parameter passed must match
	// the service name in the configuration: s.Service("trpc.app.server.fasthttp").
	thttp.RegisterNoProtocolService(s.Service("trpc.app.server.fasthttp"))

	// Start serving and listening.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
