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

package connpool

import (
	"time"
)

// Options indicates pool configuration.
type Options struct {
	// Dial Initializes the connection.
	Dial DialFunc
	// Checker checks idle connection health.
	Checker HealthChecker
	// AdditionalCheckers are additional health checkers.
	AdditionalCheckers []HealthChecker

	// MinIdle is minimal number of connections, ready for the next io.
	MinIdle int
	// MaxIdle is maximum number of idle connections, 0 means no idle.
	MaxIdle int
	// MaxActive is maximum number of active connections, 0 means no limit.
	MaxActive int
	// Wait decides wait when the max number of active connections is reached or not.
	Wait bool
	// ForceClose closes the connection, suitable for streaming scenarios.
	ForceClose bool
	// connection to ip will be push tail when ConnectionPool.put method is called.
	PushIdleConnToTail bool

	// IdleTimeout is the idle timeout of connection.
	IdleTimeout time.Duration
	// MaxConnLifetime is the maximum lifetime of the connection.
	MaxConnLifetime time.Duration
	// DialTimeout is the timeout of connection establishment.
	DialTimeout time.Duration
	// PoolIdleTimeout is the idle timeout of pool.
	PoolIdleTimeout time.Duration
}

// Option is the Options helper.
type Option func(*Options)

// WithMinIdle returns an Option which sets the number of initialized connections.
func WithMinIdle(n int) Option {
	return func(o *Options) {
		o.MinIdle = n
	}
}

// WithMaxIdle returns an Option which sets the maximum number of idle connections. 0 means no idle number limit.
func WithMaxIdle(i int) Option {
	return func(o *Options) {
		o.MaxIdle = i
	}
}

// WithMaxActive returns an Option which sets the maximum number of active connections. 0 means no number limit.
func WithMaxActive(a int) Option {
	return func(o *Options) {
		o.MaxActive = a
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

// WithAdditionalHealthChecker returns an Option which sets additional health checker.
// The additional checker will be called after the main health checker.
// This function can be called multiple times, the additional checkers will be used in order.
func WithAdditionalHealthChecker(c ...HealthChecker) Option {
	return func(o *Options) {
		o.AdditionalCheckers = append(o.AdditionalCheckers, c...)
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
