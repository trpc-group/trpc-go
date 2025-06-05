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

package objectpool

import (
	"bytes"
	"sync"
)

// BufferPool represents the buffer object pool.
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new bytes.Buffer object pool.
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Get takes the buffer from the pool.
func (p *BufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

// Put buffer back into the pool.
func (p *BufferPool) Put(buf *bytes.Buffer) {
	p.pool.Put(buf)
}
