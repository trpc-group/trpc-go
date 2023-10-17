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
	"bytes"
	"io"
	"sync"

	"github.com/golang/snappy"
)

func init() {
	RegisterCompressor(CompressTypeSnappy, NewSnappyCompressor())
	RegisterCompressor(CompressTypeStreamSnappy, NewSnappyCompressor())
	RegisterCompressor(CompressTypeBlockSnappy, NewSnappyBlockCompressor())
}

// SnappyCompress is snappy compressor using stream snappy format.
//
// There are actually two Snappy formats: block and stream. They are related,
// but different: trying to decompress block-compressed data as a Snappy stream
// will fail, and vice versa.
type SnappyCompress struct {
	writerPool *sync.Pool
	readerPool *sync.Pool
}

// NewSnappyCompressor returns a stream format snappy compressor instance.
func NewSnappyCompressor() *SnappyCompress {
	s := &SnappyCompress{}
	s.writerPool = &sync.Pool{
		New: func() interface{} {
			return snappy.NewBufferedWriter(&bytes.Buffer{})
		},
	}
	s.readerPool = &sync.Pool{
		New: func() interface{} {
			return snappy.NewReader(&bytes.Buffer{})
		},
	}
	return s
}

// Compress returns binary data compressed by snappy stream format.
func (c *SnappyCompress) Compress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}

	buf := &bytes.Buffer{}
	writer := c.getSnappyWriter(buf)
	defer func() {
		if c.writerPool != nil {
			c.writerPool.Put(writer)
		}
	}()

	if _, err := writer.Write(in); err != nil {
		writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decompress returns binary data decompressed by snappy stream format.
func (c *SnappyCompress) Decompress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}

	inReader := bytes.NewReader(in)
	reader := c.getSnappyReader(inReader)
	defer func() {
		if c.readerPool != nil {
			c.readerPool.Put(reader)
		}
	}()

	out, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return out, err
}

// SnappyBlockCompressor is snappy compressor using snappy block format.
type SnappyBlockCompressor struct{}

// NewSnappyBlockCompressor returns a block format snappy compressor instance.
func NewSnappyBlockCompressor() *SnappyBlockCompressor {
	return &SnappyBlockCompressor{}
}

// Compress returns binary data compressed by snappy block formats.
func (c *SnappyBlockCompressor) Compress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}
	return snappy.Encode(nil, in), nil
}

// Decompress returns binary data decompressed by snappy block formats.
func (c *SnappyBlockCompressor) Decompress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}
	return snappy.Decode(nil, in)
}

func (c *SnappyCompress) getSnappyWriter(buf *bytes.Buffer) *snappy.Writer {
	if c.writerPool == nil {
		return snappy.NewBufferedWriter(buf)
	}

	// get from pool
	writer, ok := c.writerPool.Get().(*snappy.Writer)
	if !ok || writer == nil {
		return snappy.NewBufferedWriter(buf)
	}
	writer.Reset(buf)
	return writer
}

func (c *SnappyCompress) getSnappyReader(inReader *bytes.Reader) *snappy.Reader {
	if c.readerPool == nil {
		return snappy.NewReader(inReader)
	}

	// get from pool
	reader, ok := c.readerPool.Get().(*snappy.Reader)
	if !ok || reader == nil {
		return snappy.NewReader(inReader)
	}
	reader.Reset(inReader)
	return reader
}
