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
	"errors"
	"strings"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"

	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestServerReusePort() {
	s.Run("EnableReusePort", s.testEnableServerReusePort)
	s.Run("DisableReusePort", func() {
		for _, enable1 := range []bool{true, false} {
			for _, enable2 := range []bool{true, false} {
				if enable1 && enable2 {
					break
				}
				s.testDisableServerReusePort(enable1, enable2)
			}
		}
	})
}
func (s *TestSuite) testEnableServerReusePort() {
	s.enableReusePort = true
	s.startServer(&TRPCService{})

	trpc.ServerConfigPath = "trpc_go_trpc_server.yaml"
	svr := trpc.NewServer(
		server.WithTransport(transport.NewServerTransport(transport.WithReusePort(true))),
		server.WithNetwork("tcp"),
		server.WithAddress(s.listener.Addr().String()),
	)
	testpb.RegisterTestTRPCService(svr.Service(trpcServiceName), &TRPCService{})

	startServe := make(chan struct{})
	go func() {
		startServe <- struct{}{}
		svr.Serve()
	}()
	<-startServe
	s.server.Close(nil)
	s.server = nil

	c := s.newTRPCClient()
	for {
		_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
		if err == nil {
			svr.Close(nil)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *TestSuite) testDisableServerReusePort(enable1, enable2 bool) {
	s.enableReusePort = enable1
	s.startServer(&TRPCService{})

	svr := trpc.NewServer(
		server.WithNetwork("tcp"),
		server.WithAddress(s.listener.Addr().String()),
		server.WithTransport(transport.NewServerTransport(transport.WithReusePort(enable2))),
	)
	require.Contains(s.T(), svr.Serve().Error(), "address already in use")
}

func (s *TestSuite) TestServerIdleTime() {
	s.Run("ServerIdleTimeLessThanHandleTime", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), s.testServerIdleTimeLessThanHandleTime)
		}
	})
	s.Run("ServerIdleTimeGreaterThanHandleTime", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), s.testServerIdleTimeGreaterThanHandleTime)
		}
	})
}

func (s *TestSuite) testServerIdleTimeLessThanHandleTime() {
	s.startServer(&TRPCService{unaryCallSleepTime: time.Second}, server.WithIdleTimeout(100*time.Millisecond))
	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(2*time.Second))
	switch {
	case s.tRPCEnv.server.async && s.tRPCEnv.client.multiplexed:
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err))
		require.Contains(s.T(), err.Error(), "client multiplexed transport ReadFrame: EOF")
	case s.tRPCEnv.server.async && !s.tRPCEnv.client.multiplexed:
		require.EqualValues(s.T(), errs.RetClientReadFrameErr, errs.Code(err))
	case !s.tRPCEnv.server.async && s.tRPCEnv.server.network == "unix":
		require.Nil(s.T(), err, "idle time won't work in unix network")
	case !s.tRPCEnv.server.async && s.tRPCEnv.server.transport == "default":
		require.Nil(s.T(), err, "idle time implemented in default transport has a bug")
	case !s.tRPCEnv.server.async && s.tRPCEnv.server.transport == "tnet":
		require.EqualValues(s.T(), errs.RetClientReadFrameErr, errs.Code(err))
	default:
		s.T().Fatal()
	}

	for s.closeServer(nil); ; {
		_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
		if errors.Is(err, errs.ErrServerClose) ||
			strings.Contains(errs.Msg(err), "connect: connection refused") ||
			strings.Contains(errs.Msg(err), "connect: no such file or directory") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *TestSuite) testServerIdleTimeGreaterThanHandleTime() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond}, server.WithIdleTimeout(time.Second))
	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(2*time.Second))
	require.Nil(s.T(), err)
}
