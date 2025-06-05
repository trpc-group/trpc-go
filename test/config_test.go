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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/test/naming"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
	"trpc.group/trpc-go/trpc-go/transport"
)

func (s *TestSuite) TestClientConfigTimeoutPriority() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	s.writeTRPCConfig(`
client:
    service:
      - name: trpc.testing.end2end.TestTRPC
        protocol: trpc
        network: tcp
        timeout: 1
`)

	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c1.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err), "full err: %+v", err)

	c2 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()), client.WithTimeout(100*time.Millisecond))
	_, err = c2.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Nil(s.T(), err)

	_, err = c2.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(1*time.Microsecond))
	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err), "full err: %+v", err)
}

func (s *TestSuite) TestClientOptionUseWrongServiceName() {
	s.startServer(&TRPCService{})
	naming.AddDiscoveryNode(trpcServiceName, s.listener.Addr().String())
	defer naming.RemoveDiscoveryNode(trpcServiceName)
	c := testpb.NewTestTRPCClientProxy()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest,
		client.WithDiscoveryName("test"), client.WithServiceName("trpc.testing.end2end.TestWrongName"))
	require.Equal(s.T(), errs.RetClientRouteErr, errs.Code(err), "full err: %+v", err)
}

func (s *TestSuite) TestClientConfigConnpool() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	// Set dial_timeout to a low value to prove that the configuration has taken effect.
	s.writeTRPCConfig(`
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      # timeout: 500  # decides context timeout.
      conn_type: connpool  # connection type is connection pool, the following options are all for connpool.
      connpool:
        # priority: option dial_timeout ≈ context timeout > yaml dial_timeout
        # when both option dial_timeout and context timeout exist, real dial timeout = min(option dial timeout, context timeout)
        dial_timeout: 1us  # connection pool: dial timeout, default 200ms.
        force_close: false  # connection pool: whether force close the connection, default false.
        idle_timeout: 50s  # connection pool: idle timeout, default 50s.
        max_active: 0  # connection pool: max active connections, default 0 (means no limit).
        max_conn_lifetime: 0s  # connection pool: max lifetime for connection, default 0s (means no limit).
        max_idle: 65536  # connection pool: max idle connections, default 65536.
        min_idle: 0  # connection pool: min idle connections, default 0.
        pool_idle_timeout: 100s  # connection pool: idle timeout to close the entire pool, default 100s.
        push_idle_conn_to_tail: false  # connection pool: recycle the connection to head/tail of the idle list, default false (head).
        wait: false  # connection pool: whether wait util timeout or return err immediately when number of total connections reach max_active, default false.
`)
	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c1.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.NotNil(s.T(), err)
	require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "err: %+v", err)
}

func (s *TestSuite) TestClientConfigMultiplexed() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	// Set multiplexed_dial_timeout to a low value to prove that the configuration has taken effect.
	s.writeTRPCConfig(`
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      # timeout: 500  # decides context timeout.
      conn_type: multiplexed  # connection type is multiplexed, the following options are all for multiplex.
      multiplexed:
        multiplexed_dial_timeout: 1us  # multiplexed: dial timeout, default 1s.
        conns_per_host: 2  # multiplexed: number of concrete(real) connections for each host, default 2.
        max_vir_conns_per_conn: 0  # multiplexed: max number of virtual connections for each concrete(real) connection, default 0 (means no limit).
        max_idle_conns_per_host: 0  # multiplexed: max number of idle concrete(real) connections for each host, used together with max_vir_conns_per_conn, default 0 (disabled).
        queue_size: 1024  # multiplexed: size of send queue for each concrete(real) connection, default 1024.
        drop_full: false  # multiplexed: whether to drop the send package when queue is full, default false.
`)
	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c1.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.NotNil(s.T(), err)
	require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "err: %+v", err)
}

