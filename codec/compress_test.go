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
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

func TestNoopCompress(t *testing.T) {
	t.Run("Compress", func(t *testing.T) {
		in := []byte("body")
		compress := codec.GetCompressor(codec.CompressTypeNoop)
		out1, err := compress.Compress(in)
		require.Nil(t, err)
		require.Equal(t, out1, in)

		out2, err := compress.Decompress(in)
		require.Nil(t, err)
		require.Equal(t, out2, in)

	})
	t.Run("codec.Compress", func(t *testing.T) {
		var in []byte
		out1, err := codec.Compress(codec.CompressTypeNoop, in)
		require.Nil(t, err)
		require.Equal(t, out1, in)

		out2, err := codec.Decompress(codec.CompressTypeNoop, in)
		require.Nil(t, err)
		require.Equal(t, out2, in)
	})
	t.Run("RegisterCompressor", func(t *testing.T) {
		emptyCompressor := &codec.NoopCompress{}
		const emptyCompressType = codec.CompressTypeGzip
		oldCompressor := codec.GetCompressor(emptyCompressType)
		codec.RegisterCompressor(emptyCompressType, emptyCompressor)
		defer func() {
			codec.RegisterCompressor(emptyCompressType, oldCompressor)
		}()

		compressor := codec.GetCompressor(1)
		require.Equal(t, emptyCompressor, compressor)

		in := []byte("body")
		out1, err := codec.Compress(0, in)
		require.Nil(t, err)
		require.Equal(t, out1, in)

		out2, err := codec.Decompress(0, in)
		require.Nil(t, err)
		require.Equal(t, out2, in)
	})

	t.Run("invalid compress type", func(t *testing.T) {
		const invalidCompressType = -1

		in := []byte("body")
		out1, err := codec.Compress(invalidCompressType, in)
		require.Nil(t, out1)
		require.NotNil(t, err)

		out2, err := codec.Decompress(invalidCompressType, in)
		require.Nil(t, out2)
		require.NotNil(t, err)
	})
}

func TestMustRegisterCompressor(t *testing.T) {
	noop := &codec.NoopCompress{}
	codec.MustRegisterCompressor(1000, noop)

	t.Run("no registered compressor", func(t *testing.T) {
		require.Nil(t, codec.GetCompressor(100))
	})

	t.Run("registered compressor", func(t *testing.T) {
		require.Equal(t, noop, codec.GetCompressor(1000))
	})

	t.Run("repeat register", func(t *testing.T) {
		require.Panics(t, func() {
			codec.MustRegisterCompressor(1000, noop)
		})
	})
}

func TestRegisterNegativeCompress(t *testing.T) {
	const negativeCompressType = -1
	codec.RegisterCompressor(negativeCompressType, &codec.NoopCompress{})
	in := []byte("body")
	bts, err := codec.Compress(negativeCompressType, in)
	require.Nil(t, err)
	out, err := codec.Decompress(negativeCompressType, bts)
	require.Nil(t, err)
	require.Equal(t, in, out)
}

func TestGzip(t *testing.T) {
	t.Run("Compress and Decompress ok", func(t *testing.T) {
		compressor := &codec.GzipCompress{}
		tests := []struct {
			input      []byte
			wantOutput []byte
		}{
			{nil, nil},
			{[]byte("A long time ago in a galaxy far, far away..."),
				[]byte{0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x72, 0x54,
					0xc8, 0xc9, 0xcf, 0x4b, 0x57, 0x28, 0xc9, 0xcc, 0x4d, 0x55, 0x48, 0x4c, 0xcf, 0x57,
					0xc8, 0xcc, 0x53, 0x48, 0x54, 0x48, 0x4f, 0xcc, 0x49, 0xac, 0xa8, 0x54, 0x48, 0x4b,
					0x2c, 0xd2, 0x1, 0x11, 0xa, 0x89, 0xe5, 0x89, 0x95, 0x7a, 0x7a, 0x7a, 0x80, 0x0, 0x0,
					0x0, 0xff, 0xff, 0x10, 0x8a, 0xa3, 0xef, 0x2c, 0x0, 0x0, 0x0},
			},
		}
		for _, tt := range tests {
			temp, err := compressor.Compress(tt.input)
			require.Nil(t, err)
			require.Equal(t, tt.wantOutput, temp)

			out, err := compressor.Decompress(temp)
			require.Nil(t, err)
			require.Equal(t, tt.input, out)
		}
	})
	t.Run("Decompress Fail", func(t *testing.T) {
		compressor := &codec.GzipCompress{}
		invalidIn := []byte("invalid input")
		_, err := compressor.Decompress(invalidIn)
		require.NotNil(t, err)
	})
}

