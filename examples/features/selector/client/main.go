// Package main is the client main package for selector demo.
package main

import (
	"errors"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var (
	exampleScheme      = "example"                        // selector name
	exampleServiceName = "trpc.examples.selector.example" // service name
)

func init() {
	// register selector
	selector.Register(exampleScheme, &exampleSelector{})
}

func main() {
	// init server
	_ = trpc.NewServer()

	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	// Send normal request.
	rsp, err := pb.NewGreeterClientProxy().SayHello(ctx, req)
	if err != nil {
		log.ErrorContextf(ctx, "SayHello err[%v] req[%s]", err, req.String())
		return
	}
	log.InfoContextf(ctx, "SayHello success rsp[%s]", rsp.String())
}

// Simulate the implementation of a selector.
type exampleSelector struct{}

// Select implements the selector plugin interface here, get a backend node by service name.
func (s *exampleSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
	if serviceName == exampleServiceName {
		return &registry.Node{
			Address: "127.0.0.1:8000",
		}, nil
	}
	return nil, errors.New("no available node")
}

// Report implements the selector plugin interface here, but do not provide
// specific implementation for the interface functionality.
func (s *exampleSelector) Report(node *registry.Node, cost time.Duration, success error) error {
	return nil
}
