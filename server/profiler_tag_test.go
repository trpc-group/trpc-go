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
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/server"
)

func TestProfileLabel(t *testing.T) {
	profileLabel := server.NewProfileLabel()
	require.Equal(t, 0, profileLabel.Len())

	profileLabel.Store("k1", "v1")
	require.Equal(t, 1, profileLabel.Len())
	value, ok := profileLabel.Load("k1")
	require.Equal(t, "v1", value)
	require.True(t, ok)
	_, ok = profileLabel.Load("k2")
	require.False(t, ok)

	profileLabel.Store("k1", "v2")
	require.Equal(t, 1, profileLabel.Len())
	value, ok = profileLabel.Load("k1")
	require.Equal(t, "v2", value)
	require.True(t, ok)
}
