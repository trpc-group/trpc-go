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

// Package main is the server main package for http demo.
package main

import (
	"context"
	"fmt"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// GreeterServerImpl service implement
type GreeterServerImpl struct {
	pb.GreeterService
}

// SayHello say hello request
func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}

	log.Debugf("SayHello recv req:%s", req)
	rsp.Msg = "Hi " + req.Msg

	return rsp, nil
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
