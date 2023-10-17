//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package main trpc-go supports stream RPC，with stream RPC,
// the client and server can establish a continuous connection to continuously send and receive data,
// thus allowing the server to provide continuous responses.
// this file is stream RPC client samples.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/examples/features/stream/proto"
	"trpc.group/trpc-go/trpc-go/log"
)

// streamType define stream type parameter and get.
var streamType = flag.String("type", "ClientStream", "Input Stream Type")

func main() {
	// Create a service object, read config file.
	trpc.NewServer()

	flag.Parse()

	log.Debugf("streamType : %v", *streamType)

	opts := []client.Option{
		// If you want to set the client receiving window size, use the client option `WithMaxWindowSize`.
		client.WithMaxWindowSize(1 * 1024 * 1024),
	}
	proxy := pb.NewTestStreamClientProxy(opts...)

	ctx := trpc.BackgroundContext()
	switch *streamType {
	case "ClientStream":
		// calling Client-side streaming func.
		err := clientStream(ctx, proxy)
		if err != nil {
			log.Error(err)
		}
	case "ServerStream":
		// calling Server-side streaming func.
		err := serverStream(ctx, proxy)
		if err != nil {
			log.Error(err)
		}
	case "BidirectionalStream":
		// calling Bidirectional streaming func.
		err := bidirectionalStream(ctx, proxy)
		if err != nil {
			log.Error(err)
		}
	default:
		log.Fatal("unknown stream type, please use ClientStream, ServerStream or BidirectionalStream")
	}
}

// clientStream Client-side streaming RPC client sample.
func clientStream(ctx context.Context, proxy pb.TestStreamClientProxy) error {
	// Use ClientStream function to Create a stream client, Different from a single RPC,
	// calling ClientStream does not need to pass in a request, and returns streamClient for send and recv.
	streamClient, err := proxy.ClientStream(ctx)
	if err != nil {
		log.ErrorContextf(ctx, "Error in ClientStream: %v", err)
		return err
	}

	for i := 0; i < 5; i++ {
		// Call Send to continuously send data.
		if err = streamClient.Send(&pb.HelloReq{Msg: fmt.Sprintf("ping : %v", i)}); err != nil {
			log.ErrorContextf(ctx, "ClientStream send error: %v", err)
			break
		}
	}

	// In Client-side streaming RPC mode, The server only returns once, so call CloseAndRecv to receive.
	rsp, err := streamClient.CloseAndRecv()
	if err != nil {
		return err
	}
	log.InfoContextf(ctx, "ClientStream reply message is: %s", rsp.GetMsg())
	return err
}

// serverStream Server-side streaming client sample.
func serverStream(ctx context.Context, proxy pb.TestStreamClientProxy) error {
	// In Server-side streaming RPC mode, The client directly fills in the parameters,
	// and the returned streamClient can be used to continuously receive the response from the server.
	streamClient, err := proxy.ServerStream(ctx, &pb.HelloReq{Msg: "ping"})
	if err != nil {
		log.ErrorContextf(ctx, "Error in ServerStream sayHello %v", err)
		return err
	}

	for {
		rsp, err := streamClient.Recv()
		// Note that errors.Is(err, io.EOF) cannot be used here to determine the end of the stream.
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		log.InfoContextf(ctx, "ServerStream reply message is: %s", rsp.GetMsg())
	}
	return nil
}

// bidirectionalStream Bidirectional streaming client sample.
func bidirectionalStream(ctx context.Context, proxy pb.TestStreamClientProxy) error {
	streamClient, err := proxy.BidirectionalStream(ctx)
	if err != nil {
		log.ErrorContextf(ctx, "Error in BidirectionalStream sayHello %v", err)
		return err
	}
	for i := 0; i < 5; i++ {
		// The client send request data to the server 5 times using a for loop.
		if err = streamClient.Send(&pb.HelloReq{Msg: "ping: " + strconv.Itoa(i)}); err != nil {
			log.ErrorContextf(ctx, "BidirectionalStream Send message error: %v", err)
			break
		}
	}

	// Call CloseSend to indicate that the request sending has ended.
	if err = streamClient.CloseSend(); err != nil {
		log.ErrorContextf(ctx, "BidirectionalStream CloseSend error is: %v", err)
		return err
	}
	for {
		// Continuously call Recv to receive server response.
		rsp, err := streamClient.Recv()
		// Note that errors.Is(err, io.EOF) cannot be used here to determine the end of the stream.
		if err == io.EOF {
			log.InfoContext(ctx, "BidirectionalStream EOF error, then close receive")
			break
		}
		if err != nil {
			log.ErrorContextf(ctx, "BidirectionalStream receive error from server: %v", err)
		}
		log.InfoContextf(ctx, "BidirectionalStream reply message is: %s", rsp.GetMsg())
	}
	return err
}
