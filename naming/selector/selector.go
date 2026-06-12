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

// Package selector determines how client chooses a backend node by service name. It contains service
// discovery, load balance and circuit breaker.
package selector

import (
	"sync"
	"sync/atomic"
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
	selectorsMu sync.Mutex
	selectors   atomic.Value // stores map[string]Selector
)

// Register registers a named Selector.
func Register(name string, s Selector) {
	selectorsMu.Lock()
	defer selectorsMu.Unlock()

	old := loadSelectors()
	next := make(map[string]Selector, len(old)+1)
	for k, v := range old {
		next[k] = v
	}
	next[name] = s
	selectors.Store(next)
}

// Get gets a named Selector.
func Get(name string) Selector {
	return loadSelectors()[name]
}

func loadSelectors() map[string]Selector {
	m, _ := selectors.Load().(map[string]Selector)
	if m == nil {
		return nil
	}
	return m
}

func unregisterForTesting(name string) {
	selectorsMu.Lock()
	defer selectorsMu.Unlock()

	old := loadSelectors()
	next := make(map[string]Selector, len(old))
	for k, v := range old {
		if k != name {
			next[k] = v
		}
	}
	selectors.Store(next)
}
