// Package main is the server main package for log demo.
package main

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// GreeterServerImpl service implement
type GreeterServerImpl struct {
	pb.GreeterService
}

// SayHello say hello request
func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := &pb.HelloReply{}

	rsp.Msg = fmt.Sprintf("trpc-go-server response: %s", req.Msg)

	log.Tracef("recv msg:%s", req)
	log.Debugf("recv msg:%s", req)
	log.Infof("recv msg:%s", req)
	log.Warnf("recv msg:%s", req)
	log.Errorf("recv msg:%s", req)

	return rsp, nil
}

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register service.
	pb.RegisterGreeterService(s, &GreeterServerImpl{})

	// Serve and listen.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
