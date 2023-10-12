// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

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
			attachment.ClientRequestAttachment(nil)
		})
	})
	t.Run("empty message", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		_, ok := attachment.ClientRequestAttachment(msg)
		require.False(t, ok)
	})
	t.Run("message contains nil attachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: nil})
		_, ok := attachment.ClientRequestAttachment(msg)
		require.False(t, ok)
	})
	t.Run("message contains non-empty Request attachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		want := bytes.NewReader([]byte("attachment"))
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: &attachment.Attachment{Request: want}})
		got, ok := attachment.ClientRequestAttachment(msg)
		require.True(t, ok)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ServerResponseAttachment() = %v, want %v", got, want)
		}
	})
}

func TestGetServerResponseAttachment(t *testing.T) {
	t.Run("nil message", func(t *testing.T) {
		require.Panics(t, func() {
			attachment.ServerResponseAttachment(nil)
		})
	})
	t.Run("empty message", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		_, ok := attachment.ServerResponseAttachment(msg)
		require.False(t, ok)
	})
	t.Run("message contains nil attachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: nil})
		_, ok := attachment.ClientRequestAttachment(msg)
		require.False(t, ok)
	})
	t.Run("message contains non-empty response attachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		want := bytes.NewReader([]byte("attachment"))
		msg.WithCommonMeta(codec.CommonMeta{attachment.ServerAttachmentKey{}: &attachment.Attachment{Response: want}})
		got, ok := attachment.ServerResponseAttachment(msg)
		require.True(t, ok)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ServerResponseAttachment() = %v, want %v", got, want)
		}
	})
}

func TestSetClientResponseAttachment(t *testing.T) {
	msg := trpc.Message(context.Background())
	var a attachment.Attachment
	msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: &a})
	attachment.SetClientResponseAttachment(msg, []byte("attachment"))
	bts, err := io.ReadAll(a.Response)

	require.Nil(t, err)
	require.Equal(t, []byte("attachment"), bts)
}

func TestSetServerAttachment(t *testing.T) {
	msg := trpc.Message(context.Background())
	attachment.SetServerRequestAttachment(msg, []byte("attachment"))
	bts, err := io.ReadAll(server.GetAttachment(msg).Request())

	require.Nil(t, err)
	require.Equal(t, []byte("attachment"), bts)
}
