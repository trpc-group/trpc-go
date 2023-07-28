// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package multiplexed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPoolOptions test the configuration items of the multiplexed pool.
func TestPoolOptions(t *testing.T) {
	opts := &PoolOption{}
	WithDialTimeout(time.Second)(opts)
	WithMaxConcurrentVirtualConnsPerConn(20)(opts)
	WithEnableMetrics()(opts)
	assert.Equal(t, opts.dialTimeout, time.Second)
	assert.Equal(t, opts.maxConcurrentVirtualConnsPerConn, 20)
	assert.Equal(t, opts.enableMetrics, true)
}
