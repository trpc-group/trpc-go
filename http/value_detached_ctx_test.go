package http

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ctxKey struct{}

func TestValueDetachedCtxDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	detachedCtx := detachCtxValue(ctx)
	deadline, ok := ctx.Deadline()
	newDeadline, newOk := detachedCtx.Deadline()
	require.Equal(t, ok, newOk)
	require.Equal(t, deadline, newDeadline)
	<-ctx.Done()
	newDeadline, newOk = detachedCtx.Deadline()
	require.Equal(t, ok, newOk, "deadline should not change after ctx done")
	require.Equal(t, deadline, newDeadline, "deadline should not change after ctx done")
}

func TestValueDetachedCtxDone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	ctx = detachCtxValue(context.WithValue(ctx, ctxKey{}, struct{}{}))
	require.Nil(t, ctx.Err(), "should not timed out yet")
	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	require.NotEqual(t, deadline, time.Time{})
	<-ctx.Done()
	// Sleep a while to wait ctx done propagate to detached ctx.
	time.Sleep(time.Millisecond * 20)
	require.NotNil(t, ctx.Err(), "ctx has timed out")
}

func TestValueDetachedCtxValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = context.WithValue(ctx, ctxKey{}, struct{}{})
	ctx = detachCtxValue(ctx)
	require.Equal(t, ctx.Value(ctxKey{}), nil)
}

func TestValueDetachedCtxGC(t *testing.T) {
	newValueCtx := func() (ctx context.Context, valGCed func() bool) {
		var i int
		iGCed := make(chan struct{})
		runtime.SetFinalizer(&i, func(*int) {
			close(iGCed)
		})
		return context.WithValue(context.Background(), ctxKey{}, &i), func() bool {
			// Wait a while before finalizer could run.
			select {
			case <-time.After(time.Millisecond * 100):
				return false
			case <-iGCed:
				return true
			}
		}
	}

	ctx, valGCed := newValueCtx()
	ctx = detachCtxValue(ctx)

	// The original ctx is only swept in second GC circle due to go's tri-color GC algorithm.
	runtime.GC()
	runtime.GC()
	require.True(t, valGCed(), "allocated val should be GCed")
}

func TestValueDetachedCtxGCCancelableCtx(t *testing.T) {
	newValueCtx := func() (ctx context.Context, cancel func(), valGCed func() bool) {
		var i int
		iGCed := make(chan struct{})
		runtime.SetFinalizer(&i, func(*int) {
			close(iGCed)
		})
		ctx, cancel = context.WithCancel(context.Background())
		return context.WithValue(ctx, ctxKey{}, &i),
			cancel,
			func() bool {
				// Wait a while before finalizer could run.
				select {
				case <-time.After(time.Millisecond * 100):
					return false
				case <-iGCed:
					return true
				}
			}
	}

	ctx, cancel, valGCed := newValueCtx()
	ctx = detachCtxValue(ctx)

	// The original ctx is not swept before second GC circle due to go's tri-color GC algorithm.
	runtime.GC()
	runtime.GC()
	require.False(t, valGCed(), "allocated resource can not be GCed before cancel original ctx")

	cancel()
	runtime.GC()
	runtime.GC()
	require.True(t, valGCed(), "allocated resource should be GCed after cancel original ctx")
}
