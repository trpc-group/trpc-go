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

package bannednodes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	. "trpc.group/trpc-go/trpc-go/naming/bannednodes"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

func TestNewFromCtx(t *testing.T) {
	ctx := NewCtx(context.Background(), true)
	_, mandatory, ok := FromCtx(ctx)
	require.True(t, ok)
	require.True(t, mandatory)

	ctx = NewCtx(context.Background(), false)
	_, mandatory, ok = FromCtx(ctx)
	require.True(t, ok)
	require.False(t, mandatory)
}

func TestBannedNodes(t *testing.T) {
	const addr = "127.0.0.1:8000"

	ctx := NewCtx(context.Background(), true)

	nodes, _, ok := FromCtx(ctx)
	require.True(t, ok)

	Add(ctx, &registry.Node{Address: addr})

	// the nodes returned by FromCtx is immutable, and the Add after FromCtx has no effect on nodes.
	num := 0
	nodes.Range(func(n *registry.Node) bool {
		num++
		return true
	})
	require.Equal(t, 0, num)

	nodes, _, ok = FromCtx(ctx)
	require.True(t, ok)

	notfound := nodes.Range(
		func(n *registry.Node) bool {
			return n.Address != addr
		},
	)
	require.False(t, notfound)
}