func TestZlib(t *testing.T) {
	t.Run("Compress and Decompress ok", func(t *testing.T) {
		compressor := &codec.GzipCompress{}
		tests := []struct {
			input      []byte
			wantOutput []byte
		}{
			{nil, nil},
			{[]byte("A long time ago in a galaxy far, far away..."),
				[]byte{0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x72, 0x54,
					0xc8, 0xc9, 0xcf, 0x4b, 0x57, 0x28, 0xc9, 0xcc, 0x4d, 0x55, 0x48, 0x4c, 0xcf, 0x57,
					0xc8, 0xcc, 0x53, 0x48, 0x54, 0x48, 0x4f, 0xcc, 0x49, 0xac, 0xa8, 0x54, 0x48, 0x4b,
					0x2c, 0xd2, 0x1, 0x11, 0xa, 0x89, 0xe5, 0x89, 0x95, 0x7a, 0x7a, 0x7a, 0x80, 0x0, 0x0,
					0x0, 0xff, 0xff, 0x10, 0x8a, 0xa3, 0xef, 0x2c, 0x0, 0x0, 0x0},
			},
		}
		for _, tt := range tests {
			temp, err := compressor.Compress(tt.input)
			require.Nil(t, err)
			require.Equal(t, tt.wantOutput, temp)

			out, err := compressor.Decompress(temp)
			require.Nil(t, err)
			require.Equal(t, tt.input, out)
		}
	})
	t.Run("Decompress Fail", func(t *testing.T) {
		compressor := &codec.GzipCompress{}
		invalidIn := []byte("invalid input")
		_, err := compressor.Decompress(invalidIn)
		require.NotNil(t, err)
	})
}

func TestSnappy(t *testing.T) {
	t.Run("Compress and Decompress ok", func(t *testing.T) {
		compressor := &codec.SnappyCompress{}
		tests := []struct {
			input      []byte
			wantOutput []byte
		}{
			{nil, nil},
			{[]byte("A long time ago in a galaxy far, far away..."),
				[]byte{0xff, 0x6, 0x0, 0x0, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x59, 0x1,
					0x30, 0x0, 0x0, 0xc0, 0xe7, 0x2c, 0x24, 0x41, 0x20, 0x6c, 0x6f, 0x6e, 0x67, 0x20,
					0x74, 0x69, 0x6d, 0x65, 0x20, 0x61, 0x67, 0x6f, 0x20, 0x69, 0x6e, 0x20, 0x61, 0x20,
					0x67, 0x61, 0x6c, 0x61, 0x78, 0x79, 0x20, 0x66, 0x61, 0x72, 0x2c, 0x20, 0x66, 0x61,
					0x72, 0x20, 0x61, 0x77, 0x61, 0x79, 0x2e, 0x2e, 0x2e},
			},
		}
		for _, tt := range tests {
			temp, err := compressor.Compress(tt.input)
			require.Nil(t, err)
			require.Equal(t, tt.wantOutput, temp)

			out, err := compressor.Decompress(temp)
			require.Nil(t, err)
			require.Equal(t, tt.input, out)
		}
	})
	t.Run("Decompress Fail", func(t *testing.T) {
		compressor := &codec.SnappyCompress{}
		invalidIn := []byte("invalid input")
		_, err := compressor.Decompress(invalidIn)
		require.NotNil(t, err)
	})
}

func TestSnappyWithPool(t *testing.T) {
	t.Run("Compress and Decompress ok", func(t *testing.T) {
		compressor := codec.NewSnappyCompressor()
		tests := []struct {
			input      []byte
			wantOutput []byte
		}{
			{nil, nil},
			{[]byte("A long time ago in a galaxy far, far away..."),
				[]byte{0xff, 0x6, 0x0, 0x0, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x59, 0x1,
					0x30, 0x0, 0x0, 0xc0, 0xe7, 0x2c, 0x24, 0x41, 0x20, 0x6c, 0x6f, 0x6e, 0x67, 0x20,
					0x74, 0x69, 0x6d, 0x65, 0x20, 0x61, 0x67, 0x6f, 0x20, 0x69, 0x6e, 0x20, 0x61, 0x20,
					0x67, 0x61, 0x6c, 0x61, 0x78, 0x79, 0x20, 0x66, 0x61, 0x72, 0x2c, 0x20, 0x66, 0x61,
					0x72, 0x20, 0x61, 0x77, 0x61, 0x79, 0x2e, 0x2e, 0x2e},
			},
		}
		for _, tt := range tests {
			temp, err := compressor.Compress(tt.input)
			require.Nil(t, err)
			require.Equal(t, tt.wantOutput, temp)

			out, err := compressor.Decompress(temp)
			require.Nil(t, err)
			require.Equal(t, tt.input, out)
		}
	})
	t.Run("Decompress Fail", func(t *testing.T) {
		compressor := codec.NewSnappyCompressor()
		invalidIn := []byte("invalid input")
		_, err := compressor.Decompress(invalidIn)
		require.NotNil(t, err)
	})
}

