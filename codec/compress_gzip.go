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
	"compress/gzip"
	"io"
	"sync"
)

func init() {
	RegisterCompressor(CompressTypeGzip, &GzipCompress{})
}

// GzipCompress is gzip compressor.
type GzipCompress struct {
	readerPool sync.Pool
	writerPool sync.Pool
}

// Compress returns binary data compressed by gzip.
func (c *GzipCompress) Compress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}

	buffer := &bytes.Buffer{}
	z, ok := c.writerPool.Get().(*gzip.Writer)
	if !ok {
		z = gzip.NewWriter(buffer)
	} else {
		z.Reset(buffer)
	}
	defer c.writerPool.Put(z)

	if _, err := z.Write(in); err != nil {
		return nil, err
	}
	if err := z.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// Decompress returns binary data decompressed by gzip.
func (c *GzipCompress) Decompress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}
	br := bytes.NewReader(in)
	z, ok := c.readerPool.Get().(*gzip.Reader)
	defer func() {
		if z != nil {
			c.readerPool.Put(z)
		}
	}()
	if !ok {
		gr, err := gzip.NewReader(br)
		if err != nil {
			return nil, err
		}
		z = gr
	} else {
		if err := z.Reset(br); err != nil {
			return nil, err
		}
	}
	out, err := io.ReadAll(z)
	if err != nil {
		return nil, err
	}
	return out, nil
}
