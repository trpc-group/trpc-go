// Package main is the server main package.
package main

import (
	"context"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/examples/features/plugin"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"

	// import plugin
	_ "trpc.group/trpc-go/trpc-go/examples/features/plugin"
)

func main() {
	// init server
	s := trpc.NewServer()

	// register service
	pb.RegisterGreeterService(s, new(greeterImpl))

	// serve and listen
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}

type greeterImpl struct {
	common.GreeterServerImpl
}

// SayHello say hello request
// rewrite SayHello
func (g *greeterImpl) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Info("[Plugin] trpc-go-server SayHello, req.msg:", req.Msg)

	// call plugin
	plugin.Record()

	rsp := &pb.HelloReply{}

	rsp.Msg = "[Plugin] trpc-go-server response: Hello " + req.Msg

	return rsp, nil
}
