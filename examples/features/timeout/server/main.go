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
	"time"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/examples/features/timeout/shared"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	s := trpc.NewServer()
	pb.RegisterGreeterService(s, &timeoutServerImpl{})
	s.Serve()
}

// timeoutServerImpl  implements service.
type timeoutServerImpl struct{}

// SayHello implements `SayHello` method.
func (t *timeoutServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}
	log.Debugf("timeoutServerImpl SayHello recv req:%s", req)
	proxy := pb.NewGreeterClientProxy()
	hi, err := proxy.SayHi(ctx, req, client.WithTarget(shared.Addr))
	if err != nil {
		log.Errorf("call SayHi fail:%v", err)
		return nil, err
	}
	rsp.Msg = "SayHello: " + hi.Msg
	return rsp, nil
}

// SayHi implements `SayHello` method.
func (t *timeoutServerImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}
	log.Debugf("timeoutServerImpl SayHi recv req:%s", req)
	time.Sleep(time.Millisecond * 1100)
	rsp.Msg = "SayHi: " + req.Msg

	return rsp, nil
}
