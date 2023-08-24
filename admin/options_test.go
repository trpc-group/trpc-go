// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithSkipServe(t *testing.T) {
	opts := []Option{
		WithVersion(testVersion),
		WithAddr(defaultListenAddr),
		WithTLS(false),
		WithReadTimeout(defaultReadTimeout),
		WithWriteTimeout(defaultWriteTimeout),
		WithConfigPath(testConfigPath),
	}
	t.Run("enable SkipServe option", func(t *testing.T) {
		require.True(t, NewServer(append(opts, WithSkipServe(true))...).config.skipServe)
	})
	t.Run("disable SkipServe option", func(t *testing.T) {
		require.False(t, NewServer(append(opts, WithSkipServe(false))...).config.skipServe)
	})
}
