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

package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssignPreUnmarshal(t *testing.T) {
	var dst string
	require.NoError(t, assignPreUnmarshal(&dst, nil))

	require.ErrorContains(t, assignPreUnmarshal(nil, "value"), "destination must be a non-nil pointer")

	var nilSrc *string
	require.NoError(t, assignPreUnmarshal(&dst, nilSrc))

	require.ErrorContains(t, assignPreUnmarshal(&dst, 1), "cannot assign")

	require.NoError(t, assignPreUnmarshal(&dst, "value"))
	require.Equal(t, "value", dst)
}
