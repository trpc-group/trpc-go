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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"

	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

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
	case s.tRPCEnv.server.transport == "tnet" && s.tRPCEnv.server.network == "tcp":
		require.Equal(s.T(), errs.RetClientReadFrameErr, errs.Code(err),
			"tnet does not support graceful stop yet, and it's idle timeout implementation should be revised")
	case s.tRPCEnv.server.transport == "tnet" && s.tRPCEnv.server.network == "unix":
		// tnet does not support unix, on which tnet transport will fall back to original transport.
		fallthrough
	default:
		require.Nil(s.T(), err, "connection is only closed when there is no active request")
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
	if s.tRPCEnv.server.transport == "tnet" && s.tRPCEnv.server.network == "tcp" {
		s.T().Skip("tnet does not support graceful stop yet, and it's idle timeout implementation should be revised")
	}
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond}, server.WithIdleTimeout(time.Second))
	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(2*time.Second))
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestListenerClosed() {
	s.Run("MultiplexedOrConnectionPool", func() {
		for _, e := range allTRPCEnvs {
			if e.client.multiplexed || !e.client.disableConnectionPool {
				s.tRPCEnv = e
				s.Run(e.String(), s.testListenerClosedOnMultiplexedOrConnectionPool)
			}
		}
	})
	s.Run("ShortConnection", func() {
		for _, e := range allTRPCEnvs {
			if !e.client.multiplexed && e.client.disableConnectionPool {
				s.tRPCEnv = e
				s.Run(e.String(), s.testListenerClosedOnShortConnection)
			}
		}
	})
}

func (s *TestSuite) testListenerClosedOnMultiplexedOrConnectionPool() {
	s.startServer(&TRPCService{})
	s.T().Cleanup(func() {
		s.closeServer(nil)
	})
	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(2*time.Second))
	require.Nil(s.T(), err)

	require.Nil(s.T(), s.listener.Close())

	_, err = c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(2*time.Second))
	require.Nil(s.T(), err, "Already Accepted connections are not closed.")
}

func (s *TestSuite) testListenerClosedOnShortConnection() {
	s.startServer(&TRPCService{})
	s.T().Cleanup(func() {
		s.closeServer(nil)
	})
	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(2*time.Second))
	require.Nil(s.T(), err)

	require.Nil(s.T(), s.listener.Close())

	_, err = c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(2*time.Second))
	require.NotNilf(s.T(), err, "expected: %s, got: %v ", "connect: connection refused, or connect: no such file or director", err)
}

func (s *TestSuite) TestTnetConcurrentSafe() {
	tests := []struct {
		network   string
		transport string
	}{
		{
			network:   "tcp",
			transport: "tnet",
		},
		{
			network:   "udp",
			transport: "tnet",
		},
	}
	for _, tt := range tests {
		s.tRPCEnv = &trpcEnv{server: &trpcServerEnv{network: tt.network, transport: tt.transport},
			client: &trpcClientEnv{}}
		s.Run(s.tRPCEnv.String(), s.testTnetConcurrentSafe)
	}
}

func (s *TestSuite) testTnetConcurrentSafe() {
	serverAddr := "127.0.0.1:8965"
	go func() {
		service := server.New(server.WithAddress(serverAddr),
			server.WithProtocol("trpc"),
			server.WithServiceName(trpcServiceName),
			server.WithNetwork(s.tRPCEnv.server.network),
			server.WithTransport(transport.GetServerTransport(s.tRPCEnv.server.transport)))
		svr := &server.Server{}
		svr.AddService(trpcServiceName, service)
		testpb.RegisterTestTRPCService(svr.Service(trpcServiceName), &TRPCService{})
		svr.Serve()
	}()
	time.Sleep(10 * time.Millisecond)
	ctx := trpc.BackgroundContext()
	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			c := testpb.NewTestTRPCClientProxy(client.WithTarget(fmt.Sprintf("ip://%v", serverAddr)),
				client.WithNetwork(s.tRPCEnv.server.network),
				client.WithTransport(transport.GetClientTransport(s.tRPCEnv.server.transport)))
			for j := 0; j < 10; j++ {
				_, err := c.EmptyCall(ctx, &testpb.Empty{}, client.WithTimeout(10*time.Millisecond))
				assert.Nil(s.T(), err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
