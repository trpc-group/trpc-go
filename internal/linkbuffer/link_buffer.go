// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package linkbuffer

import "io"

// Buffer is the interface of link buffer.
type Buffer interface {
	Reader
	Writer
	Release()
}

// Reader is the interface to read from link buffer.
type Reader interface {
	io.Reader
	ReadN(size int) ([]byte, int)
	ReadAll() [][]byte
	ReadNext() []byte
}

// Writer is the interface to write to link buffer.
type Writer interface {
	io.Writer
	Append(...[]byte)
	Prepend(...[]byte)
	Alloc(size int) []byte
	Prelloc(size int) []byte
	Len() int
	Merge(Reader)
}
