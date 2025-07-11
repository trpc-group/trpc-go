//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package test to end-to-end testing.
//
//go:generate trpc create -p ./protocols/test.proto --rpconly -o ./protocols --protodir . --mock=false
package test

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	reuseport "github.com/kavu/go_reuseport"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"
	"trpc.group/trpc-go/trpc-go/transport/tnet"

	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
	"trpc.group/trpc-go/trpc-go/test/testdata"
)

// TestRunSuite run test suite in TestSuite.
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// TestSuite is an end-to-end test suite.
type TestSuite struct {
	suite.Suite

	service  server.Service
	server   *server.Server
	listener net.Listener

	enableReusePort  bool
	tRPCEnv          *trpcEnv
	httpServerEnv    *httpServerEnv
	restfulServerEnv *restfulServerEnv
	// autoIncrID is used to avoid possible naming conflicts in restful test
	// and port conflict in unix transport test
	autoIncrID int

	defaultSimpleRequest *testpb.SimpleRequest
}

// SetupAllSuite will run before the tests in the suite are run.
func (s *TestSuite) SetupSuite() {
	require.Nil(s.T(), os.Chdir(testdata.BasePath()))
	transport.RegisterServerTransport("default", transport.DefaultServerTransport)
	transport.RegisterServerTransport("tnet", tnet.DefaultServerTransport)

	const argSize = 271
	const respSize = 314
	payload, err := newPayload(testpb.PayloadType_COMPRESSIBLE, argSize)
	require.Nil(s.T(), err)
	s.defaultSimpleRequest = &testpb.SimpleRequest{
		ResponseType: testpb.PayloadType_COMPRESSIBLE,
		ResponseSize: respSize,
		Payload:      payload,
	}
}

// SetUpTest will be run before every test in the suite.
func (s *TestSuite) SetupTest() {
	s.tRPCEnv = &trpcEnv{
		server: &trpcServerEnv{network: "tcp", async: true},
		client: &trpcClientEnv{multiplexed: false, disableConnectionPool: false},
	}
	s.httpServerEnv = &httpServerEnv{async: true}
	s.enableReusePort = false
}

// TearDownTest will be run after every test in the suite.
func (s *TestSuite) TearDownTest() {
	s.closeServer(nil)
}

// startServer only starts the server. It does not create a client to it.
func (s *TestSuite) startServer(service interface{}, opts ...server.Option) {
	var (
		l   net.Listener
		err error
	)

	if s.tRPCEnv.server.network == "unix" {
		unixSockFile := fmt.Sprintf("test%d.sock", s.autoIncrID)
		l, err = net.Listen(s.tRPCEnv.server.network, unixSockFile)
		if err != nil && strings.Contains(err.Error(), "bind: address already in use") {
			os.Remove(unixSockFile)
			l, err = net.Listen(s.tRPCEnv.server.network, unixSockFile)
		}
		s.autoIncrID++
	} else {
		if s.enableReusePort {
			l, err = reuseport.Listen("tcp", defaultServerAddress)
		} else {
			l, err = net.Listen("tcp", defaultServerAddress)
		}
	}
	require.Nil(s.T(), err)
	s.listener = l
	s.T().Logf("server address: %v", l.Addr())

	var svr *server.Server
	switch ts := service.(type) {
	case *TRPCService:
		svr = s.startTRPCServer(ts, opts...)
	case *StreamingService:
		svr = s.startStreamingServer(ts, opts...)
	case *testHTTPService:
		svr = s.startHTTPServer(ts, opts...)
	case *testRESTfulService:
		svr = s.newRESTfulServer(ts, opts...)
	default:
		require.Fail(s.T(), "unsupported service type.")
	}
	require.NotNil(s.T(), svr)

	s.server = svr
	go svr.Serve()
}

func (s *TestSuite) startTRPCServer(ts testpb.TestTRPCService, opts ...server.Option) *server.Server {
	service := server.New(
		append(
			opts,
			server.WithServiceName(trpcServiceName),
			server.WithProtocol("trpc"),
			server.WithNetwork(s.tRPCEnv.server.network),
			server.WithListener(s.listener),
			server.WithServerAsync(s.tRPCEnv.server.async),
			server.WithTransport(transport.GetServerTransport(s.tRPCEnv.server.transport)),
		)...,
	)
	svr := &server.Server{}
	svr.AddService(trpcServiceName, service)
	testpb.RegisterTestTRPCService(svr.Service(trpcServiceName), ts)
	return svr
}

func (s *TestSuite) startStreamingServer(ts testpb.TestStreamingService, opts ...server.Option) *server.Server {
	trpc.ServerConfigPath = "trpc_go_streaming_server.yaml"
	svr := trpc.NewServer(
		append(
			opts,
			server.WithListener(s.listener),
			server.WithServerAsync(s.tRPCEnv.server.async),
		)...,
	)
	testpb.RegisterTestStreamingService(svr.Service(streamingServiceName), ts)
	return svr
}

