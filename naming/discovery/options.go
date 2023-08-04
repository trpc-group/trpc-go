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
