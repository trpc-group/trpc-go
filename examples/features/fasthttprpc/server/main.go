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

// Package main provides an echo server.
package main

//go:generate trpc create -p ../proto/echo/echo.proto -o ../proto/echo --alias --protocol http --api-version 2 --rpconly --mock=false --nogomod=true

import (
	"context"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	pb "trpc.group/trpc-go/trpc-go/examples/features/fasthttprpc/proto/echo"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// for custom field Json alias in pb file.
	codec.Marshaler.OrigName = false

	// Create a server.
	s := trpc.NewServer()
	// Register echoService into the server.
	pb.RegisterEchoService(s.Service("trpc.examples.echo.Echo"), &echoService{})

	// Start the server.
	if err := s.Serve(); err != nil {
		log.Fatalf("server serving: %v", err)
	}
}

type echoService struct{}

// UnaryEcho echos request's message.
func (s *echoService) UnaryEcho(ctx context.Context, request *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{
		Code:    219,
		Message: request.Message,
	}, nil
}
