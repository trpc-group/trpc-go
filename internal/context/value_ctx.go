// Package context provides extensions to context.Context.
package context

import (
	"context"
)

// NewContextWithValues will use the valuesCtx's Value function.
// Effects of the returned context:
//
//	Whether it has timed out or canceled: decided by ctx.
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
