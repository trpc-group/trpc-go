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

package http

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValueDetachedContextScavenger(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &valueDetachedCtx{ctx: ctx}
	s := newScavenger()
	go s.scavengeValueDetachedCtx()
	require.True(t, s.collect(c))
	require.Eventually(t, func() bool {
		return len(s.cases) == 2 && len(s.ctxs) == 2 && s.ctxs[1] == c
	}, time.Second, time.Millisecond)

	cancel()
	require.Eventually(t, func() bool {
		return len(s.cases) == 1 && len(s.ctxs) == 1 && s.inprocess.Load() == 0
	}, time.Second, time.Millisecond)
}

func TestValueDetachedContextScavengerLimited(t *testing.T) {
	oldLimitedCases := limitedCases
	limitedCases = 10
	defer func() {
		limitedCases = oldLimitedCases
	}()

	s := newScavenger()
	go s.scavengeValueDetachedCtx()
	var cancels []context.CancelFunc
	var failed bool
	for i := uint32(0); i < limitedCases+20; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancels = append(cancels, cancel)
		if !s.collect(&valueDetachedCtx{ctx: ctx}) {
			failed = true
		}
	}
	require.True(t, failed)

	for _, cancel := range cancels {
		cancel()
	}
	require.Eventually(t, func() bool {
		return len(s.cases) == 1 && len(s.ctxs) == 1 && s.inprocess.Load() == 0
	}, time.Second, time.Millisecond)
}
