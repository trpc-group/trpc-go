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

// Package allocator implements byte slice pooling management
// to reduce the pressure of memory allocation.
package allocator

import (
	"fmt"
	"sync"
)

const maxPowerToRoundUpInt = 63

var defaultAllocator = NewClassAllocator()

// Malloc gets a []byte from pool. The second return param is used to Free.
func Malloc(size int) ([]byte, interface{}) {
	return defaultAllocator.Malloc(size)
}

// Free releases the bytes to pool.
func Free(bts interface{}) {
	defaultAllocator.Free(bts)
}

// NewClassAllocator creates a new ClassAllocator.
func NewClassAllocator() *ClassAllocator {
	var pools [maxPowerToRoundUpInt]*sync.Pool
	for i := range pools {
		size := 1 << i
		pools[i] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		}
	}
	return &ClassAllocator{pools: pools}
}

// ClassAllocator is a bytes pool. The size of bytes satisfies 1 << n.
type ClassAllocator struct {
	pools [maxPowerToRoundUpInt]*sync.Pool
}

// Malloc gets a []byte from pool. The second return param is used to Free.
// We may also use first return param to Free bytes, but this causes an additional heap allocation.
// See https://github.com/golang/go/issues/8618 for more details.
func (a *ClassAllocator) Malloc(size int) ([]byte, interface{}) {
	if size <= 0 {
		panic(fmt.Sprintf("invalid alloc size %d", size))
	}
	v := a.pools[powerToRoundUp(size)].Get()
	return v.([]byte)[:size], v
}

// Free releases the bytes to pool.
func (a *ClassAllocator) Free(bts interface{}) {
	cap := cap(bts.([]byte))
	if cap == 0 {
		panic("free an empty bytes")
	}
	power := powerToRoundUp(cap)
	if 1<<power != cap {
		panic(fmt.Sprintf("cap %d of bts must be power of two", cap))
	}
	a.pools[power].Put(bts)
}

func powerToRoundUp(n int) int {
	powerOfTwo, power := 1, 0
	for ; n-powerOfTwo > 0; power++ {
		powerOfTwo <<= 1
	}
	return power
}
