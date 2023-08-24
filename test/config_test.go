// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/test/naming"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestClientConfigTimeoutPriority() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	s.writeTRPCConfig(`
client:
    service:
      - name: trpc.testing.end2end.TestTRPC
        protocol: trpc
        network: tcp
        timeout: 10
`)

	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c1.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err))

	c2 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()), client.WithTimeout(100*time.Millisecond))
	_, err = c2.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Nil(s.T(), err)

	_, err = c2.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(10*time.Microsecond))
	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err))
}

func (s *TestSuite) TestClientConfigLoadWrongServiceName() {
	s.startServer(&TRPCService{})

	s.writeTRPCConfig(`
client:
    service:
      - name: trpc.testing.end2end.TestGRPC
        protocol: trpc
        network: tcp
`)

	naming.AddDiscoveryNode(trpcServiceName, s.listener.Addr().String())
	defer naming.RemoveDiscoveryNode(trpcServiceName)

	c := testpb.NewTestTRPCClientProxy()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithDiscoveryName("test"))
	require.NotNil(s.T(), errs.RetClientConnectFail, errs.Code(err))
}

func (s *TestSuite) TestClientCalleeField() {
	s.writeTRPCConfig(`
global:
  namespace: Development
  env_name: test
server:
  app: testing
  server: end2end
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      ip: 127.0.0.1
      port: 17832
client:
  service:
    - callee: trpc.testing.end2end.TestTRPC
      name: test-servic-name
      protocol: trpc
      network: tcp
      target: "test://trpc.testing.end2end.TestTRPC"
`)

	s.startServer(&TRPCService{})

	naming.AddSelectorNode(trpcServiceName, s.listener.Addr().String())
	defer naming.RemoveSelectorNode(trpcServiceName)

	c1 := testpb.NewTestTRPCClientProxy()
	_, err := c1.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(s.T(), err)

	c2 := testpb.NewTestTRPCClientProxy()
	_, err = c2.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{}, client.WithDiscoveryName("test"), client.WithServiceName("wrong-service-name"))
	require.Nil(s.T(), err, "callee field is valid.")
}

func (s *TestSuite) TestServiceNameFormat() {
	s.writeTRPCConfig(`
global:
  namespace: Development
  env_name: test
server:
  app: testing
  server: end2end
  service:
    - name: not-follow-trpc-dot-app-dot-server-dot-service-format
      protocol: trpc
      network: tcp
`)
	const nonstandardServerName = "not-follow-trpc-dot-app-dot-server-dot-service-format"
	s.startServer(&TRPCService{})
	naming.AddDiscoveryNode(nonstandardServerName, s.listener.Addr().String())
	defer naming.RemoveDiscoveryNode(nonstandardServerName)

	c := s.newTRPCClient()
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{}, client.WithServiceName(nonstandardServerName))
	require.Nil(s.T(), err, "any service name is ok, even it doesn't follow trpc.app.server.service format.")
}

func (s *TestSuite) TestServerLoadWrongYamlFile() {
	defer func() {
		p := recover()
		require.NotNil(s.T(), p)
		require.Contains(s.T(), fmt.Sprint(p), "load config fail: yaml")
	}()

	s.writeTRPCConfig(`
server:
app: testing
  server: end2end
    service:
      - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
`)
	s.startTRPCServerWithListener(&TRPCService{})
}

func (s *TestSuite) TestServerLoadYamlFileOk() {
	trpc.ServerConfigPath = "trpc_go_trpc_server.yaml"
	s.startTRPCServerWithListener(&TRPCService{})

	c := s.newTRPCClient()
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestServerCantFindYamlFile() {
	defer func() {
		p := recover()
		require.NotNil(s.T(), p)
		require.Contains(
			s.T(),
			fmt.Sprintf("%v", p),
			fmt.Sprintf("open %s: no such file or directory", defaultConfigPath),
		)
	}()

	trpc.ServerConfigPath = defaultConfigPath
	s.startTRPCServerWithListener(&TRPCService{})

	c := s.newTRPCClient()
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestServerOptionConfigProtocolPriority() {
	s.writeTRPCConfig(`
