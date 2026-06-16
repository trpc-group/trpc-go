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

package transport

import "io"

type readCountingReader struct {
	r         io.Reader
	readBytes int
}

func newReadCountingReader(r io.Reader) *readCountingReader {
	return &readCountingReader{r: r}
}

func (r *readCountingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.readBytes += n
	return n, err
}

func (r *readCountingReader) ReadBytes() int {
	return r.readBytes
}

func (r *readCountingReader) ResetReadBytes() {
	r.readBytes = 0
}
