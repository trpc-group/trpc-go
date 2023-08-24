// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var (
	exampleScheme      = "example"
	exampleServiceName = "selector.example.trpc.test"
)

func TestClient(t *testing.T) {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	node := &registry.Node{}
	rsphead := &trpcpb.ResponseProtocol{}
	opts := []client.Option{
		client.WithServiceName("trpc.test.helloworld"),
		client.WithProtocol("trpc"),
		client.WithNetwork("tcp4"),
		client.WithTarget(fmt.Sprintf("%s://%s", exampleScheme, exampleServiceName)),
		client.WithSelectorNode(node),
		client.WithRspHead(rsphead),
	}

	proxy := pb.NewGreeterClientProxy()

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := proxy.SayHello(ctx, req, opts...)
	log.Debugf("req:%s, rsp:%s, err:%v", req, rsp, err)
	assert.NotNil(t, err)
}

func init() {
	selector.Register(exampleScheme, &exampleSelector{})
}

type exampleSelector struct{}

// Select 通过 service name 获取一个后端节点
func (s *exampleSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
	fmt.Println(serviceName)
	if serviceName == exampleServiceName {
		return &registry.Node{
			Address: "127.0.0.1:8000",
		}, nil
	}

	return nil, errors.New("no available node")
}

// Report 上报当前请求成功或失败
func (s *exampleSelector) Report(node *registry.Node, cost time.Duration, success error) error {
	return nil
}
