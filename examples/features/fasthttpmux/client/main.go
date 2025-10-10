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
	fasthttpNoProtocolProxyPost()
	fasthttpNoProtocolProxyGet()

	fasthttpNoProtocolClientPost()
	fasthttpNoProtocolClientGet()
}

// fasthttpNoProtocolProxyPost sends an POST request using FastHTTPClientProxy.
// Note: tRPC-Go framework configuration loading is omitted here assuming it's already loaded in a typical RPC handler.
func fasthttpNoProtocolProxyPost() {
	// Create a FastHTTPClientProxy, and use Noop serialization.
	fcp := thttp.NewFastHTTPClientProxy("trpc.app.server.fasthttp",
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:8080"),
	)

	// Create a FastHTTPClientReqHeader with the POST method.
	reqHeader := &thttp.FastHTTPClientReqHeader{
		Method: fasthttp.MethodPost,
		// Add a custom header "Hello": "fcp-post".
		// Notice: "hello" -> "Hello". But we can get "fcp-post" by string(req.Header.Peek("hello")).
		DecorateRequest: func(r *fasthttp.Request) *fasthttp.Request {
			r.Header.Add("hello", "fcp-post")
			return r
		},
	}

	// Create FastHTTPClientRspHeader to store the response header.
	rspHeader := &thttp.FastHTTPClientRspHeader{}

	// Create a Body containing the request data.
	req := &codec.Body{Data: []byte("Hello, I am fcp!")}

	// Create an empty Body to store the response data.
	rsp := &codec.Body{}

	// Send a v1 POST request.
	if err := fcp.Post(context.Background(), "/v1/hello", req, rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHeader),
	); err != nil {
		log.Warn("Error getting response:", err)
		return
	}
	// Get the "reply" field from the HTTP response header.
	replyHead := rspHeader.Response.Header.Peek("reply")
	log.Infof("Msg is %q, response head is %q", rsp.Data, replyHead)

	reqHeader.Request = nil
	// Send a v2 POST request.
	if err := fcp.Post(context.Background(), "/v2/hello", req, rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHeader),
	); err != nil {
		log.Warn("Error getting response:", err)
		return
	}

	// Get the "reply" field from the HTTP response header.
	replyHead = rspHeader.Response.Header.Peek("reply")
	log.Infof("Msg is %q, response head is %q", rsp.Data, replyHead)

	// After invocation, remember to release the req and rsp.
	fasthttp.ReleaseRequest(reqHeader.Request)
	fasthttp.ReleaseResponse(rspHeader.Response)
}

// fasthttpNoProtocolProxyGet sends an GET request using FastHTTPClientProxy.
func fasthttpNoProtocolProxyGet() {
	// Create a FastHTTPClientProxy, and use Noop serialization.
	fcp := thttp.NewFastHTTPClientProxy("trpc.app.server.fasthttp",
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:8080"),
	)

	// Create a FastHTTPClientReqHeader with the GET method.
	reqHeader := &thttp.FastHTTPClientReqHeader{
		Method: fasthttp.MethodGet,
		// Add a custom header "Hello": "fcp-get".
		// Notice: "hello" -> "Hello". But we can get "fcp-get" by string(req.Header.Peek("hello"))
		DecorateRequest: func(req *fasthttp.Request) *fasthttp.Request {
			req.Header.Add("hello", "fcp-get")
			return req
		},
	}

	// Create FastHTTPClientRspHeader to store the response header.
	rspHeader := &thttp.FastHTTPClientRspHeader{}

	// Create an empty Body to store the response data.
	rsp := &codec.Body{}

	// Send a v1 GET request.
	if err := fcp.Get(context.Background(), "/v1/hello", rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHeader),
	); err != nil {
		log.Warn("Error getting response:", err)
		return
	}

	// Get the "reply" field from the HTTP response header.
	replyHead := rspHeader.Response.Header.Peek("reply")
	log.Infof("Msg is %q, response head is %q", rsp.Data, replyHead)

	reqHeader.Request = nil
	// Send a v2 GET request.
	if err := fcp.Get(context.Background(), "/v2/hello", rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHeader),
	); err != nil {
		log.Warn("Error getting response:", err)
		return
	}

	// Get the "reply" field from the HTTP response header.
	replyHead = rspHeader.Response.Header.Peek("reply")
	log.Infof("Msg is %q, response head is %q", rsp.Data, replyHead)

	// After invocation, remember to release the req and rsp.
	defer func() {
		fasthttp.ReleaseRequest(reqHeader.Request)
		fasthttp.ReleaseResponse(rspHeader.Response)
	}()
}

func fasthttpNoProtocolClientGet() {
	fc := thttp.NewFastHTTPClient("trpc.app.server.fasthttp")

	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	// After invocation, remember to release the req and rsp.
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	// v1
	req.SetRequestURI("http://127.0.0.1:8080/v1/hello")
	req.Header.Add("hello", "fc-get")

	if err := fc.Do(req, rsp); err != nil {
		log.Warn("Error getting response:", err)
		return
	}
	log.Infof("Msg is %q, response head is %q", rsp.Body(), rsp.Header.Peek("reply"))

	req.Reset()
	rsp.Reset()

	// v2
	req.SetRequestURI("http://127.0.0.1:8080/v2/hello")
	req.Header.Add("hello", "fc-get")

	if err := fc.Do(req, rsp); err != nil {
		log.Warn("Error getting response:", err)
		return
	}
	log.Infof("Msg is %q, response head is %q", rsp.Body(), rsp.Header.Peek("reply"))
}

func fasthttpNoProtocolClientPost() {
	fc := thttp.NewFastHTTPClient("trpc.app.server.fasthttp")

	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	// After invocation, remember to release the req and rsp.
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("http://127.0.0.1:8080/v1/hello")
	req.Header.Add("hello", "fc-post")

	if err := fc.Do(req, rsp); err != nil {
		log.Warn("Error getting response:", err)
		return
	}
	log.Infof("Msg is %q, response head is %q", rsp.Body(), rsp.Header.Peek("reply"))

	req.Reset()
	rsp.Reset()

	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("http://127.0.0.1:8080/v2/hello")
	req.Header.Add("hello", "fc-post")

	if err := fc.Do(req, rsp); err != nil {
		log.Warn("Error getting response:", err)
		return
	}
	log.Infof("Msg is %q, response head is %q", rsp.Body(), rsp.Header.Peek("reply"))
}
