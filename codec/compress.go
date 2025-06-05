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
	"errors"

	"github.com/spf13/cast"
)

// Compressor is body compress and decompress interface.
type Compressor interface {
	Compress(in []byte) (out []byte, err error)
	Decompress(in []byte) (out []byte, err error)
}

// CompressType is the mode of body compress or decompress.
const (
	CompressTypeNoop = iota
	CompressTypeGzip
	CompressTypeSnappy
	CompressTypeZlib
	CompressTypeStreamSnappy
	CompressTypeBlockSnappy
	CompressTypeStreamLZ4
	CompressTypeBlockLZ4
	maxIndexForCompressionFastAccess = 64
)

var (
	primaryCompressors  [maxIndexForCompressionFastAccess + 1]Compressor
	fallbackCompressors = make(map[int]Compressor)
)

// RegisterCompressor register a specific compressor, which will
// be called by init function defined in third package.
func RegisterCompressor(compressType int, s Compressor) {
	if compressType >= 0 && compressType <= maxIndexForCompressionFastAccess {
		primaryCompressors[compressType] = s
		return
	}
	fallbackCompressors[compressType] = s
}

// MustRegisterCompressor register a specific compressor, which will
// panic if the compressor has been registered.
//
// In most cases, the framework uses the init + RegisterCompressor method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegisterCompressor to forcibly register a component 'xxx', while the framework
// uses init + RegisterCompressor to register another component 'yyy', conflicts may occur. If the init function
// for MustRegisterCompressor is executed before the conflicting init function, MustRegisterCompressor might not raise
// an error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegisterCompressor and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegisterCompressor(compressType int, s Compressor) {
	if GetCompressor(compressType) != nil {
		panic("compressor already registered for type: " + cast.ToString(compressType))
	}
	RegisterCompressor(compressType, s)
}

// GetCompressor returns a specific compressor by type.
func GetCompressor(compressType int) Compressor {
	if compressType >= 0 && compressType <= maxIndexForCompressionFastAccess {
		return primaryCompressors[compressType]
	}
	return fallbackCompressors[compressType]
}

// Compress returns the compressed data, the data is compressed
// by a specific compressor.
func Compress(compressorType int, in []byte) ([]byte, error) {
	// Explicitly check for noop to avoid accessing the map.
	if compressorType == CompressTypeNoop {
		return in, nil
	}
	if len(in) == 0 {
		return nil, nil
	}
	compressor := GetCompressor(compressorType)
	if compressor == nil {
		return nil, errors.New("compressor not registered")
	}
	return compressor.Compress(in)
}

// Decompress returns the decompressed data, the data is decompressed
// by a specific compressor.
func Decompress(compressorType int, in []byte) ([]byte, error) {
	// Explicitly check for noop to avoid accessing the map.
	if compressorType == CompressTypeNoop {
		return in, nil
	}
	if len(in) == 0 {
		return nil, nil
	}
	compressor := GetCompressor(compressorType)
	if compressor == nil {
		return nil, errors.New("compressor not registered")
	}
	return compressor.Decompress(in)
}
