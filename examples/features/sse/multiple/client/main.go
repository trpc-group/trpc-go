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

// Package main provides a client example for multiple cases between SSE and common HTTP response based on tRPC-Go.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"

	"github.com/r3labs/sse/v2"
)

func main() {
	// Handle the multiple cases of SSE and common HTTP response, automatically.
	if err := autoReadBody(); err != nil {
		log.Fatalf("auto read body failed, err: %v", err)
	}
}

// autoReadBody reads the body in auto mode.
// You only need to implement the sseHandler to tell the framework how to deal with the sse.Event.
func autoReadBody() error {
	c := thttp.NewClientProxy(
		"trpc.app.server.ServiceSSE",
		client.WithTarget("ip://127.0.0.1:8080"),
	)
	header := http.Header{}
	header.Set("Cache-Control", "no-cache")
	header.Set("Accept", "text/event-stream")
	header.Set(thttp.Connection, "keep-alive")
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
		Header: header,
	}

	// Disable manual body reading in order to
	// enable the framework's automatic body reading capability,
	// so that the client-side streaming reads could be done by the framework.
	var data []byte
	rspHead := &thttp.ClientRspHeader{
		// Enable automatic body reading capability.
		ManualReadBody: false,
		// SSECondition tells the framework whether to invoke the SSEHandler or not.
		// The default SSECondition always returns true.
		// Leave it empty to use the default one, or you can implement your own SSECondition.
		SSECondition: func(r *http.Response) bool {
			return r.Header.Get("Content-Type") == "text/event-stream"
		},
		ResponseHandler: &rspHandler{
			// This function tells the framework how to deal with the http.Response,
			// if the server sends a response that is not an SSE event.
			fn: func(r *http.Response) error {
				bs, err := io.ReadAll(r.Body)
				if err != nil {
					return fmt.Errorf("read body failed, err: %v", err)
				}
				msg := string(bs)
				fmt.Printf("Process common response: %s\n", msg)
				data = append(data, msg...)
				return nil
			},
		},
		SSEHandler: &sseHandler{
			// This function tells the framework how to deal with the sse.Event.
			fn: func(e *sse.Event) error {
				if string(e.Event) == "message" {
					fmt.Printf("Processing event: %s, data: %s\n", e.Event, e.Data)
					data = append(data, e.Data...)
				} else {
					fmt.Printf("Ignored event: %s, data: %s\n", e.Event, e.Data)
				}
				return nil
			},
		},
	}

	req := &codec.Body{Data: []byte("hello")}
	rsp := &codec.Body{}

	for i := 0; i < 4; i++ {
		data = []byte{} // clear the data before each request.
		err := c.Post(context.Background(), "/v1/hello", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHeader),
			client.WithRspHead(rspHead),
			client.WithTimeout(time.Minute),
		)
		if err != nil {
			return fmt.Errorf("post err: %v", err)
		}

		fmt.Printf("Received data: %s\n\n", string(data))
	}

	return nil
}

// sseHandler defines the event handler, implements the SSEHandler interface.
type sseHandler struct {
	fn func(e *sse.Event) error
}

// Handle implements the SSEHandler interface.
func (h *sseHandler) Handle(e *sse.Event) error {
	return h.fn(e)
}

// rspHandler implements the RspHandler interface.
type rspHandler struct {
	fn func(r *http.Response) error
}

// Handle implements the ResponseHandler interface.
func (h *rspHandler) Handle(r *http.Response) error {
	return h.fn(r)
}
