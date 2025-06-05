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
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata"
)

func main() {
	// Create a server and register a service.
	s := trpc.NewServer()
	pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter1"), greeter)
	// Start serving.
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
