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

// Package main is the server main package.
package main

import (
	"context"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/examples/features/plugin"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"

	// import plugin
	_ "trpc.group/trpc-go/trpc-go/examples/features/plugin"
)

func main() {
	// init server
	s := trpc.NewServer()

	// register service
	pb.RegisterGreeterService(s, new(greeterImpl))

	// serve and listen
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}

type greeterImpl struct {
	common.GreeterServerImpl
}

// SayHello say hello request
// rewrite SayHello
func (g *greeterImpl) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Info("[Plugin] trpc-go-server SayHello, req.msg:", req.Msg)

	// call plugin
	plugin.Record()

	rsp := &pb.HelloReply{}

	rsp.Msg = "[Plugin] trpc-go-server response: Hello " + req.Msg

	return rsp, nil
}
