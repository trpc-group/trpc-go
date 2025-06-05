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

package codec

import (
	"bufio"
	"io"
)

// DefaultReaderSize is the default size of reader in bytes.
const DefaultReaderSize = 4 * 1024

// readerSizeConfig is the default size of buffer when framer read package.
var readerSizeConfig = DefaultReaderSize

// NewReaderSize returns a reader with read buffer. Size <= 0 means no buffer.
func NewReaderSize(r io.Reader, size int) io.Reader {
	if size <= 0 {
		return r
	}
	return bufio.NewReaderSize(r, size)
}

// NewReader returns reader with the default buffer size.
func NewReader(r io.Reader) io.Reader {
	return bufio.NewReaderSize(r, readerSizeConfig)
}

// GetReaderSize returns size of read buffer in bytes.
func GetReaderSize() int {
	return readerSizeConfig
}

// SetReaderSize sets the size of read buffer in bytes.
func SetReaderSize(size int) {
	readerSizeConfig = size
}

// FramerBuilder defines how to build a framer. In general, each connection
// build a framer.
type FramerBuilder interface {
	New(io.Reader) Framer
}

// Framer defines how to read a data frame.
type Framer interface {
	ReadFrame() ([]byte, error)
}

// SafeFramer is a special framer, provides an isSafe() method
// to describe if it is safe when concurrent read.
type SafeFramer interface {
	Framer
	// IsSafe returns if this framer is safe when concurrent read.
	IsSafe() bool
}

// IsSafeFramer returns if this framer is safe when concurrent read. The input
// parameter f should implement SafeFramer interface. If not , this method will return false.
func IsSafeFramer(f interface{}) bool {
	framer, ok := f.(SafeFramer)
	if ok && framer.IsSafe() {
		return true
	}
	return false
}

// Decoder defines the decode logic of transport response frame data.
type Decoder interface {
	// Decode parse frame head, package head and package body from response.
	Decode() (TransportResponseFrame, error)

	// UpdateMsg update Msg content, the first input param is parsed response data.
	UpdateMsg(interface{}, Msg) error
}

// TransportResponseFrame is the interface should be implemented
// by the response package data.
type TransportResponseFrame interface {
	// GetRequestID returns the stream id when in stream mode,
	// returns request id when one-request-one-response mode.
	GetRequestID() uint32

	// GetResponseBuf returns the whole frame when in stream mode,
	// returns the package body when in one-request-one-response mode.
	GetResponseBuf() []byte
}
