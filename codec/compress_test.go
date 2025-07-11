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
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

func TestCompress(t *testing.T) {
	in := []byte("body")

	compress := codec.GetCompressor(0)
	out1, err := compress.Compress(in)
	assert.Nil(t, err)
	assert.Equal(t, out1, in)
	out1, err = codec.Compress(0, in)
	assert.Nil(t, err)
	assert.Equal(t, out1, in)
	out2, err := compress.Decompress(in)
	assert.Nil(t, err)
	assert.Equal(t, out2, in)
	out2, err = codec.Decompress(0, in)
	assert.Nil(t, err)
	assert.Equal(t, out2, in)

	empty := &codec.NoopCompress{}
	codec.RegisterCompressor(1, empty)

	compress = codec.GetCompressor(1)
	assert.Equal(t, empty, compress)
	in = nil
	out3, err := codec.Compress(0, in)
	assert.Nil(t, err)
	assert.Equal(t, out3, in)

	in = nil
	out4, err := compress.Decompress(in)
	assert.Nil(t, err)
	assert.Equal(t, out4, in)
	out4, err = codec.Decompress(0, in)
	assert.Nil(t, err)
	assert.Equal(t, out4, in)
	t.Run("invalid compress type", func(t *testing.T) {
		const invalidCompressType = -1

		in = []byte("body")
		out5, err := codec.Compress(invalidCompressType, in)
		assert.Nil(t, out5)
		assert.NotNil(t, err)

		in = []byte("body")
		out6, err := codec.Decompress(invalidCompressType, in)
		assert.Nil(t, out6)
		assert.NotNil(t, err)
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

	compress := &codec.GzipCompress{}

	emptyIn := []byte{}

	out1, err := compress.Compress(emptyIn)
	assert.Nil(t, err)
	assert.Equal(t, len(out1), 0)

	out2, err := compress.Decompress(emptyIn)
	assert.Nil(t, err)
	assert.Equal(t, len(out2), 0)

	in := []byte("A long time ago in a galaxy far, far away...")

	out3, err := compress.Compress(in)
	assert.Nil(t, err)
	assert.Equal(t, []byte{0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x72, 0x54,
		0xc8, 0xc9, 0xcf, 0x4b, 0x57, 0x28, 0xc9, 0xcc, 0x4d, 0x55, 0x48, 0x4c, 0xcf, 0x57,
		0xc8, 0xcc, 0x53, 0x48, 0x54, 0x48, 0x4f, 0xcc, 0x49, 0xac, 0xa8, 0x54, 0x48, 0x4b,
		0x2c, 0xd2, 0x1, 0x11, 0xa, 0x89, 0xe5, 0x89, 0x95, 0x7a, 0x7a, 0x7a, 0x80, 0x0, 0x0,
		0x0, 0xff, 0xff, 0x10, 0x8a, 0xa3, 0xef, 0x2c, 0x0, 0x0, 0x0}, out3)

	out4, err := compress.Decompress(out3)
	assert.Nil(t, err)
	assert.Equal(t, out4, in)

	invalidIn := []byte("hahahahah")
	_, err = compress.Decompress(invalidIn)
	assert.NotNil(t, err)
}

func TestZlib(t *testing.T) {

	compress := &codec.ZlibCompress{}

	emptyIn := []byte{}

	out1, err := compress.Compress(emptyIn)
	assert.Nil(t, err)
	assert.Equal(t, len(out1), 0)

	out2, err := compress.Decompress(emptyIn)
	assert.Nil(t, err)
	assert.Equal(t, len(out2), 0)

	in := []byte("A long time ago in a galaxy far, far away...")

	out3, err := compress.Compress(in)
	assert.Nil(t, err)

	out4, err := compress.Decompress(out3)
	assert.Nil(t, err)
	assert.Equal(t, out4, in)

	invalidIn := []byte("hahahahah")
	_, err = compress.Decompress(invalidIn)
	assert.NotNil(t, err)
}

func TestSnappy(t *testing.T) {
	compress := &codec.SnappyCompress{}
	testSnappyCompressor(t, compress)
}

func TestSnappyWithPool(t *testing.T) {
	compress := codec.NewSnappyCompressor()
	testSnappyCompressor(t, compress)
}

func TestSnappyBlockFormat(t *testing.T) {
	compress := codec.NewSnappyBlockCompressor()
	testSnappyBlockCompressor(t, compress)
}

func testSnappyCompressor(t *testing.T, compress *codec.SnappyCompress) {
	emptyIn := []byte{}

	out1, err := compress.Compress(emptyIn)
	assert.Nil(t, err)
	assert.Equal(t, len(out1), 0)

	out2, err := compress.Decompress(emptyIn)
	assert.Nil(t, err)
	assert.Equal(t, len(out2), 0)

	in := []byte("A long time ago in a galaxy far, far away...")

	out3, err := compress.Compress(in)
	assert.Nil(t, err)
	assert.Equal(t, []byte{0xff, 0x6, 0x0, 0x0, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x59, 0x1,
		0x30, 0x0, 0x0, 0xc0, 0xe7, 0x2c, 0x24, 0x41, 0x20, 0x6c, 0x6f, 0x6e, 0x67, 0x20,
		0x74, 0x69, 0x6d, 0x65, 0x20, 0x61, 0x67, 0x6f, 0x20, 0x69, 0x6e, 0x20, 0x61, 0x20,
		0x67, 0x61, 0x6c, 0x61, 0x78, 0x79, 0x20, 0x66, 0x61, 0x72, 0x2c, 0x20, 0x66, 0x61,
		0x72, 0x20, 0x61, 0x77, 0x61, 0x79, 0x2e, 0x2e, 0x2e}, out3)

	out4, err := compress.Decompress(out3)
	assert.Nil(t, err)
	assert.Equal(t, out4, in)

	invalidIn := []byte("hahahahah")
	_, err = compress.Decompress(invalidIn)
	assert.NotNil(t, err)
}

func testSnappyBlockCompressor(t *testing.T, compress *codec.SnappyBlockCompressor) {
	emptyIn := []byte{}

	out1, err := compress.Compress(emptyIn)
	assert.Nil(t, err)
	assert.Equal(t, len(out1), 0)

	out2, err := compress.Decompress(emptyIn)
	assert.Nil(t, err)
	assert.Equal(t, len(out2), 0)

	in := []byte("A long time ago in a galaxy far, far away...")

	out3, err := compress.Compress(in)
	assert.Nil(t, err)
	assert.Equal(t, []byte{0x2c, 0xac, 0x41, 0x20, 0x6c, 0x6f, 0x6e, 0x67, 0x20, 0x74,
		0x69, 0x6d, 0x65, 0x20, 0x61, 0x67, 0x6f, 0x20, 0x69, 0x6e, 0x20, 0x61, 0x20,
		0x67, 0x61, 0x6c, 0x61, 0x78, 0x79, 0x20, 0x66, 0x61, 0x72, 0x2c, 0x20, 0x66,
		0x61, 0x72, 0x20, 0x61, 0x77, 0x61, 0x79, 0x2e, 0x2e, 0x2e}, out3)

	out4, err := compress.Decompress(out3)
	assert.Nil(t, err)
	assert.Equal(t, out4, in)

	invalidIn := []byte("hahahahah")
	_, err = compress.Decompress(invalidIn)
	assert.NotNil(t, err)
}

func BenchmarkGzipCompress_Compress(b *testing.B) {
	in := make([]byte, 10280)
	rand.Read(in)
	compress := &codec.GzipCompress{}
	for i := 0; i < b.N; i++ {
		compress.Compress(in)
	}
}

func BenchmarkGzipCompress_Decompress(b *testing.B) {
	in := make([]byte, 10280)
	rand.Read(in)
	compress := &codec.GzipCompress{}
	compressBytes, _ := compress.Compress(in)

	for i := 0; i < b.N; i++ {
		compress.Decompress(compressBytes)
	}
}

func BenchmarkSnappyBlockCompress_Compress(b *testing.B) {
	in := make([]byte, 10280)
	rand.Read(in)
	compress := &codec.SnappyBlockCompressor{}
	for i := 0; i < b.N; i++ {
		compress.Compress(in)
	}
}

func BenchmarkSnappyBlockCompress_Decompress(b *testing.B) {
	in := make([]byte, 10280)
	rand.Read(in)
	compress := &codec.SnappyBlockCompressor{}
	compressBytes, _ := compress.Compress(in)

	for i := 0; i < b.N; i++ {
		compress.Decompress(compressBytes)
	}
}

func BenchmarkSnappyCompress_Compress_Pool(b *testing.B) {
	in := make([]byte, 10280)
	rand.Read(in)
	compress := codec.NewSnappyCompressor()
	for i := 0; i < b.N; i++ {
		compress.Compress(in)
	}
}

func BenchmarkSnappyCompress_Compress_NoPool(b *testing.B) {
	in := make([]byte, 10280)
	rand.Read(in)
	compress := &codec.SnappyCompress{}
	for i := 0; i < b.N; i++ {
		compress.Compress(in)
	}
}

func BenchmarkSnappyCompress_Decompress_Pool(b *testing.B) {
	in := make([]byte, 10280)
	rand.Read(in)
	compress := codec.NewSnappyCompressor()
	compressBytes, _ := compress.Compress(in)

	for i := 0; i < b.N; i++ {
		compress.Decompress(compressBytes)
	}
}

func BenchmarkSnappyCompress_Decompress_NoPool(b *testing.B) {
	in := make([]byte, 10280)
	rand.Read(in)
	compress := &codec.SnappyCompress{}
	compressBytes, _ := compress.Compress(in)

	for i := 0; i < b.N; i++ {
		compress.Decompress(compressBytes)
	}
}
