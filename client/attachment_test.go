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

package client

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
)

func TestAttachment(t *testing.T) {
	t.Run("sizer interface hasn't been implemented", func(t *testing.T) {
		want := []byte("attachment")

		attm := NewAttachment(bytes.NewBuffer(want))
		require.Equal(t, attachment.NoopAttachment{}, attm.Response())

		msg := codec.Message(context.Background())
		setAttachment(msg, &attm.attachment)
		a, err := attachment.ClientRequestSizedAttachment(msg)
		require.Nil(t, err)
		require.EqualValues(t, len(want), a.Size())

		got := make([]byte, len(want))
		require.Nil(t, a.ReadAll(got))
		require.Equal(t, got, []byte("attachment"))
	})
	t.Run("sizer interface has been implemented", func(t *testing.T) {
		want := []byte("attachment")
		attm := NewAttachment(bytes.NewReader(want))
		require.Equal(t, attachment.NoopAttachment{}, attm.Response())

		msg := codec.Message(context.Background())
		setAttachment(msg, &attm.attachment)
		a, err := attachment.ClientRequestSizedAttachment(msg)
		require.Nil(t, err)
		require.EqualValues(t, len(want), a.Size())

		got := make([]byte, len(want))
		require.Nil(t, a.ReadAll(got))
		require.Equal(t, got, []byte("attachment"))
	})

}
