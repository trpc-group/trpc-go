// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package selector determines how client chooses a backend node by service name. It contains service
// discovery, load balance and circuit breaker.
package selector

import (
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
)

// Register registers a named Selector
func Register(name string, s Selector) {
	selectors[name] = s
}

// Get gets a named Selector.
func Get(name string) Selector {
	s := selectors[name]
	return s
}

func unregisterForTesting(name string) {
	delete(selectors, name)
}
