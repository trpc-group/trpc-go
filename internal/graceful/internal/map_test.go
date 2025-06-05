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

package graceful

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	var mp map[string]map[int]float64
	mp = appendMap(mp, "a", 1, 0.1)
	mp1, ok := mp["a"]
	require.True(t, ok)
	require.Equal(t, 0.1, mp1[1])

	require.Equal(t, 0, len(mp["b"]))
	require.Equal(t, 1, len(mp["a"]))
	deleteMap(mp, "a", 2)
	require.Equal(t, 1, len(mp["a"]))
	deleteMap(mp, "a", 1)
	require.Equal(t, 0, len(mp["a"]))
}
