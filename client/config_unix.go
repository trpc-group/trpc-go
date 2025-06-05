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
	"fmt"

	"trpc.group/trpc-go/trpc-go/transport"
	tnettransport "trpc.group/trpc-go/trpc-go/transport/tnet"
	tnetmultiplexed "trpc.group/trpc-go/trpc-go/transport/tnet/multiplex"
)

// tnetClientPoolOption return transport roundtrip option for tnet.
func (cfg *BackendConfig) tnetClientPoolOption() (transport.RoundTripOption, error) {
	switch *cfg.ConnType {
	case ConnTypeShort:
		return transport.WithDisableConnectionPool(), nil
	case ConnTypeConnPool:
		return cfg.tnetClientConnPoolOption(), nil
	case ConnTypeMultiplexedPool:
		return cfg.tnetClientMultiplexedPoolOption(), nil
	default:
		return nil,
			fmt.Errorf("transport %v doesn't support connection type %v; supported connection types are [%v, %v, %v]",
				cfg.Transport, *cfg.ConnType, ConnTypeShort, ConnTypeConnPool, ConnTypeMultiplexedPool)
	}
}

// tnetClientPoolOption return transport roundtrip option for tnet connection pool.
func (cfg *BackendConfig) tnetClientConnPoolOption() transport.RoundTripOption {
	// tnet connection pool options is the same as gonet.
	return transport.WithDialPool(tnettransport.NewConnectionPool(cfg.connpoolOptions()...))
}

// tnetClientPoolOption return transport roundtrip option for tnet multiplexed pool.
func (cfg *BackendConfig) tnetClientMultiplexedPoolOption() transport.RoundTripOption {
	var opts []tnetmultiplexed.OptPool
	if cfg.Multiplexed.MultiplexedDialTimeout != nil {
		opts = append(opts, tnetmultiplexed.WithDialTimeout(*cfg.Multiplexed.MultiplexedDialTimeout))
	}
	if cfg.Multiplexed.MaxVirConnsPerConn != nil {
		opts = append(opts, tnetmultiplexed.WithMaxConcurrentVirtualConnsPerConn(*cfg.Multiplexed.MaxVirConnsPerConn))
	}
	// Option enable_metrics is only used in tnet.
	if cfg.Multiplexed.EnableMetrics != nil && *cfg.Multiplexed.EnableMetrics {
		opts = append(opts, tnetmultiplexed.WithEnableMetrics())
	}
	return transport.WithMultiplexedPool(tnettransport.NewMultiplexdPool(opts...))
}
