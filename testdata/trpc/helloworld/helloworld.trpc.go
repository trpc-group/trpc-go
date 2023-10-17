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

// Code generated by trpc-go/trpc-cmdline. DO NOT EDIT.
// source: helloworld.proto

package helloworld

import (
	"context"
	"fmt"

	_ "trpc.group/trpc-go/trpc-go"
	_ "trpc.group/trpc-go/trpc-go/http"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/server"
)

/* ************************************ Service Definition ************************************ */

// GreeterService defines service
type GreeterService interface {
	SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error)
	SayHi(ctx context.Context, req *HelloRequest) (*HelloReply, error)
}

func GreeterService_SayHello_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (rspBody interface{}, err error) {
	req := &HelloRequest{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	if len(filters) == 0 {
		return svr.(GreeterService).SayHello(ctx, req)
	}
	handleFunc := func(ctx context.Context, reqBody interface{}) (interface{}, error) {
		return svr.(GreeterService).SayHello(ctx, reqBody.(*HelloRequest))
	}
	return filters.Filter(ctx, req, handleFunc)
}

func GreeterService_SayHi_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (rspBody interface{}, err error) {
	req := &HelloRequest{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, reqBody interface{}) (interface{}, error) {
		return svr.(GreeterService).SayHi(ctx, reqBody.(*HelloRequest))
	}

	return filters.Filter(ctx, req, handleFunc)
}

// GreeterServer_ServiceDesc descriptor for server.RegisterService
var GreeterServer_ServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.test.helloworld.Greeter",
	HandlerType: ((*GreeterService)(nil)),
	Methods: []server.Method{
		{
			Name: "/trpc.test.helloworld.Greeter/SayHello",
			Func: GreeterService_SayHello_Handler,
		},
		{
			Name: "/trpc.test.helloworld.Greeter/SayHi",
			Func: GreeterService_SayHi_Handler,
		},
	},
}

// RegisterGreeterService register service
func RegisterGreeterService(s server.Service, svr GreeterService) {
	if err := s.Register(&GreeterServer_ServiceDesc, svr); err != nil {
		panic(fmt.Sprintf("Greeter register error:%v", err))
	}

}

/* ************************************ Client Definition ************************************ */

// GreeterClientProxy defines service client proxy
type GreeterClientProxy interface {
	SayHello(ctx context.Context, req *HelloRequest, opts ...client.Option) (rsp *HelloReply, err error)

	SayHi(ctx context.Context, req *HelloRequest, opts ...client.Option) (rsp *HelloReply, err error)
}

type GreeterClientProxyImpl struct {
	client client.Client
	opts   []client.Option
}

var NewGreeterClientProxy = func(opts ...client.Option) GreeterClientProxy {
	return &GreeterClientProxyImpl{client: client.DefaultClient, opts: opts}
}

func (c *GreeterClientProxyImpl) SayHello(ctx context.Context, req *HelloRequest, opts ...client.Option) (rsp *HelloReply, err error) {

	ctx, msg := codec.WithCloneMessage(ctx)

	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithCalleeServiceName(GreeterServer_ServiceDesc.ServiceName)
	msg.WithCalleeApp("test")
	msg.WithCalleeServer("helloworld")
	msg.WithCalleeService("Greeter")
	msg.WithCalleeMethod("SayHello")
	msg.WithSerializationType(codec.SerializationTypePB)

	callopts := make([]client.Option, 0, len(c.opts)+len(opts))
	callopts = append(callopts, c.opts...)
	callopts = append(callopts, opts...)

	rsp = &HelloReply{}

	err = c.client.Invoke(ctx, req, rsp, callopts...)
	if err != nil {
		return nil, err
	}
	codec.PutBackMessage(msg)

	return rsp, nil
}

func (c *GreeterClientProxyImpl) SayHi(ctx context.Context, req *HelloRequest, opts ...client.Option) (rsp *HelloReply, err error) {

	ctx, msg := codec.WithCloneMessage(ctx)

	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHi")
	msg.WithCalleeServiceName(GreeterServer_ServiceDesc.ServiceName)
	msg.WithCalleeApp("test")
	msg.WithCalleeServer("helloworld")
	msg.WithCalleeService("Greeter")
	msg.WithCalleeMethod("SayHi")
	msg.WithSerializationType(codec.SerializationTypePB)

	callopts := make([]client.Option, 0, len(c.opts)+len(opts))
	callopts = append(callopts, c.opts...)
	callopts = append(callopts, opts...)

	rsp = &HelloReply{}

	err = c.client.Invoke(ctx, req, rsp, callopts...)
	if err != nil {
		return nil, err
	}
	codec.PutBackMessage(msg)

	return rsp, nil
}
