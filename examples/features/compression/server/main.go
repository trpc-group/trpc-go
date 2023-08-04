// Package main is the main package.
package main

import (
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	s := trpc.NewServer()

	// register service
	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})

	// start serve
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
