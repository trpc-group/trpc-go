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

package server_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	"trpc.group/trpc-go/trpc-go/server"
)

func TestAttachment(t *testing.T) {
	t.Run("sizer interface hasn't been implemented", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		attm := server.GetAttachment(msg)
		require.Equal(t, attachment.NoopAttachment{}, attm.Request())

		want := []byte("attachment")
		attm.SetResponse(bytes.NewBuffer(want))
		responseAttm, err := attachment.ServerResponseSizedAttachment(msg)
		require.Nil(t, err)
		require.EqualValues(t, len(want), responseAttm.Size())

		got := make([]byte, len(want))
		require.Nil(t, responseAttm.ReadAll(got))
		require.Equal(t, want, got)
	})
	t.Run("sizer interface has been implemented", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		attm := server.GetAttachment(msg)
		require.Equal(t, attachment.NoopAttachment{}, attm.Request())

		want := []byte("attachment")
		attm.SetResponse(bytes.NewReader(want))
		responseAttm, err := attachment.ServerResponseSizedAttachment(msg)
		require.Nil(t, err)
		require.EqualValues(t, len(want), responseAttm.Size())

		got := make([]byte, len(want))
		require.Nil(t, responseAttm.ReadAll(got))
		require.Equal(t, want, got)
	})
}
