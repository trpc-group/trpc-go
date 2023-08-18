// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
)

func TestAttachment(t *testing.T) {
	attm := NewAttachment(bytes.NewReader([]byte("attachment")))
	require.Equal(t, attachment.NoopAttachment{}, attm.Response())

	msg := codec.Message(context.Background())
	setAttachment(msg, &attm.attachment)
	attcher := attachment.GetClientRequestAttachment(msg)
	bts, err := io.ReadAll(attcher)
	require.Nil(t, err)
	require.Equal(t, []byte("attachment"), bts)
}
