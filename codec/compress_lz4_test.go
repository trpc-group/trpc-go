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

package codec_test

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
	"github.com/stretchr/testify/require"
)

func TestLZ4Compression(t *testing.T) {
	cases := [][]byte{
		[]byte("hello"),
		[]byte(nil),
	}
	compressors := []codec.Compressor{
		codec.NewLZ4BlockCompressor(),
		codec.NewLZ4StreamCompressor(),
		&codec.LZ4BlockCompressor{},
		&codec.LZ4StreamCompressor{},
	}
	for _, compressor := range compressors {
		for _, c := range cases {
			bs, err := compressor.Compress(c)
			require.Nil(t, err)
			t.Logf("compressed: %q", bs)
			hh, err := compressor.Decompress(bs)
			require.Nil(t, err)
			t.Logf("decompressed: %q", hh)
			require.Equal(t, c, hh)
		}
	}
}
