// Package main is the client main package for config demo.
package main

import (
	"context"
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var addr = "ip://127.0.0.1:8000"

func main() {
	ctx, _ := context.WithTimeout(context.TODO(), time.Millisecond*2000)

	// Init proxy.
	clientProxy := pb.NewGreeterClientProxy(client.WithTarget(addr))

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	// Send request.
	rsp, err := clientProxy.SayHello(ctx, req)
	if err != nil {
		fmt.Println("Say hi err:%v", err)
		return
	}
	fmt.Printf("Get msg: %s\n", rsp.GetMsg())
}
