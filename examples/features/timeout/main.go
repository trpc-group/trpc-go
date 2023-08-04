// Package main is the main package.
package main

import (
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	s := trpc.NewServer()
	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})
	s.Serve()
}
