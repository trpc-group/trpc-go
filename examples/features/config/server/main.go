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

// Package main is the server main package for config demo.
package main

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/examples/features/common"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// Parse configuration files in yaml format.
	conf, err := config.Load("server/custom.yaml", config.WithCodec("yaml"), config.WithProvider("file"))
	if err != nil {
		fmt.Println(err)
		return
	}

	// The format of the configuration file corresponds to custom struct.
	var custom customStruct
	if err := conf.Unmarshal(&custom); err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Get config - custom : %v \n", custom)

	fmt.Printf("test : %s \n", conf.GetString("custom.test", ""))
	fmt.Printf("key1 : %s \n", conf.GetString("custom.test_obj.key1", ""))
	fmt.Printf("key2 : %t \n", conf.GetBool("custom.test_obj.key2", false))
	fmt.Printf("key2 : %d \n", conf.GetInt32("custom.test_obj.key3", 0))

	// Init server.
	s := trpc.NewServer()

	// Register service.
	greeterImpl := &greeterImpl{
		customConf: conf.GetString("custom.test", ""),
	}
	pb.RegisterGreeterService(s, greeterImpl)

	// Serve and listen.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}

}

// customStruct it defines the struct of the custom configuration file read.
type customStruct struct {
	Custom struct {
		Test    string `yaml:"test"`
		TestObj struct {
			Key1 string `yaml:"key1"`
			Key2 bool   `yaml:"key2"`
			Key3 int32  `yaml:"key3"`
		} `yaml:"test_obj"`
	} `yaml:"custom"`
}

// greeterImpl it implements `pb.RegisterGreeterService` interface, which is used to implement the service logic.
type greeterImpl struct {
	common.GreeterServerImpl

	customConf string
}

// SayHello say hello request. Rewrite SayHello to inform server config.
func (g *greeterImpl) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	fmt.Printf("trpc-go-server SayHello, req.msg:%s\n", req.Msg)

	rsp := &pb.HelloReply{}
	rsp.Msg = "trpc-go-server response: Hello " + req.Msg + ". Custom config from server: " + g.customConf
	fmt.Printf("trpc-go-server SayHello, rsp.msg:%s\n", rsp.Msg)

	return rsp, nil
}
