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

package main

import (
	"context"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/examples/helloworld/pb"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	c := pb.NewGreeterClientProxy(client.WithTarget("ip://127.0.0.1:8000"))
	rsp, err := c.Hello(context.Background(), &pb.HelloRequest{Msg: "world"})
	if err != nil {
		log.Error(err)
	}
	log.Info(rsp.Msg)
}
