// Package main is the server main package for loadbalance demo.
package main

import (
	"context"
	"fmt"
	"log"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// ServerImpl implements service.
type ServerImpl struct {
	pb.GreeterService
}

// SayHello sends sayhello request.
func (s *ServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	if req.Msg == "" {
		err := errs.New(10001, "missing required field: Msg")
		return nil, err
	}
	log.Printf("Received msg from client : %v", req.Msg)
	return &pb.HelloReply{Msg: "Hello " + req.Msg}, nil
}

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register service.
	pb.RegisterGreeterService(s, &ServerImpl{})

	// Serve and listen.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
