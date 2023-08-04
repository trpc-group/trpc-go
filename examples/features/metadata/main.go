// Package main is the main package.
package main

import (
	"context"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// Create a server.
	s := trpc.NewServer(server.WithFilter(getMetaData))
	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})
	// Start serving.
	s.Serve()
}

func getMetaData(ctx context.Context, req interface{}, f filter.ServerHandleFunc) (interface{}, error) {
	msg := codec.Message(ctx)
	md := msg.ServerMetaData()
	// Extract metadata for processing in the filter.
	for k, v := range md {
		log.Debugf("get metadata key : %s, value : %s", k, string(v))
	}
	return f(ctx, req)
}
