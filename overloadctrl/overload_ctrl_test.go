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

package overloadctrl_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
)

func TestNoop(t *testing.T) {
	noopT := overloadctrl.NoopOC{}
	token, err := noopT.Acquire(context.Background(), "")
	require.Nil(t, err)
	require.Equal(t, overloadctrl.NoopToken{}, token)
	token.OnResponse(context.Background(), nil)
	require.True(t, true, "nothing should happen")
}

func TestIsNoop(t *testing.T) {
	noopOC := overloadctrl.NoopOC{}
	require.True(t, overloadctrl.IsNoop(noopOC))
	impl := &overloadctrl.Impl{}
	require.True(t, overloadctrl.IsNoop(impl))
	impl.OverloadController = overloadctrl.NoopOC{}
	require.True(t, overloadctrl.IsNoop(impl))
	impl.OverloadController = testOC{}
	require.False(t, overloadctrl.IsNoop(impl))
	require.False(t, overloadctrl.IsNoop(testOC{}))
}

type testOC struct{}

func (testOC) Acquire(ctx context.Context, addr string) (overloadctrl.Token, error) {
	return nil, nil
}
