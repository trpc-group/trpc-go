// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package loadbalance

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-go/naming/bannednodes"
	"trpc.group/trpc-go/trpc-go/naming/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomEmptyList(t *testing.T) {
	b := NewRandom()
	_, err := b.Select("", nil)
	assert.Equal(t, err, ErrNoServerAvailable)
}

func TestRandomGet(t *testing.T) {
	b := NewRandom()
	node, err := b.Select("", []*registry.Node{testNode})
	assert.Nil(t, err)
	assert.Equal(t, node, testNode)
}

func TestRandom_SelectMandatoryBanned(t *testing.T) {
	candidates := []*registry.Node{
		{ServiceName: "1", Address: "1"},
		{ServiceName: "2", Address: "2"},
	}

	ctx := context.Background()
	ctx = bannednodes.NewCtx(ctx, true)

	b := NewRandom()
	_, err := b.Select("", candidates, WithContext(ctx))
	require.Nil(t, err)

	_, err = b.Select("", candidates, WithContext(ctx))
	require.Nil(t, err)

	_, err = b.Select("", candidates, WithContext(ctx))
	require.NotNil(t, err)

	nodes, mandatory, ok := bannednodes.FromCtx(ctx)
	require.True(t, ok)
	require.True(t, mandatory)
	var n int
	nodes.Range(func(*registry.Node) bool {
		n++
		return true
	})
	require.Equal(t, 2, n)
}

func TestRandom_SelectOptionalBanned(t *testing.T) {
	candidates := []*registry.Node{
		{ServiceName: "1", Address: "1"},
		{ServiceName: "2", Address: "2"},
	}

	ctx := context.Background()
	ctx = bannednodes.NewCtx(ctx, false)

	b := NewRandom()
	n1, err := b.Select("", candidates, WithContext(ctx))
	require.Nil(t, err)

	n2, err := b.Select("", candidates, WithContext(ctx))
	require.Nil(t, err)
	require.NotEqual(t, n1.Address, n2.Address)

	_, err = b.Select("", candidates, WithContext(ctx))
	require.Nil(t, err)

	nodes, mandatory, ok := bannednodes.FromCtx(ctx)
	require.True(t, ok)
	require.False(t, mandatory)
	var n int
	nodes.Range(func(*registry.Node) bool {
		n++
		return true
	})
	require.Equal(t, 3, n)
}
