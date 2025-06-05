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

package trpc

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
)

// TestCloneContext test CloneContext
func TestCloneContext(t *testing.T) {
	calleeMethod := "1"
	// add msg
	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithCalleeMethod(calleeMethod)
	// add timeout
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	// add custom key value
	type ctxKey struct{}
	ctx = context.WithValue(ctx, ctxKey{}, "value")
	clonedCtx := CloneContext(ctx)
	updateCalleeMethod := "2"
	codec.Message(clonedCtx).WithCalleeMethod(updateCalleeMethod)
	// cancel
	cancel()
	// test msg cloning
	assert.Equal(t, codec.Message(ctx).CalleeMethod(), calleeMethod)
	assert.Equal(t, codec.Message(clonedCtx).CalleeMethod(), updateCalleeMethod)
	// check timeout
	assert.Equal(t, ctx.Err(), context.Canceled)
	assert.Nil(t, clonedCtx.Err())
	// check getting kv
	assert.Equal(t, ctx.Value(ctxKey{}).(string), "value")
	assert.Equal(t, clonedCtx.Value(ctxKey{}).(string), "value")
}

func TestGetMetaData(t *testing.T) {
	type args struct {
		ctx context.Context
		key string
	}
	ctx, msg := codec.WithNewMessage(context.Background())
	md := make(map[string][]byte)
	md["testKey"] = []byte("testValue")
	msg.WithServerMetaData(md)
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "test case",
			args: args{
				ctx: ctx,
				key: "testKey",
			},
			want: []byte("testValue"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMetaData(tt.args.ctx, tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMetaData() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetIP test GetIP.
func TestGetIP(t *testing.T) {
	nicName := []string{"en1", "utun0"}
	for _, name := range nicName {
		got := GetIP(name)
		t.Logf("get ip by name: %v, ip: %v",
			name, got)
		assert.LessOrEqual(t, 0, len(got))
	}

	// Test None Existed NIC
	NoneExistNIC := "ethNoneExist"
	ip := GetIP(NoneExistNIC)
	assert.Empty(t, ip)
}
func TestGoAndWait(t *testing.T) {
	err := GoAndWait(
		func() error {
			return nil
		},
		func() error {
			return errors.New("go and wait3 test error")
		},
	)
	assert.NotNil(t, err)
}

func TestGo(t *testing.T) {
	timeout := time.Millisecond * 100
	t.Run("default go is async", func(t *testing.T) {
		start, done := make(chan struct{}), make(chan struct{})
		require.Nil(t, Go(context.Background(), timeout, func(context.Context) {
			done <- <-start
		}))
		select {
		case <-done:
			t.Error("should not done")
		default:
		}
		start <- struct{}{}
		<-done
	})
	t.Run("syncGo", func(t *testing.T) {
		var cost time.Duration
		start := time.Now()
		require.Nil(t, NewSyncGoer().Go(context.Background(), timeout, func(ctx context.Context) {
			<-ctx.Done()
			cost = time.Since(start)
		}))
		require.Greater(t, cost, timeout)
	})
	t.Run("async timeout", func(t *testing.T) {
		done := make(chan struct{})
		start := time.Now()
		require.Nil(t, NewAsyncGoer(0, 1024, false).
			Go(context.Background(), timeout, func(ctx context.Context) {
				done <- <-ctx.Done()
			}))
		select {
		case <-time.After(timeout * 2):
			t.Error("async does not timeout as expected")
		case <-done:
			require.Greater(t, time.Since(start), timeout)
		}
	})
	t.Run("async panic recover", func(t *testing.T) {
		require.Nil(t, NewAsyncGoer(0, 1024, true).
			Go(context.Background(), timeout, func(ctx context.Context) {
				panic(t.Name())
			}))
		// We must sleep a while to wait async panic happen.
		time.Sleep(timeout * 2)
	})
}

func TestGoAndWaitWithPanic(t *testing.T) {
	err := GoAndWait(
		func() error {
			return nil
		},
		func() error {
			panic("go and wait2 test panic")
		},
	)
	assert.NotNil(t, err)
}

// TestNetInterfaceIPEnumAllIP
func TestNetInterfaceIPEnumAllIP(t *testing.T) {
	p := &netInterfaceIP{}
	ips := p.enumAllIP()
	for k, v := range ips {
		t.Logf("enum ips nic: %s, ips: %+v",
			k, *v)
	}
	assert.LessOrEqual(t, 0, len(ips))
}

// TestNetInterfaceIPGetIPByNic
func TestNetInterfaceIPGetIPByNic(t *testing.T) {
	p := &netInterfaceIP{}
	got := p.getIPByNic("en1")
	t.Logf("get ip by nic name en1 ip: %v", got)
	assert.LessOrEqual(t, 0, len(got))
	got = p.getIPByNic("utun0")
	t.Logf("get ip by nic name utun0 ip: %v", got)
	assert.LessOrEqual(t, 0, len(got))
}

// TestDeduplicate
func TestDeduplicate(t *testing.T) {
	a := []string{"a1", "a2"}
	b := []string{"b1", "b2", "a2"}
	r := Deduplicate(a, b)
	assert.Equal(t, r, []string{"a1", "a2", "b1", "b2"})
}
