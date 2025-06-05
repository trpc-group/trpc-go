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

package overloadctrl_test

import (
	"context"
	"strings"
	"testing"

	"trpc.group/trpc-go/trpc-go/overloadctrl"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestImpl(t *testing.T) {
	ctx := context.Background()
	t.Run("empty", func(t *testing.T) {
		var impl overloadctrl.Impl
		require.Nil(t, yaml.Unmarshal([]byte(``), &impl))
		require.Nil(t, impl.Build(overloadctrl.GetClient, &overloadctrl.ServiceMethodInfo{
			ServiceName: "test",
			MethodName:  overloadctrl.AnyMethod,
		}))
		token, err := impl.Acquire(ctx, "")
		require.Nil(t, err)
		require.Equal(t, overloadctrl.NoopToken{}, token)
	})
	t.Run("not found", func(t *testing.T) {
		var impl overloadctrl.Impl
		require.Nil(t, yaml.Unmarshal([]byte(`
not_exist
`), &impl))
		require.NotNil(t, impl.Build(overloadctrl.GetClient, &overloadctrl.ServiceMethodInfo{
			ServiceName: "test",
			MethodName:  overloadctrl.AnyMethod,
		}))
	})

	testClientOC := overloadctrl.NoopOC{}
	overloadctrl.RegisterClient("test_client_oc",
		func(*overloadctrl.ServiceMethodInfo) overloadctrl.OverloadController {
			return testClientOC
		})
	t.Run("ok", func(t *testing.T) {
		var impl overloadctrl.Impl
		require.Nil(t, yaml.Unmarshal([]byte(`
test_client_oc`), &impl))
		require.Nil(t, impl.Build(overloadctrl.GetClient, &overloadctrl.ServiceMethodInfo{
			ServiceName: "test",
			MethodName:  overloadctrl.AnyMethod,
		}))
		require.Equal(t, testClientOC, impl.OverloadController)
		token, err := impl.Acquire(ctx, "")
		require.Nil(t, err)
		require.Equal(t, overloadctrl.NoopToken{}, token)
	})
	t.Run("marshal_unmarshal", func(t *testing.T) {
		name := "test_client_oc"
		var impl overloadctrl.Impl
		require.Nil(t, yaml.Unmarshal([]byte(name), &impl))
		data, err := yaml.Marshal(&impl)
		require.Nil(t, err)
		require.Equal(t, name, strings.TrimRightFunc(string(data), func(r rune) bool {
			return r == '\n'
		}))
	})
}
