// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package selector

import (
	"errors"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
)

// Errors when route failed.
var (
	ErrReportNodeEmpty             = errors.New("selector report node empty")
	ErrReportMetaDataEmpty         = errors.New("selector report metadata empty")
	ErrReportNoCircuitBreaker      = errors.New("selector report not circuitbreaker")
	ErrReportInvalidCircuitBreaker = errors.New("selector report circuitbreaker invalid")
)

// DefaultSelector is the default Selector.
var DefaultSelector Selector = &TrpcSelector{}

// TrpcSelector is the default Selector implementation of tRPC. It automatically combines pluggable
// modules, like, service discovery, load balance and circuit breaker.
type TrpcSelector struct{}

// Select returns an available node by service name.
func (s *TrpcSelector) Select(serviceName string, opt ...Option) (*registry.Node, error) {
	if serviceName == "" {
		return nil, errors.New("service name empty")
	}

	opts := &Options{
		Discovery:            discovery.DefaultDiscovery,
		DiscoveryOptions:     make([]discovery.Option, 0, defaultDiscoveryOptionsSize),
		LoadBalancer:         loadbalance.DefaultLoadBalancer,
		LoadBalanceOptions:   make([]loadbalance.Option, 0, defaultLoadBalanceOptionsSize),
		ServiceRouter:        servicerouter.DefaultServiceRouter,
		ServiceRouterOptions: make([]servicerouter.Option, 0, defaultServiceRouterOptionsSize),
		CircuitBreaker:       circuitbreaker.DefaultCircuitBreaker,
	}
	for _, o := range opt {
		o(opts)
	}

	if opts.Discovery == nil {
		return nil, errors.New("discovery not exists")
	}
	list, err := opts.Discovery.List(serviceName, opts.DiscoveryOptions...)
	if err != nil {
		return nil, err
	}

	if opts.ServiceRouter == nil {
		return nil, errors.New("servicerouter not exists")
	}
	list, err = opts.ServiceRouter.Filter(serviceName, list, opts.ServiceRouterOptions...)
	if err != nil {
		return nil, err
	}

	if opts.LoadBalancer == nil {
		return nil, errors.New("loadbalancer not exists")
	}

	if opts.CircuitBreaker == nil {
		return nil, errors.New("circuitbreaker not exists")
	}

	node, err := opts.LoadBalancer.Select(serviceName, list, opts.LoadBalanceOptions...)
	if err != nil {
		return nil, err
	}

	if len(node.Metadata) == 0 {
		node.Metadata = make(map[string]interface{})
	}
	node.Metadata["circuitbreaker"] = opts.CircuitBreaker
	return node, nil
}

// Report reports result.
func (s *TrpcSelector) Report(node *registry.Node, cost time.Duration, err error) error {
	if node == nil {
		return ErrReportNodeEmpty
	}
	if node.Metadata == nil {
		return ErrReportMetaDataEmpty
	}
	breaker, ok := node.Metadata["circuitbreaker"]
	if !ok {
		return ErrReportNoCircuitBreaker
	}
	circuitbreaker, ok := breaker.(circuitbreaker.CircuitBreaker)
	if !ok {
		return ErrReportInvalidCircuitBreaker
	}
	return circuitbreaker.Report(node, cost, err)
}
