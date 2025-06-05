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

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/restful/server/pb"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// init trpc server
	server := trpc.NewServer()
	// Register the greeter service with the server
	pb.RegisterGreeterService(server.Service("trpc.test.helloworld.Greeter"), new(greeterService))
	// Run the server
	if err := server.Serve(); err != nil {
		log.Fatal(err)
	}
}

// greeterService is used to implement pb.GreeterService.
type greeterService struct {
	// unimplementedGreeterServiceServer is the unimplemented greeter service server
	pb.UnimplementedGreeter
}

// SayHello implements pb.GreeterService.
func (g greeterService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.InfoContextf(ctx, "[restful] Received SayHello request with req: %v", req)
	// handle request
	rsp := &pb.HelloReply{
		Message: "[restful] SayHello Hello " + req.Name,
	}
	return rsp, nil
}

// Message implements pb.GreeterService.
func (g greeterService) Message(ctx context.Context, req *pb.MessageRequest) (*pb.MessageInfo, error) {
	log.InfoContextf(ctx, "[restful] Received Message request with req: %v", req)
	// handle request
	rsp := &pb.MessageInfo{
		Message: fmt.Sprintf("[restful] Message name: %s,subfield: %s",
			req.GetName(), req.GetSub().GetSubfield()),
	}
	return rsp, nil
}

// UpdateMessage implements pb.GreeterService.
func (g greeterService) UpdateMessage(ctx context.Context, req *pb.UpdateMessageRequest) (*pb.MessageInfo, error) {
	log.InfoContextf(ctx, "[restful] Received UpdateMessage request with req: %v", req)
	// handle request
	rsp := &pb.MessageInfo{
		Message: fmt.Sprintf("[restful] UpdateMessage message_id: %s,message: %s",
			req.GetMessageId(), req.GetMessage().GetMessage()),
	}
	return rsp, nil
}

// UpdateMessageV2 implements pb.GreeterService.
func (g greeterService) UpdateMessageV2(ctx context.Context, req *pb.UpdateMessageV2Request) (*pb.MessageInfo, error) {
	log.InfoContextf(ctx, "[restful] Received UpdateMessageV2 request with req: %v", req)
	// handle request
	rsp := &pb.MessageInfo{
		Message: fmt.Sprintf("[restful] UpdateMessageV2 message_id: %s,message: %s",
			req.GetMessageId(), req.GetMessage()),
	}
	return rsp, nil
}
