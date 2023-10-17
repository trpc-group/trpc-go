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

// Package main is the client main package for config demo.
package main

import (
	"context"
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var addr = "ip://127.0.0.1:8000"

func main() {
	ctx, _ := context.WithTimeout(context.TODO(), time.Millisecond*2000)

	// Init proxy.
	clientProxy := pb.NewGreeterClientProxy(client.WithTarget(addr))

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	// Send request.
	rsp, err := clientProxy.SayHello(ctx, req)
	if err != nil {
		fmt.Println("Say hi err:%v", err)
		return
	}
	fmt.Printf("Get msg: %s\n", rsp.GetMsg())
}
