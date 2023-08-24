// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package discovery

import (
	"context"
)

// Options is the call options.
type Options struct {
	Ctx       context.Context
	Namespace string
}

// Option modifies the Options.
type Option func(*Options)

// WithContext returns an Option which sets ctx.
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Ctx = ctx
	}
}

// WithNamespace returns an Option which sets namespace.
func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Namespace = namespace
	}
}
