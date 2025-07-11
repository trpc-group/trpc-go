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

package filter_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/rpcz"
)

func TestFilterChain(t *testing.T) {
	ctx := context.Background()
	req, rsp := "req", "rsp"
	sc := filter.ServerChain{filter.NoopServerFilter}
	_, err := sc.Filter(ctx, req,
		func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
			return nil, nil
		})
	require.Nil(t, err)
	cc := filter.ClientChain{filter.NoopClientFilter}
	require.Nil(t, cc.Filter(ctx, req, rsp,
		func(ctx context.Context, req, rsp interface{}) error {
			return nil
		}))
}

func TestNamedFilter(t *testing.T) {
	const filterName = "filterName"
	filter.Register(filterName, filter.NoopServerFilter, filter.NoopClientFilter)
	require.NotNil(t, filter.GetClient(filterName))
	require.NotNil(t, filter.GetServer(filterName))
	ctx := context.Background()
	span, end := rpcz.NewRPCZ(&rpcz.Config{Fraction: 1, Capacity: 1}).NewChild("child")
	defer end.End()
	ctx = rpcz.ContextWithSpan(ctx, span)
	span.SetAttribute(rpcz.TRPCAttributeFilterNames, []string{filterName})
	cc := filter.ClientChain{filter.NoopClientFilter}
	require.Nil(t, cc.Filter(ctx, nil, nil,
		func(ctx context.Context, req, rsp interface{}) error { return nil }))
	sc := filter.ServerChain{filter.NoopServerFilter}
	_, err := sc.Filter(ctx, nil,
		func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil })
	require.Nil(t, err)
}

func TestChainConcurrentHandle(t *testing.T) {
	const concurrentN = 4
	var calledTimes [concurrentN]int32
	cc := filter.ClientChain{
		func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
			atomic.AddInt32(&calledTimes[0], 1)
			return f(ctx, req, rsp)
		},
		func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
			atomic.AddInt32(&calledTimes[1], 1)
			var eg errgroup.Group
			for i := 0; i < concurrentN; i++ {
				eg.Go(func() error {
					return f(ctx, req, rsp)
				})
			}
			return eg.Wait()
		},
		func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) (err error) {
			atomic.AddInt32(&calledTimes[2], 1)
			return f(ctx, req, rsp)
		},
		func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) (err error) {
			atomic.AddInt32(&calledTimes[3], 1)
			return f(ctx, req, rsp)
		},
	}
	require.Nil(t, cc.Filter(context.Background(), nil, nil,
		func(ctx context.Context, req, rsp interface{}) (err error) {
			return nil
		}))
	require.Equal(t, int32(1), atomic.LoadInt32(&calledTimes[0]))
	require.Equal(t, int32(1), atomic.LoadInt32(&calledTimes[1]))
	require.Equal(t, int32(concurrentN), atomic.LoadInt32(&calledTimes[2]))
	require.Equal(t, int32(concurrentN), atomic.LoadInt32(&calledTimes[3]))
}
