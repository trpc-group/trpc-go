// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package naming

import (
	"fmt"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
)

type testSelector struct {
	nodeInfo map[string]*registry.Node
	mu       sync.RWMutex
}

func init() {
	selector.Register("test", ts)
}

var ts = &testSelector{
	nodeInfo: make(map[string]*registry.Node),
}

// Select a node from ts.nodeInfo.
func (s *testSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
	ts.mu.RLock()
	node, ok := s.nodeInfo[serviceName]
	ts.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no available node")
	}
	return node, nil
}

// Report nothing.
func (s *testSelector) Report(node *registry.Node, cost time.Duration, err error) error {
	return nil
}

// AddSelectorNode adds a selector node to ts.nodeInfo.
func AddSelectorNode(serviceName, address string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.nodeInfo[serviceName] = &registry.Node{Address: address}
}

// RemoveSelectorNode removes a selector node from ts.nodeInfo.
func RemoveSelectorNode(serviceName string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.nodeInfo, serviceName)
}
