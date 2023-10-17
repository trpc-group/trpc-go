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

// Package main is the main package for discovery demo.
package main

import (
	"context"
	"fmt"
	"log"

	trpc "trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	s := trpc.NewServer()
	pb.RegisterGreeterService(s, &impl{})
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}

type impl struct{}

func (i *impl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received msg from client : %v", req.Msg)
	return &pb.HelloReply{Msg: "Hello " + req.Msg}, nil
}

func (i *impl) SayHi(_ context.Context, _ *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{}, nil
}
