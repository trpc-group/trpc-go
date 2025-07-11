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

package admin

import (
	"time"
)

// Option Service configuration options.
type Option func(*configuration)

// WithAddr returns an Option which sets the address bound to admin, default: ":9028".
// Supported formats:
// 1. :80
// 2. 0.0.0.0:80
// 3. localhost:80
// 4. 127.0.0.0:8001
func WithAddr(addr string) Option {
	return func(config *configuration) {
		config.addr = addr
	}
}

// WithTLS returns an Option which sets whether to use HTTPS.
func WithTLS(isTLS bool) Option {
	return func(config *configuration) {
		config.enableTLS = isTLS
	}
}

// WithVersion returns an Option which sets the version number.
func WithVersion(version string) Option {
	return func(config *configuration) {
		config.version = version
	}
}

// WithReadTimeout returns an Option which sets read timeout.
func WithReadTimeout(readTimeout time.Duration) Option {
	return func(config *configuration) {
		if readTimeout > 0 {
			config.readTimeout = readTimeout
		}
	}
}

// WithWriteTimeout returns an Option which sets write timeout.
func WithWriteTimeout(writeTimeout time.Duration) Option {
	return func(config *configuration) {
		if writeTimeout > 0 {
			config.writeTimeout = writeTimeout
		}
	}
}

// WithConfigPath returns an Option which sets the framework configuration file path.
func WithConfigPath(configPath string) Option {
	return func(config *configuration) {
		config.configPath = configPath
	}
}

// WithSkipServe sets whether to skip starting the admin service.
func WithSkipServe(isSkip bool) Option {
	return func(config *configuration) {
		config.skipServe = isSkip
	}
}
