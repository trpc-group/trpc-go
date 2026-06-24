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

// Package main is the client main package for SSE demo.
package main

import (
	"context"
	stdhttp "net/http"
	"strings"

	"github.com/r3labs/sse/v2"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

type eventHandler struct{}

func (eventHandler) Handle(event *sse.Event) error {
	log.Infof("event id=%s type=%s data=%s", event.ID, event.Event, event.Data)
	return nil
}

func main() {
	proxy := thttp.NewClientProxy(
		"trpc.examples.sse.Events",
		client.WithTarget("ip://127.0.0.1:8080"),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
	)

	rspHead := &thttp.ClientRspHeader{
		SSECondition: func(rsp *stdhttp.Response) bool {
			return strings.Contains(rsp.Header.Get("Content-Type"), "text/event-stream")
		},
		SSEHandler: eventHandler{},
	}
	if err := proxy.Get(context.Background(), "/events", nil, client.WithRspHead(rspHead)); err != nil {
		log.Fatalf("get events: %v", err)
	}
}
