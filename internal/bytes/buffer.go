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

// Package bytes extends std/bytes to provide versatile utilities for buffers.
package bytes

import (
	"bytes"
	"sync"
)

var nopCloserBufferPool sync.Pool

func init() {
	nopCloserBufferPool = sync.Pool{
		New: func() interface{} {
			return &NopCloserBuffer{}
		},
	}
}

// NopCloserBuffer implements io.Closer, but the implementation is nop.
type NopCloserBuffer struct {
	bytes.Buffer
}

// Close implements io.Closer, it does nothing.
func (*NopCloserBuffer) Close() error {
	return nil
}

// GetNopCloserBuffer gets a NopCloserBuffer from pool.
func GetNopCloserBuffer() *NopCloserBuffer {
	return nopCloserBufferPool.Get().(*NopCloserBuffer)
}

// PutNopCloserBuffer puts a NopCloserBuffer to pool.
func PutNopCloserBuffer(b *NopCloserBuffer) {
	b.Reset()
	nopCloserBufferPool.Put(b)
}
