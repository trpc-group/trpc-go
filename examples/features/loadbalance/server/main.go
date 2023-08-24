// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package main is the server main package for loadbalance demo.
package main

import (
	"context"
	"fmt"
	"log"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// ServerImpl implements service.
type ServerImpl struct {
	pb.GreeterService
}

// SayHello sends sayhello request.
func (s *ServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	if req.Msg == "" {
		err := errs.New(10001, "missing required field: Msg")
		return nil, err
	}
	log.Printf("Received msg from client : %v", req.Msg)
	return &pb.HelloReply{Msg: "Hello " + req.Msg}, nil
}

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register service.
	pb.RegisterGreeterService(s, &ServerImpl{})

	// Serve and listen.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
