// Package main provides an echo server.
package main

//go:generate trpc create -p ../proto/echo/echo.proto -o ../proto/echo --alias --protocol http --api-version 2 --rpconly --mock=false --nogomod=true

import (
	"context"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	pb "trpc.group/trpc-go/trpc-go/examples/features/httprpc/proto/echo"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Use custom JSON aliases from the proto file.
	codec.Marshaler.UseProtoNames = false

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
		Code:    0,
		Message: request.Message,
	}, nil
}
