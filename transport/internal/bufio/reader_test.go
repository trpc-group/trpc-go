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

package bufio_test

import (
	"io"
	"testing"

	. "trpc.group/trpc-go/trpc-go/transport/internal/bufio"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	r := reader{bts: []byte("abcdefg")}
	br := NewReader(&r, 4)
	buf := make([]byte, 4)

	n, err := br.Read(buf[:1])
	require.Nil(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "a", string(buf[:n]))
	require.Equal(t, 4, r.n)

	n, err = br.Read(buf[:2])
	require.Nil(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "bc", string(buf[:n]))
	require.Equal(t, 4, r.n)

	br.Unbuffer()
	require.Equal(t, 1, br.Buffered())

	n, err = br.Read(buf[:2])
	require.Nil(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "d", string(buf[:n]))
	require.Equal(t, 4, r.n)
	require.Equal(t, 0, br.Buffered())

	n, err = br.Read(buf[:2])
	require.Nil(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "ef", string(buf[:n]))
	require.Equal(t, 6, r.n)
	require.Equal(t, 0, br.Buffered())

	n, err = br.Read(buf[:0])
	require.Nil(t, err)
	require.Equal(t, 0, n)
}

type reader struct {
	bts []byte
	n   int
}

func (r *reader) Read(p []byte) (int, error) {
	n := copy(p, r.bts[r.n:])
	r.n += n
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}
