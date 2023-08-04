// Package main provides an echo client.
package main

import (
	"bytes"
	"io"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/examples/features/attachment/proto/echo"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Create an attachment.
	a := client.NewAttachment(bytes.NewReader([]byte("client attachment")))

	// Call UnaryEcho that send attachment along with messages.
	c := pb.NewEchoClientProxy(client.WithTarget("ip://127.0.0.1:8000"))
	rsp, err := c.UnaryEcho(trpc.BackgroundContext(), &pb.EchoRequest{Message: "message"}, client.WithAttachment(a))
	if err != nil {
		log.Errorf("calling UnaryEcho: %v", err)
		return
	}
	log.Infof("received message: %s", rsp.GetMessage())

	// Read attachment returned from the server.
	attachment, err := io.ReadAll(a.Response())
	if err != nil {
		log.Errorf("reading attachment: %v", err)
	}
	log.Infof("received attachment: %s", attachment)
}
