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

package config

// WithCodec returns an option which sets the codec's name.
func WithCodec(name string) LoadOption {
	return func(c *TrpcConfig) {
		c.decoder = GetCodec(name)
	}
}

// WithProvider returns an option which sets the provider's name.
func WithProvider(name string) LoadOption {
	return func(c *TrpcConfig) {
		c.p = GetProvider(name)
	}
}

// WithExpandEnv replaces ${var} in raw bytes with environment value of var.
// Note, method TrpcConfig.Bytes will return the replaced bytes.
func WithExpandEnv() LoadOption {
	return func(c *TrpcConfig) {
		c.expandEnv = true
	}
}

// WithWatch returns an option to start watch model
func WithWatch() LoadOption {
	return func(c *TrpcConfig) {
		c.watch = true
	}
}

// WithWatchHook returns an option to set log func for config change logger
func WithWatchHook(f func(msg WatchMessage)) LoadOption {
	return func(c *TrpcConfig) {
		c.watchHook = f
	}
}

// options is config option.
type options struct{}

// Option is the option for config provider sdk.
type Option func(*options)
