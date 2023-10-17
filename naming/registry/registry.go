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

// Package registry registers servers. A server report itself on start.
package registry

import (
	"errors"
	"sync"
)

// ErrNotImplement is the not implemented error.
var ErrNotImplement = errors.New("not implement")

// DefaultRegistry is the default registry.
var DefaultRegistry Registry = &NoopRegistry{}

// SetDefaultRegistry sets the default registry.
func SetDefaultRegistry(r Registry) {
	DefaultRegistry = r
}

// Registry is the interface that defines a register.
type Registry interface {
	Register(service string, opt ...Option) error
	Deregister(service string) error
}

var (
	registries = make(map[string]Registry)
	lock       = sync.RWMutex{}
)

// Register registers a named registry. Each service has its own registry.
func Register(name string, s Registry) {
	lock.Lock()
	registries[name] = s
	lock.Unlock()
}

// Get gets a named registry.
func Get(name string) Registry {
	lock.RLock()
	r := registries[name]
	lock.RUnlock()
	return r
}

// NoopRegistry is the noop registry.
type NoopRegistry struct{}

// Register always returns ErrNotImplement.
func (noop *NoopRegistry) Register(service string, opt ...Option) error {
	return ErrNotImplement
}

// Deregister always return ErrNotImplement.
func (noop *NoopRegistry) Deregister(service string) error {
	return ErrNotImplement
}
