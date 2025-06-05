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

	"github.com/pierrec/lz4/v4"
)

func init() {
	RegisterCompressor(CompressTypeStreamLZ4, NewLZ4StreamCompressor())
	RegisterCompressor(CompressTypeBlockLZ4, NewLZ4BlockCompressor())
}

// LZ4StreamCompressor is lz4 compressor using stream lz4 format.
//
// There are actually two LZ4 formats: block and stream. They are related,
// but different: trying to decompress block-compressed data as a LZ4 stream
// will fail, and vice versa.
type LZ4StreamCompressor struct {
	writerPool *sync.Pool
	readerPool *sync.Pool
}

// NewLZ4StreamCompressor returns a stream format lz4 compressor instance.
func NewLZ4StreamCompressor() *LZ4StreamCompressor {
	s := &LZ4StreamCompressor{}
	s.writerPool = &sync.Pool{
		New: func() interface{} {
			return lz4.NewWriter(&bytes.Buffer{})
		},
	}
	s.readerPool = &sync.Pool{
		New: func() interface{} {
			return lz4.NewReader(&bytes.Buffer{})
		},
	}
	return s
}

// Compress returns binary data compressed by lz4 stream format.
func (c *LZ4StreamCompressor) Compress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}

	buf := &bytes.Buffer{}
	writer := c.getLZ4Writer(buf)
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

// Decompress returns binary data decompressed by lz4 stream format.
func (c *LZ4StreamCompressor) Decompress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}

	inReader := bytes.NewReader(in)
	reader := c.getLZ4Reader(inReader)
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

// LZ4BlockCompressor is lz4 compressor using lz4 block format.
type LZ4BlockCompressor struct {
	compressorPool *sync.Pool
}

// NewLZ4BlockCompressor returns a block format lz4 compressor instance.
func NewLZ4BlockCompressor() *LZ4BlockCompressor {
	return &LZ4BlockCompressor{
		compressorPool: &sync.Pool{
			New: func() interface{} {
				return &lz4.Compressor{}
			},
		},
	}
}

// Compress returns binary data compressed by lz4 block formats.
func (c *LZ4BlockCompressor) Compress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}
	cc := c.getLZ4Compressor()
	defer func() {
		if c.compressorPool != nil {
			c.compressorPool.Put(cc)
		}
	}()
	out := make([]byte, lz4.CompressBlockBound(len(in)))
	n, err := cc.CompressBlock(in, out)
	if err != nil {
		return nil, err
	}
	return out[:n], nil
}

// Decompress returns binary data decompressed by lz4 block formats.
func (c *LZ4BlockCompressor) Decompress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}
	// I have no idea how to get the size of a dst buffer in a decent way. 🤷‍♂️
	// https://github.com/pierrec/lz4/blob/v4.1.18/example_test.go#L51
	const somePossibleExpansionFactor = 10
	out := make([]byte, somePossibleExpansionFactor*len(in))
	n, err := lz4.UncompressBlock(in, out)
	if err != nil {
		return nil, err
	}
	return out[:n], nil
}

func (c *LZ4BlockCompressor) getLZ4Compressor() *lz4.Compressor {
	if c.compressorPool == nil {
		return &lz4.Compressor{}
	}

	compressor, ok := c.compressorPool.Get().(*lz4.Compressor)
	if !ok || compressor == nil {
		return &lz4.Compressor{}
	}
	return compressor
}

func (c *LZ4StreamCompressor) getLZ4Writer(buf *bytes.Buffer) *lz4.Writer {
	if c.writerPool == nil {
		return lz4.NewWriter(buf)
	}

	writer, ok := c.writerPool.Get().(*lz4.Writer)
	if !ok || writer == nil {
		return lz4.NewWriter(buf)
	}
	writer.Reset(buf)
	return writer
}

func (c *LZ4StreamCompressor) getLZ4Reader(inReader *bytes.Reader) *lz4.Reader {
	if c.readerPool == nil {
		return lz4.NewReader(inReader)
	}

	reader, ok := c.readerPool.Get().(*lz4.Reader)
	if !ok || reader == nil {
		return lz4.NewReader(inReader)
	}
	reader.Reset(inReader)
	return reader
}
