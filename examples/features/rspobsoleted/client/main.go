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
	"flag"

	"trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/examples/features/rspobsoleted/proto"
	"trpc.group/trpc-go/trpc-go/log"
)

func init() {
	flag.StringVar(&trpc.ServerConfigPath, "conf", "./trpc_go.yaml", "trpc-go yaml path")
}

func main() {
	flag.Parse()
	// Configurations are loaded following the logic of trpc.NewServer.
	cfg, err := trpc.LoadConfig(trpc.ServerConfigPath)
	if err != nil {
		panic("load config fail: " + err.Error())
	}
	trpc.SetGlobalConfig(cfg)
	if err := trpc.Setup(cfg); err != nil {
		panic("setup plugin fail: " + err.Error())
	}
	// Create client proxy.
	proxy := pb.NewRspObsoletedExampleClientProxy()
	ctx := trpc.BackgroundContext()
	// Do RPC call.
	reply, err := proxy.Hello(ctx, &pb.Request{Msg: []byte("helloworld")})
	if err != nil {
		log.Fatalf("err: %v", err)
	}
	log.Debugf("simple  rpc   receive: %q", reply.Msg)
}
