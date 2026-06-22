// Package main provides an echo client.
package main

import (
	"io"
	"strconv"
	"sync"
	"time"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/examples/features/stop/proto/echo"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		testBidiStreamingRPC()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		testBidiStreamingRPC()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		testBidiStreamingRPC()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		testUnaryRPC()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		testServerStreamingRPC()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		testClientStreamingRPC()
	}()
	wg.Wait()
}

func testServerStreamingRPC() {
	c := pb.NewTestStreamingClientProxy(client.WithTarget("ip://127.0.0.1:9001"))
	stream, err := c.StreamingOutputCall(trpc.BackgroundContext(), &pb.StreamingFullDuplexCallRequest{
		Message: "server-streaming",
	})
	if err != nil {
		log.Errorf("newing stream from StreamingOutputCall failed: %v", err)
		return
	}
	for {
		reply, err := stream.Recv()
		if err == io.EOF {
			log.Infof("StreamingOutputCall received EOF\n")
			break
		}
		if err != nil {
			log.Errorf("StreamingOutputCall receive error from server : %v", err)
			return
		}
		log.Infof("StreamingOutputCall receive message from server is %s\n", reply.Message)
		time.Sleep(time.Second)
	}
}

func testClientStreamingRPC() {
	c := pb.NewTestStreamingClientProxy(client.WithTarget("ip://127.0.0.1:9001"))
	stream, err := c.StreamingInputCall(trpc.BackgroundContext())
	if err != nil {
		log.Errorf("newing stream from StreamingInputCall failed: %v", err)
		return
	}
	for i := 0; i < 10; i++ {
		msg := "client-streaming" + strconv.Itoa(i)
		if err := stream.Send(&pb.StreamingFullDuplexCallRequest{Message: msg}); err != nil {
			log.Errorf("StreamingInputCall send message %s failed: %v", msg, err)
			return
		}
		time.Sleep(time.Second)
	}
	reply, err := stream.CloseAndRecv()
	if err != nil {
		log.Errorf("StreamingInputCall close and receive failed: %v", err)
		return
	}
	log.Infof("StreamingInputCall receive message from server is %s\n", reply.Message)
}

func testBidiStreamingRPC() {
	c := pb.NewTestStreamingClientProxy(client.WithTarget("ip://127.0.0.1:9001"))
	stream, err := c.FullDuplexCall(trpc.BackgroundContext())
	if err != nil {
		log.Errorf("newing stream from FullDuplexCall failed: %v", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func(sendStream pb.TestStreaming_FullDuplexCallClient) {
		defer wg.Done()
		for i := 0; i < 60; i++ {
			msg := "test" + strconv.Itoa(i)
			if err := sendStream.Send(&pb.StreamingFullDuplexCallRequest{
				Message: msg,
			}); err != nil {
				log.Errorf("sendStream send message %s failed: %v", msg, err)
			}
			time.Sleep(time.Second)
		}
		if err := sendStream.CloseSend(); err != nil {
			log.Errorf("sendStream closeSend failed: %v", err)
			return
		}
	}(stream)

	wg.Add(1)
	go func(recvStream pb.TestStreaming_FullDuplexCallClient) {
		defer wg.Done()
		for {
			reply, err := recvStream.Recv()
			// 注意这里不能使用 errors.Is(err, io.EOF) 来判断流结束
			if err == io.EOF {
				log.Infof("FullDuplexCall received EOF\n")
				break
			}
			if err != nil {
				log.Errorf("FullDuplexCall receive error from server : %v", err)
				return
			}

			log.Infof("FullDuplexCall receive message from server is %s\n", reply.Message)
			time.Sleep(time.Second)
		}
	}(stream)

	wg.Wait()
}

func testUnaryRPC() {
	for i := 0; i < 60; i++ {
		c := pb.NewEchoClientProxy(client.WithTarget("ip://127.0.0.1:9000"), client.WithMultiplexed(true))
		rsp, err := c.UnaryEcho(trpc.BackgroundContext(), &pb.EchoRequest{Message: "unary rpc test " + strconv.Itoa(i)})
		if err != nil {
			log.Errorf("calling UnaryEcho: %v", err)
		} else {
			log.Infof("UnaryEcho received message: %s", rsp.GetMessage())
		}
		time.Sleep(time.Second)
	}
}
