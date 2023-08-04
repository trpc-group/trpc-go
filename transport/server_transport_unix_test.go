//go:build !windows
// +build !windows

package transport_test

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestServerTransport_ListenAndServeUnix(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	t.Run("disable reuse port", func(t *testing.T) {
		require.Nil(t, transport.NewServerTransport(
			transport.WithReusePort(false),
		).ListenAndServe(
			context.Background(),
			transport.WithListenNetwork("unix"),
			transport.WithListenAddress(fmt.Sprintf("test%d.sock", rand.Int63())),
			transport.WithServerFramerBuilder(&framerBuilder{}),
		))
	})

	t.Run("enable reuse port", func(t *testing.T) {
		require.Nil(t, transport.NewServerTransport(
			transport.WithReusePort(true),
		).ListenAndServe(
			context.Background(),
			transport.WithListenNetwork("unix"),
			transport.WithListenAddress(fmt.Sprintf("test%d.sock", rand.Int63())),
			transport.WithServerFramerBuilder(&framerBuilder{}),
		))
	})
}

func TestGetPassedListenerErr(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	addr := listener.Addr().String()
	ln := listener.(*net.TCPListener)
	file, _ := ln.File()

	_ = os.Setenv(transport.EnvGraceFirstFd, fmt.Sprint(file.Fd()))
	_ = os.Setenv(transport.EnvGraceRestartFdNum, "1")

	_, err = transport.GetPassedListener("tcp", fmt.Sprintf("localhost:%d", savedListenerPort))
	assert.NotNil(t, err)

	// Simulate fd derived from environment.
	_, err = transport.GetPassedListener("tcp", addr)
	assert.Nil(t, err)

	_ = os.Setenv(transport.EnvGraceRestart, "true")
	fb := transport.GetFramerBuilder("trpc")

	st := transport.NewServerTransport(transport.WithReusePort(false))
	err = st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("tcp"),
		transport.WithListenAddress(addr),
		transport.WithServerFramerBuilder(fb))
	assert.NotNil(t, err)

	err = st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress(addr),
		transport.WithServerFramerBuilder(fb))
	assert.NotNil(t, err)

	_ = os.Setenv(transport.EnvGraceRestart, "")
}
