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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package client

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestConnTypeShortWithTNet(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
transport: tnet
conn_type: short  # connection type is short pool.
`), &backendConfig))
	opts, err := backendConfig.genOptions()
	require.Nil(t, err)
	require.False(t, opts.EnableMultiplexed)
	o := &transport.RoundTripOptions{}
	for _, opt := range opts.CallOptions {
		opt(o)
	}
	require.True(t, o.DisableConnectionPool)
	require.False(t, o.EnableMultiplexed)
	require.Nil(t, o.Multiplexed)
	require.Nil(t, o.Pool)
}

func TestConnTypeConnPoolWithTNet(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
transport: tnet
conn_type: connpool  # connection type is connection pool, the following options are all for connpool.
connpool:
  dial_timeout: 200ms  # connection pool: dial timeout, default 200ms.
  force_close: false  # connection pool: whether force close the connection, default false.
  idle_timeout: 50s  # connection pool: idle timeout, default 50s.
  max_active: 0  # connection pool: max active connections, default 0 (means no limit).
  max_conn_lifetime: 0s  # connection pool: max lifetime for connection, default 0s (means no limit).
  max_idle: 65536  # connection pool: max idle connections, default 65536.
  min_idle: 0  # connection pool: min idle connections, default 0.
  pool_idle_timeout: 100s  # connection pool: idle timeout to close the entire pool, default 100s.
  push_idle_conn_to_tail: false  # connection pool: recycle the connection to head/tail of the idle list, default false (head).
  wait: false  # connection pool: whether wait util timeout or return err immediately when number of total connections reach max_active, default false.
`), &backendConfig))
	opts, err := backendConfig.genOptions()
	require.Nil(t, err)
	require.False(t, opts.EnableMultiplexed)
	o := &transport.RoundTripOptions{}
	for _, opt := range opts.CallOptions {
		opt(o)
	}
	require.False(t, o.DisableConnectionPool)
	require.False(t, o.EnableMultiplexed)
	require.Nil(t, o.Multiplexed)
	require.NotNil(t, o.Pool)
}

func TestConnTypeMultiplexedWithTNet(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
transport: tnet
conn_type: multiplexed  # connection type is multiplexed, the following options are all for multiplex.
multiplexed:
  multiplexed_dial_timeout: 1s  # multiplexed: dial timeout, default 1s.
  max_vir_conns_per_conn: 0  # multiplexed: max number of virtual connections for each concrete(real) connection, default 0 (means no limit).
  enable_metrics: true 
`), &backendConfig))
	opts, err := backendConfig.genOptions()
	require.Nil(t, err)
	require.True(t, opts.EnableMultiplexed)
	o := &transport.RoundTripOptions{}
	for _, opt := range opts.CallOptions {
		opt(o)
	}
	require.False(t, o.DisableConnectionPool)
	require.True(t, o.EnableMultiplexed)
	require.NotNil(t, o.Multiplexed)
	require.Nil(t, o.Pool)
}

func TestConnTypeHTTPPoolWithTNet(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
transport: tnet
conn_type: httppool  # connection type is http pool.
`), &backendConfig))
	_, err := backendConfig.genOptions()
	require.NotNil(t, err)
}
