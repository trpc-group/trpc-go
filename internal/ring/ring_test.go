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

package ring_test

import (
	"math"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/internal/ring"
)

var (
	defaultMsg = []byte("Hello World!!")
	defaultCap = uint32(3)
)

func TestNew(t *testing.T) {
	r := ring.New[[]byte](0)
	assert.Equal(t, uint32(1), r.Cap())

	r = ring.New[[]byte](1)
	assert.Equal(t, uint32(1), r.Cap())

	r = ring.New[[]byte](math.MaxUint32)
	assert.Equal(t, uint32(1), r.Cap())

	r = ring.New[[]byte](2)
	assert.Equal(t, uint32(1), r.Cap())

	r = ring.New[[]byte](3)
	assert.Equal(t, uint32(3), r.Cap())
	assert.Equal(t, "Ring: Cap=3, Head=0, Tail=0, Size=0\n", r.String())

	r = ring.New[[]byte](4)
	assert.Equal(t, uint32(3), r.Cap())

	r = ring.New[[]byte](7)
	assert.Equal(t, uint32(7), r.Cap())
}

func TestPutGet(t *testing.T) {
	r := ring.New[[]byte](defaultCap)
	assert.NotNil(t, r)
	assert.Equal(t, defaultCap, r.Cap())

	// normal Put.
	err := r.Put(defaultMsg)
	assert.Nil(t, err)

	// normal Get.
	val, left := r.Get()
	assert.NotEmpty(t, val)
	assert.Equal(t, defaultMsg, val)
	assert.Equal(t, uint32(0), left)

	// Get an empty ring.
	assert.Equal(t, true, r.IsEmpty())
	val, left = r.Get()
	assert.Empty(t, val)
	assert.Equal(t, uint32(0), left)

	// Check a full queue.
	for i := uint32(0); i < r.Cap(); i++ {
		err := r.Put(defaultMsg)
		assert.Nil(t, err)
	}
	assert.Equal(t, true, r.IsFull())
	assert.Equal(t, r.Cap(), r.Size())

	// insert into a full queue.
	err = r.Put(defaultMsg)
	assert.Equal(t, ring.ErrQueueFull, err)
}

func TestPutGetOrder(t *testing.T) {
	r := ring.New[uint32](defaultCap)
	for i := uint32(0); i < r.Cap(); i++ {
		err := r.Put(i)
		assert.Nil(t, err)
	}

	for i := uint32(0); i < r.Cap(); i++ {
		val, _ := r.Get()
		assert.Equal(t, i, val)
	}
}

func TestGetsWithFull(t *testing.T) {
	r := ring.New[[]byte](defaultCap)
	assert.NotNil(t, r)

	for i := uint32(0); i < r.Cap(); i++ {
		err := r.Put(defaultMsg)
		assert.Nil(t, err)
	}
	values := make([][]byte, 0, r.Cap())
	count, left := r.Gets(&values)
	assert.Equal(t, r.Cap(), count)
	assert.Equal(t, uint32(0), left)
	assert.Equal(t, uint32(len(values)), defaultCap)

	for _, x := range values {
		assert.Equal(t, x, defaultMsg)
	}
	// the Get queue is empty, execute Gets.
	assert.Equal(t, true, r.IsEmpty())
	count, _ = r.Gets(&values)
	assert.Equal(t, uint32(0), count)
}

func TestGetsWithAskedSize(t *testing.T) {
	r := ring.New[[]byte](defaultCap)
	assert.NotNil(t, r)

	for i := uint32(0); i < r.Cap(); i++ {
		err := r.Put(defaultMsg)
		assert.Nil(t, err)
	}
	values := make([][]byte, 0, r.Cap()-1)
	count, left := r.Gets(&values)
	assert.Equal(t, r.Cap()-1, count)
	assert.Equal(t, uint32(1), left)
}

func TestConcurrentGetPut(t *testing.T) {
	r := ring.New[[]byte](1024)
	cpus := runtime.NumCPU()

	// starts send goroutine, every goroutine sends N packages.
	wg := &sync.WaitGroup{}
	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go startPutMsgs(r, wg, 10000)
	}
	// starts receive goroutine, every goroutine receives N packages.
	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go startGetMsgs(r, wg, 10000)
	}
	wg.Wait()
	assert.Equal(t, true, r.IsEmpty())
}

func BenchmarkTestChannel(b *testing.B) {
	ch := make(chan interface{}, 1024)
	cpus := runtime.NumCPU()
	wg := &sync.WaitGroup{}
	b.SetBytes(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				ch <- defaultMsg
			}
			wg.Done()
		}()
	}
	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				<-ch
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkTestRingBuffer(b *testing.B) {
	r := ring.New[[]byte](1024)
	cpus := runtime.NumCPU()

	wg := &sync.WaitGroup{}
	b.SetBytes(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go startGetMsgs(r, wg, b.N)
	}
	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go startPutMsgs(r, wg, b.N)
	}
	wg.Wait()
}

func startPutMsgs(r *ring.Ring[[]byte], wg *sync.WaitGroup, num int) {
	for {
		if num <= 0 {
			break
		}
		err := r.Put(defaultMsg)
		if err == nil {
			num = num - 1
		}
	}
	wg.Done()
}

func startGetMsgs(r *ring.Ring[[]byte], wg *sync.WaitGroup, num int) {
	for {
		if num <= 0 {
			break
		}
		val, _ := r.Get()
		if val != nil {
			num = num - 1
		}
	}
	wg.Done()
}
