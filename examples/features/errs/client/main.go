// Package main is the client main package for error demo.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"trpc.group/trpc-go/trpc-go/client"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var addr = flag.String("addr", "ip://127.0.0.1:8000", "the address to connect to")

func main() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	// Init proxy.
	clientProxy := pb.NewGreeterClientProxy(client.WithTarget(*addr))

	// Send SayHello request.
	for _, reqMsg := range []string{"trpc-go-client", ""} {
		log.Printf("Calling SayHello with Name:%q", reqMsg)
		rsp, err := clientProxy.SayHello(ctx, &pb.HelloRequest{Msg: reqMsg})
		if err != nil {
			log.Printf("Received error: %v", err)
			continue
		}
		log.Printf("Received response: %s", rsp.Msg)
	}

	// Send SayHello request with nil.
	_, err := clientProxy.SayHello(ctx, nil)
	if err != nil {
		log.Printf("Received error: %v", err)
	}
}
