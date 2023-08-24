// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/server"
)

func (s *TestSuite) TestProxyServer() {
	s.startServer(&TRPCService{})
	proxyServer, proxyAddr := startProxyServer(s.T(), &proxyServiceImpl{targetAddr: s.listener.Addr()})
	s.T().Cleanup(func() {
		if err := proxyServer.Close(nil); err != nil {
			s.T().Log(err)
		}
	})

	c := s.newTRPCClient(client.WithTarget(fmt.Sprintf("%s://%v", "ip", proxyAddr)))
	req := s.defaultSimpleRequest
	rsp, err := c.UnaryCall(context.Background(), req)

	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(req.ResponseSize))
}

func startProxyServer(t *testing.T, s proxyService) (*server.Server, net.Addr) {
	const serviceName = "trpc.testing.end2end.testTRPCProxy"
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	svr := &server.Server{}
	svr.AddService(serviceName, server.New(
		server.WithServiceName(serviceName),
		server.WithCurrentSerializationType(codec.SerializationTypeNoop),
		server.WithListener(l),
		server.WithProtocol("trpc"),
		server.WithNetwork("tcp"),
	))

	if err := svr.Service(serviceName).Register(&server.ServiceDesc{
		ServiceName: trpcServiceName,
		HandlerType: (*proxyService)(nil),
		Methods: []server.Method{
			{
				Name: "/trpc.testing.end2end.TestTRPC/UnaryCall",
				Func: proxyServiceForwardHandler,
			},
		}}, s); err != nil {
		t.Fatal(err)
	}

	go func() {
		t.Log(svr.Serve())
	}()

	return svr, l.Addr()
}

type proxyService interface {
	Forward(ctx context.Context, req *codec.Body) (*codec.Body, error)
}

type proxyServiceImpl struct {
	targetAddr net.Addr
}

func (s *proxyServiceImpl) Forward(_ context.Context, req *codec.Body) (*codec.Body, error) {
	ctx, msg := codec.WithCloneMessage(trpc.BackgroundContext())
	msg.WithCalleeServiceName("trpc.testing.end2end.TestTRPC")
	msg.WithClientRPCName("/trpc.testing.end2end.TestTRPC/UnaryCall")
	rsp := &codec.Body{}
	err := client.DefaultClient.Invoke(ctx, req, rsp,
		[]client.Option{
			client.WithProtocol("trpc"),
			client.WithSerializationType(codec.SerializationTypePB),
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithTarget(fmt.Sprintf("%s://%v", "ip", s.targetAddr)),
			client.WithTimeout(time.Second),
		}...,
	)
	return rsp, err
}

func proxyServiceForwardHandler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	req := &codec.Body{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, req interface{}) (interface{}, error) {
		return svr.(proxyService).Forward(ctx, req.(*codec.Body))
	}
	return filters.Filter(ctx, req, handleFunc)
}