server:
  app: testing
  server: end2end
  service:
    - name: trpc.testing.end2end.TestHTTP
      protocol: trpc
      network: tcp
`)
	l, err := net.Listen("tcp", defaultServerAddress)
	require.Nil(s.T(), err)
	s.listener = l
	s.T().Logf("server address %v", l.Addr())

	svr := trpc.NewServer(server.WithListener(s.listener), server.WithProtocol("http"))
	testpb.RegisterTestHTTPService(svr, &testHTTPService{})
	go svr.Serve()
	s.server = svr

	bts, err := json.Marshal(s.defaultSimpleRequest)
	require.Nil(s.T(), err)

	rsp, err := http.Post(s.unaryCallCustomURL(), "application/json", bytes.NewReader(bts))
	require.Nil(s.T(), err)
	defer rsp.Body.Close()
	bts, err = io.ReadAll(rsp.Body)
	require.Nil(s.T(), err)

	c := s.newTRPCClient()
	_, err = c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err))
}

func (s *TestSuite) TestServerFromConfigOk() {
	cfg := trpc.Config{}
	err := yaml.Unmarshal(
		[]byte(`
server:
  app: testing
  server: end2end
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
`),
		&cfg,
	)
	require.Nil(s.T(), err)

	s.startTrpcServerFromConfig(&TRPCService{}, &cfg)

	c := s.newTRPCClient()
	_, err = c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestConfigLoad() {
	const validConfigFile = "trpc_go_trpc_server.yaml"
	s.Run("CodecNotExist", func() {
		_, err := config.Load(validConfigFile, config.WithCodec("not-yaml"))
		require.ErrorIs(s.T(), err, config.ErrCodecNotExist)
	})
	s.Run("ProviderNotExist", func() {
		_, err := config.Load(validConfigFile, config.WithProvider("not-file"))
		require.ErrorIs(s.T(), err, config.ErrProviderNotExist)
	})
	s.Run("ProviderNotExist", func() {
		require.ErrorIs(s.T(), config.DefaultConfigLoader.Reload("not-valid-config-file.yaml"), config.ErrConfigNotExist)
	})
	s.Run("LoadOk", func() {
		cfg, err := config.DefaultConfigLoader.Load(validConfigFile)
		require.Nil(s.T(), err)
		trpcConfig := trpc.Config{}
		require.Nil(s.T(), cfg.Unmarshal(&trpcConfig))
		s.startTrpcServerFromConfig(&TRPCService{}, &trpcConfig)
		defer s.closeServer(nil)
		c := s.newTRPCClient()
		_, err = c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
		require.Nil(s.T(), err)
	})
}

func (s *TestSuite) TestServerFromConfigFailed() {
	cfg := trpc.Config{}
	err := yaml.Unmarshal(
		[]byte(`
server:
  app: testing
  server: end2end
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
`),
		&cfg,
	)
	require.Nil(s.T(), err)
	cfg.Server.Service[0].IP = "wrong-ip"

	svr := trpc.NewServerWithConfig(&cfg)
	testpb.RegisterTestTRPCService(svr, &TRPCService{})
	require.NotNil(s.T(), svr.Serve())
}

func (s *TestSuite) startTrpcServerFromConfig(ts testpb.TestTRPCService, cfg *trpc.Config, opts ...server.Option) {
	l, err := net.Listen("tcp", defaultServerAddress)
	require.Nil(s.T(), err)
	s.listener = l
	s.T().Logf("server address: %v", l.Addr())

	svr := trpc.NewServerWithConfig(
		cfg,
		append(
			opts,
			server.WithListener(s.listener),
			server.WithServerAsync(s.tRPCEnv.server.async),
		)...,
	)
	require.NotNil(s.T(), svr)

	testpb.RegisterTestTRPCService(svr.Service(trpcServiceName), ts)
	s.server = svr
	go svr.Serve()
}

// writeTRPCConfig writes config  to the temporary directory.
func (s *TestSuite) writeTRPCConfig(config string) {
	path := filepath.Join(s.T().TempDir(), defaultConfigPath)
	require.Nil(s.T(), os.WriteFile(path, []byte(config), 0644))
	trpc.ServerConfigPath = path
	_ = trpc.NewServer()
}
