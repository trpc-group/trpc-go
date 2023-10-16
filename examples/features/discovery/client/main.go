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

// Package main is the client main package for discovery demo.
package main

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/registry"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var serviceAddrMap sync.Map

const (
	exampleScheme      = "discoveryExample"
	exampleServiceName = "trpc.examples.discovery.Discovery"
)

type exampleDiscovery struct{}

// List returns node list of the server.
func (*exampleDiscovery) List(serviceName string, opt ...discovery.Option) ([]*registry.Node, error) {
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

func main() {
	// Service address map.
	serviceAddrMap.Store(exampleServiceName, []string{
		"127.0.0.1:8000",
		"127.0.0.1:8001",
		"127.0.0.1:8002",
	})

	// Register server using service address map.
	discovery.Register(exampleScheme, &exampleDiscovery{})

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	// Init proxy.
	clientProxy := pb.NewGreeterClientProxy(client.WithServiceName(exampleServiceName),
		client.WithDiscoveryName(exampleScheme))

	// Send 10 SayHello requests.
	for i := 0; i < 10; i++ {
		req := &pb.HelloRequest{
			Msg: "trpc-go-client " + strconv.Itoa(i),
		}
		if _, err := clientProxy.SayHello(ctx, req); err != nil {
			log.Printf("Received error from client "+strconv.Itoa(i)+": %v", err)
		}
	}
}
