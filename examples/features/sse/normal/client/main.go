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

// Package main provides a client example for SSE based on tRPC-Go.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"

	"github.com/r3labs/sse/v2"
)

func main() {
	// Read the body in manual mode.
	if err := manualReadBody(); err != nil {
		log.Fatalf("manual read body failed, err: %v", err)
	}

	// Recommended: Read the body in auto mode.
	if err := autoReadBody(); err != nil {
		log.Fatalf("auto read body failed, err: %v", err)
	}
}

// manualReadBody reads the body manually.
// You are required to do a stream read on rspHead.Response.Body and close it manually.
func manualReadBody() error {
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

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req := &codec.Body{Data: []byte("hello")}
	rsp := &codec.Body{}
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

	// Do stream reads directly from rspHead.Response.Body.
	body := rspHead.Response.Body
	// Do remember to close the body.
	defer body.Close()

	// You can do some extra work such as understanding, and proxy the raw stream data to another sse client.
	// Here just use io.Copy to read the raw stream data and print it to stdout.
	if _, err := io.Copy(os.Stdout, body); err != nil {
		return fmt.Errorf("copy body err: %v", err)
	}
	return nil
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

	fmt.Printf("Received data: %s\n", string(data))
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