func (s *TestSuite) startHTTPServer(ts testpb.TestHTTPService, opts ...server.Option) *server.Server {
	svr := &server.Server{}
	svr.AddService(
		httpServiceName,
		server.New(append([]server.Option{
			server.WithServiceName(httpServiceName),
			server.WithNetwork("tcp"),
			server.WithProtocol("http"),
			server.WithServerAsync(s.httpServerEnv.async),
			server.WithListener(s.listener),
		}, opts...)...),
	)
	testpb.RegisterTestHTTPService(svr.Service(httpServiceName), ts)
	s.server = svr
	return svr
}

func (s *TestSuite) newRESTfulServer(ts testpb.TestRESTfulService, opts ...server.Option) *server.Server {
	s.autoIncrID++
	serviceName := fmt.Sprintf("trpc.testing.end2end.TestRESTful%d", s.autoIncrID)
	service := server.New(append([]server.Option{
		server.WithServiceName(serviceName),
		server.WithProtocol("restful"),
		server.WithNetwork("tcp"),
		server.WithListener(s.listener),
	}, opts...)...)
	svr := &server.Server{}
	svr.AddService(serviceName, service)
	testpb.RegisterTestRESTfulService(svr, ts)
	return svr
}

// serverAddress return server address(ip://x.x.x.x:port) used at client.WithTarget.
func (s *TestSuite) serverAddress() string {
	if s.listener == nil {
		return ""
	}

	var uriScheme string
	switch s.tRPCEnv.server.network {
	case "tcp":
		uriScheme = "ip"
	case "unix":
		uriScheme = "unix"
	default:
	}

	return fmt.Sprintf("%s://%v", uriScheme, s.listener.Addr())
}

func (s *TestSuite) unaryCallDefaultURL() string {
	// "default url: http://ip:port/package.service/method"
	return fmt.Sprintf("http://%v/%s/UnaryCall", s.listener.Addr(), httpServiceName)
}

func (s *TestSuite) unaryCallCustomURL() string {
	return fmt.Sprintf("http://%v/UnaryCall", s.listener.Addr())
}

func (s *TestSuite) startTRPCServerWithConfig(
	service testpb.TestTRPCService,
	cfg *trpc.Config,
	opts ...server.Option) {
	svr := trpc.NewServerWithConfig(cfg, opts...)
	require.NotNil(s.T(), svr)
	testpb.RegisterTestTRPCService(svr, service)
	s.server = svr
	go svr.Serve()
}

func (s *TestSuite) startTRPCServerWithListener(ts testpb.TestTRPCService, opts ...server.Option) {
	l, err := net.Listen("tcp", defaultServerAddress)
	require.Nil(s.T(), err)
	s.listener = l
	svr := trpc.NewServer(append(opts, server.WithListener(s.listener))...)
	testpb.RegisterTestTRPCService(svr, ts)
	go svr.Serve()
	s.server = svr
}

// newTRPCClient creates a tRPC client connected to this service that the test may use.
// The newly created client will be available in the client field of TestSuite.
func (s *TestSuite) newTRPCClient(opts ...client.Option) testpb.TestTRPCClientProxy {
	s.T().Logf("client dial to %s", s.serverAddress())
	internalOption := []client.Option{
		client.WithNetwork(s.tRPCEnv.server.network),
		client.WithTarget(s.serverAddress()),
		client.WithTimeout(time.Second),
		client.WithMultiplexed(s.tRPCEnv.client.multiplexed),
	}
	if s.tRPCEnv.client.disableConnectionPool {
		internalOption = append(internalOption, client.WithDisableConnectionPool())
	}
	return testpb.NewTestTRPCClientProxy(
		append(
			internalOption,
			opts...,
		)...,
	)
}

func (s *TestSuite) newHTTPRPCClient(opts ...client.Option) testpb.TestHTTPClientProxy {
	s.T().Logf("client dial to %s", s.serverAddress())
	return testpb.NewTestHTTPClientProxy(append([]client.Option{
		client.WithProtocol("http"),
		client.WithTarget(s.serverAddress()),
		client.WithTimeout(time.Second)}, opts...)...)
}

// newStreamingClient creates a tRPC streaming client connected to this service that the test may use.
// The newly created client will be available in the client field of TestSuite.
func (s *TestSuite) newStreamingClient(opts ...client.Option) testpb.TestStreamingClientProxy {
	s.T().Logf("client dial to %s", s.serverAddress())
	const defaultTimeout = 1 * time.Second
	return testpb.NewTestStreamingClientProxy(
		append(
			[]client.Option{
				client.WithTarget(s.serverAddress()),
				client.WithTimeout(defaultTimeout),
				client.WithMultiplexed(s.tRPCEnv.client.multiplexed),
			},
			opts...,
		)...,
	)
}

func (s *TestSuite) closeServer(ch chan struct{}) {
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.T().Log(err)
		}
	}
	if s.server != nil {
		if err := s.server.Close(ch); err != nil {
			s.T().Log(err)
		}
		s.server = nil
	}
	s.listener = nil
}
