// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package main provides an echo server.
package main

//go:generate trpc create -p ../proto/echo/echo.proto --api-version 2 --rpconly -o ../proto/echo --protodir . --mock=false

import (
	"bytes"
	"context"
	"fmt"
	"io"

	trpc "trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/examples/features/attachment/proto/echo"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
)

func main() {
	// Create a server.
	s := trpc.NewServer()

	// Register echoService into the server.
	pb.RegisterEchoService(s, &echoService{})

	// Start the server.
	if err := s.Serve(); err != nil {
		log.Fatalf("server serving: %v", err)
	}
}

type echoService struct{}

// UnaryEcho echos request's message and attachment.
func (s *echoService) UnaryEcho(ctx context.Context, request *pb.EchoRequest) (*pb.EchoResponse, error) {
	// Get and read attachment send by client
	a := server.GetAttachment(trpc.Message(ctx))
	bts, err := io.ReadAll(a.Request())
	if err != nil {
		return nil, fmt.Errorf("reading attachment: %w", err)
	}
	log.Infof("received attachment: %s", bts)

	// send server's attachment to client
	a.SetResponse(bytes.NewReader([]byte("server attachment")))

	return &pb.EchoResponse{Message: request.GetMessage()}, nil
}
