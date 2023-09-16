// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build !windows
// +build !windows

package server_test

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestStartNewProcess(t *testing.T) {
	// If the process is started by graceful restart,
	// exit here in case of infinite loop.
	if len(os.Getenv(transport.EnvGraceRestart)) > 0 {
		t.SkipNow()
	}
	s := &server.Server{}
	cfg := &trpc.Config{}
	cfg.Server.Admin.IP = "127.0.0.1"
	cfg.Server.Admin.Port = 9028
	opts := []admin.Option{
		admin.WithVersion(trpc.Version()),
		admin.WithAddr(fmt.Sprintf("%s:%d", cfg.Server.Admin.IP, cfg.Server.Admin.Port)),
		admin.WithTLS(cfg.Server.Admin.EnableTLS),
	}

	adminService := admin.NewServer(opts...)
	s.AddService(admin.ServiceName, adminService)

	service := server.New(server.WithAddress("127.0.0.1:9080"),
		server.WithNetwork("tcp"),
		server.WithProtocol("trpc"),
		server.WithServiceName("trpc.test.helloworld.Greeter1"))

	s.AddService("trpc.test.helloworld.Greeter1", service)
	err := s.Register(nil, nil)
	assert.NotNil(t, err)

	impl := &GreeterServerImpl{}
	err = s.Register(&GreeterServerServiceDesc, impl)
	assert.Nil(t, err)
	go func() {
		var netOpError *net.OpError
		assert.ErrorAs(
			t,
			s.Serve(),
			&netOpError,
			`it is normal to have "use of closed network connection" error during hot restart`,
		)
	}()
	time.Sleep(time.Second * 1)

	log.Info(os.Environ())
	// The environment variable is not set for parent process.
	// It will be set by the child process started by graceful restart.
	if os.Getenv(transport.EnvGraceRestart) == "" {
		fpid := os.Getpid()
		// graceful restart
		cpid, err := s.StartNewProcess("-test.run=Test[^StartNewProcess$]")
		assert.Nil(t, err)
		assert.NotEqual(t, fpid, cpid)
		t.Logf("fpid:%v, cpid:%v", fpid, cpid)
	}
	// Sleep 10s, let the parent process rewrite test coverage. The child process will exit quickly.
	time.Sleep(time.Second * 10)
	err = s.Close(nil)
	assert.Nil(t, err)
}

func TestCloseOldListenerDuringHotRestart(t *testing.T) {
	// If the process is started by graceful restart,
	// exit here in case of infinite loop.
	if len(os.Getenv(transport.EnvGraceRestart)) > 0 {
		t.SkipNow()
	}
	s := &server.Server{}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	service := server.New(
		server.WithNetwork("tcp"),
		server.WithProtocol("trpc"),
		server.WithServiceName("trpc.test.helloworld.Greeter1"),
		server.WithListener(ln),
	)

	s.AddService("trpc.test.helloworld.Greeter1", service)
	err = s.Register(&GreeterServerServiceDesc, &GreeterServerImpl{})
	go func() {
		err = s.Serve()
		assert.Nil(t, err)
	}()
	time.Sleep(time.Second)

	log.Info(os.Environ())
	// The environment variable is not set for parent process.
	// It will be set by the child process started by graceful restart.
	if os.Getenv(transport.EnvGraceRestart) == "" {
		fpid := os.Getpid()
		// Graceful restart
		cpid, err := s.StartNewProcess("-test.run=^TestCloseOldListenerDuringHotRestart$")
		require.Nil(t, err)
		require.NotEqual(t, fpid, cpid)
		t.Logf("fpid:%v, cpid:%v", fpid, cpid)
		time.Sleep(time.Second)
		// Child will not be up in this test case, so trying to connect won't work.
		_, err = net.Dial("tcp", ln.Addr().String())
		t.Logf("dial err: %+v", err)
		require.NotNil(t, err)
	}
	// Sleep 1s, let the parent process rewrite test coverage. The child process will exit quickly.
	time.Sleep(time.Second)
	err = s.Close(nil)
	require.Nil(t, err)
}