func (s *TestSuite) TestClientConfigShort() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	// Specify conn_type as short, which will ignore the configurations of connpool and multiplex.
	s.writeTRPCConfig(`
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      # timeout: 500  # decides context timeout.
      conn_type: short
      # although connpool and multiplexed configurations are provided,
      # conn_type is specified as short, the following configs will not take effect. 
      connpool:
        dial_timeout: 1us  # connection pool: dial timeout, default 200ms.
      multiplexed:
        multiplexed_dial_timeout: 1us  # multiplexed: dial timeout, default 1s.
`)
	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c1.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestClientConfigHTTPPool() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	// Specify conn_type as httppool, which will ignore the configurations of connpool and multiplex.
	s.writeTRPCConfig(`
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      # timeout: 500  # decides context timeout.
      conn_type: httppool
      # although connpool and multiplexed configurations are provided,
      # conn_type is specified as httppool, the following configs will not take effect. 
      connpool:
        dial_timeout: 1us  # connection pool: dial timeout, default 200ms.
      multiplexed:
        multiplexed_dial_timeout: 1us  # multiplexed: dial timeout, default 1s.
`)
	c := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestClientConfigTNetConnpool() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	// Set dial_timeout to a low value to prove that the configuration has taken effect.
	s.writeTRPCConfig(`
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      # timeout: 500  # decides context timeout.
      transport: tnet
      conn_type: connpool  # connection type is connection pool, the following options are all for connpool.
      connpool:
        # priority: option dial_timeout ≈ context timeout > yaml dial_timeout
        # when both option dial_timeout and context timeout exist, real dial timeout = min(option dial timeout, context timeout)
        dial_timeout: 1us  # connection pool: dial timeout, default 200ms.
        force_close: false  # connection pool: whether force close the connection, default false.
        idle_timeout: 50s  # connection pool: idle timeout, default 50s.
        max_active: 0  # connection pool: max active connections, default 0 (means no limit).
        max_conn_lifetime: 0s  # connection pool: max lifetime for connection, default 0s (means no limit).
        max_idle: 65536  # connection pool: max idle connections, default 65536.
        min_idle: 0  # connection pool: min idle connections, default 0.
        pool_idle_timeout: 100s  # connection pool: idle timeout to close the entire pool, default 100s.
        push_idle_conn_to_tail: false  # connection pool: recycle the connection to head/tail of the idle list, default false (head).
        wait: false  # connection pool: whether wait util timeout or return err immediately when number of total connections reach max_active, default false.
`)
	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c1.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.NotNil(s.T(), err)
	require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "err: %+v", err)
}

func (s *TestSuite) TestClientConfigTNetMultiplexed() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	// Set multiplexed_dial_timeout to a low value to prove that the configuration has taken effect.
	s.writeTRPCConfig(`
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      # timeout: 500  # decides context timeout.
      transport: tnet
      conn_type: multiplexed  # connection type is multiplexed, the following options are all for multiplex.
      multiplexed:
        multiplexed_dial_timeout: 1us  # multiplexed: dial timeout, default 1s.
        max_vir_conns_per_conn: 0  # multiplexed: max number of virtual connections for each concrete(real) connection, default 0 (means no limit).
        enable_metrics: true  # tnet-muLtiplex: whether to enable metrics, used together with 'transport: tnet', default false.
`)
	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c1.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.NotNil(s.T(), err)
	require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "err: %+v", err)
}

func (s *TestSuite) TestClientConfigTNetShort() {
	s.startServer(&TRPCService{unaryCallSleepTime: 10 * time.Millisecond})

	// Specify conn_type as short, which will ignore the configurations of connpool and multiplex.
	s.writeTRPCConfig(`
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      # timeout: 500  # decides context timeout.
      transport: tnet
      conn_type: short
      # although connpool and multiplexed configurations are provided,
      # conn_type is specified as short, the following configs will not take effect. 
      connpool:
        dial_timeout: 1us  # connection pool: dial timeout, default 200ms.
      multiplexed:
        multiplexed_dial_timeout: 1us  # multiplexed: dial timeout, default 1s.
`)
	c1 := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
	_, err := c1.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestClientConfigTNetHTTPPool() {
	defer func() {
		require.NotNil(s.T(), recover())
	}()

	// Specify conn_type as httppool.
	s.writeTRPCConfig(`
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      transport: tnet
      conn_type: httppool
`)
}

func (s *TestSuite) TestClientConfigHTTP_Connpool() {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "trpc-protocol-with-http-transport",
			config: `
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      transport: http
      conn_type: connpool
`,
		},
		{
			name: "http-protocol",
			config: `
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: http
      conn_type: connpool
`,
		},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			defer func() {
				require.NotNil(s.T(), recover())
			}()

			// Specify conn_type as connpool.
			s.writeTRPCConfig(tt.config)
		})
	}
}

func (s *TestSuite) TestClientConfigHTTP_Multiplexed() {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "trpc-protocol-with-http-transport",
			config: `
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      transport: http
      conn_type: multiplexed
`,
		},
		{
			name: "http-protocol",
			config: `
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: http
      conn_type: multiplexed
`,
		},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			defer func() {
				require.NotNil(s.T(), recover())
			}()

			// Specify conn_type as multiplexed.
			s.writeTRPCConfig(tt.config)
		})
	}
}

