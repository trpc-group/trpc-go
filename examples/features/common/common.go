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

// Package common provides common function for trpc-go example.
package common

import (
	"context"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// GreeterServerImpl service implement
type GreeterServerImpl struct{}

// SayHello say hello request
func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}
	// implement business logic here ...
	// ...

	log.Debugf("recv req:%s", req)

	proxy := pb.NewGreeterClientProxy()
	hi, err := proxy.SayHi(ctx, req, client.WithTarget("ip://127.0.0.1:8000"))
	if err != nil {
		log.Errorf("say hi fail:%v", err)
		return nil, err
	}
	rsp.Msg = "Hello " + hi.Msg
	return rsp, nil
}

// SayHi say hi request
func (s *GreeterServerImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}
	// implement business logic here ...
	// ...

	log.Debugf("SayHi recv req:%s", req)
	rsp.Msg = "Hi " + req.Msg

	return rsp, nil
}
