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

// Package main is the main package.
package main

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/examples/features/timeout/proto/chat"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	sayHi(4 * time.Second)
}

func sayHi(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(trpc.BackgroundContext(), timeout)
	defer cancel()
	c := pb.NewChatClientProxy(client.WithTarget("ip://127.0.0.1:8001"), client.WithTimeout(timeout))
	rsp, err := c.UnarySayHi(ctx, &pb.SayHiRequest{
		Message: "trpc-go-client",
	})
	if err != nil {
		log.Error(err)
	} else {
		log.Info("rsp message: %s", rsp.Message)
	}
}
