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
	"testing"

	"trpc.group/trpc-go/trpc-go/overloadctrl"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	require.Nil(t, overloadctrl.GetClient("not_exist"))
	require.Nil(t, overloadctrl.GetServer("not_exist"))
	overloadctrl.RegisterClient("test_noop",
		func(info *overloadctrl.ServiceMethodInfo) overloadctrl.OverloadController {
			return overloadctrl.NoopOC{}
		})
	require.NotNil(t, overloadctrl.GetClient("test_noop"))
	overloadctrl.RegisterServer("test_noop",
		func(info *overloadctrl.ServiceMethodInfo) overloadctrl.OverloadController {
			return overloadctrl.NoopOC{}
		})
	require.NotNil(t, overloadctrl.GetServer("test_noop"))
}
