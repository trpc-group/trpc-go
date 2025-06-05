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

package main

import (
	"context"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"

	pb "trpc.group/trpc-go/trpc-go/testdata"
)

func main() {
	cfg, err := trpc.LoadConfig("../trpc_go.yaml")
	if err != nil {
		log.Fatalf("load config fail: %+v", err)
	}
	trpc.SetGlobalConfig(cfg)

	if err := trpc.Setup(cfg); err != nil {
		log.Fatalf("setup error: %+v", err)
	}

	// 创建一个客户端调用代理，名词解释见客户端开发文档。
	proxy := pb.NewGreeterClientProxy(client.WithServiceName("trpc.test.helloworld.Greeter1"))
	// 填充请求参数。
	req := &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
	// 调用目标地址为前面启动的服务监听的地址，并使用 timeout_800 标签寻址配置。
	rsp, err := proxy.SayHello(context.Background(), req, client.WithTag("timeout_800"))
	if err != nil {
		log.Errorf("could not greet: %v", err)
	} else {
		log.Debugf("response: %v", rsp)
	}

	// 调用目标地址为前面启动的服务监听的地址，并使用 timeout_1500 标签寻址配置。
	rsp, err = proxy.SayHello(context.Background(), req, client.WithTag("timeout_1500"))
	if err != nil {
		log.Errorf("could not greet: %v", err)
	} else {
		log.Debugf("response: %v", rsp)
	}
}
