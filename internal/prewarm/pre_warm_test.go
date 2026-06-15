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

package prewarm_test

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/prewarm"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestEnabled(t *testing.T) {
	require.True(t, prewarm.Enabled(context.Background(),
		transport.WithPreWarm(transport.PreWarmOptions{ConnsPerNode: 1})))
	require.False(t, prewarm.Enabled(context.Background(),
		transport.WithPreWarm(transport.PreWarmOptions{})))
	require.False(t, prewarm.Enabled(context.Background()))
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name string
		opts *transport.RoundTripOptions
		ok   bool
	}{
		{name: "tcp", opts: &transport.RoundTripOptions{
			Network: "tcp", Pool: connpool.DefaultConnectionPool,
			PreWarm: &transport.PreWarmOptions{ConnsPerNode: 1},
		}, ok: true},
		{name: "tcp4", opts: &transport.RoundTripOptions{
			Network: "tcp4", Pool: connpool.DefaultConnectionPool,
			PreWarm: &transport.PreWarmOptions{ConnsPerNode: 1},
		}, ok: true},
		{name: "tcp6", opts: &transport.RoundTripOptions{
			Network: "tcp6", Pool: connpool.DefaultConnectionPool,
			PreWarm: &transport.PreWarmOptions{ConnsPerNode: 1},
		}, ok: true},
		{name: "unix", opts: &transport.RoundTripOptions{
			Network: "unix", Pool: connpool.DefaultConnectionPool,
			PreWarm: &transport.PreWarmOptions{ConnsPerNode: 1},
		}, ok: true},
		{name: "udp", opts: &transport.RoundTripOptions{
			Network: "udp", Pool: connpool.DefaultConnectionPool,
			PreWarm: &transport.PreWarmOptions{ConnsPerNode: 1},
		}},
		{name: "disabled pool", opts: &transport.RoundTripOptions{
			Network: "tcp", DisableConnectionPool: true,
			PreWarm: &transport.PreWarmOptions{ConnsPerNode: 1},
		}},
		{name: "zero connections", opts: &transport.RoundTripOptions{
			Network: "tcp", Pool: connpool.DefaultConnectionPool,
			PreWarm: &transport.PreWarmOptions{},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := prewarm.ValidateConfig(tt.opts)
			require.Equal(t, tt.ok, err == nil)
		})
	}
}

func TestPreWarmConnPool(t *testing.T) {
	var gets atomic.Int32
	pool := &countingPool{gets: &gets}
	nodes := []*registry.Node{{Network: "tcp", Address: "127.0.0.1:1"}, {Address: "127.0.0.1:2"}}
	opts := &transport.RoundTripOptions{
		Network: "tcp",
		Pool:    pool,
		PreWarm: &transport.PreWarmOptions{ConnsPerNode: 2},
	}
	require.NoError(t, prewarm.PreWarm(context.Background(), nodes, opts))
	require.Equal(t, int32(4), gets.Load())
}

func TestPreWarmConnPoolError(t *testing.T) {
	opts := &transport.RoundTripOptions{
		Network: "tcp",
		Pool:    errPool{},
		PreWarm: &transport.PreWarmOptions{ConnsPerNode: 1},
	}
	err := prewarm.PreWarm(context.Background(), []*registry.Node{{Address: "127.0.0.1:1"}}, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "establish connection")
}

func TestPreWarmMultiplexed(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer l.Close()

	opts := &transport.RoundTripOptions{
		Network:           "tcp",
		Multiplexed:       multiplexed.New(),
		EnableMultiplexed: true,
		Msg:               codec.Message(trpc.BackgroundContext()),
		FramerBuilder:     trpc.DefaultFramerBuilder,
		PreWarm:           &transport.PreWarmOptions{ConnsPerNode: 1},
	}
	nodes := []*registry.Node{{Address: l.Addr().String()}}
	require.NoError(t, prewarm.PreWarm(context.Background(), nodes, opts))
}

func TestPreWarmContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := &transport.RoundTripOptions{
		Network: "tcp",
		Pool:    connpool.DefaultConnectionPool,
		PreWarm: &transport.PreWarmOptions{ConnsPerNode: 1},
	}
	err := prewarm.PreWarm(ctx, []*registry.Node{{Address: "127.0.0.1:1"}}, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), context.Canceled.Error())
}

type countingPool struct {
	gets *atomic.Int32
}

func (p *countingPool) Get(_ string, _ string, _ connpool.GetOptions) (net.Conn, error) {
	p.gets.Add(1)
	return nopConn{}, nil
}

type errPool struct{}

func (errPool) Get(_ string, _ string, _ connpool.GetOptions) (net.Conn, error) {
	return nil, errors.New("get connection failed")
}

type nopConn struct{}

func (nopConn) Read(_ []byte) (int, error)       { return 0, io.EOF }
func (nopConn) Write(p []byte) (int, error)      { return len(p), nil }
func (nopConn) Close() error                     { return nil }
func (nopConn) LocalAddr() net.Addr              { return dummyAddr("local") }
func (nopConn) RemoteAddr() net.Addr             { return dummyAddr("remote") }
func (nopConn) SetDeadline(time.Time) error      { return nil }
func (nopConn) SetReadDeadline(time.Time) error  { return nil }
func (nopConn) SetWriteDeadline(time.Time) error { return nil }

type dummyAddr string

func (a dummyAddr) Network() string { return string(a) }
func (a dummyAddr) String() string  { return string(a) }
