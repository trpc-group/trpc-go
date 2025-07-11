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

func init() {
	RegisterCompressor(CompressTypeNoop, &NoopCompress{})
}

// NoopCompress is an empty compressor
type NoopCompress struct {
}

// Compress returns the origin data.
func (c *NoopCompress) Compress(in []byte) ([]byte, error) {
	return in, nil
}

// Decompress returns the origin data.
func (c *NoopCompress) Decompress(in []byte) ([]byte, error) {
	return in, nil
}
