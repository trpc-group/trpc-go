// Package main is the main package.
package main

import (
	"context"
	"fmt"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/restful/pb"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// init trpc server
	server := trpc.NewServer()
	// Register the greeter service with the server
	pb.RegisterGreeterService(server, new(greeterService))
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
		Message: fmt.Sprintf("[restful] Message name:%s,subfield:%s",
			req.GetName(), req.GetSub().GetSubfield()),
	}
	return rsp, nil
}

// UpdateMessage implements pb.GreeterService.
func (g greeterService) UpdateMessage(ctx context.Context, req *pb.UpdateMessageRequest) (*pb.MessageInfo, error) {
	log.InfoContextf(ctx, "[restful] Received UpdateMessage request with req: %v", req)
	// handle request
	rsp := &pb.MessageInfo{
		Message: fmt.Sprintf("[restful] UpdateMessage message_id:%s,message:%s",
			req.GetMessageId(), req.GetMessage().GetMessage()),
	}
	return rsp, nil
}

// UpdateMessageV2 implements pb.GreeterService.
func (g greeterService) UpdateMessageV2(ctx context.Context, req *pb.UpdateMessageV2Request) (*pb.MessageInfo, error) {
	log.InfoContextf(ctx, "[restful] Received UpdateMessageV2 request with req: %v", req)
	// handle request
	rsp := &pb.MessageInfo{
		Message: fmt.Sprintf("[restful] UpdateMessageV2 message_id:%s,message:%s",
			req.GetMessageId(), req.GetMessage()),
	}
	return rsp, nil
}
