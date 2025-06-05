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

// Package main provides a server example for multiple cases between SSE and common HTTP response based on tRPC-Go.
package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"trpc.group/trpc-go/trpc-go"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"

	"github.com/r3labs/sse/v2"
)

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register the handle function for the "/v1/hello" endpoint.
	thttp.HandleFunc("/v1/hello", handle)

	// When registering the NoProtocolService, the parameter passed must match the service name in the configuration: s.Service("trpc.app.server.stdhttp").
	thttp.RegisterNoProtocolService(s.Service("trpc.app.server.ServiceSSE"))

	// Start serving and listening.
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

var isSSE atomic.Bool

// handle is a function that processes HTTP requests.
// After the request is processed, isSSE will be set to the opposite value.
func handle(w http.ResponseWriter, r *http.Request) error {
	defer func() { isSSE.Store(!isSSE.Load()) }()
	if isSSE.Load() {
		return sseHandlerFunc(w, r)
	}
	return normalHandlerFunc(w, r)
}

// sseHandlerFunc is a handler that processes SSE responses.
func sseHandlerFunc(w http.ResponseWriter, r *http.Request) error {
	// The following code is NECESSARY to implement the server side of SSE(server-sent events).
	// For more information on SSE, please refer to
	// https://html.spec.whatwg.org/multipage/server-sent-events.html#server-sent-events

	// Beginning of necessary code.
	// The Flusher interface is implemented by ResponseWriters that support streaming.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return fmt.Errorf("http: ResponseWriter from %T does not implement http.Flusher", w)
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set(thttp.Connection, "keep-alive")
	// End of necessary code.

	w.Header().Set("Access-Control-Allow-Origin", "*")

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return fmt.Errorf("http: Read request body: %v", err)
	}
	msg := string(bs)
	for i := 0; i < 3; i++ {
		e := sse.Event{Event: []byte("message"), Data: []byte(msg + strconv.Itoa(i))}
		if err := thttp.WriteSSE(w, e); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return fmt.Errorf("thttp WriteSSE: %v", err)
		}
		// Flush the events to the client, so that the events are immediately sent to the client
		// instead of being buffered. If not, the events may not be sent to the client until the buffer is full.
		flusher.Flush()
		// Simulate the processing delay.
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// normalHandlerFunc is a handler that processes common HTTP responses.
func normalHandlerFunc(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set(thttp.Connection, "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return fmt.Errorf("http: Read request body: %v", err)
	}
	msg := string(bs)

	var data []byte
	for i := 0; i < 3; i++ {
		data = append(data, []byte(msg+strconv.Itoa(i))...)
	}

	_, err = w.Write(data)
	return err
}
