// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package main is the server main package for error demo.
package main

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// GreeterServerImpl service implement
type GreeterServerImpl struct {
	pb.GreeterService
}

// SayHello say hello request
func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	if req.Msg == "" {
		err := errs.New(10001, "request missing required field: Msg")
		return nil, err
	}

	return &pb.HelloReply{Msg: "Hello " + req.Msg}, nil
}

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register service.
	pb.RegisterGreeterService(s, &GreeterServerImpl{})

	// Serve and listen.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
