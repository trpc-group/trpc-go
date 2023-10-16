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

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// Create a server and register service.
	s := trpc.NewServer()
	pb.RegisterGreeterService(s, &greeterServiceImpl{})

	// Start serving.
	if err := s.Serve(); err != nil {
		log.Fatalf("service serves error: %v", err)
	}
}

// greeterServiceImpl greeter service implement.
type greeterServiceImpl struct{}

// SayHello Say hello request.
func (s *greeterServiceImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}

	log.Debugf("SayHello recv req:%s", req)

	// We can get the specified key-value through GetMetaData api.
	// If there is not the key-value, nil will be returned.
	val := trpc.GetMetaData(ctx, "say-hello-client")
	log.Debugf("SayHello get key: say-hello-client, value: %s", val)

	// Set the transmitted fields through ctx to return to the upstream caller.
	trpc.SetMetaData(ctx, "say-hello-server", []byte("hello"))

	return rsp, nil
}

// SayHi Say hi request.
func (s *greeterServiceImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}

	log.Debugf("SayHi recv req:%s", req)

	// Get all key-value pairs of metadata by for-range.
	msg := codec.Message(ctx)
	md := msg.ServerMetaData()
	for key, val := range md {
		log.Debugf("SayHi get key: %s, value: %s", key, string(val))
	}

	trpc.SetMetaData(ctx, "say-hi-server", []byte("hi"))

	return rsp, nil
}
