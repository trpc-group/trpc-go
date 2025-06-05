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

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/helloworld/pb"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	s := trpc.NewServer()
	pb.RegisterGreeterService(s, &Greeter{})
	if err := s.Serve(); err != nil {
		log.Error(err)
	}
}

type Greeter struct{}

func (g Greeter) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Infof("got hello request: %s", req.Msg)
	return &pb.HelloReply{Msg: "Hello " + req.Msg + "!"}, nil
}
