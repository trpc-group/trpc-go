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

// Package objectpool provides object pool.
package objectpool

import (
	"sync"
)

// BytesPool represents the bytes array object pool.
type BytesPool struct {
	pool sync.Pool
}

// NewBytesPool creates a new bytes array object pool.
func NewBytesPool(size int) *BytesPool {
	return &BytesPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get takes the bytes array from the object pool.
func (p *BytesPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put bytes array back into object pool.
func (p *BytesPool) Put(b []byte) {
	p.pool.Put(b)
}
