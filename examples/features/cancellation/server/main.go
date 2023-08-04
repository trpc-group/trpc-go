// Package main is the server main package for cancellation demo.
package main

import (
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register service.
	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})

	// Serve and listen.
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}
