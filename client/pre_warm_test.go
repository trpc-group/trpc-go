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

package client_test

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestInitializableClientPreWarm(t *testing.T) {
	cli, ok := client.New().(client.InitializableClient)
	require.True(t, ok)

	pool := &recordPool{}
	require.NoError(t, cli.Init(context.Background(),
		client.WithTarget("prewarm-selector://backend"),
		client.WithPool(pool),
		client.WithPreWarm(transport.PreWarmOptions{ConnsPerNode: 2}),
	))
	require.Equal(t, []string{"10.0.0.1:8000", "10.0.0.1:8000"}, pool.addresses)
}

func TestInitializableClientPreWarmDisabled(t *testing.T) {
	cli, ok := client.New().(client.InitializableClient)
	require.True(t, ok)

	pool := &recordPool{}
	require.NoError(t, cli.Init(context.Background(),
		client.WithTarget("prewarm-selector://backend"),
		client.WithPool(pool),
		client.WithPreWarm(transport.PreWarmOptions{}),
	))
	require.Empty(t, pool.addresses)
}

func TestInitializableClientPreWarmInvalidConfig(t *testing.T) {
	cli, ok := client.New().(client.InitializableClient)
	require.True(t, ok)

	err := cli.Init(context.Background(),
		client.WithTarget("prewarm-selector://backend"),
		client.WithDisableConnectionPool(),
		client.WithPreWarm(transport.PreWarmOptions{ConnsPerNode: 1}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "connection pool is disabled")
}

func TestInitializableClientPreWarmTimeout(t *testing.T) {
	cli, ok := client.New().(client.InitializableClient)
	require.True(t, ok)

	err := cli.Init(context.Background(),
		client.WithTarget("prewarm-selector://backend"),
		client.WithPool(&blockingPool{}),
		client.WithPreWarm(transport.PreWarmOptions{ConnsPerNode: 1, Timeout: time.Millisecond}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), context.DeadlineExceeded.Error())
}

type preWarmSelector struct{}

func (preWarmSelector) Select(serviceName string, _ ...selector.Option) (*registry.Node, error) {
	if serviceName == "backend" {
		return &registry.Node{Network: "tcp", Address: "10.0.0.1:8000"}, nil
	}
	return nil, errors.New("unknown service")
}

func (preWarmSelector) Report(*registry.Node, time.Duration, error) error { return nil }

func init() {
	selector.Register("prewarm-selector", preWarmSelector{})
}

type recordPool struct {
	mu        sync.Mutex
	addresses []string
}

func (p *recordPool) Get(_ string, address string, _ connpool.GetOptions) (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.addresses = append(p.addresses, address)
	return preWarmNopConn{}, nil
}

type blockingPool struct{}

func (blockingPool) Get(_ string, _ string, opts connpool.GetOptions) (net.Conn, error) {
	<-opts.Ctx.Done()
	return nil, opts.Ctx.Err()
}

type preWarmNopConn struct{}

func (preWarmNopConn) Read(_ []byte) (int, error)       { return 0, io.EOF }
func (preWarmNopConn) Write(p []byte) (int, error)      { return len(p), nil }
func (preWarmNopConn) Close() error                     { return nil }
func (preWarmNopConn) LocalAddr() net.Addr              { return newUnresolvedAddr("tcp", "local") }
func (preWarmNopConn) RemoteAddr() net.Addr             { return newUnresolvedAddr("tcp", "remote") }
func (preWarmNopConn) SetDeadline(time.Time) error      { return nil }
func (preWarmNopConn) SetReadDeadline(time.Time) error  { return nil }
func (preWarmNopConn) SetWriteDeadline(time.Time) error { return nil }
