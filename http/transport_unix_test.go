// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build !windows
// +build !windows

package http_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"
)

// TestPassedListener tests passing listener.
func TestPassedListener(t *testing.T) {
	ctx := context.Background()
	addr := "127.0.0.1:28084"
	key := "TRPC_TEST_HTTP_PASSED_LISTENER"
	if value := os.Getenv(key); value == "1" {
		time.Sleep(1 * time.Second)
		os.Unsetenv(key)
		// child process, tests whether the listener fd can be got.
		ln, err := transport.GetPassedListener("tcp", addr)
		require.Nil(t, err)
		require.NotNil(t, ln)
		require.Nil(t, ln.(net.Listener).Close())
		return
	}

	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	option := transport.WithListenAddress(addr)
	handler := transport.WithHandler(transport.Handler(&h{}))
	err := tp.ListenAndServe(ctx, option, handler)
	require.Nil(t, err)
	os.Setenv(key, "1")
	s := server.Server{}
	os.Args = os.Args[0:1]
	os.Args = append(os.Args, "-test.run", "^TestPassedListener$")
	time.Sleep(time.Millisecond)
	cpid, err := s.StartNewProcess()
	require.Nil(t, err)

	process, err := os.FindProcess(int(cpid))
	require.Nil(t, err)
	ps, err := process.Wait()
	require.Nil(t, err)
	require.Equal(t, 0, ps.ExitCode())
}
