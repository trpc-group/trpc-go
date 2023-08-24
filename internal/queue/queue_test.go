// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package queue

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestQueue test queue related functions
func TestQueue(t *testing.T) {
	done := make(chan struct{})
	q := New[[]byte](done)

	// Get it normally
	v := []byte("hello world")
	q.Put(v)
	e, ok := q.Get()
	assert.True(t, ok)
	assert.Equal(t, []byte("hello world"), e)

	// no data blocking
	t1 := time.Now()
	go func() {
		time.Sleep(500 * time.Millisecond)
		q.Put(v)
	}()
	e, ok = q.Get()

	assert.True(t, ok)
	assert.Equal(t, []byte("hello world"), e)
	t2 := int64(time.Now().Sub(t1))
	assert.GreaterOrEqual(t, t2, int64(500*time.Millisecond))

	// queue Done
	time.Sleep(200 * time.Millisecond)
	close(done)
	e, ok = q.Get()
	assert.False(t, ok)
	assert.Nil(t, e)
}

// TestConcurrentQueue test queue concurrency
func TestConcurrentQueue(t *testing.T) {
	done := make(chan struct{})
	q := New[[]byte](done)
	wg := &sync.WaitGroup{}
	wg.Add(3)

	// a goroutine write
	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			v := []byte("hello world")
			q.Put(v)
		}
	}()

	// write another goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			v := []byte("hello world")
			q.Put(v)
		}
	}()

	// a goroutine read
	go func() {
		defer wg.Done()
		for i := 0; i < 10000; i++ {
			e, ok := q.Get()
			assert.True(t, ok)
			assert.Equal(t, []byte("hello world"), e)
		}
	}()
	wg.Wait()
}
