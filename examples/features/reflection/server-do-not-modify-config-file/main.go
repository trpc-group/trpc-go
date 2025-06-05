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

package main

import (
	"trpc.group/trpc-go/trpc-go"
	ecpb "trpc.group/trpc-go/trpc-go/examples/features/reflection/proto"
	"trpc.group/trpc-go/trpc-go/examples/features/reflection/service"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/reflection"
	"trpc.group/trpc-go/trpc-go/server"
	hwpb "trpc.group/trpc-go/trpc-go/testdata"
)

func main() {
	s := trpc.NewServer()
	hwpb.RegisterGreeterService(s.Service("trpc.test.helloworld.GreeterXXX"), &service.Greeter{})
	ecpb.RegisterEchoService(s.Service("trpc.examples.echo.EchoYYY"), &service.Echo{})
	service := server.New(server.WithServiceName("trpc.reflection.v1.ServerReflection"),
		server.WithProtocol("trpc"),
		server.WithNetwork("tcp"),
		server.WithAddress("127.0.0.1:8002"),
	)
	s.AddService("trpc.reflection.v1.ServerReflection", service)
	reflection.Register(service, s)
	if err := s.Serve(); err != nil {
		log.Fatalf("server serving: %v", err)
	}
}
