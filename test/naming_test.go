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
	"fmt"
	"net"

	"github.com/stretchr/testify/require"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/test/naming"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestIPDiscovery() {
	s.startServer(&TRPCService{})

	discovery.Register("test-ip-discovery", &discovery.IPDiscovery{})

	c := testpb.NewTestTRPCClientProxy()
	_, err := c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithServiceName(s.listener.Addr().String()),
		client.WithDiscoveryName("test-ip-discovery"),
	)
	require.Nil(s.T(), err)

	discovery.Register("test-ip-discovery", &discovery.IPDiscovery{})
	_, err = c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithServiceName("localhost"),
		client.WithDiscoveryName("test-ip-discovery"),
	)
	require.NotNil(s.T(), err)
	require.Contains(s.T(), err.Error(), "missing port in address")
}

func (s *TestSuite) TestIPSelector() {
	s.startServer(&TRPCService{})

	selector.Register("test-ip-selector", selector.NewIPSelector())

	c := testpb.NewTestTRPCClientProxy()
	_, err := c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithTarget(fmt.Sprintf("test-ip-selector://%s", s.listener.Addr().String())),
	)
	require.Nil(s.T(), err)

	_, err = c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithTarget(fmt.Sprintf("test-ip-selector://%s", "127.0.0.1:-1")),
	)
	require.NotNil(s.T(), err)
}

func (s *TestSuite) TestTRPCSelector() {
	s.startServer(&TRPCService{})

	selector.Register("test-trpc-selector", &selector.TrpcSelector{})

	naming.AddDiscoveryNode(trpcServiceName, s.listener.Addr().String())
	defer naming.RemoveDiscoveryNode(trpcServiceName)

	c := testpb.NewTestTRPCClientProxy()
	_, err := c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithTarget(fmt.Sprintf("test-trpc-selector://%s", trpcServiceName)),
		client.WithDiscoveryName("test"),
	)
	require.Nil(s.T(), err)

	_, err = c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithTarget(fmt.Sprintf("test-trpc-selector://%s", "wrong-service-know")),
		client.WithDiscoveryName("test"),
	)
	require.Equal(s.T(), errs.RetClientRouteErr, errs.Code(err))
	require.Contains(s.T(), err.Error(), "can't discover wrong-service-know")
}

func (s *TestSuite) TestCustomSelector() {
	s.startServer(&TRPCService{})

	c := testpb.NewTestTRPCClientProxy()
	_, err := c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithTarget(fmt.Sprintf("test://%s", trpcServiceName)),
	)
	require.Equal(s.T(), errs.RetClientRouteErr, errs.Code(err))
	require.Contains(s.T(), err.Error(), "no available node")

	naming.AddSelectorNode(trpcServiceName, s.listener.Addr().String())
	defer naming.RemoveSelectorNode(trpcServiceName)

	_, err = c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithTarget(fmt.Sprintf("test://%s", trpcServiceName)),
	)
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestCustomDiscovery() {
	s.startServer(&TRPCService{})

	c := testpb.NewTestTRPCClientProxy()
	_, err := c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithServiceName(trpcServiceName),
		client.WithDiscoveryName("test"),
	)
	require.Equal(s.T(), errs.RetClientRouteErr, errs.Code(err))

	naming.AddDiscoveryNode(trpcServiceName, s.listener.Addr().String())
	defer naming.RemoveDiscoveryNode(trpcServiceName)

	_, err = c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithServiceName(trpcServiceName),
		client.WithDiscoveryName("test"),
	)
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestNoServiceOnAddress() {
	trpc.ServerConfigPath = "trpc_go_trpc_server.yaml"

	l, err := net.Listen("tcp", defaultServerAddress)
	require.Nil(s.T(), err)
	s.listener = l
	s.T().Logf("server address: %v", l.Addr())

	svr := trpc.NewServer(server.WithListener(s.listener))
	require.NotNil(s.T(), svr)
	go svr.Serve()
	s.server = svr

	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err = c1.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Equal(s.T(), errs.RetServerNoFunc, errs.Code(err))
}

func (s *TestSuite) TestServiceOnAddress() {
	s.startServer(&TRPCService{})

	c := testpb.NewTestTRPCClientProxy()
	_, err := c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithServiceName(trpcServiceName),
	)
	require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err))
	require.Contains(s.T(), err.Error(), "missing port in address")

	_, err = c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithTarget(s.serverAddress()),
	)
	require.Nil(s.T(), err)
}
