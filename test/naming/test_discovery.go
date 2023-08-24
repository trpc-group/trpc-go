// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package naming

import (
	"fmt"
	"sync"

	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

func init() {
	discovery.Register("test", td)
}

var td = &testDiscovery{
	nodesInfo: make(map[string][]*registry.Node),
}

type testDiscovery struct {
	nodesInfo map[string][]*registry.Node
	mu        sync.RWMutex
}

// List return a registry.Node from td.nodesInfo.
func (d *testDiscovery) List(serviceName string, opt ...discovery.Option) (nodes []*registry.Node, err error) {
	d.mu.RLock()
	nodes, ok := d.nodesInfo[serviceName]
	d.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("can't discover %s", serviceName)
	}
	return nodes, nil
}

// AddDiscoveryNode add a node to td.nodesInfo.
func AddDiscoveryNode(serviceName, address string) {
	td.mu.Lock()
	defer td.mu.Unlock()

	_, ok := td.nodesInfo[serviceName]
	if !ok {
		td.nodesInfo[serviceName] = []*registry.Node{
			{Address: address},
		}
	} else {
		td.nodesInfo[serviceName] = append(td.nodesInfo[serviceName], &registry.Node{Address: address})
	}
}

// RemoveDiscoveryNode remove a node from td.nodesInfo.
func RemoveDiscoveryNode(serviceName string) {
	td.mu.Lock()
	defer td.mu.Unlock()
	delete(td.nodesInfo, serviceName)
}
