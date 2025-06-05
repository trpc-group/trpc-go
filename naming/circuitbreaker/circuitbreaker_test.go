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

package circuitbreaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

type testCircuitBreaker struct{}

// Available determines whether the circuit breaker is available.
func (cb *testCircuitBreaker) Available(node *registry.Node) bool {
	return true
}

// Report reports the result.
func (cb *testCircuitBreaker) Report(node *registry.Node, cost time.Duration, err error) error {
	return nil
}

func unregister(t *testing.T, name string) {
	t.Helper()

	lock.Lock()
	delete(circuitbreakers, name)
	lock.Unlock()
}

func TestCircuitBreakerRegister(t *testing.T) {
	want := &testCircuitBreaker{}
	Register("cb", want)
	t.Cleanup(func() {
		unregister(t, "cb")
	})
	require.Equal(t, want, Get("cb"))
}

func TestCircuitBreakerGet(t *testing.T) {
	want := &testCircuitBreaker{}
	Register("cb", &testCircuitBreaker{})
	t.Cleanup(func() {
		unregister(t, "cb")
	})
	require.Equal(t, want, Get("cb"))
	require.Nil(t, Get("not_exist"))
}

func TestNoopCircuitBreaker(t *testing.T) {
	noop := &NoopCircuitBreaker{}
	assert.True(t, noop.Available(nil))
	assert.Nil(t, noop.Report(nil, 0, nil))
}

func TestSetDefaultCircuitBreaker(t *testing.T) {
	noop := &NoopCircuitBreaker{}
	SetDefaultCircuitBreaker(noop)
	assert.Equal(t, DefaultCircuitBreaker, noop)
}
