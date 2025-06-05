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

package lru_test

import (
	"testing"
	"time"

	. "trpc.group/trpc-go/trpc-go/internal/lru"
	"github.com/stretchr/testify/require"
)

func TestLRU(t *testing.T) {
	const ttl = time.Millisecond * 200
	var cnt int
	lru := NewLRU(ttl, func() interface{} {
		cnt++
		return cnt
	})
	require.Equal(t, 1, lru.Get("a"))
	time.Sleep(ttl / 2)
	require.Equal(t, 2, lru.Get("b"))
	time.Sleep(ttl / 2)
	require.Equal(t, 3, lru.Get("c"))
	require.Equal(t, 4, lru.Get("a"))
	require.Equal(t, 2, lru.Get("b"))
	time.Sleep(ttl / 2)
	require.Equal(t, 3, lru.Get("c"))
	time.Sleep(ttl / 2)
	require.Equal(t, 3, lru.Get("c"))
	require.Equal(t, 5, lru.Get("a"))
	require.Equal(t, 6, lru.Get("b"))
}
