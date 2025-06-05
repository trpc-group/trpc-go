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

// Package main is the client main package for mTLS demo.
package main

import (
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata"
)

func main() {
	// Set up mTLS client options.
	options := []client.Option{
		client.WithTarget("ip://localhost:8080"),
		client.WithTLS(
			"../../../testdata/client.crt",
			"../../../testdata/client.key",
			"../../../testdata/ca.pem",
			"localhost",
		),
	}
	// new client
	proxy := pb.NewGreeterClientProxy(options...)
	ctx := trpc.BackgroundContext()
	// start rpc call
	rsp, err := proxy.SayHi(ctx, &pb.HelloRequest{Msg: "test mTLS message"})
	if err != nil {
		log.ErrorContextf(ctx, "say hi err: %v", err)
		return
	}
	log.InfoContextf(ctx, "get msg: %s", rsp.GetMsg())
}
