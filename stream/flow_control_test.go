// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package stream

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSendControl test flow control sender related implementation.
func TestSendControl(t *testing.T) {
	done := make(chan struct{})
	sc := newSendControl(defaultInitWindowSize, done)
	err := sc.GetWindow(100)
	assert.Nil(t, err)

	// Available window drops less than 0.
	err = sc.GetWindow(uint32(defaultInitWindowSize - 99))
	assert.Nil(t, err)

	// block
	t1 := time.Now()
	go func() {
		time.Sleep(500 * time.Millisecond)
		sc.UpdateWindow(201)
	}()
	err = sc.GetWindow(200)
	assert.Nil(t, err)
	t2 := int64(time.Now().Sub(t1))
	assert.GreaterOrEqual(t, t2, int64(500*time.Millisecond))
}

// TestReceiveControl test.
func TestReceiveControl(t *testing.T) {
	fb := func(uint32) error {
		return nil
	}
	rc := newReceiveControl(defaultInitWindowSize, fb)
	err := rc.OnRecv(100)
	assert.Nil(t, err)

	n := atomic.LoadUint32(&rc.left)
	assert.Equal(t, defaultInitWindowSize-uint32(100), n)

	// need to send updates.
	err = rc.OnRecv(defaultInitWindowSize / 4)
	assert.Nil(t, err)

	// test for feedback errors.
	fb = func(uint32) error {
		return errors.New("feedback error")
	}
	err = rc.OnRecv(100)
	assert.Nil(t, err)

	rc = newReceiveControl(defaultInitWindowSize, fb)
	err = rc.OnRecv(defaultInitWindowSize / 4)
	assert.NotNil(t, err)
}
