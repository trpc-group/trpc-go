// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Code generated by trpc-go/trpc-cmdline v2.0.17. DO NOT EDIT.
// source: helloworld.proto

package proto

import (
	"context"
	"errors"
	"fmt"

	_ "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	_ "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/server"
)

// START ======================================= Server Service Definition ======================================= START

// RPCZService defines service
type RPCZService interface {
	// Hello Defined Hello RPC
	Hello(ctx context.Context, req *HelloReq) (*HelloRsp, error)
}

func RPCZService_Hello_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	req := &HelloReq{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, reqbody interface{}) (interface{}, error) {
		return svr.(RPCZService).Hello(ctx, reqbody.(*HelloReq))
	}

	var rsp interface{}
	rsp, err = filters.Filter(ctx, req, handleFunc)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

// RPCZServer_ServiceDesc descriptor for server.RegisterService
var RPCZServer_ServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.examples.rpcz.RPCZ",
	HandlerType: ((*RPCZService)(nil)),
	Methods: []server.Method{
		{
			Name: "/trpc.examples.rpcz.RPCZ/Hello",
			Func: RPCZService_Hello_Handler,
		},
	},
}

// RegisterRPCZService register service
func RegisterRPCZService(s server.Service, svr RPCZService) {
	if err := s.Register(&RPCZServer_ServiceDesc, svr); err != nil {
		panic(fmt.Sprintf("RPCZ register error:%v", err))
	}
}

// START --------------------------------- Default Unimplemented Server Service --------------------------------- START

type UnimplementedRPCZ struct{}

// Hello Defined Hello RPC
func (s *UnimplementedRPCZ) Hello(ctx context.Context, req *HelloReq) (*HelloRsp, error) {
	return nil, errors.New("rpc Hello of service RPCZ is not implemented")
}

// END --------------------------------- Default Unimplemented Server Service --------------------------------- END

// END ======================================= Server Service Definition ======================================= END

// START ======================================= Client Service Definition ======================================= START

// RPCZClientProxy defines service client proxy
type RPCZClientProxy interface {
	// Hello Defined Hello RPC
	Hello(ctx context.Context, req *HelloReq, opts ...client.Option) (rsp *HelloRsp, err error)
}

type RPCZClientProxyImpl struct {
	client client.Client
	opts   []client.Option
}

var NewRPCZClientProxy = func(opts ...client.Option) RPCZClientProxy {
	return &RPCZClientProxyImpl{client: client.DefaultClient, opts: opts}
}

func (c *RPCZClientProxyImpl) Hello(ctx context.Context, req *HelloReq, opts ...client.Option) (*HelloRsp, error) {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName("/trpc.examples.rpcz.RPCZ/Hello")
	msg.WithCalleeServiceName(RPCZServer_ServiceDesc.ServiceName)
	msg.WithCalleeApp("examples")
	msg.WithCalleeServer("rpcz")
	msg.WithCalleeService("RPCZ")
	msg.WithCalleeMethod("Hello")
	msg.WithSerializationType(codec.SerializationTypePB)
	callopts := make([]client.Option, 0, len(c.opts)+len(opts))
	callopts = append(callopts, c.opts...)
	callopts = append(callopts, opts...)
	rsp := &HelloRsp{}
	if err := c.client.Invoke(ctx, req, rsp, callopts...); err != nil {
		return nil, err
	}
	return rsp, nil
}

// END ======================================= Client Service Definition ======================================= END
