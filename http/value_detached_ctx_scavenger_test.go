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

package http

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValueDetachedContextScavenger(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &valueDetachedCtx{
		ctx: ctx,
	}
	s := newScavenger()
	go s.scavengeValueDetachedCtx()
	require.Equal(t, 1, len(s.cases))
	require.Equal(t, 1, len(s.ctxs))
	s.collect(c)
	require.Eventually(t, func() bool {
		return len(s.cases) == 2 && len(s.ctxs) == 2 && s.ctxs[1] == c
	}, time.Second, time.Millisecond)
	cancel()
	require.Eventually(t, func() bool {
		return len(s.cases) == 1 && len(s.ctxs) == 1
	}, time.Second, time.Millisecond)
}

func TestValueDetachedContextScavengerMultiple(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	c1 := &valueDetachedCtx{
		ctx: ctx1,
	}
	s := newScavenger()
	go s.scavengeValueDetachedCtx()
	require.Eventually(t, func() bool {
		return len(s.cases) == 1 && len(s.ctxs) == 1
	}, time.Second, time.Millisecond)
	s.collect(c1)
	require.Eventually(t, func() bool {
		return len(s.cases) == 2 && len(s.ctxs) == 2
	}, time.Second, time.Millisecond)
	ctx2, cancel2 := context.WithCancel(context.Background())
	c2 := &valueDetachedCtx{
		ctx: ctx2,
	}
	s.collect(c2)
	require.Eventually(t, func() bool {
		return len(s.cases) == 3 && len(s.ctxs) == 3
	}, time.Second, time.Millisecond)
	cancel1()
	require.Eventually(t, func() bool {
		return len(s.cases) == 2 && len(s.ctxs) == 2
	}, time.Second, time.Millisecond)
	cancel2()
	require.Eventually(t, func() bool {
		return len(s.cases) == 1 && len(s.ctxs) == 1
	}, time.Second, time.Millisecond)
}

func TestValueDetachedContextScavengerShrinkCapacity(t *testing.T) {
	// Create a scavenger and start its goroutine.
	s := newScavenger()
	go s.scavengeValueDetachedCtx()

	// Wait for initial setup.
	require.Eventually(t, func() bool {
		return len(s.cases) == 1 && len(s.ctxs) == 1
	}, time.Second, time.Millisecond)

	// Create 2000 contexts to exceed minSelectCasesCap (1024).
	var cancels []context.CancelFunc
	for i := 0; i < 2000; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancels = append(cancels, cancel)
		c := &valueDetachedCtx{
			ctx: ctx,
		}
		s.collect(c)
	}

	// Wait for all contexts to be collected.
	require.Eventually(t, func() bool {
		return len(s.cases) == 2001 && len(s.ctxs) == 2001
	}, time.Second, time.Millisecond)

	// Record the capacity.
	originalCap := cap(s.cases)

	// Cancel 1500 contexts to trigger capacity shrinking.
	for i := 0; i < 1500; i++ {
		cancels[i]()
	}

	// Wait for capacity to shrink.
	require.Eventually(t, func() bool {
		currentCap := cap(s.cases)
		return currentCap < originalCap && currentCap >= 1024
	}, time.Second, time.Millisecond)

	// Clean up remaining contexts.
	for i := 1500; i < 2000; i++ {
		cancels[i]()
	}
}

func TestValueDetachedContextScavengerLimited(t *testing.T) {
	oldLimitedCases := limitedCases
	limitedCases = 10
	defer func() {
		limitedCases = oldLimitedCases
	}()
	// Create a scavenger and start its goroutine.
	s := newScavenger()
	go s.scavengeValueDetachedCtx()

	// Wait for initial setup.
	require.Eventually(t, func() bool {
		return len(s.cases) == 1 && len(s.ctxs) == 1
	}, time.Second, time.Millisecond)

	// Create contexts slightly more than limitedCases to test the limit.
	var cancels []context.CancelFunc
	var successCount int
	var failCount int

	// Create contexts in batches to avoid timeout.
	for i := uint32(0); i < limitedCases+100; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancels = append(cancels, cancel)
		c := &valueDetachedCtx{
			ctx: ctx,
		}
		// Count successful and failed collections.
		if s.collect(c) {
			successCount++
		} else {
			failCount++
		}
	}

	// Verify that we collected approximately limitedCases contexts.
	// The exact number might be slightly less due to concurrent processing.
	require.True(t, successCount <= int(limitedCases),
		"Should not collect more than limitedCases contexts, got %d.", successCount)
	require.True(t, failCount > 0,
		"Should have some failed collections when exceeding limit, got %d failures.", failCount)

	// Try to collect one more context, it should return false.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &valueDetachedCtx{
		ctx: ctx,
	}
	require.False(t, s.collect(c),
		"Should return false when trying to collect beyond limit.")

	// Clean up all contexts.
	for _, cancel := range cancels {
		cancel()
	}

	// Wait for cleanup to complete.
	require.Eventually(t, func() bool {
		return len(s.cases) == 1 && len(s.ctxs) == 1 && s.inprocess.Load() == 0
	}, 5*time.Second, time.Millisecond,
		"Should clean up all contexts, but got %d in-process.", s.inprocess.Load())
}

func TestValueDetachedContextScavengerLimited2(t *testing.T) {
	oldLimitedCases := limitedCases
	limitedCases = 10
	defer func() {
		limitedCases = oldLimitedCases
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < 1000; i++ {
		detachCtxValue(ctx)
	}
}
