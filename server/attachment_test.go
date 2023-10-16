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
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	"trpc.group/trpc-go/trpc-go/server"
)

func TestAttachment(t *testing.T) {
	msg := trpc.Message(context.Background())
	attm := server.GetAttachment(msg)
	require.Equal(t, attachment.NoopAttachment{}, attm.Request())

	attm.SetResponse(bytes.NewReader([]byte("attachment")))
	responseAttm, ok := attachment.ServerResponseAttachment(msg)
	require.True(t, ok)
	bts, err := io.ReadAll(responseAttm)
	require.Nil(t, err)
	require.Equal(t, []byte("attachment"), bts)
}
