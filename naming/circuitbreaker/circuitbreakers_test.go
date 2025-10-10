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

package circuitbreaker_test

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	. "trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
)

func TestCircuitBreakers_ErrRateToOpen(t *testing.T) {
	cb := NewLRUCircuitBreakers(
		WithErrRateToOpen(0.5),
		WithMinRequestsToOpen(1))
	require.True(t, cb.Available("a"))
	repeat(2, func() { cb.Report("a", false) })
	repeat(3, func() { cb.Report("a", true) })
	require.True(t, cb.Available("a"))
	cb.Report("a", false)
	require.False(t, cb.Available("a"))
}

func TestCircuitBreakers_ContinuousFailuresToOpen(t *testing.T) {
	windowInterval := time.Millisecond * 200
	newCB := func(opts ...Opt) *LRUCircuitBreakers {
		return NewLRUCircuitBreakers(append([]Opt{
			WithErrRateToOpen(1.1), // a value over 1.0 means disabled
			WithContinuousFailuresToOpen(2),
			WithSlidingWindowInterval(windowInterval),
			WithSlidingWindowSize(2),
			WithMinRequestsToOpen(1)},
			opts...)...)
	}
	t.Run("continuous failures opens circuit breaker", func(t *testing.T) {
		cb := newCB()
		require.True(t, cb.Available("a"))
		cb.Report("a", false)
		require.True(t, cb.Available("a"))
		cb.Report("a", false)
		require.False(t, cb.Available("a"))
	})
	t.Run("long ago failures are not ignored", func(t *testing.T) {
		cb := newCB()
		require.True(t, cb.Available("a"))
		cb.Report("a", false)
		require.True(t, cb.Available("a"))
		time.Sleep(windowInterval * 2)
		cb.Report("a", false)
		require.False(t, cb.Available("a"))
	})
	t.Run("very long ago failures does be ignored", func(t *testing.T) {
		cb := newCB(WithOpenDuration(0))
		require.True(t, cb.Available("a"))
		cb.Report("a", false)
		require.True(t, cb.Available("a"))
		time.Sleep(windowInterval * 3)
		require.True(t, cb.Available("b")) // to trigger idle GC on "a"
		cb.Report("a", false)
		require.True(t, cb.Available("a")) // still available even 2 continuous errors
	})
}

func TestCircuitBreakers_MinRequestsToOpen(t *testing.T) {
	cb := NewLRUCircuitBreakers(WithMinRequestsToOpen(10))
	repeat(9, func() { cb.Report("a", false) })
	require.True(t, cb.Available("a"))
	cb.Report("a", false)
	require.False(t, cb.Available("a"))
}

func TestCircuitBreaker_OpenDuration(t *testing.T) {
	openDuration := time.Millisecond * 200
	cb := NewLRUCircuitBreakers(
		WithOpenDuration(openDuration),
		WithMinRequestsToOpen(3))
	repeat(3, func() { cb.Report("a", false) })
	require.False(t, cb.Available("a"))
	time.Sleep(openDuration / 2)
	repeat(6, func() { cb.Report("a", true) })
	require.False(t, cb.Available("a"))
	time.Sleep(openDuration / 2)
	require.True(t, cb.Available("a"))
}

func TestCircuitBreaker_HalfOpenedToClosed(t *testing.T) {
	cb := NewLRUCircuitBreakers(
		WithOpenDuration(0),
		WithMinRequestsToOpen(3),
		WithTotalRequestsToClose(5),
		WithSuccessRequestsToClose(4)) // closed
	repeat(3, func() { cb.Report("a", false) })
	require.False(t, cb.Available("a"))                      // opened
	repeat(4, func() { require.True(t, cb.Available("a")) }) // halfOpened
	cb.Report("a", true)
	require.True(t, cb.Available("a"))  // last available count
	require.False(t, cb.Available("a")) // still halfOpened
	repeat(2, func() { cb.Report("a", true) })
	require.False(t, cb.Available("a"))
	cb.Report("a", true)
	require.True(t, cb.Available("a")) // closed
	require.True(t, cb.Available("a")) // still closed
}

