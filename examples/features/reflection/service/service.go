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

package service

import (
	"context"

	ecpb "trpc.group/trpc-go/trpc-go/examples/features/reflection/proto"
	"trpc.group/trpc-go/trpc-go/log"
	hwpb "trpc.group/trpc-go/trpc-go/testdata"
)

// Greeter implements hello world service.
type Greeter struct{}

// SayHello says hello to request.
func (s *Greeter) SayHello(_ context.Context, req *hwpb.HelloRequest) (*hwpb.HelloReply, error) {
	log.Debugf("SayHello recv req: %s", req)
	return &hwpb.HelloReply{
		Msg: "Hello " + req.GetMsg(),
	}, nil
}

// SayHi says hi to request.
func (s *Greeter) SayHi(_ context.Context, req *hwpb.HelloRequest) (*hwpb.HelloReply, error) {
	log.Debugf("SayHi recv req: %s", req)
	return &hwpb.HelloReply{
		Msg: "Hello " + req.GetMsg(),
	}, nil
}

// Echo implements echo service.
type Echo struct {
	ecpb.UnimplementedEcho
}

// UnaryEcho echo to request.
func (s *Echo) UnaryEcho(_ context.Context, req *ecpb.EchoRequest) (*ecpb.EchoResponse, error) {
	log.Debugf("UnaryEcho recv req: %s", req)
	return &ecpb.EchoResponse{
		Message: req.GetMessage(),
	}, nil
}
