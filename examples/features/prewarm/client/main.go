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

// Package main is the client main package for prewarm demo.
package main

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/examples/helloworld/pb"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

func main() {
	ctx := context.Background()
	opts := []client.Option{
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithPreWarm(transport.PreWarmOptions{
			ConnsPerNode: 2,
			Timeout:      time.Second,
		}),
	}

	initializable, ok := client.DefaultClient.(client.InitializableClient)
	if !ok {
		log.Fatal("default client does not support initialization")
	}
	if err := initializable.Init(ctx, opts...); err != nil {
		log.Fatalf("prewarm client: %v", err)
	}

	proxy := pb.NewGreeterClientProxy(opts...)
	rsp, err := proxy.Hello(ctx, &pb.HelloRequest{Msg: "prewarm"})
	if err != nil {
		log.Fatalf("hello: %v", err)
	}
	log.Infof("hello response: %s", rsp.Msg)
}
