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

// Package bufio provide a buffered reader which can stop buffering in the future.
package bufio

import "io"

// NewReader create a new Reader.
func NewReader(rd io.Reader, size int) *Reader {
	return &Reader{
		rd:  rd,
		buf: make([]byte, size),
	}
}

// Reader is an buffered Reader.
type Reader struct {
	rd         io.Reader
	buf        []byte
	r, w       int
	err        error
	unbuffered bool
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		if r.w > r.r {
			return 0, nil
		}
		return 0, r.readErr()
	}
	if r.r == r.w {
		if r.err != nil {
			return 0, r.readErr()
		}
		if len(p) >= len(r.buf) || r.unbuffered {
			return r.rd.Read(p)
		}
		r.r, r.w = 0, 0
		var n int
		n, r.err = r.rd.Read(r.buf)
		if n == 0 {
			return 0, r.readErr()
		}
		r.w += n
	}

	n := copy(p, r.buf[r.r:r.w])
	r.r += n
	return n, nil
}

// Unbuffer stops the buffering of Reader.
func (r *Reader) Unbuffer() { r.unbuffered = true }

// Buffered returns how many bytes is currently buffered.
func (r *Reader) Buffered() int { return r.w - r.r }

func (r *Reader) readErr() error {
	err := r.err
	r.err = nil
	return err
}
