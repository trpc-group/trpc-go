// Package main is the client main package.
package main

import (
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// init server
	_ = trpc.NewServer()

	ctx := trpc.BackgroundContext()
	// call service
	hello, err := pb.NewGreeterClientProxy().SayHello(ctx, &pb.HelloRequest{
		Msg: "client",
	})
	// handle error
	if err != nil {
		log.Fatal(err)
	}
	// print response
	log.Infof("recv rsp:%s", hello.Msg)
}