func TestCircuitBreaker_HalfOpenedToOpened(t *testing.T) {
	cb := NewLRUCircuitBreakers(
		WithOpenDuration(0),
		WithMinRequestsToOpen(3),
		WithTotalRequestsToClose(5),
		WithSuccessRequestsToClose(4)) // closed
	repeat(3, func() { cb.Report("a", false) })
	require.False(t, cb.Available("a"))                      // opened
	repeat(3, func() { require.True(t, cb.Available("a")) }) // halfOpened
	cb.Report("a", false)
	require.True(t, cb.Available("a")) // still halfOpened
	cb.Report("a", false)
	require.False(t, cb.Available("a"))                      // opened
	repeat(5, func() { require.True(t, cb.Available("a")) }) // halfOpened
	require.False(t, cb.Available("a"))
}

func TestCircuitBreaker_ErrRateIsLimitedWithinWindowInterval(t *testing.T) {
	cb := NewLRUCircuitBreakers(
		WithMinRequestsToOpen(1),
		WithErrRateToOpen(0.5),
		WithContinuousFailuresToOpen(math.MaxInt),
		WithSlidingWindowSize(2),
		WithSlidingWindowInterval(time.Millisecond*200))
	repeat(5, func() { cb.Report("a", false) })
	time.Sleep(time.Millisecond * 200) // 5 failures are rolled out
	repeat(4, func() { cb.Report("a", true) })
	require.True(t, cb.Available("a"))
}

func TestCircuitBreaker_IdleGCed(t *testing.T) {
	cb := NewLRUCircuitBreakers(
		WithOpenDuration(0),
		WithMinRequestsToOpen(1),
		WithTotalRequestsToClose(3),
		WithSlidingWindowInterval(time.Millisecond*200))
	repeat(2, func() { cb.Report("a", false) })
	require.False(t, cb.Available("a"))
	repeat(3, func() { require.True(t, cb.Available("a")) })
	require.False(t, cb.Available("a"))
	time.Sleep(time.Millisecond * 200 * 4)
	require.True(t, cb.Available("b"))
	require.True(t, cb.Available("a"))
}

func TestCircuitBreaker_IndividualAddr(t *testing.T) {
	cb := NewLRUCircuitBreakers(
		WithOpenDuration(0),
		WithMinRequestsToOpen(2),
		WithTotalRequestsToClose(2))
	require.True(t, cb.Available("a"))
	require.True(t, cb.Available("b"))
	repeat(2, func() { cb.Report("a", false) })
	require.False(t, cb.Available("a"))
	repeat(2, func() { require.True(t, cb.Available("a")) })
	repeat(2, func() { require.False(t, cb.Available("a")) })
	repeat(2, func() { require.True(t, cb.Available("b")) })
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewLRUCircuitBreakers(
		WithOpenDuration(0),
		WithMinRequestsToOpen(10),
		WithSuccessRequestsToClose(12),
		WithSlidingWindowInterval(time.Millisecond*200))
	concurrentRepeat(9, func() {
		cb.Report("a", false)
	})
	require.True(t, cb.Available("a"))
	cb.Report("a", false)

	var availables int32
	concurrentRepeat(12, func() {
		if cb.Available("a") {
			atomic.AddInt32(&availables, 1)
		}
	})
	// the one not available is opened stat.
	require.Equal(t, 11, int(availables))

	availables = 0
	concurrentRepeat(5, func() {
		if cb.Available("a") {
			atomic.AddInt32(&availables, 1)
		}
	})
	// there remains only one available for 12 total requests to close.
	require.Equal(t, 1, int(availables))
}

func repeat(n int, f func()) {
	for i := 0; i < n; i++ {
		f()
	}
}

func concurrentRepeat(n int, f func()) {
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			f()
			wg.Done()
		}()
	}
	wg.Wait()
}
