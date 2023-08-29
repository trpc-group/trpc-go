// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

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
