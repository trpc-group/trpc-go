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

package attachment_test

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	"trpc.group/trpc-go/trpc-go/server"
)

func TestGetClientRequestAttachment(t *testing.T) {
	t.Run("nil message", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = attachment.ClientRequestSizedAttachment(nil)
		})
	})
	t.Run("empty message", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		a, err := attachment.ClientRequestSizedAttachment(msg)
		require.Nil(t, err)
		require.Empty(t, a)
	})
	t.Run("message contains nil SizedAttachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: nil})
		a, err := attachment.ClientRequestSizedAttachment(msg)
		require.Nil(t, err)
		require.Empty(t, a)
	})
	t.Run("message contains non-empty Request SizedAttachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		want := []byte("SizedAttachment")
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: &attachment.Attachment{Request: bytes.NewReader(want)}})

		a, err := attachment.ClientRequestSizedAttachment(msg)
		require.Nil(t, err)

		got := make([]byte, len(want))
		require.Nil(t, a.ReadAll(got))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ClientRequestSizedAttachment() = %v, want %v", got, want)
		}
	})
}

func TestGetServerResponseAttachment(t *testing.T) {
	t.Run("nil message", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = attachment.ServerResponseSizedAttachment(nil)
		})
	})
	t.Run("empty message", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		a, err := attachment.ServerResponseSizedAttachment(msg)
		require.Nil(t, err)
		require.Empty(t, a)
	})
	t.Run("message contains nil SizedAttachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: nil})
		a, err := attachment.ServerResponseSizedAttachment(msg)
		require.Nil(t, err)
		require.Empty(t, a)
	})
	t.Run("message contains non-empty response SizedAttachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		want := []byte("SizedAttachment")
		msg.WithCommonMeta(codec.CommonMeta{attachment.ServerAttachmentKey{}: &attachment.Attachment{Response: bytes.NewReader(want)}})
		a, err := attachment.ServerResponseSizedAttachment(msg)
		require.Nil(t, err)

		got := make([]byte, len(want))
		require.Nil(t, a.ReadAll(got))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ServerResponseSizedAttachment() = %v, want %v", got, want)
		}
	})
}

func TestSetClientResponseAttachment(t *testing.T) {
	msg := trpc.Message(context.Background())
	var a attachment.Attachment
	msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: &a})
	attachment.SetClientResponseAttachment(msg, []byte("SizedAttachment"))
	bts, err := io.ReadAll(a.Response)

	require.Nil(t, err)
	require.Equal(t, []byte("SizedAttachment"), bts)
}

func TestSetServerAttachment(t *testing.T) {
	msg := trpc.Message(context.Background())
	attachment.SetServerRequestAttachment(msg, []byte("SizedAttachment"))
	bts, err := io.ReadAll(server.GetAttachment(msg).Request())

	require.Nil(t, err)
	require.Equal(t, []byte("SizedAttachment"), bts)
}
