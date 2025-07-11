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

package codec

import (
	"bytes"
	"compress/zlib"
	"io"
)

func init() {
	RegisterCompressor(CompressTypeZlib, &ZlibCompress{})
}

// ZlibCompress is zlib compressor.
type ZlibCompress struct {
}

// Compress returns binary data compressed by zlib.
func (c *ZlibCompress) Compress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}
	var (
		buffer bytes.Buffer
		out    []byte
	)
	writer := zlib.NewWriter(&buffer)
	if _, err := writer.Write(in); err != nil {
		writer.Close()
		return out, err
	}
	if err := writer.Close(); err != nil {
		return out, err
	}
	return buffer.Bytes(), nil
}

// Decompress returns binary data decompressed by zlib.
func (c *ZlibCompress) Decompress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}
	reader, err := zlib.NewReader(bytes.NewReader(in))
	if err != nil {
		var out []byte
		return out, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
