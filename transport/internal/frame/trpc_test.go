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

package frame

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainTRPCStreamHeader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.False(t, ContainTRPCStreamHeader([]byte("")))
	})
	t.Run("not TRPC Frame", func(t *testing.T) {
		bts := make([]byte, trpcFrameHeadLen)
		binary.BigEndian.PutUint16(bts, trpcMagicVALUE-1)
		require.False(t, ContainTRPCStreamHeader(bts))
	})
	t.Run("not TRPC Stream Frame", func(t *testing.T) {
		bts := make([]byte, trpcFrameHeadLen)
		binary.BigEndian.PutUint16(bts, trpcMagicVALUE)
		bts[2] = trpcStreamFrameType - 1
		require.False(t, ContainTRPCStreamHeader(bts))
	})
	t.Run("TRPC Stream Frame", func(t *testing.T) {
		bts := make([]byte, trpcFrameHeadLen)
		binary.BigEndian.PutUint16(bts, trpcMagicVALUE)
		bts[2] = trpcStreamFrameType
		require.True(t, ContainTRPCStreamHeader(bts))
	})
}
