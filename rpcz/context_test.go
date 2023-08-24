// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rpcz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpanFromContext(t *testing.T) {
	t.Run("empty context", func(t *testing.T) {
		require.Panicsf(t, func() { SpanFromContext(nil) }, "should panic because of nil pointer dereference")
	})
	t.Run("background context", func(t *testing.T) {
		require.Equal(t, GlobalRPCZ, SpanFromContext(context.Background()))
	})
	rootSpan := newSpan("root span", SpanID(1), nil)
	t.Run("context has root span", func(t *testing.T) {
		ctx := ContextWithSpan(context.Background(), rootSpan)
		require.Equal(t, rootSpan, SpanFromContext(ctx))
	})
	t.Run("context has child span", func(t *testing.T) {
		childSpan := newSpan("child span", SpanID(2), nil)
		ctx := ContextWithSpan(context.Background(), childSpan)
		require.Equal(t, childSpan, SpanFromContext(ctx))
	})
}

func TestCurrentSpanKey(t *testing.T) {
	type notSpanKey struct{}
	t.Run("key is not spanKey", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, notSpanKey{}, &span{})
		s := SpanFromContext(ctx)
		require.IsType(t, &RPCZ{}, s)
	})
	t.Run("key is spanKey", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, spanKey{}, &span{})
		s := SpanFromContext(ctx)
		require.IsType(t, &span{}, s)
	})
}

func TestNewSpanContext(t *testing.T) {
	t.Run("new noop span", func(t *testing.T) {
		ctx := context.Background()
		span, _, newCtx1 := NewSpanContext(ctx, "server")
		require.Equal(t, ctx, newCtx1)
		require.Equal(t, GlobalRPCZ, span)

		NewSpanContext(ctx, "filter")
		span, _, newCtx2 := NewSpanContext(newCtx1, "server")
		require.Equal(t, newCtx1, newCtx2)
		require.Equal(t, GlobalRPCZ, span)
	})

	t.Run("new *span", func(t *testing.T) {
		ctx := ContextWithSpan(context.Background(), &span{})
		s, _, newCtx1 := NewSpanContext(ctx, "server")
		require.NotEqual(t, ctx, newCtx1)
		require.NotEqual(t, noopSpan{}, s)

		NewSpanContext(ctx, "filter")
		s, _, newCtx2 := NewSpanContext(newCtx1, "server")
		require.NotEqual(t, newCtx1, newCtx2)
		require.NotEqual(t, noopSpan{}, s)
	})
}
