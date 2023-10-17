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
	"fmt"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// GreeterServerImpl service implement
type GreeterServerImpl struct{}

// SayHello say hello request
func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}

	rsp.Msg = fmt.Sprintf("trpc-go-server response: Hello %s", req.Msg)

	// We can dynamically adjust the logging level at runtime.
	log.SetLevel("console", log.LevelNames[req.Msg])

	log.Tracef("recv msg:%s", req)
	log.Debugf("recv msg:%s", req)
	log.Infof("recv msg:%s", req)
	log.Warnf("recv msg:%s", req)
	log.Errorf("recv msg:%s", req)

	return rsp, nil
}

// SayHi say hi request
func (s *GreeterServerImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}

	rsp.Msg = fmt.Sprintf("trpc-go-server response: Hello %s", req.Msg)

	// We can dynamically adjust the logging level at runtime.
	log.SetLevel("file", log.LevelNames[req.Msg])

	log.Tracef("recv msg:%s", req)
	log.Debugf("recv msg:%s", req)
	log.Infof("recv msg:%s", req)
	log.Warnf("recv msg:%s", req)
	log.Errorf("recv msg:%s", req)

	return rsp, nil
}

func main() {
	s := trpc.NewServer()
	pb.RegisterGreeterService(s, &GreeterServerImpl{})
	err := s.Serve()
	if err != nil {
		panic(err)
	}
}
