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

// Package selector determines how client chooses a backend node by service name. It contains service
// discovery, load balance and circuit breaker.
package selector

import (
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// Selector is the interface that defines the selector.
type Selector interface {
	// Select gets a backend node by service name.
	Select(serviceName string, opt ...Option) (*registry.Node, error)
	// Report reports request status.
	Report(node *registry.Node, cost time.Duration, err error) error
}

var (
	selectors = make(map[string]Selector)
	lock      = sync.RWMutex{}
)

// Register registers a named Selector.
func Register(name string, s Selector) {
	lock.Lock()
	selectors[name] = s
	lock.Unlock()
}

// Get gets a named Selector.
func Get(name string) Selector {
	lock.RLock()
	s := selectors[name]
	lock.RUnlock()
	return s
}

func unregisterForTesting(name string) {
	lock.Lock()
	delete(selectors, name)
	lock.Unlock()
}
