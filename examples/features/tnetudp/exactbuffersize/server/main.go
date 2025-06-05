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

// Package main is the client main package for tnetudp demo.
package main

import (
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
	pb "trpc.group/trpc-go/trpc-go/testdata"
	"trpc.group/trpc-go/trpc-go/transport/tnet"
)

func main() {
	// Use tnet transport with ExactUDPBufferSize enabled to allocate the exact buffer size for UDP packets.
	tnetTransport := tnet.NewServerTransport(tnet.WithServerExactUDPBufferSizeEnabled(true))
	s := trpc.NewServer(server.WithTransport(tnetTransport))

	// Register service.
	pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter"), &common.GreeterServerImpl{})

	// Start serve.
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
