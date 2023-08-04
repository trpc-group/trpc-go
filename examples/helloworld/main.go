// Package main is the main package.
package main

import (
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// Create a server and register a service.
	s := trpc.NewServer()
	pb.RegisterGreeterService(s, greeter)
	// Start serving.
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
