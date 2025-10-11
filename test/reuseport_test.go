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

package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/internal/reuseport"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestReusePort(t *testing.T) {
	var l1, err1 = reuseport.Listen("tcp", "127.0.0.1:55321")
	require.Nil(t, err1)
	var l2, err2 = reuseport.Listen("tcp", "127.0.0.1:55321")
	require.Nil(t, err2)
	name1 := "trpc.testing.end2end.TestReusePort1"
	name2 := "trpc.testing.end2end.TestReusePort2"
	service1 := server.New(server.WithServiceName(name1),
		server.WithProtocol("trpc"),
		server.WithListener(l1),
		server.WithTransport(transport.DefaultServerTransport))
	service2 := server.New(server.WithServiceName(name2),
		server.WithProtocol("trpc"),
		server.WithListener(l2),
		server.WithTransport(transport.DefaultServerTransport))
	svr1 := &server.Server{}
	svr2 := &server.Server{}
	svr1.AddService(name1, service1)
	svr2.AddService(name2, service2)
	testpb.RegisterTestTRPCService(svr1.Service(name1), &TRPCService{})
	testpb.RegisterTestTRPCService(svr2.Service(name2), &TRPCService{})
	go svr1.Serve()
	go svr2.Serve()
	time.Sleep(1 * time.Second)

	closeSvr := func() {
		require.Nil(t, l1.Close())
		require.Nil(t, l2.Close())
		require.Nil(t, svr1.Close(nil))
		require.Nil(t, svr2.Close(nil))
	}

	defer closeSvr()

	c := testpb.NewTestTRPCClientProxy(client.WithTarget("ip://127.0.0.1:55321"),
		client.WithTimeout(time.Second),
		client.WithDisableConnectionPool(),
		client.WithTransport(transport.DefaultClientTransport))
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(t, err)
}
