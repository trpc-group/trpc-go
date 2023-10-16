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
// the client and server can establish a continuous connection to send and receive data continuously,
// allowing the server to provide continuous responses
// this file is stream RPC server samples.
package main

import (
	"fmt"
	"io"

	pb "trpc.group/trpc-go/trpc-go/examples/features/stream/proto"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
)

func main() {
	// Create a service object,
	// If you want to set the server receiving window size, use `server option WithMaxWindowSize`.
	s := trpc.NewServer(server.WithMaxWindowSize(1 * 1024 * 1024))

	// Register the current implementation into the service object.
	pb.RegisterTestStreamService(s, &testStreamImpl{})

	// Start the service and block here.
	if err := s.Serve(); err != nil {
		log.Fatalf("service serves error: %v", err)
	}
}

// testStreamImpl TestStream service implement.
type testStreamImpl struct {
	pb.UnimplementedTestStream
}

// ClientStream Client-side streaming,
// ClientStream passes pb.TestStream_ClientStreamServer as a parameter, returns error,
// pb.TestStream_ClientStreamServer provides interfaces such as Recv() and SendAndClose() for streaming interaction.
func (s *testStreamImpl) ClientStream(stream pb.TestStream_ClientStreamServer) error {
	for {
		// The server uses a for loop to recv data from the client.
		req, err := stream.Recv()
		// If EOF is returned, it means that the client stream has ended.
		if err == io.EOF {
			// Use `SendAndClose` function to send data and close the stream.
			log.Info("ClientStream receive EOF, then close receive and send pong message")
			return stream.SendAndClose(&pb.HelloRsp{Msg: "pong"})
		}
		if err != nil {
			log.Errorf("ClientStream receive error: %v", err)
			return err
		}
		log.Infof("ClientStream receive Msg: %s", req.GetMsg())
	}
}

// ServerStream Server-side streaming,
// passes in a request and pb.TestStream_ServerStreamServer as parameters, and returns an error,
// b.TestStream_ServerStreamServer provides Send() interface for streaming interaction.
func (s *testStreamImpl) ServerStream(req *pb.HelloReq, stream pb.TestStream_ServerStreamServer) error {
	log.Infof("ServerStream receive Msg: %s", req.GetMsg())
	for i := 0; i < 5; i++ {
		// Continuously call Send to send the response.
		if err := stream.SendMsg(&pb.HelloRsp{Msg: fmt.Sprintf(" pong: %v", i)}); err != nil {
			return err
		}
	}
	return nil
}

// BidirectionalStream Bidirectional streaming,
// BidirectionalStream passes pb.TestStream_BidirectionalStreamServer as a parameter, returns error,
// pb.TestStream_BidirectionalStreamServer provides interfaces
// such as Recv() and SendAndClose() for streaming interaction.
func (s *testStreamImpl) BidirectionalStream(stream pb.TestStream_BidirectionalStreamServer) error {
	for {
		// The server uses a for loop to recv data from the client.
		req, err := stream.Recv()
		// If EOF is returned, it means that the client stream has ended.
		if err == io.EOF {
			log.Info("BidirectionalStream EOF error, then close receive")
			return nil
		}
		if err != nil {
			log.Errorf("ClientStream receive error: %v", err)
			return err
		}
		log.Infof("BidirectionalStream receive Msg: %s", req.GetMsg())
		if err = stream.Send(&pb.HelloRsp{Msg: fmt.Sprintf("pong: :%v", req.GetMsg())}); err != nil {
			return err
		}
	}
}
