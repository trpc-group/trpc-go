// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package multiplexed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPoolOptions test the configuration items of the multiplexed pool.
func TestPoolOptions(t *testing.T) {
	opts := &PoolOptions{}
	WithConnectNumber(50000)(opts)
	WithQueueSize(20000)(opts)
	WithDropFull(true)(opts)
	assert.Equal(t, opts.connectNumberPerHost, 50000)
	assert.Equal(t, opts.sendQueueSize, 20000)
	assert.Equal(t, opts.dropFull, true)
}
