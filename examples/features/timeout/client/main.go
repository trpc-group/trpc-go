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
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/examples/features/timeout/shared"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	fmt.Println("== testSayHello begin ==")
	testSayHello()
	fmt.Println("== testSayHello end ==")

	fmt.Println("== testSayHi begin ==")
	testSayHi()
	fmt.Println("== testSayHi end ==")
}

// testSayHello is the test cases for SayHello method.
func testSayHello() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithTarget(shared.Addr),
		// Setting the timeout value for this call to 2000ms.
		client.WithTimeout(time.Millisecond * 2000),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHello(ctx, req)
	fmt.Println(rsp, err)
}

// testSayHi is the test cases for method.
func testSayHi() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithTarget(shared.Addr),
		// Setting the timeout value for this call to 1000ms.
		client.WithTimeout(time.Millisecond * 1000),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	// This rpc calling would timeout.
	rsp, err := clientProxy.SayHi(ctx, req)
	// Would print timeout error.
	fmt.Println(rsp, err)
}
