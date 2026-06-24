// Package main provides an echo server.
package main

//go:generate trpc create -p ../proto/greeter/greeter.proto --api-version 2 --rpconly -o ../proto/greeter --protodir . --mock=false --nogomod

import (
	"context"

	trpc "trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/examples/features/sendonly/proto/greeter"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Create a server.
	s := trpc.NewServer()

	// Register greeterService into the server.
	pb.RegisterGreeterService(s.Service("trpc.examples.sendonly.greeter"), &greeterService{})

	// Start the server.
	if err := s.Serve(); err != nil {
		log.Fatalf("server serving: %v", err)
	}
}

type greeterService struct{}

func (*greeterService) Notify(ctx context.Context, req *pb.NotifyRequest) (*pb.NotifyResponse, error) {
	log.Infof("request message: %s", req.Message)
	return &pb.NotifyResponse{Message: "ack"}, nil
}
