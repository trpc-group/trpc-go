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

// Package main is the main package.
package main

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata"
)

var greeter = &greeterServiceImpl{
	proxy: pb.NewGreeterClientProxy(),
}

// greeterServiceImpl implements greeter service.
type greeterServiceImpl struct {
	proxy pb.GreeterClientProxy
}

// SayHello says hello request.
// trpc-cli -func "/trpc.test.helloworld.Greeter/SayHello" -target "ip://127.0.0.1:8000" -body '{"msg":"hellotrpc"}'
func (s *greeterServiceImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	time.Sleep(time.Second)
	rsp := &pb.HelloReply{
		Msg: req.Msg,
	}
	return rsp, nil
}

// SayHi says hi request.
func (s *greeterServiceImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Debugf("SayHi recv req: %s", req)

	rsp := &pb.HelloReply{
		Msg: "Hi " + req.Msg,
	}
	return rsp, nil
}
