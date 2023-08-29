// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package http

import (
	"context"
	"sync"
	"time"
)

// valueDetachedCtx removes all values associated with ctx while
// ensuring the transitivity of ctx timeout/cancel.
// After the original ctx timeout/cancel, valueDetachedCtx must release
// the original ctx to ensure that the resources associated with
// original ctx can be GC normally.
type valueDetachedCtx struct {
	mu  sync.Mutex
	ctx context.Context
}

// detachCtxValue creates a new valueDetachedCtx from ctx.
func detachCtxValue(ctx context.Context) context.Context {
	if ctx.Done() == nil {
		return context.Background()
	}
	c := valueDetachedCtx{ctx: ctx}
	go func() {
		<-ctx.Done()
		deadline, ok := ctx.Deadline()
		c.mu.Lock()
		c.ctx = &ctxRemnant{
			deadline:    deadline,
			hasDeadline: ok,
			err:         ctx.Err(),
			done:        ctx.Done(),
		}
		c.mu.Unlock()
	}()
	return &c
}

// Deadline implements the Deadline method of Context.
func (c *valueDetachedCtx) Deadline() (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ctx.Deadline()
}

// Done implements Done method of Context.
func (c *valueDetachedCtx) Done() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ctx.Done()
}

// Err implements Err method of Context.
func (c *valueDetachedCtx) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ctx.Err()
}

// Value always returns nil.
func (c *valueDetachedCtx) Value(_ interface{}) interface{} {
	return nil
}

// ctxRemnant is the remnant of valueDetachedCtx after timeout/cancel,
// retains some information of the original ctx, ensure that the original ctx
// can be GC normally.
type ctxRemnant struct {
	deadline    time.Time
	hasDeadline bool
	err         error
	done        <-chan struct{}
}

// Deadline returns the saved readline information.
func (c *ctxRemnant) Deadline() (time.Time, bool) {
	return c.deadline, c.hasDeadline
}

// Done returns saved Done channel.
func (c *ctxRemnant) Done() <-chan struct{} {
	return c.done
}

// Err returns saved error.
func (c *ctxRemnant) Err() error {
	return c.err
}

// Value always returns nil.
func (c *ctxRemnant) Value(_ interface{}) interface{} {
	return nil
}
