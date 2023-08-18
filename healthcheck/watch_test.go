// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package healthcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	require.Nil(t, watchers["testService"], "testService watcher")
	Watch("testService", func(status Status) {})
	require.NotNil(t, watchers["testService"], "testService watcher")
}

func TestGetWatchers(t *testing.T) {
	Watch("testService", func(status Status) {})
	ws := GetWatchers()
	require.NotNil(t, ws["testService"])
}
