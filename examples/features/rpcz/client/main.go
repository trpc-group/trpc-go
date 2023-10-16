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

// Package main this file is RPCZ client samples.
package main

import (
	pb "trpc.group/trpc-go/trpc-go/examples/features/rpcz/proto"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Create a service object, read config file.
	trpc.NewServer()

	ctx := trpc.BackgroundContext()
	rsp, err := pb.NewRPCZClientProxy().Hello(ctx, &pb.HelloReq{Msg: "111"})
	if err != nil {
		log.ErrorContextf(ctx, "Error in SayHello: %v", err)
		return
	}

	log.InfoContextf(ctx, "SayHello reply message is: %s", rsp.GetMsg())
}
