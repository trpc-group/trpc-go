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

package rpcz_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	irpcz "trpc.group/trpc-go/trpc-go/internal/rpcz"
	"trpc.group/trpc-go/trpc-go/rpcz"
)

func TestFilterNames(t *testing.T) {
	old := rpcz.GlobalRPCZ
	defer func() { rpcz.GlobalRPCZ = old }()
	rpcz.GlobalRPCZ = rpcz.NewRPCZ(&rpcz.Config{Fraction: 1.0, Capacity: 10})
	ctx := context.Background()
	_, ok := irpcz.FilterNames(ctx)
	require.False(t, ok)
	span, end, ctx := rpcz.NewSpanContext(ctx, "")
	defer end.End()
	filterNames := []string{"f1", "f2"}
	span.SetAttribute(rpcz.TRPCAttributeFilterNames, filterNames)
	filterNamesFromContext, ok := irpcz.FilterNames(ctx)
	require.True(t, ok)
	require.Equal(t, filterNames, filterNamesFromContext)
	const index = 0
	require.Equal(t, filterNames[index], irpcz.FilterName(filterNamesFromContext, index))
	require.NotEqual(t, filterNames[index], irpcz.FilterName(filterNamesFromContext, len(filterNames)))
}
