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
	"fmt"
	"testing"
)

// goos: linux
// goarch: amd64
// pkg: trpc.group/trpc-go/trpc-go/codec
// cpu: AMD EPYC 7K62 48-Core Processor
// compress_old-16   67606471  17.87 ns/op  0 B/op  0 allocs/op
// compress_new-16  160804280  7.479 ns/op  0 B/op  0 allocs/op
func BenchmarkCompressionSliceAndMap(b *testing.B) {
	const customCompressType = 6
	oldRegisterCompressor(customCompressType, &NoopCompress{})
	backup := GetCompressor(customCompressType)
	RegisterCompressor(customCompressType, &NoopCompress{})
	b.Cleanup(func() {
		RegisterCompressor(customCompressType, backup)
	})
	bs1 := []byte("hello")
	var bs2 []byte
	b.Run("compress old", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bs2, _ = oldCompress(customCompressType, bs1)
			bs1, _ = oldDecompress(customCompressType, bs2)
		}
	})
	b.Run("compress new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bs2, _ = Compress(customCompressType, bs1)
			bs1, _ = Decompress(customCompressType, bs2)
		}
	})
	fmt.Printf("%q %q\n", bs1, bs2)
}

func init() {
	oldRegisterCompressor(CompressTypeGzip, &GzipCompress{})
	oldRegisterCompressor(CompressTypeNoop, &NoopCompress{})
	oldRegisterCompressor(CompressTypeSnappy, NewSnappyCompressor())
	oldRegisterCompressor(CompressTypeStreamSnappy, NewSnappyCompressor())
	oldRegisterCompressor(CompressTypeBlockSnappy, NewSnappyBlockCompressor())
	oldRegisterCompressor(CompressTypeZlib, &ZlibCompress{})
}

var oldCompressors = make(map[int]Compressor)

func oldRegisterCompressor(compressType int, s Compressor) {
	oldCompressors[compressType] = s
}

func oldGetCompressor(compressType int) Compressor {
	return oldCompressors[compressType]
}

func oldCompress(compressorType int, in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	compressor := oldGetCompressor(compressorType)
	if compressor == nil {
		return nil, errors.New("compressor not registered")
	}
	return compressor.Compress(in)
}

func oldDecompress(compressorType int, in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	compressor := oldGetCompressor(compressorType)
	if compressor == nil {
		return nil, errors.New("compressor not registered")
	}
	return compressor.Decompress(in)
}