func TestSnappyBlockCompressor(t *testing.T) {
	t.Run("Compress and Decompress ok", func(t *testing.T) {
		compressor := codec.NewSnappyBlockCompressor()
		tests := []struct {
			input      []byte
			wantOutput []byte
		}{
			{nil, nil},
			{[]byte("A long time ago in a galaxy far, far away..."),
				[]byte{0x2c, 0xac, 0x41, 0x20, 0x6c, 0x6f, 0x6e, 0x67, 0x20, 0x74,
					0x69, 0x6d, 0x65, 0x20, 0x61, 0x67, 0x6f, 0x20, 0x69, 0x6e, 0x20, 0x61, 0x20,
					0x67, 0x61, 0x6c, 0x61, 0x78, 0x79, 0x20, 0x66, 0x61, 0x72, 0x2c, 0x20, 0x66,
					0x61, 0x72, 0x20, 0x61, 0x77, 0x61, 0x79, 0x2e, 0x2e, 0x2e},
			},
		}
		for _, tt := range tests {
			temp, err := compressor.Compress(tt.input)
			require.Nil(t, err)
			require.Equal(t, tt.wantOutput, temp)

			out, err := compressor.Decompress(temp)
			require.Nil(t, err)
			require.Equal(t, tt.input, out)
		}
	})
	t.Run("Decompress Fail", func(t *testing.T) {
		compressor := codec.NewSnappyBlockCompressor()
		invalidIn := []byte("invalid input")
		_, err := compressor.Decompress(invalidIn)
		require.NotNil(t, err)
	})
}

func BenchmarkGzipCompress_Compress(b *testing.B) {
	bts := newRandBytes(b, 10280)
	compress := &codec.GzipCompress{}
	for i := 0; i < b.N; i++ {
		_, _ = compress.Compress(bts)
	}
}

func BenchmarkGzipCompress_Decompress(b *testing.B) {
	compress := &codec.GzipCompress{}
	compressBytes, _ := compress.Compress(newRandBytes(b, 10280))

	for i := 0; i < b.N; i++ {
		_, _ = compress.Decompress(compressBytes)
	}
}

func BenchmarkSnappyBlockCompress_Compress(b *testing.B) {
	bts := newRandBytes(b, 10280)
	compress := &codec.SnappyBlockCompressor{}
	for i := 0; i < b.N; i++ {
		_, _ = compress.Compress(bts)
	}
}

func BenchmarkSnappyBlockCompress_Decompress(b *testing.B) {
	compress := &codec.SnappyBlockCompressor{}
	compressBytes, _ := compress.Compress(newRandBytes(b, 10280))

	for i := 0; i < b.N; i++ {
		_, _ = compress.Decompress(compressBytes)
	}
}

func BenchmarkSnappyCompress_Compress_Pool(b *testing.B) {
	bts := newRandBytes(b, 10280)
	compress := codec.NewSnappyCompressor()
	for i := 0; i < b.N; i++ {
		_, _ = compress.Compress(bts)
	}
}

func BenchmarkSnappyCompress_Compress_NoPool(b *testing.B) {
	bts := newRandBytes(b, 10280)
	compress := &codec.SnappyCompress{}
	for i := 0; i < b.N; i++ {
		_, _ = compress.Compress(bts)
	}
}

func BenchmarkSnappyCompress_Decompress_Pool(b *testing.B) {
	compress := codec.NewSnappyCompressor()
	compressBytes, _ := compress.Compress(newRandBytes(b, 10280))

	for i := 0; i < b.N; i++ {
		_, _ = compress.Decompress(compressBytes)
	}
}

func BenchmarkSnappyCompress_Decompress_NoPool(b *testing.B) {
	compress := &codec.SnappyCompress{}
	compressBytes, _ := compress.Compress(newRandBytes(b, 10280))

	for i := 0; i < b.N; i++ {
		_, _ = compress.Decompress(compressBytes)
	}
}

func newRandBytes(t *testing.B, n int) []byte {
	bts := make([]byte, n)
	if _, err := rand.Read(bts); err != nil {
		t.Fatal(err)
	}
	return bts
}
