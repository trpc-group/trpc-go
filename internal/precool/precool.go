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

// Package precool implements service-level precool checks.
package precool

import (
	"fmt"
	"sync"

	"trpc.group/trpc-go/trpc-go/precool"
)

// Option configures a Check.
type Option func(*Check)

// WithUnregisteredServiceStatus sets the status returned for unregistered services.
func WithUnregisteredServiceStatus(status precool.Status) Option {
	return func(pc *Check) {
		pc.unregisteredServiceStatus = status
	}
}

// Check provides service-level precool detection.
type Check struct {
	unregisteredServiceStatus precool.Status

	mu    sync.RWMutex
	funcs map[string]precool.Func
}

// New creates a new Check instance.
func New(opts ...Option) *Check {
	pc := &Check{
		unregisteredServiceStatus: precool.Unknown,
		funcs:                     make(map[string]precool.Func),
	}
	for _, opt := range opts {
		opt(pc)
	}
	return pc
}

// Register registers a service with a custom precool strategy.
func (pc *Check) Register(name string, fn precool.Func) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if _, ok := pc.funcs[name]; ok {
		return fmt.Errorf("service %s has been registered", name)
	}
	pc.funcs[name] = fn
	return nil
}

// Unregister unregisters a service from precool check.
func (pc *Check) Unregister(name string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.funcs, name)
}

// CheckService returns the precool status of a specific service.
func (pc *Check) CheckService(name string) precool.Status {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	fn, ok := pc.funcs[name]
	if !ok {
		return pc.unregisteredServiceStatus
	}
	return fn()
}

// CheckServer returns the aggregate precool status of the entire server.
func (pc *Check) CheckServer() precool.Status {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if len(pc.funcs) == 0 {
		return precool.Success
	}
	var (
		hasFailure bool
		hasOngoing bool
		allSuccess = true
	)
	for _, fn := range pc.funcs {
		switch fn() {
		case precool.Failure:
			hasFailure = true
			allSuccess = false
		case precool.Ongoing:
			hasOngoing = true
			allSuccess = false
		case precool.Success:
		default:
			allSuccess = false
		}
	}
	if hasFailure {
		return precool.Unknown
	}
	if hasOngoing {
		return precool.Ongoing
	}
	if allSuccess {
		return precool.Success
	}
	return precool.Failure
}
