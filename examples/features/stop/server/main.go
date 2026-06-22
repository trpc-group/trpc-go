// Package main provides an echo server.
package main

//go:generate trpc create -p ../proto/echo/echo.proto --api-version 2 --rpconly -o ../proto/echo --protodir . --mock=false

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/examples/features/stop/proto/echo"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Create a server.
	s := trpc.NewServer()

	// Register echoService into the server.
	pb.RegisterEchoService(s.Service("trpc.examples.stop.test-unary-rpc"), &echoService{})
	pb.RegisterTestStreamingService(s.Service("trpc.examples.stop.test-streaming"), &testStreaming{})

	// Start the server.
	if err := s.Serve(); err != nil {
		log.Fatalf("server serving: %v", err)
	}
}

type testStreaming struct{}

func (s *testStreaming) StreamingOutputCall(req *pb.StreamingFullDuplexCallRequest, stream pb.TestStreaming_StreamingOutputCallServer) error {
	for i := 0; i < 60; i++ {
		msg := fmt.Sprintf("%s-server-stream-%d", req.Message, i)
		if err := stream.Send(&pb.StreamingFullDuplexCallResponse{Message: msg}); err != nil {
			log.Errorf("StreamingOutputCall send message %s failed: %v", msg, err)
			return err
		}
		time.Sleep(time.Second)
	}
	return nil
}

func (s *testStreaming) StreamingInputCall(stream pb.TestStreaming_StreamingInputCallServer) error {
	var messages []string
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			time.Sleep(5 * time.Second)
			return stream.SendAndClose(&pb.StreamingFullDuplexCallResponse{Message: strings.Join(messages, ";")})
		}
		if err != nil {
			return err
		}

		log.Infof("StreamingInputCall receive message from client: %s", in.Message)
		messages = append(messages, in.Message)
	}
}

func (s *testStreaming) FullDuplexCall(stream pb.TestStreaming_FullDuplexCallServer) error {
	var messages []string
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.Send(&pb.StreamingFullDuplexCallResponse{Message: strings.Join(messages, ";")})
		}
		if err != nil {
			return err
		}

		log.Infof("FullDuplexCall receive message from client: %s", in.Message)
		messages = append(messages, in.Message)
		if err := stream.Send(&pb.StreamingFullDuplexCallResponse{Message: in.Message}); err != nil {
			log.Errorf("send message %s failed", in.Message)
		}
		time.Sleep(time.Second)
	}
}

type echoService struct{}

// UnaryEcho echos request's message and attachment.
func (s *echoService) UnaryEcho(_ context.Context, request *pb.EchoRequest) (*pb.EchoResponse, error) {
	log.Infof("UnaryEcho receive message from client: %s", request.GetMessage())
	return &pb.EchoResponse{Message: request.GetMessage()}, nil
}
