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

package circuitbreaker

import (
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/registry"

	"github.com/stretchr/testify/assert"
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

func TestCircuitBreakerRegister(t *testing.T) {
	Register("cb", &testCircuitBreaker{})
	assert.NotNil(t, Get("cb"))
	unregisterForTesting("cb")
}

func TestCircuitBreakerGet(t *testing.T) {
	Register("cb", &testCircuitBreaker{})
	assert.NotNil(t, Get("cb"))
	unregisterForTesting("cb")
	assert.Nil(t, Get("not_exist"))
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
