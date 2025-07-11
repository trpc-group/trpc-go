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

package servicerouter

import (
	"context"
)

// Options defines the call options.
type Options struct {
	Ctx                  context.Context
	SourceSetName        string
	DestinationSetName   string
	DisableServiceRouter bool
	Namespace            string
	SourceNamespace      string
	SourceServiceName    string
	SourceEnvName        string
	DestinationEnvName   string
	EnvTransfer          string
	EnvKey               string
	SourceMetadata       map[string]string
	DestinationMetadata  map[string]string
}

// Option modifies the Options.
type Option func(*Options)

// WithContext returns an Option which sets request context.
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

// WithDisableServiceRouter returns an Option which disables service router.
func WithDisableServiceRouter() Option {
	return func(o *Options) {
		o.DisableServiceRouter = true
	}
}

// WithSourceNamespace returns an Option which sets caller namespace.
func WithSourceNamespace(namespace string) Option {
	return func(o *Options) {
		o.SourceNamespace = namespace
	}
}

// WithSourceServiceName returns an Option which sets caller service name.
func WithSourceServiceName(serviceName string) Option {
	return func(o *Options) {
		o.SourceServiceName = serviceName
	}
}

// WithSourceEnvName returns an Option which sets caller environment name.
func WithSourceEnvName(envName string) Option {
	return func(o *Options) {
		o.SourceEnvName = envName
	}
}

// WithDestinationEnvName returns an Option which sets callee environment name.
func WithDestinationEnvName(envName string) Option {
	return func(o *Options) {
		o.DestinationEnvName = envName
	}
}

// WithEnvTransfer returns an Option which sets transparent environment information.
func WithEnvTransfer(envTransfer string) Option {
	return func(o *Options) {
		o.EnvTransfer = envTransfer
	}
}

// WithEnvKey returns an Option which sets environment key.
func WithEnvKey(key string) Option {
	return func(o *Options) {
		o.EnvKey = key
	}
}

// WithSourceSetName returns an Option which sets caller set name.
func WithSourceSetName(sourceSetName string) Option {
	return func(o *Options) {
		o.SourceSetName = sourceSetName
	}
}

// WithDestinationSetName returns an Option which sets callee set name.
func WithDestinationSetName(destinationSetName string) Option {
	return func(o *Options) {
		o.DestinationSetName = destinationSetName
	}
}

// WithSourceMetadata returns an Option which sets caller metadata.
func WithSourceMetadata(key string, val string) Option {
	return func(o *Options) {
		if o.SourceMetadata == nil {
			o.SourceMetadata = make(map[string]string)
		}
		o.SourceMetadata[key] = val
	}
}

// WithDestinationMetadata returns an Option which sets callee metadata.
func WithDestinationMetadata(key string, val string) Option {
	return func(o *Options) {
		if o.DestinationMetadata == nil {
			o.DestinationMetadata = make(map[string]string)
		}
		o.DestinationMetadata[key] = val
	}
}
