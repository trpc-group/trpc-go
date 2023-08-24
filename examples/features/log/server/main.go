// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package main is the server main package for log demo.
package main

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-go"
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

	rsp.Msg = fmt.Sprintf("trpc-go-server response: %s", req.Msg)

	log.Tracef("recv msg:%s", req)
	log.Debugf("recv msg:%s", req)
	log.Infof("recv msg:%s", req)
	log.Warnf("recv msg:%s", req)
	log.Errorf("recv msg:%s", req)

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
