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

// Package main provides a proxy example for multiple cases between SSE and common HTTP response based on tRPC-Go.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/server"

	"github.com/r3labs/sse/v2"
)

const (
	network = "tcp"
	address = "127.0.0.1:8081"
)

// You can run the command to test after the server is started.
// curl -X POST 'http://127.0.0.1:8081?data=hello'
func main() {
	// Start a local server to proxy the stream response.
	ln, err := net.Listen(network, address)
	if err != nil {
		panic(fmt.Errorf("listen err: %v", err))
	}
	defer ln.Close()

	// Register the auto service to the server.
	serviceName := "trpc.app.server.ServiceAutoProxy"
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork(network),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
	)
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(autoProxyHandler))
	// If you want to use the manual proxy, you can use the following code:
	// thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(manualProxyHandler))
	_ = manualProxyHandler

	s := &server.Server{}
	s.AddService(serviceName, service)
	//s.AddService(manualServiceName, manualService)
	if err := s.Serve(); err != nil {
		panic(fmt.Errorf("serve err: %v", err))
	}
}

// autoProxyHandler is the handler for the auto proxy.
func autoProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Prepare the response header.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set(thttp.Connection, "keep-alive")

	// Start a client.
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

				// Send the data to the client.
				_, _ = w.Write([]byte("This is a common response: "))
				_, _ = w.Write(bs)
				fmt.Printf("Process common response: %s\n", string(bs))
				return nil
			},
		},
		SSEHandler: &sseHandler{
			// This function tells the framework how to deal with the sse.Event.
			fn: func(e *sse.Event) error {
				if string(e.Event) != "message" {
					fmt.Printf("Ignored event: %s, data: %s\n", e.Event, e.Data)
					return nil
				}

				fmt.Printf("Processing event: %s, data: %s\n", e.Event, e.Data)
				// Send the data to the client.
				_, _ = w.Write([]byte("This is an SSE response: "))
				_, _ = w.Write(append(e.Data, '\n'))
				flusher.Flush() // This is SSE, DO remember to flush the response.
				return nil
			},
		},
	}

	// Get the data from the request, like 127.0.0.1:8081?data=xxx
	req := &codec.Body{Data: []byte(r.FormValue("data"))}
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
		http.Error(w, fmt.Sprintf("post err: %v", err), http.StatusInternalServerError)
	}
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

// manualProxyHandler is the handler for the manual proxy.
// It simply reads the response data and replaces keywords with ***.
func manualProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Prepare the response header.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set(thttp.Connection, "keep-alive")

	// Start a client.
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
	req := &codec.Body{Data: []byte(r.FormValue("data"))}
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
		http.Error(w, fmt.Sprintf("post err: %v", err), http.StatusInternalServerError)
	}

	// Read the body and replace keywords
	body := rspHead.Response.Body
	defer body.Close()
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		for _, keyword := range []string{"data:", "event:", "retry:", "id:"} {
			line = strings.ReplaceAll(line, keyword, "***")
		}
		_, _ = w.Write([]byte(line + "\n"))
		flusher.Flush()
	}
	if err := scanner.Err(); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}
}
