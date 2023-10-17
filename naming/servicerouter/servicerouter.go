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

// Package servicerouter is service router which filters server instances. It is between service
// discovery and load balance.
package servicerouter

import (
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// DefaultServiceRouter is the default service router which is initialized by configuration.
var DefaultServiceRouter ServiceRouter = &NoopServiceRouter{}

// SetDefaultServiceRouter set the default service router.
func SetDefaultServiceRouter(s ServiceRouter) {
	DefaultServiceRouter = s
}

// ServiceRouter is the interface that defines the service router.
type ServiceRouter interface {
	Filter(serviceName string, nodes []*registry.Node, opt ...Option) ([]*registry.Node, error)
}

var (
	servicerouters = make(map[string]ServiceRouter)
)

// Register registers a named service router.
func Register(name string, s ServiceRouter) {
	servicerouters[name] = s
}

// Get gets a named service router.
func Get(name string) ServiceRouter {
	return servicerouters[name]
}

// NoopServiceRouter is the noop service router.
type NoopServiceRouter struct {
}

// Filter returns all nodes.
func (*NoopServiceRouter) Filter(serviceName string, nodes []*registry.Node, opt ...Option) ([]*registry.Node, error) {
	return nodes, nil
}

func unregisterForTesting(name string) {
	delete(servicerouters, name)
}
