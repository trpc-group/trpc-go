// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package connpool

import (
	"time"
)

// Options indicates pool configuration.
type Options struct {
	MinIdle   int // Initialize the number of connections, ready for the next io.
	MaxIdle   int // Maximum number of idle connections, 0 means no idle.
	MaxActive int // Maximum number of active connections, 0 means no limit.
	// Whether to wait when the maximum number of active connections is reached.
	Wait               bool
	IdleTimeout        time.Duration // idle connection timeout.
	MaxConnLifetime    time.Duration // Maximum lifetime of the connection.
	DialTimeout        time.Duration // Connection establishment timeout.
	ForceClose         bool
	Dial               DialFunc
	Checker            HealthChecker
	PushIdleConnToTail bool          // connection to ip will be push tail when ConnectionPool.put method is called
	PoolIdleTimeout    time.Duration // ConnectionPool idle timeout
}

// Option is the Options helper.
type Option func(*Options)

// WithMinIdle returns an Option which sets the number of initialized connections.
func WithMinIdle(n int) Option {
	return func(o *Options) {
		o.MinIdle = n
	}
}

// WithMaxIdle returns an Option which sets the maximum number of idle connections.
func WithMaxIdle(m int) Option {
	return func(o *Options) {
		o.MaxIdle = m
	}
}

// WithMaxActive returns an Option which sets the maximum number of active connections.
func WithMaxActive(s int) Option {
	return func(o *Options) {
		o.MaxActive = s
	}
}

// WithWait returns an Option which sets whether to wait when the number of connections reaches the limit.
func WithWait(w bool) Option {
	return func(o *Options) {
		o.Wait = w
	}
}

// WithIdleTimeout returns an Option which sets the idle connection time, after which it may be closed.
func WithIdleTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.IdleTimeout = t
	}
}

// WithMaxConnLifetime returns an Option which sets the maximum lifetime of
// the connection, after which it may be closed.
func WithMaxConnLifetime(t time.Duration) Option {
	return func(o *Options) {
		o.MaxConnLifetime = t
	}
}

// WithDialTimeout returns an Option which sets the default timeout time for
// the connection pool to establish a connection.
func WithDialTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.DialTimeout = t
	}
}

// WithForceClose returns an Option which sets whether to force the connection to be closed.
func WithForceClose(f bool) Option {
	return func(o *Options) {
		o.ForceClose = f
	}
}

// WithDialFunc returns an Option which sets dial function.
func WithDialFunc(d DialFunc) Option {
	return func(o *Options) {
		o.Dial = d
	}
}

// WithHealthChecker returns an Option which sets health checker.
func WithHealthChecker(c HealthChecker) Option {
	return func(o *Options) {
		o.Checker = c
	}
}

// WithPushIdleConnToTail returns an Option which sets PushIdleConnToTail flag.
func WithPushIdleConnToTail(c bool) Option {
	return func(o *Options) {
		o.PushIdleConnToTail = c
	}
}

// WithPoolIdleTimeout returns an Option which sets pool idle timeout.
// after the timeout, ConnectionPool resource may be cleaned up.
func WithPoolIdleTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.PoolIdleTimeout = t
	}
}
