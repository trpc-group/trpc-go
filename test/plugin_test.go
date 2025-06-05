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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestTimeoutAtPluginRegister() {
	defer func() {
		p := recover()
		require.NotNil(s.T(), p)
		require.Contains(s.T(), fmt.Sprint(p), "setup plugin fail")
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
		require.Contains(s.T(), fmt.Sprint(p), "plugin test: plugin1 no registered or imported")
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
	require.Panics(s.T(), func() {
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
	}, "depends plugin test-timeout not exists")
}

func (s *TestSuite) TestPluginObeyRegistrationOrder() {
	plugin.Register("timeout", &timeOutPlugin{})
	plugin.Register("dependTimeoutPlugin", &dependTimeoutPlugin{})
	require.NotPanics(s.T(), func() {
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
    dependTimeoutPlugin:
`)
	})
}

type dependTimeoutPlugin struct{}

func (p *dependTimeoutPlugin) Type() string {
	return "test"
}

func (p *dependTimeoutPlugin) Setup(_ string, _ plugin.Decoder) error {
	return nil
}

func (p *dependTimeoutPlugin) DependsOn() []string {
	return []string{"test-timeout"}
}

func (s *TestSuite) TestPluginOnFinish() {
	s.T().Run("ok", func(t *testing.T) {
		ch := make(chan string, 1)
		plugin.Register("timeout", &timeOutPlugin{})
		plugin.Register("dependTimeoutPlugin", &dependTimeoutPlugin{})
		plugin.Register("hasOnFinishPlugin", &hasOnFinishPlugin{ch: ch, onFinishErr: nil})
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
    dependTimeoutPlugin:
  config:
    hasOnFinishPlugin:
`)
		require.Contains(s.T(), <-ch, "all other plugins' loading has been done")
	})
	s.T().Run("failed", func(t *testing.T) {
		ch := make(chan string, 1)
		require.Panics(t, func() {
			plugin.Register("timeout", &timeOutPlugin{})
			plugin.Register("dependTimeoutPlugin", &dependTimeoutPlugin{})
			plugin.Register("hasOnFinishPlugin", &hasOnFinishPlugin{ch: ch, onFinishErr: errors.New("failed")})
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
    dependTimeoutPlugin:
  config:
    hasOnFinishPlugin:
`)
		})
	})

}

type hasOnFinishPlugin struct {
	ch          chan string
	onFinishErr error
}

func (p *hasOnFinishPlugin) Type() string {
	return "config"
}

func (p *hasOnFinishPlugin) Setup(_ string, _ plugin.Decoder) error {
	return nil
}

func (p *hasOnFinishPlugin) OnFinish(name string) error {
	p.ch <- fmt.Sprintf("%s: all other plugins' loading has been done", name)
	return p.onFinishErr
}

func (s *TestSuite) TestPluginWithSameType() {
	// set logger to file
	logDir := s.T().TempDir()
	logger := log.NewZapLog(log.Config{
		{
			Writer: log.OutputFile,
			WriteConfig: log.WriteConfig{

				LogPath:   logDir,
				Filename:  "trpc.log",
				WriteMode: log.WriteSync,
			},
			Level: "DEBUG",
		},
	})
	dftLogger := log.DefaultLogger
	log.SetLogger(logger)
	defer log.SetLogger(dftLogger)

	// register service and plugin
	oldPath := trpc.ServerConfigPath
	trpc.ServerConfigPath = "trpc_go_trpc_server_with_plugin.yaml"
	defer func() { trpc.ServerConfigPath = oldPath }()
	plugin.Register("default", &displayPlugin{})
	plugin.Register("timeout", &displayPlugin{})
	svr := trpc.NewServer()
	testpb.RegisterTestTRPCService(svr.Service("trpc.testing.end2end.TestTRPC"), &TRPCService{})

	// read log from file
	fp := filepath.Join(logDir, "trpc.log")
	buf, err := os.ReadFile(fp)
	s.Nil(err)

	s.Contains(string(buf), "timeout-key")
	s.Contains(string(buf), "default-key")
}

type displayPlugin struct {
	Key string `yaml:"key"`
}

func (p *displayPlugin) Type() string {
	return "test"
}

func (p *displayPlugin) Setup(name string, decoder plugin.Decoder) error {
	if err := decoder.Decode(&p); err != nil {
		return err
	}

	log.Infof("[plugin] init displayPlugin success, key: %v", p.Key)

	return nil
}
