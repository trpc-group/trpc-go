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

package msg_test

import (
	"context"
	"testing"

	trpc "trpc.group/trpc-go/trpc-go"
	imsg "trpc.group/trpc-go/trpc-go/transport/internal/msg"

	"github.com/stretchr/testify/require"
)

func TestWithLocalAddr(t *testing.T) {
	t.Run("empty address", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		got := imsg.WithLocalAddr(msg, "tcp", "")
		require.Equal(t, msg, got)
		require.Nil(t, msg.LocalAddr())
	})
	t.Run("non-empty address", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		got := imsg.WithLocalAddr(msg, "tcp", "localhost:8080")
		require.Equal(t, msg, got)
		require.Equal(t, "localhost:8080", msg.LocalAddr().String())
	})
}
