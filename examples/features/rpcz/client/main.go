// Package main this file is RPCZ client samples.
package main

import (
	"trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/examples/features/rpcz/proto"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Create a service object, read config file.
	trpc.NewServer()

	ctx := trpc.BackgroundContext()
	rsp, err := pb.NewRPCZClientProxy().Hello(ctx, &pb.HelloReq{Msg: "111"})
	if err != nil {
		log.ErrorContextf(ctx, "Error in SayHello: %v", err)
		return
	}

	log.InfoContextf(ctx, "SayHello reply message is: %s", rsp.GetMsg())
}
