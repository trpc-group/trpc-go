//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
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
	"time"

	"trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/examples/features/timeout/proto/chat"
	"trpc.group/trpc-go/trpc-go/log"
)

//go:generate trpc create -p ../proto/chat/chat.proto --api-version 2 --rpconly -o ../proto/chat --protodir .. --mock=false --nogomod

func main() {
	s := trpc.NewServer()
	pb.RegisterChatService(s.Service("trpc.examples.timeout.forward-chat"), &chat{
		client: pb.NewChatClientProxy(),
	})
	if err := s.Serve(); err != nil {
		log.Error(err)
	}
}

// timeoutServerImpl  implements service.
type chat struct {
	client pb.ChatClientProxy
}

func (c *chat) UnarySayHi(ctx context.Context, req *pb.SayHiRequest) (*pb.SayHiResponse, error) {
	time.Sleep(6 * time.Second)
	rsp, err := c.client.UnarySayHi(ctx, req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &pb.SayHiResponse{Message: "SayHi: " + rsp.Message}, nil
}
