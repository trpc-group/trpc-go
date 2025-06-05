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

package multiplexed

import (
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

// PoolOptions represents some settings for the connection pool.
type PoolOptions struct {
	connectNumberPerHost        int           // The number of connections per address.
	sendQueueSize               int           // The length of each Connection send queue.
	dropFull                    bool          // Whether to drop the request when the queue is full.
	dialTimeout                 time.Duration // Connection timeout, default 1s.
	maxVirConnsPerConn          int           // Max number of virtual connections per real connection, 0 means no limit.
	maxIdleConnsPerHost         int           // The maximum number of idle connections for each peer ip:port.
	maxReconnectCount           int           // The maximum number of reconnection attempts, 0 means reconnect is disable.
	initialBackoff              time.Duration // The initial backoff time during the first reconnection attempt.
	maxBackoff                  time.Duration // The maximum backoff time between reconnection attempts.
	reconnectCountResetInterval time.Duration // The interval after which the reconnectCount is reset.
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

// WithRecvQueueSize returns an Option which sets the queue length of the VirtualConnection to receive data,
// if the data exceeds the queue length, a packet loss error will be returned
//
// Deprecated: receive queue size is unlimited now.
func WithRecvQueueSize(n int) PoolOption {
	return func(opts *PoolOptions) {}
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

// WithMaxReconnectCount sets the maxReconnectCount reconnection.
// Depending on the value of the input parameter n,
// the behavior of the function varies:
//   - If n is 0, the reconnect will be disable.
//   - If n is less than 0, a warning is logged and maxReconnectCount will be set to defaultMaxReconnectCount.
//   - Otherwise, maxReconnectCount is set to n and the maxBackoff and reconnectionCountResetInterval are adjusted.
func WithMaxReconnectCount(n int) PoolOption {
	return func(opts *PoolOptions) {
		opts.maxReconnectCount = n
	}
}

// WithInitialBackoff sets the initialBackoff for reconnection.
// Depending on the value of the input parameter d and the value of maxReconnectCount,
// the behavior of the function varies:
//   - If maxReconnectCount is 0, the reconnect is disable, so make no changes.
//   - If d is less than or equal to 0, a warning is logged and initialBackoff will be set to defaultInitialBackoff.
//   - Otherwise, initialBackoff is set to d and the maxBackoff and reconnectionCountResetInterval are adjusted.
func WithInitialBackoff(d time.Duration) PoolOption {
	return func(opts *PoolOptions) {
		opts.initialBackoff = d
	}
}

// WithReconnectCountResetInterval sets the reconnectCountResetInterval for reconnection.
// Due to the presence of dialTimeout, users need to set reconnectCountResetInterval to a larger value.
// details: https://git.woa.com/trpc-go/trpc-go/issues/990.
func WithReconnectCountResetInterval(d time.Duration) PoolOption {
	return func(opts *PoolOptions) {
		opts.reconnectCountResetInterval = d
	}
}

// checkReconnectParams checks the params for reconnect.
// When reconnect is disabled by setting maxReconnectCount == 0, invoke disableReconnect().
func (o *PoolOptions) checkReconnectParams() error {
	if o.maxReconnectCount == 0 {
		log.Info("disable reconnect for the multiplex connection pool")
		o.disableReconnect()
		return nil
	}

	if o.maxReconnectCount < 0 {
		return fmt.Errorf("failed to set maxReconnectCount = %v,"+
			"maxReconnectCount should be equal or greater than 0", o.maxReconnectCount)
	}

	if o.initialBackoff <= 0 {
		return fmt.Errorf("failed to set initialBackoff = %v,"+
			"initialBackoff should be greater than 0", o.initialBackoff)
	}

	// maxBackoff and reconnectCountResetInterval are calculated by linear Backoff strategy.
	o.maxBackoff = o.initialBackoff * time.Duration(o.maxReconnectCount)

	// By default, reconnectCountResetInterval is 2 *(sum(backoffTime) + sum(dialTime)).
	// reconnectCountResetInterval must be greater than sum(backoffTime) + sum(dialTime).
	minReconnectCountResetInterval := o.initialBackoff * time.Duration((1+o.maxReconnectCount)*o.maxReconnectCount) / 2
	// To avoid the impact of the last dial during retries,
	// reconnectCountResetInterval needs to include the dialTimeout duration.
	// Details: https://git.woa.com/trpc-go/trpc-go/issues/990.
	dt := defaultDialTimeout
	if o.dialTimeout != 0 {
		dt = o.dialTimeout
	}
	minReconnectCountResetInterval += time.Duration(o.maxReconnectCount) * dt

	// reconnectCountResetInterval is not set by user.
	if o.reconnectCountResetInterval == 0 {
		o.reconnectCountResetInterval = 2 * minReconnectCountResetInterval
	}

	// reconnectCountResetInterval is set by user mistakenly.
	if o.reconnectCountResetInterval <= minReconnectCountResetInterval {
		return fmt.Errorf("failed to set reconnectCountResetInterval = %v,"+
			"initialBackoff should be greater than %v", o.reconnectCountResetInterval, minReconnectCountResetInterval)
	}

	return nil
}

// disableReconnect just set all reconnect params to 0.
func (o *PoolOptions) disableReconnect() {
	// maxReconnectCount equal to zero means that Reconnect is disable.
	o.maxReconnectCount = 0
	o.initialBackoff = 0
	o.maxBackoff = 0
	o.reconnectCountResetInterval = 0
}