func (s *TestSuite) TestClientConfigHTTP_HTTPPool() {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "trpc-protocol-with-http-transport",
			config: `
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      transport: http
      conn_type: httppool  # connection type is httppool, the following options are all for httppool.
      httppool:
        max_idle_conns: 100  # httppool: max number of idle connections, default 0 (means no limit).
        max_idle_conns_per_host: 10  # httppool: max number of idle connections per-host, default 2.
        max_conns_per_host: 20  # httppool: max number of connections, default 0 (means no limit).
        idle_conn_timeout: 1s  # httppool: idle timeout, default 0s (means no limit).
`,
		},
		{
			name: "http-protocol",
			config: `
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: http
      conn_type: httppool  # connection type is httppool, the following options are all for httppool.
      httppool:
        max_idle_conns: 100  # httppool: max number of idle connections, default 0 (means no limit).
        max_idle_conns_per_host: 10  # httppool: max number of idle connections per-host, default 2.
        max_conns_per_host: 20  # httppool: max number of connections, default 0 (means no limit).
        idle_conn_timeout: 1s  # httppool: idle timeout, default 0s (means no limit).
`,
		},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.startServer(&testHTTPService{TRPCService{UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
				return &testpb.SimpleResponse{}, nil
			}}})
			defer s.closeServer(nil)

			// Specify conn_type as httppool.
			s.writeTRPCConfig(tt.config)
			c := testpb.NewTestHTTPClientProxy(
				client.WithProtocol(protocol.HTTP),
				client.WithTarget(s.serverAddress()),
			)
			_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(time.Second))
			require.Nil(s.T(), err)
		})
	}
}

func (s *TestSuite) TestClientConfigHTTP_Short() {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "trpc-protocol-with-http-transport",
			config: `
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      transport: http
      conn_type: short  # connection type is short.
`,
		},
		{
			name: "http-protocol",
			config: `
client:
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: http
      conn_type: short  # connection type is short.
`,
		},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.startServer(&testHTTPService{TRPCService{UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
				return &testpb.SimpleResponse{}, nil
			}}})
			defer s.closeServer(nil)

			// Specify conn_type as httppool.
			s.writeTRPCConfig(tt.config)
			c := testpb.NewTestHTTPClientProxy(
				client.WithProtocol(protocol.HTTP),
				client.WithTarget(s.serverAddress()),
			)
			_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(time.Second))
			require.Nil(s.T(), err)
		})
	}
}

func (s *TestSuite) TestClientCalleeField() {
	s.Run("service.name is different from callee, but don't configure callee", func() {
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
    - name: test-service-name
      protocol: trpc
      network: tcp
      target: "test://test-service-name"
`)
		const name = "test-service-name"
		s.startServer(&TRPCService{})

		naming.AddSelectorNode(name, s.listener.Addr().String())
		defer naming.RemoveSelectorNode(name)

		c := testpb.NewTestTRPCClientProxy()
		_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
		require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "callee field is the service name of PB, which is used for matching client proxy and configuration")
	})
	s.Run("service.name is different from callee, and configure callee correctly", func() {
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
      name: test-service-name
      protocol: trpc
      network: tcp
      target: "test://test-service-name"
`)
		const name = "test-service-name"
		s.startServer(&TRPCService{})

		naming.AddSelectorNode(name, s.listener.Addr().String())
		defer naming.RemoveSelectorNode(name)

		c := testpb.NewTestTRPCClientProxy()
		_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
		require.Nil(s.T(), err)
	})
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
	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err), "full err: %+v", err)
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

func (s *TestSuite) TestTnetTCP() {
	cfg := trpc.Config{}
	err := yaml.Unmarshal(
		[]byte(`
server:
  app: testing
  server: end2end
  service:
    - name: trpc.testing.end2end.TestTRPC
      ip: 127.0.0.1
      port: 9876
      protocol: trpc
      network: tcp
      transport: tnet
`),
		&cfg,
	)
	require.Nil(s.T(), err)

	svr := trpc.NewServerWithConfig(&cfg)
	testpb.RegisterTestTRPCService(svr.Service(trpcServiceName), &TRPCService{})
	go svr.Serve()
	defer svr.Close(nil)
	time.Sleep(10 * time.Millisecond)

	c := testpb.NewTestTRPCClientProxy(client.WithNetwork("tcp"),
		client.WithTransport(transport.GetClientTransport("tnet")),
		client.WithTarget("ip://127.0.0.1:9876"))
	_, err = c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestTnetUDP() {

	cfg := trpc.Config{}
	err := yaml.Unmarshal(
		[]byte(`
server:
  app: testing
  server: end2end
  service:
    - name: trpc.testing.end2end.TestTRPC
      ip: 127.0.0.1
      port: 9876
      protocol: trpc
      network: udp
      transport: tnet
`),
		&cfg,
	)
	require.Nil(s.T(), err)

	svr := trpc.NewServerWithConfig(&cfg)
	testpb.RegisterTestTRPCService(svr.Service(trpcServiceName), &TRPCService{})
	go svr.Serve()
	defer svr.Close(nil)
	time.Sleep(10 * time.Millisecond)

	c := testpb.NewTestTRPCClientProxy(client.WithNetwork("udp"),
		client.WithTransport(transport.GetClientTransport("tnet")),
		client.WithTarget("ip://127.0.0.1:9876"))
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
	cfg.Server.Service[0].IP = "8.8.8.8"

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
