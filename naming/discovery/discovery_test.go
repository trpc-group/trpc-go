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

package discovery

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/naming/registry"

	"github.com/stretchr/testify/assert"
)

var testNode *registry.Node = &registry.Node{
	ServiceName: "testservice",
	Address:     "testservice.ip.1:16721",
	Network:     "tcp",
}

type testDiscovery struct{}

// List 获取节点列表
func (d *testDiscovery) List(serviceName string, opt ...Option) ([]*registry.Node, error) {
	return []*registry.Node{testNode}, nil
}

func TestDiscoveryRegister(t *testing.T) {
	Register("test-discovery", &testDiscovery{})
	assert.NotNil(t, Get("test-discovery"))
	unregisterForTesting("test-discovery")
}

func TestDiscoveryGet(t *testing.T) {
	Register("test-discovery", &testDiscovery{})
	assert.NotNil(t, Get("test-discovery"))
	unregisterForTesting("test-discovery")
	assert.Nil(t, Get("not_exist"))
}

func TestDiscoveryList(t *testing.T) {
	Register("test-discovery", &testDiscovery{})
	d := Get("test-discovery")
	list, err := d.List("test-service", nil)
	assert.Nil(t, err)
	assert.Equal(t, list[0], testNode)
	unregisterForTesting("test-discovery")
}

func TestSetDefaultDiscovery(t *testing.T) {
	noop := &testDiscovery{}
	SetDefaultDiscovery(noop)
	assert.Equal(t, DefaultDiscovery, noop)
}
