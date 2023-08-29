// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rpcz

import "context"

type (
	spanKey    struct{}
	notSpanKey struct{}
)

// SpanFromContext returns Span from ctx.
// If no Span isn't currently set in ctx, returns GlobalRPCZ.
func SpanFromContext(ctx context.Context) Span {
	if s, ok := ctx.Value(spanKey{}).(Span); ok {
		return s
	}
	return GlobalRPCZ
}

// ContextWithSpan returns a copy of parent with span set as the current Span.
func ContextWithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, spanKey{}, span)
}

// NewSpanContext creates a new span and binds it to a new ctx by combining SpanFromContext, Span.NewChildSpan and
// ContextWithSpan. It returns the original ctx if new span is the same as original one to avoid unnecessary valueCtx.
// Ender used to end this span immediately，and doesn't wait for related work to stop.
// Ender doesn't end context.Context.
func NewSpanContext(ctx context.Context, name string) (Span, Ender, context.Context) {
	span := SpanFromContext(ctx)
	newSpan, end := span.NewChild(name)
	if newSpan == span {
		return newSpan, end, ctx
	}
	return newSpan, end, ContextWithSpan(ctx, newSpan)
}
