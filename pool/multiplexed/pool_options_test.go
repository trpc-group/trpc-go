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

package multiplexed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPoolOptions test the configuration items of the multiplexed pool.
func TestPoolOptions(t *testing.T) {
	opts := &PoolOptions{}
	WithConnectNumber(50000)(opts)
	WithQueueSize(20000)(opts)
	WithRecvQueueSize(10000)(opts)
	WithDropFull(true)(opts)
	WithMaxReconnectCount(100)(opts)
	WithInitialBackoff(1 * time.Second)(opts)
	WithReconnectCountResetInterval(10000 * time.Second)(opts)

	assert.Equal(t, 50000, opts.connectNumberPerHost)
	assert.Equal(t, 20000, opts.sendQueueSize)
	assert.Equal(t, true, opts.dropFull)
	assert.Equal(t, 100, opts.maxReconnectCount)
	assert.Equal(t, 1*time.Second, opts.initialBackoff)
	assert.Equal(t, 10000*time.Second, opts.reconnectCountResetInterval)
}

func TestDisableReconnect(t *testing.T) {
	opts := &PoolOptions{}
	WithMaxReconnectCount(0)(opts)
	WithInitialBackoff(1 * time.Second)(opts)
	assert.Equal(t, 1*time.Second, opts.initialBackoff)
	opts.checkReconnectParams()
	assert.Equal(t, 0, opts.maxReconnectCount)
	assert.Equal(t, time.Duration(0), opts.initialBackoff)
}

func TestFixReconnectParams(t *testing.T) {
	opts := &PoolOptions{}
	WithMaxReconnectCount(10)(opts)
	WithInitialBackoff(1 * time.Second)(opts)
	assert.Nil(t, opts.checkReconnectParams())
	assert.Equal(t, 10*time.Second, opts.maxBackoff)
	assert.Equal(t, 130*time.Second, opts.reconnectCountResetInterval)

	opts = &PoolOptions{}
	WithMaxReconnectCount(-1)(opts)
	WithInitialBackoff(1 * time.Second)(opts)
	assert.NotNil(t, opts.checkReconnectParams())

	opts = &PoolOptions{}
	WithMaxReconnectCount(1)(opts)
	WithInitialBackoff(0 * time.Second)(opts)
	assert.NotNil(t, opts.checkReconnectParams())

	opts = &PoolOptions{}
	WithMaxReconnectCount(-1)(opts)
	WithInitialBackoff(0 * time.Second)(opts)
	assert.NotNil(t, opts.checkReconnectParams())
}
