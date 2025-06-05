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

// Package context provides extensions to context.Context.
package context

import (
	"context"
)

// NewContextWithValues will use the valuesCtx's Value function.
// Effects of the returned context:
//
//	Whether has timed out or canceled: decided by ctx.
//	Retrieve value using key: first use valuesCtx.Value, then ctx.Value.
func NewContextWithValues(ctx, valuesCtx context.Context) context.Context {
	return &valueCtx{Context: ctx, values: valuesCtx}
}

type valueCtx struct {
	context.Context
	values context.Context
}

// Value re-implements context.Value, valueCtx.values.Value has the highest
// priority.
func (c *valueCtx) Value(key interface{}) interface{} {
	if v := c.values.Value(key); v != nil {
		return v
	}
	return c.Context.Value(key)
}
