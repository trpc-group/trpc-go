// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var addr = "ip://127.0.0.1:8000"

func TestSayHello(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithTarget(addr),
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
		client.WithTarget(addr),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHi(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}
