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

package codec_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
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
			require.NoError(t, err)
			hh, err := compressor.Decompress(bs)
			require.NoError(t, err)
			require.Equal(t, c, hh)
		}
	}
}

func TestLZ4CompressorsAreRegistered(t *testing.T) {
	require.NotNil(t, codec.GetCompressor(codec.CompressTypeStreamLZ4))
	require.NotNil(t, codec.GetCompressor(codec.CompressTypeBlockLZ4))

	in := []byte("hello")
	for _, compressType := range []int{codec.CompressTypeStreamLZ4, codec.CompressTypeBlockLZ4} {
		bs, err := codec.Compress(compressType, in)
		require.NoError(t, err)
		out, err := codec.Decompress(compressType, bs)
		require.NoError(t, err)
		require.Equal(t, in, out)
	}
}

func TestLZ4DecompressInvalidInput(t *testing.T) {
	invalid := []byte("invalid")

	_, err := codec.NewLZ4StreamCompressor().Decompress(invalid)
	require.Error(t, err)

	_, err = codec.NewLZ4BlockCompressor().Decompress(invalid)
	require.Error(t, err)
}
