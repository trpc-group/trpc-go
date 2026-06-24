// Package main provides an echo client.
package main

import (
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/examples/features/sendonly/proto/greeter"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	c := pb.NewGreeterClientProxy(client.WithTarget("ip://127.0.0.1:8000"))
	rsp, err := c.Notify(
		trpc.BackgroundContext(),
		&pb.NotifyRequest{Message: "hello"},
		client.WithSendOnly(),
		client.WithNetwork("udp"),
	)
	if err != nil {
		log.Errorf("calling Notify: %v", err)
		return
	}
	if len(rsp.Message) == 0 {
		log.Info(" send only successfully, message of response is empty")
	} else {
		log.Errorf(" send only failed, message of response is %s", rsp.GetMessage())
	}
}
