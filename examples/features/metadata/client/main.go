// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package main is the main package.
package main

import (
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// Load config.
	trpc.NewServer()

	proxy := pb.NewGreeterClientProxy()

	// Call SayHello.
	// Client obtain server metadata by setting a response head of each protocol.
	helloHead := &trpc.ResponseProtocol{}
	sayHelloOpts := []client.Option{
		client.WithMetaData("key1", []byte("val1")),
		client.WithMetaData("key2", []byte("val2")),
		client.WithMetaData("say-hello-client", []byte("hello")),
		client.WithRspHead(helloHead),
	}
	_, err := proxy.SayHello(trpc.BackgroundContext(), &pb.HelloRequest{Msg: "trpc-go-client"}, sayHelloOpts...)
	if err != nil {
		log.Error(err)
	}
	// Get key-value pairs from TransInfo that transmitted by the framework (map[string][]byte).
	log.Debugf("say hello trans info: key: say-hello-server, val: %s", string(helloHead.TransInfo["say-hello-server"]))

	// Call SayHi.
	hiHead := &trpc.ResponseProtocol{}
	sayHiOpts := []client.Option{
		client.WithMetaData("key1", []byte("val1")),
		client.WithMetaData("key2", []byte("val2")),
		client.WithMetaData("say-hi-client", []byte("hi")),
		client.WithRspHead(hiHead),
	}
	_, err = proxy.SayHi(trpc.BackgroundContext(), &pb.HelloRequest{Msg: "trpc-go-client"}, sayHiOpts...)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("say hi trans info: key: say-hi-server, val: %s", string(hiHead.TransInfo["say-hi-server"]))
}
