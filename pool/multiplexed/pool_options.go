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

package multiplexed

import "time"

// PoolOptions represents some settings for the connection pool.
type PoolOptions struct {
	connectNumberPerHost int           // Set the number of connections per address.
	sendQueueSize        int           // Set the length of each Connection send queue.
	dropFull             bool          // Whether the queue is full or not.
	dialTimeout          time.Duration // Connection timeout, default 1s.
	maxVirConnsPerConn   int           // Max number of virtual connections per real connection, 0 means no limit.
	maxIdleConnsPerHost  int           // Set the maximum number of idle connections for each peer ip:port.
}

// PoolOption is the Options helper.
type PoolOption func(*PoolOptions)

// WithConnectNumber returns an Option which sets the number of connections for each peer in the connection pool.
func WithConnectNumber(number int) PoolOption {
	return func(opts *PoolOptions) {
		opts.connectNumberPerHost = number
	}
}

// WithQueueSize returns an Option which sets the length of each Connection sending queue in the connection pool.
func WithQueueSize(n int) PoolOption {
	return func(opts *PoolOptions) {
		opts.sendQueueSize = n
	}
}

// WithDropFull returns an Option which sets whether to drop the request when the queue is full.
func WithDropFull(drop bool) PoolOption {
	return func(opts *PoolOptions) {
		opts.dropFull = drop
	}
}

// WithDialTimeout returns an Option which sets the connection timeout.
func WithDialTimeout(d time.Duration) PoolOption {
	return func(opts *PoolOptions) {
		opts.dialTimeout = d
	}
}

// WithMaxVirConnsPerConn returns an Option which sets the maximum number of virtual
// connections per real connection, 0 means no limit.
func WithMaxVirConnsPerConn(n int) PoolOption {
	return func(opts *PoolOptions) {
		opts.maxVirConnsPerConn = n
	}
}

// WithMaxIdleConnsPerHost returns an Option which sets the maximum number of idle connections
// for each peer ip:port, this value should not be less than ConnectNumber,
// This option is usually used with MaxVirConnsPerConn in streaming scenarios
// to dynamically adjust the number of connections,
// This option takes effect only when MaxVirConnsPerConn is set, 0 means no limit.
func WithMaxIdleConnsPerHost(n int) PoolOption {
	return func(opts *PoolOptions) {
		opts.maxIdleConnsPerHost = n
	}
}
