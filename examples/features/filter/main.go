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

// Package main is the main package.
package main

import (
	"context"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// Create a server.
	s := trpc.NewServer(server.WithFilter(serverFilter))
	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})
	// Start serving.
	s.Serve()
}

func serverFilter(ctx context.Context, req interface{}, f filter.ServerHandleFunc) (interface{}, error) {
	log.Debug("begin server filter")
	defer log.Debug("end server filter")
	msg := codec.Message(ctx)
	md := msg.ServerMetaData()
	// Extract metadata for processing in the filter.
	if md == nil {
		log.Debug("get filter msg nil")
		return f(ctx, req)
	}
	if string(md["test_filter"]) != "ok" {
		log.Debug("get filter msg error")
		return f(ctx, req)
	}
	log.Debugf("get test_filter : %s", string(md["test_filter"]))
	return f(ctx, req)
}
