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

// Package main is the client main package for cancellation demo.
package main

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var addr = "ip://127.0.0.1:8000"

func main() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)

	// Init proxy.
	clientProxy := pb.NewGreeterClientProxy(client.WithTarget(addr))

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	// Send normal request.
	rsp, err := clientProxy.SayHello(ctx, req)
	if err != nil {
		log.ErrorContextf(ctx, "SayHello err[%v] req[%s]", err, req.String())
		return
	}
	log.InfoContextf(ctx, "SayHello success rsp[%s]", rsp.String())

	// Cancel context.
	cancel()

	// Send canceled request.
	reqCanceled := &pb.HelloRequest{
		Msg: "trpc-go-client-canceled",
	}
	rsp, err = clientProxy.SayHello(ctx, reqCanceled)
	if err != nil {
		log.ErrorContextf(ctx, "canceled SayHello err[%v] req[%s]", err, req.String())
		return
	}
	log.InfoContextf(ctx, "canceled SayHello success rsp[%s]", rsp.String())
}
