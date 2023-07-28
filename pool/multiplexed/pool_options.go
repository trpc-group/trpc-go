// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package multiplexed

import "time"

// PoolOption represents some settings for the multiplexed pool.
type PoolOption struct {
	dialTimeout                      time.Duration
	maxConcurrentVirtualConnsPerConn int
	enableMetrics                    bool
}

// OptPool is function to modify PoolOption.
type OptPool func(*PoolOption)

// WithDialTimeout returns an OptPool which sets dial timeout.
func WithDialTimeout(timeout time.Duration) OptPool {
	return func(o *PoolOption) {
		o.dialTimeout = timeout
	}
}

// WithMaxConcurrentVirtualConnsPerConn returns an OptPool which sets the number
// of concurrent virtual connections per connection.
func WithMaxConcurrentVirtualConnsPerConn(max int) OptPool {
	return func(o *PoolOption) {
		o.maxConcurrentVirtualConnsPerConn = max
	}
}

// WithEnableMetrics returns an OptPool which enable metrics.
func WithEnableMetrics() OptPool {
	return func(o *PoolOption) {
		o.enableMetrics = true
	}
}
