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

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

var testNodes = []*registry.Node{
	{
		ServiceName: "testservice",
		Address:     "testservice.ip.1:16721",
		Network:     "tcp",
	},
}

type testDiscovery struct{}

// List 获取节点列表
func (d *testDiscovery) List(serviceName string, opt ...Option) ([]*registry.Node, error) {
	return testNodes, nil
}

func TestDiscoveryRegister(t *testing.T) {
	want := &testDiscovery{}
	Register("test-discovery", want)
	t.Cleanup(func() {
		unregister(t, "test-discovery")
	})
	require.Equal(t, want, Get("test-discovery"))
}

func TestDiscoveryGet(t *testing.T) {
	want := &testDiscovery{}
	Register("test-discovery", want)
	t.Cleanup(func() {
		unregister(t, "test-discovery")
	})
	require.Equal(t, want, Get("test-discovery"))
	require.Nil(t, Get("not_exist"))
}

func TestDiscoveryList(t *testing.T) {
	want := &testDiscovery{}
	Register("test-discovery", want)
	t.Cleanup(func() {
		unregister(t, "test-discovery")
	})
	d := Get("test-discovery")
	nodes, err := d.List("test-service", nil)
	assert.Nil(t, err)
	assert.Equal(t, testNodes, nodes)

}

func TestSetDefaultDiscovery(t *testing.T) {
	noop := &testDiscovery{}
	SetDefaultDiscovery(noop)
	assert.Equal(t, DefaultDiscovery, noop)
}

func unregister(t *testing.T, name string) {
	t.Helper()

	lock.Lock()
	delete(discoveries, name)
	lock.Unlock()
}
