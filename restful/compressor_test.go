// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package restful_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/restful"
)

type anonymousCompressor struct {
	restful.Compressor
}

func (anonymousCompressor) Name() string { return "" }

type mockCompressor struct {
	restful.Compressor
}

func (mockCompressor) Name() string            { return "mock" }
func (mockCompressor) ContentEncoding() string { return "mock" }

type reader struct {
	io.Reader
}

func (reader) Read([]byte) (int, error) {
	return 0, errors.New("mock error")
}

func TestRegisterCompressor(t *testing.T) {
	for _, test := range []struct {
		compressor  restful.Compressor
		expectPanic bool
		desc        string
	}{
		{
			compressor:  nil,
			expectPanic: true,
			desc:        "register nil compressor test",
		},
		{
			compressor:  anonymousCompressor{},
			expectPanic: true,
			desc:        "register anonymous compressor test",
		},
		{
			compressor:  mockCompressor{},
			expectPanic: false,
			desc:        "register mock compressor test",
		},
	} {
		register := func() { restful.RegisterCompressor(test.compressor) }
		if test.expectPanic {
			require.Panics(t, register, test.desc)
		} else {
			require.NotPanics(t, register, test.desc)
		}
		var c restful.Compressor
		if !test.expectPanic {
			c = restful.GetCompressor(test.compressor.Name())
			require.True(t, reflect.DeepEqual(c, test.compressor), test.desc)
		}
	}
}

func TestGZIPCompressor(t *testing.T) {
	g := &restful.GZIPCompressor{}

	require.Equal(t, "gzip", g.Name())
	require.Equal(t, "gzip", g.ContentEncoding())

	input := []byte("foobar foo bar baz")
	buf := new(bytes.Buffer)
	w, err := g.Compress(buf)
	require.Nil(t, err)
	_, err = w.Write(input)
	require.Nil(t, err)
	err = w.Close()
	require.Nil(t, err)
	wrong := reader{}
	_, err = g.Decompress(wrong)
	require.NotNil(t, err)
	r, err := g.Decompress(buf)
	require.Nil(t, err)
	out, err := io.ReadAll(r)
	require.Nil(t, err)
	require.Equal(t, input, out)
}
