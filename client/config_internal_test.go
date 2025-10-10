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

package client

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestConnTypeConnPool(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
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

func TestConnTypeMultiplexed(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
conn_type: multiplexed  # connection type is multiplexed, the following options are all for multiplex.
multiplexed:
  multiplexed_dial_timeout: 1s  # multiplexed: dial timeout, default 1s.
  conns_per_host: 2  # multiplexed: number of concrete(real) connections for each host, default 2.
  max_vir_conns_per_conn: 0  # multiplexed: max number of virtual connections for each concrete(real) connection, default 0 (means no limit).
  max_idle_conns_per_host: 0  # multiplexed: max number of idle concrete(real) connections for each host, used together with max_vir_conns_per_conn, default 0 (disabled).
  queue_size: 1024  # multiplexed: size of send queue for each concrete(real) connection, default 1024.
  drop_full: false  # multiplexed: whether to drop the send package when queue is full, default false.
  max_reconnect_count: 10 # multiplexed: the maximum number of reconnection attempts, 0 means reconnect is disable.
  initial_backoff: 5ms # multiplexed: the initial backoff time during the first reconnection attempt.
  reconnect_count_reset_interval: 600s # multiplexed: the time to reset the reconnect counts.
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

func TestConnTypeShort(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
conn_type: short
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

func TestConnTypeHTTPPool(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
conn_type: httppool  # connection type is http pool.
`), &backendConfig))
	_, err := backendConfig.genOptions()
	require.Nil(t, err)
}

func TestConnTypeShortWithHTTP(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
transport: http
conn_type: short  # connection type is short pool.
`), &backendConfig))
	opts, err := backendConfig.genOptions()
	require.Nil(t, err)
	o := &transport.RoundTripOptions{}
	for _, opt := range opts.CallOptions {
		opt(o)
	}
	require.True(t, o.DisableConnectionPool)
	require.False(t, o.EnableMultiplexed)
	require.Nil(t, o.Multiplexed)
	require.Nil(t, o.Pool)
}

func TestConnTypeConnPoolWithHTTP(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
transport: http
conn_type: connpool  # connection type is connection pool.
`), &backendConfig))
	_, err := backendConfig.genOptions()
	require.NotNil(t, err)
}

func TestConnTypeMultiplexedWithHTTP(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
transport: http
conn_type: multiplexed  # connection type is multiplexed.
`), &backendConfig))
	_, err := backendConfig.genOptions()
	require.NotNil(t, err)
}

func TestConnTypeHTTPPoolWithHTTP(t *testing.T) {
	backendConfig := BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
transport: http
conn_type: httppool  # connection type is httppool, the following options are all for httppool.
httppool:
  max_idle_conns: 100  # httppool: max number of idle connections, default 0 (means no limit).
  max_idle_conns_per_host: 10  # httppool: max number of idle connections per-host, default 2.
  max_conns_per_host: 20  # httppool: max number of connections, default 0 (means no limit).
  idle_conn_timeout: 1s  # httppool: idle timeout, default 0s (means no limit).
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
	require.Nil(t, o.Pool)
	require.Equal(t, 100, o.HTTPOpts.Pool.MaxIdleConns)
	require.Equal(t, 10, o.HTTPOpts.Pool.MaxIdleConnsPerHost)
	require.Equal(t, 20, o.HTTPOpts.Pool.MaxConnsPerHost)
	require.Equal(t, time.Second, o.HTTPOpts.Pool.IdleConnTimeout)
}

func TestGetConfigWithTag(t *testing.T) {
	RegisterConfig(nil)
	defer RegisterConfig(nil)
	_, err := GetConfig(t.Name(), "")
	require.NotNil(t, err)

	cfg1 := &BackendConfig{
		Callee:      t.Name(),
		ServiceName: t.Name(), // backend service name
		Tag:         "tag1",
		Target:      "ip://1.1.1.1:1111", // backend address
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
	}

	RegisterClientConfig(cfg1.Callee, cfg1)

	conf, err := getConfigWithTag(cfg1.Callee, cfg1.ServiceName, cfg1.Tag)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if conf == nil {
		t.Error("Expected config, got nil")
	}

	conf, err = getConfigWithTag(cfg1.Callee, cfg1.ServiceName, cfg1.Tag+"non-existed")
	expectedErr := errors.New(
		"client config: callee TestGetConfigWithTag service name TestGetConfigWithTag tag tag1non-existed not found")
	if err == nil || err.Error() != expectedErr.Error() {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if conf != nil {
		t.Error("Expected nil config, got non-nil")
	}

	conf, err = getConfigWithTag(cfg1.Callee, cfg1.ServiceName+"non-existed", cfg1.Tag)
	expectedErr = errors.New(
		"client config: callee TestGetConfigWithTag service name TestGetConfigWithTagnon-existed tag tag1 not found")
	if err == nil || err.Error() != expectedErr.Error() {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if conf != nil {
		t.Error("Expected nil config, got non-nil")
	}

	conf, err = getConfigWithTag(cfg1.Callee+"non-existed", cfg1.ServiceName, cfg1.Tag)
	expectedErr = errors.New(
		"client config: callee TestGetConfigWithTagnon-existed service name TestGetConfigWithTag tag tag1 not found")
	if err == nil || err.Error() != expectedErr.Error() {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if conf != nil {
		t.Error("Expected nil config, got non-nil")
	}
}
