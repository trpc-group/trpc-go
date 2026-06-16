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

package restful

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPutBackBodyBufferNilSafe(t *testing.T) {
	require.NotPanics(t, func() {
		putBackBodyBuffer(nil)
	})
}

func TestPutBackBodyBufferKeepsSmallBuffer(t *testing.T) {
	buffer := bytes.NewBufferString("payload")
	require.LessOrEqual(t, buffer.Cap(), maxPooledBodyBufferSize)

	putBackBodyBuffer(buffer)

	require.Zero(t, buffer.Len())
	got := bodyBufferPool.Get().(*bytes.Buffer)
	putBackBodyBuffer(got)
}

func TestPutBackBodyBufferDropsOversizedBuffer(t *testing.T) {
	buffer := bodyBufferPool.Get().(*bytes.Buffer)
	buffer.Grow(maxPooledBodyBufferSize + 1)
	putBackBodyBuffer(buffer)

	for i := 0; i < 8; i++ {
		got := bodyBufferPool.Get().(*bytes.Buffer)
		require.NotSame(t, buffer, got)
		putBackBodyBuffer(got)
	}
}
