//go:build !windows
// +build !windows

package transport_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestST_UnixDomain(t *testing.T) {
	// Disable reuse port
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Millisecond * 100) // Ensure the unix listener is closed.
	})
	require.Nil(t, transport.NewServerTransport(
		transport.WithReusePort(false),
	).ListenAndServe(
		ctx,
		transport.WithListenNetwork("unix"),
		transport.WithListenAddress(fmt.Sprintf("%s/test.sock", t.TempDir())),
		transport.WithServerFramerBuilder(&framerBuilder{}),
	))

	// Enable reuse port
	require.Nil(t, transport.NewServerTransport(
		transport.WithReusePort(true),
	).ListenAndServe(
		ctx,
		transport.WithListenNetwork("unix"),
		transport.WithListenAddress(fmt.Sprintf("%s/test.sock", t.TempDir())),
		transport.WithServerFramerBuilder(&framerBuilder{}),
	))
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
