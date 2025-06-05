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

package httppool

import (
	"time"
)

// Options indicates pool configuration.
type Options struct {
	// MaxIdleConns controls the maximum number of idle connections across all hosts, default 0, which means no limit.
	MaxIdleConns int
	// MaxIdleConnsPerHost controls the maximum idle connections to keep per-host, default 2.
	MaxIdleConnsPerHost int
	// MaxConnsPerHost optionally limits the total number of connections per host, default 0, which means no limit.
	MaxConnsPerHost int
	// IdleConnTimeout is the maximum amount of time an idle connection will remain idle before closing,
	// default 0, which means no limit.
	IdleConnTimeout time.Duration
}

// Option is the Options helper.
type Option func(*Options)

// WithMaxIdleConns returns an Option which sets the maximum number of idle connections across all hosts,
// default 0, which means no limit.
func WithMaxIdleConns(m int) Option {
	return func(o *Options) {
		o.MaxIdleConns = m
	}
}

// WithMaxIdleConnsPerHost returns an Option which sets the maximum idle connections to keep per-host, default 2.
func WithMaxIdleConnsPerHost(m int) Option {
	return func(o *Options) {
		o.MaxIdleConnsPerHost = m
	}
}

// WithMaxConnsPerHost returns an Option which sets the total number of connections per host,
// default 0, which means no limit.
func WithMaxConnsPerHost(m int) Option {
	return func(o *Options) {
		o.MaxConnsPerHost = m
	}
}

// WithIdleConnTimeout returns an Option which sets the maximum amount of time an idle connection
// will remain idle before closing, default 0, which means no limit.
func WithIdleConnTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.IdleConnTimeout = t
	}
}
