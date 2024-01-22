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
	"sync"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/examples/features/common"

	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	// Parse configuration files in yaml format.
	// Load default codec is `yaml` and provider is `file`
	c, err := config.Load("custom.yaml", config.WithCodec("yaml"), config.WithProvider("file"))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("test : %s \n", c.GetString("custom.test", ""))
	fmt.Printf("key1 : %s \n", c.GetString("custom.test_obj.key1", ""))
	fmt.Printf("key2 : %t \n", c.GetBool("custom.test_obj.key2", false))
	fmt.Printf("key2 : %d \n", c.GetInt32("custom.test_obj.key3", 0))

	// print
	// test : customConfigFromServer
	// key1 : value1
	// key2 : true
	// key3 : 1234

	// The format of the configuration file corresponds to custom struct.
	var custom customStruct
	if err := c.Unmarshal(&custom); err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Get config - custom : %v \n", custom)
	// print: Get config - custom : {{customConfigFromServer {value1 true 1234}}}

	// Init server.
	s := trpc.NewServer()

	config.RegisterProvider(p)
	// Register service.
	imp := &greeterImpl{}
	imp.once, _ = config.Load(p.Name(), config.WithProvider(p.Name()))
	imp.watch, _ = config.Load(p.Name(), config.WithProvider(p.Name()), config.WithWatch())

	pb.RegisterGreeterService(s, imp)

	// Serve and listen.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}

}

const cf = `custom :
  test : number_%d
  test_obj :
    key1 : value_%d
    key2 : %t
    key3 : %d`

var p = &provider{}

// mock provider to trigger config change
type provider struct {
	mu        sync.Mutex
	data      []byte
	num       int
	callbacks []config.ProviderCallback
}

func (p *provider) Name() string {
	return "test"
}

func (p *provider) Read(s string) ([]byte, error) {
	if s != p.Name() {
		return nil, fmt.Errorf("not found config %s", s)
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.data == nil {
		p.num++
		p.data = []byte(fmt.Sprintf(cf, p.num, p.num, p.num%2 == 0, p.num))
	}
	return p.data, nil
}

func (p *provider) Watch(callback config.ProviderCallback) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.callbacks = append(p.callbacks, callback)
}

func (p *provider) update() {
	p.mu.Lock()
	p.num++
	p.data = []byte(fmt.Sprintf(cf, p.num, p.num, p.num%2 == 0, p.num))
	callbacks := p.callbacks
	p.mu.Unlock()
	for _, callback := range callbacks {
		callback(p.Name(), p.data)
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

	once  config.Config
	watch config.Config
}

// SayHello say hello request. Rewrite SayHello to inform server config.
func (g *greeterImpl) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	fmt.Printf("trpc-go-server SayHello, req.msg:%s\n", req.Msg)

	if req.Msg == "change config" {
		p.update()
	}

	rsp := &pb.HelloReply{}
	rsp.Msg = "trpc-go-server response: Hello " + req.Msg +
		fmt.Sprintf("\nload once config: %s", g.once.GetString("custom.test", "")) +
		fmt.Sprintf("\nstart watch config: %s", g.watch.GetString("custom.test", ""))

	fmt.Printf("trpc-go-server SayHello, rsp.msg:%s\n", rsp.Msg)

	return rsp, nil
}

// first print
//
// trpc-go-server SayHello, rsp.msg:trpc-go-server response: Hello trpc-go-client
// load once config: number_1
// start watch config:number_1
//
// second print
//
// trpc-go-server SayHello, req.msg:change config
// trpc-go-server SayHello, rsp.msg:trpc-go-server response: Hello change config
// load once config: number_1
// start watch config:number_2
