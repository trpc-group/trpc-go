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
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata"
)

func main() {
	s := trpc.NewServer()

	// Register service.
	pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter"), &common.GreeterServerImpl{})

	// Start serve.
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
