// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var target = "ip://127.0.0.1:8000"

var serviceAddrMap sync.Map

const (
	exampleScheme      = "example"
	exampleServiceName = "selector.example.trpc.test"
)

func init() {
	serviceAddrMap.Store(exampleServiceName, []string{
		"127.0.0.1:8000",
		"127.0.0.1:8001",
		"127.0.0.1:8002",
	})
	discovery.Register(exampleScheme, &exampleDiscovery{})
}

type exampleDiscovery struct{}

// List 获取节点列表
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

func TestSayHello(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		// client.WithTarget(target),
		client.WithServiceName(exampleServiceName),
		client.WithDiscoveryName(exampleScheme),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHello(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}

func TestSayHi(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		// client.WithTarget(target),
		client.WithServiceName(exampleServiceName),
		client.WithDiscoveryName(exampleScheme),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHi(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}
