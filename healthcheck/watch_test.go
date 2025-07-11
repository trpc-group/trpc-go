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

package healthcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	serviceName := t.Name()
	require.Nil(t, watchers[serviceName])
	Watch(serviceName, func(status Status) {})
	require.NotNil(t, watchers[serviceName])
	delete(watchers, serviceName)
}

func TestGetWatchers(t *testing.T) {
	serviceName := t.Name()
	Watch(serviceName, func(status Status) {})
	ws := GetWatchers()
	require.NotNil(t, ws[serviceName])
	delete(ws, serviceName)
}
