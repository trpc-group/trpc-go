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
	"time"

	"github.com/stretchr/testify/require"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

func (s *TestSuite) TestTimeoutAtPluginRegister() {
	defer func() {
		p := recover()
		require.NotNil(s.T(), p)
		require.Contains(s.T(), fmt.Sprint(p), "setup plugin fail: setup plugin test-timeout timeout")
	}()
	plugin.Register("timeout", &timeOutPlugin{Timeout: 30 * time.Second})
	trpc.ServerConfigPath = "trpc_go_trpc_server_with_plugin.yaml"
	s.startTRPCServerWithListener(&TRPCService{})
}

type timeOutPlugin struct {
	Timeout time.Duration
}

func (p *timeOutPlugin) Type() string {
	return "test"
}

func (p *timeOutPlugin) Setup(_ string, dec plugin.Decoder) error {
	if err := dec.Decode(p); err != nil {
		return err
	}
	time.Sleep(p.Timeout)
	return nil
}

func (s *TestSuite) TestPluginDuplicateRegistered() {
	defer func() {
		p := recover()
		require.NotNil(s.T(), p)
		require.Contains(s.T(), fmt.Sprint(p), "mapping key \"timeout\" already defined")
	}()
	plugin.Register("timeout", &timeOutPlugin{})
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
plugins:
  test:
    timeout:
    timeout:
`)
}

func (s *TestSuite) TestInternalPluginRegisterOk() {
	plugin.Register("test-internal", log.DefaultLogFactory)
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
plugins:
  log:
    test-internal:
      - writer: console                 
        level: debug
`)
	_, ok := trpc.GlobalConfig().Plugins["log"]["test-internal"]
	require.True(s.T(), ok)
}

func (s *TestSuite) TestCustomPluginRegisterOk() {
	plugin.Register("timeout", &timeOutPlugin{})
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
plugins:
  test:
    timeout:
`)
	_, ok := trpc.GlobalConfig().Plugins["test"]["timeout"]
	require.True(s.T(), ok)
}

func (s *TestSuite) TestPluginRegisterFailed() {
	defer func() {
		p := recover()
		require.NotNil(s.T(), p, "please import plugin-package before start server.")
		require.Contains(s.T(), fmt.Sprint(p), "plugin test:plugin1 no registered or imported")
	}()
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
plugins:
  test:
    plugin1:
`)
}

func (s *TestSuite) TestPluginViolateRegistrationOrder() {
	defer func() {
		p := recover()
		require.NotNil(s.T(), p)
		require.Contains(s.T(), fmt.Sprint(p), "depends plugin test-Timeout not exists")
	}()

	plugin.Register("dependTimeoutPlugin", &dependTimeoutPlugin{})
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
plugins:
  test:
    dependTimeoutPlugin:
`)
}

type dependTimeoutPlugin struct{}

func (p *dependTimeoutPlugin) Type() string {
	return "test"
}

func (p *dependTimeoutPlugin) Setup(_ string, _ plugin.Decoder) error {
	return nil
}

func (p *dependTimeoutPlugin) DependsOn() []string {
	return []string{"test-Timeout"}
}
