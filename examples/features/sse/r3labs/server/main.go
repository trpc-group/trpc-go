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

// Package main provides a server example for SSE based on https://github.com/r3labs/sse.
package main

import (
	"fmt"
	"net"
	"net/http"

	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"

	"github.com/r3labs/sse/v2"
)

func main() {
	const (
		network = "tcp"
		address = "127.0.0.1:8081"
	)
	ln, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return
	}
	defer ln.Close()

	const pattern = "/events"
	serviceName := "trpc.app.server.Service" + pattern
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork(network),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
	)

	svr := sse.New()
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, svr.ServeHTTP)
	thttp.RegisterNoProtocolServiceMux(service, mux)

	// Create a stream named "test".
	stream := "test"
	svr.CreateStream(stream)
	// Publish 1 event.
	publishSingeEvent(svr, stream)
	// Publish 3 events.
	publishMultipleEvents(svr, stream)

	s := &server.Server{}
	s.AddService(serviceName, service)
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// Publish an event to the stream.
func publishSingeEvent(svr *sse.Server, stream string) {
	svr.Publish(stream, &sse.Event{Data: []byte("data")})
}

// Publish multiple events to the stream.
func publishMultipleEvents(svr *sse.Server, stream string) {
	for i := 0; i < 3; i++ {
		svr.Publish(stream, &sse.Event{Data: []byte(fmt.Sprintf("data %d", i))})
	}
}
