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

package loadbalance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

var testNode = &registry.Node{
	ServiceName: "testservice",
	Address:     "loadbalance.ip.1:16721",
	Network:     "tcp",
}

type testLoadBalance struct{}

// Select acquires a node.
func (tlb *testLoadBalance) Select(serviceName string, list []*registry.Node, opt ...Option) (*registry.Node, error) {
	return testNode, nil
}

func TestLoadBalanceRegister(t *testing.T) {
	want := &testLoadBalance{}
	Register("tlb", want)
	t.Cleanup(func() {
		unregister(t, "tlb")
	})
	require.Equal(t, want, Get("tlb"))
}

func TestLoadBalanceGet(t *testing.T) {
	want := &testLoadBalance{}
	Register("tlb", &testLoadBalance{})
	t.Cleanup(func() {
		unregister(t, "tlb")
	})
	require.Equal(t, want, Get("tlb"))
	require.Nil(t, Get("not_exist"))
}

func TestLoadBalanceSelect(t *testing.T) {
	Register("tlb", &testLoadBalance{})
	t.Cleanup(func() {
		unregister(t, "tlb")
	})
	lb := Get("tlb")
	node, err := lb.Select("test-service", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, testNode, node)
}

func TestSetDefaultLoadBalancer(t *testing.T) {
	noop := &testLoadBalance{}
	SetDefaultLoadBalancer(noop)
	assert.Equal(t, DefaultLoadBalancer, noop)
}

func unregister(t *testing.T, name string) {
	t.Helper()

	lock.Lock()
	delete(loadbalancers, name)
	lock.Unlock()
}
