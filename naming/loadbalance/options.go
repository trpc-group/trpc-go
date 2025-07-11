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

package loadbalance

import (
	"context"
	"time"
)

// Options is the call options.
type Options struct {
	Ctx             context.Context // request context
	Interval        time.Duration   // refresh interval
	Namespace       string          // namespace
	Key             string          // hash key
	LoadBalanceType string          // load balance type
	Replicas        int             // virtual node coefficient of consistent hash
}

// Option modifies the Options.
type Option func(*Options)

// WithContext returns an Option which set request ctx.
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Ctx = ctx
	}
}

// WithNamespace returns an Option which set namespace.
func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Namespace = namespace
	}
}

// WithInterval returns an Option which set load balance refresh interval.
func WithInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.Interval = interval
	}
}

// WithKey returns an Option which set the hash key of status route.
func WithKey(k string) Option {
	return func(o *Options) {
		o.Key = k
	}
}

// WithReplicas returns an Option which set the virtual node coefficient.
func WithReplicas(r int) Option {
	return func(o *Options) {
		o.Replicas = r
	}
}

// WithLoadBalanceType returns an Option which set load balance type.
func WithLoadBalanceType(name string) Option {
	return func(opts *Options) {
		opts.LoadBalanceType = name
	}
}
