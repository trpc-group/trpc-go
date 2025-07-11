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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package multiplex

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateRWMutex(t *testing.T) {
	var mu stateRWMutex
	require.True(t, mu.rLock())
	mu.rUnlock()

	require.True(t, mu.lock())
	mu.closeLocked()
	mu.unlock()

	// Lock return false when mutex is already closed.
	require.False(t, mu.rLock())
	require.False(t, mu.lock())
}
