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

// Package packetbuffer implements functions for the manipulation
// of byte slices.
package packetbuffer

import (
	"io"
)

// PacketBuffer a variable-sized buffer of bytes.
type PacketBuffer struct {
	buf         []byte // contents are the bytes buf[read : write]
	read, write int    // read at &buf[read], write at &buf[write]
}

// New return a PacketBuffer.
func New(buf []byte) *PacketBuffer {
	return &PacketBuffer{buf: buf}
}

// Read try gets len(b) data from buffer, the current position is
// advanced by `n` bytes returned. It returns the number of bytes
// read (0 <= n <= len(b)), if Read returns n < len(b) or buf is
// empty return io.EOF.
func (r *PacketBuffer) Read(b []byte) (int, error) {
	if r.write == r.read {
		return 0, io.EOF
	}
	var err error
	if r.write-r.read < len(b) {
		err = io.EOF
	}
	n := copy(b, r.buf[r.read:r.write])
	r.read += n
	return n, err
}

// UnRead returns the number of bytes between the read
// position and the write position of the buffer.
func (r *PacketBuffer) UnRead() int {
	return r.write - r.read
}

// Reset reset the read/write position of the buffer.
func (r *PacketBuffer) Reset() {
	r.read = 0
	r.write = 0
}

// Advance advance the write position of the buffer.
func (r *PacketBuffer) Advance(n int) {
	r.write += n
}

// Bytes returns a slice starting at the write position and
// the end of the buffer.
func (r *PacketBuffer) Bytes() []byte {
	return r.buf[r.write:]
}
