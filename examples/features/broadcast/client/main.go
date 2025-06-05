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

// Package main is the client main package for broadcast demo.
package main

import (
	"context"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
	"trpc.group/trpc-go/trpc-go/transport"

	pb "trpc.group/trpc-go/trpc-go/examples/features/broadcast/proto"
)

const (
	FakeDiscovery      = "fake_discovery"
	FakeServiceRouter  = "fake_service_router"
	FakeTransportName  = "fake_transport"
	exampleServiceName = "trpc.examples.broadcast.example"
)

var serviceAddrMap sync.Map

func init() {
	// Service address map.
	serviceAddrMap.Store(exampleServiceName, []string{
		"127.0.0.1:8000",
		"127.0.0.1:8001",
		"127.0.0.1:8002",
	})

	// Register.
	discovery.Register(FakeDiscovery, &fakeDiscovery{})
	discovery.DefaultDiscovery = &fakeDiscovery{}
	servicerouter.Register(FakeServiceRouter, &fakeServiceRouter{})
	servicerouter.DefaultServiceRouter = &fakeServiceRouter{}
	transport.RegisterClientTransport(FakeTransportName, &fakeTransport{})
	transport.DefaultClientTransport = &fakeTransport{}
}

func main() {
	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}

	// Init proxy and broadcast.
	clientProxy := pb.NewGreeterClientProxy(client.WithServiceName("trpc.examples.broadcast.example"))
	replies, err := clientProxy.BroadcastSayHello(context.Background(), req)

	if err != nil {
		log.Errorf("Fail to Broadcast: %v", err)
	}
	// handle responses
	for _, reply := range replies {
		if reply.Err != nil {
			log.Errorf("error from node %s: %v", reply.Node.Address, reply.Err)
		} else {
			log.Debugf("broadcast rpc receive from node: %s, with: %+v", reply.Node.Address, reply.Rsp)
		}
	}
}

// ================================================================ //
type fakeDiscovery struct{}

func (d *fakeDiscovery) List(serviceName string, opt ...discovery.Option) ([]*registry.Node, error) {
	var registryNodes []*registry.Node

	if serviceAddr, ok := serviceAddrMap.Load(exampleServiceName); ok {
		if addrs, ok := serviceAddr.([]string); ok {
			for _, addr := range addrs {
				registryNodes = append(registryNodes, &registry.Node{
					ServiceName: serviceName,
					Address:     addr,
				})
			}
		}
	}
	return registryNodes, nil
}

type fakeServiceRouter struct{}

func (r *fakeServiceRouter) Filter(serviceName string, nodes []*registry.Node, opt ...servicerouter.Option) ([]*registry.Node, error) {
	opts := &servicerouter.Options{}
	for _, o := range opt {
		o(opts)
	}
	if opts.Broadcast {
		return nodes, nil
	}
	return nodes, nil
}

type fakeTransport struct {
	send  func() error
	recv  func() ([]byte, error)
	close func()
}

func (c *fakeTransport) RoundTrip(ctx context.Context, req []byte,
	roundTripOpts ...transport.RoundTripOption) (rsp []byte, err error) {
	time.Sleep(time.Millisecond * 2)
	return req, nil
}

func (c *fakeTransport) Send(ctx context.Context, req []byte, opts ...transport.RoundTripOption) error {
	if c.send != nil {
		return c.send()
	}
	return nil
}

func (c *fakeTransport) Recv(ctx context.Context, opts ...transport.RoundTripOption) ([]byte, error) {
	if c.recv != nil {
		return c.recv()
	}
	return []byte("test"), nil
}

func (c *fakeTransport) Init(ctx context.Context, opts ...transport.RoundTripOption) error {
	return nil
}
func (c *fakeTransport) Close(ctx context.Context) {
	if c.close != nil {
		c.close()
	}
}
