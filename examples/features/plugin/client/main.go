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

// Package main is the client main package.
package main

import (
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// init server
	_ = trpc.NewServer()

	ctx := trpc.BackgroundContext()
	// call service
	hello, err := pb.NewGreeterClientProxy().SayHello(ctx, &pb.HelloRequest{
		Msg: "client",
	})
	// handle error
	if err != nil {
		log.Fatal(err)
	}
	// print response
	log.Infof("recv rsp:%s", hello.Msg)
}
