// Package main is the client main package for log demo.
package main

import (
	"context"
	"log"
	"time"

	"trpc.group/trpc-go/trpc-go/client"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var addr = "ip://127.0.0.1:8080"

func main() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	// Init proxy: set up a connection to the server.
	clientProxy := pb.NewGreeterClientProxy(client.WithTarget(addr))

	// Send SayHello request.
	rsp, err := clientProxy.SayHello(ctx, &pb.HelloRequest{Msg: "Hello"})
	if err != nil {
		log.Printf("Received error: %v", err)
	}
	log.Printf("Received response: %s", rsp.Msg)
}
