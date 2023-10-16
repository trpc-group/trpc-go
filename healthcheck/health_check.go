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

// Package healthcheck is used to check service status.
package healthcheck

import (
	"fmt"
	"sync"
)

// Status is the status of server or individual services.
type Status int

const (
	// Unknown is the initial status of a service.
	Unknown Status = iota
	// Serving indicate the status of service is ok.
	Serving
	// NotServing indicate the service is not available now.
	NotServing
)

// New creates a new HealthCheck.
func New(opts ...Opt) *HealthCheck {
	hc := HealthCheck{
		unregisteredServiceStatus: Unknown,
		serviceStatuses:           make(map[string]Status),
		statusServices: map[Status]map[string]struct{}{
			Unknown:    make(map[string]struct{}),
			Serving:    make(map[string]struct{}),
			NotServing: make(map[string]struct{}),
		},
		serviceWatchers: make(map[string][]func(status Status)),
	}
	for _, opt := range opts {
		opt(&hc)
	}
	return &hc
}

// HealthCheck is the struct to implement health check.
type HealthCheck struct {
	unregisteredServiceStatus Status

	rwm             sync.RWMutex
	serviceStatuses map[string]Status
	statusServices  map[Status]map[string]struct{}
	serviceWatchers map[string][]func(status Status)
}

// Register registers a new service with initial status Unknown and returns a function to update status.
func (hc *HealthCheck) Register(name string) (update func(Status), err error) {
	hc.rwm.Lock()
	defer hc.rwm.Unlock()
	if _, ok := hc.serviceStatuses[name]; ok {
		return nil, fmt.Errorf("service %s has been registered", name)
	}
	hc.serviceStatuses[name] = Unknown
	hc.statusServices[Unknown][name] = struct{}{}
	for _, onStatusChanged := range hc.serviceWatchers[name] {
		onStatusChanged(Unknown)
	}
	return func(status Status) {
		hc.rwm.Lock()
		defer hc.rwm.Unlock()
		delete(hc.statusServices[hc.serviceStatuses[name]], name)
		hc.statusServices[status][name] = struct{}{}
		hc.serviceStatuses[name] = status

		for _, onStatusChanged := range hc.serviceWatchers[name] {
			onStatusChanged(status)
		}
	}, nil
}

// Unregister unregisters the registered service.
func (hc *HealthCheck) Unregister(name string) {
	hc.rwm.Lock()
	defer hc.rwm.Unlock()
	delete(hc.statusServices[hc.serviceStatuses[name]], name)
	delete(hc.serviceStatuses, name)
}

// CheckService returns the status of a service.
func (hc *HealthCheck) CheckService(name string) Status {
	hc.rwm.RLock()
	defer hc.rwm.RUnlock()
	status, ok := hc.serviceStatuses[name]
	if !ok {
		return hc.unregisteredServiceStatus
	}
	return status
}

// CheckServer returns the status of the entire server.
func (hc *HealthCheck) CheckServer() Status {
	hc.rwm.RLock()
	defer hc.rwm.RUnlock()
	if len(hc.statusServices[Serving]) == len(hc.serviceStatuses) {
		return Serving
	}
	if len(hc.statusServices[Unknown]) != 0 {
		return Unknown
	}
	return NotServing
}

// Watch registers a service status watcher.
func (hc *HealthCheck) Watch(serviceName string, onStatusChanged func(Status)) {
	hc.rwm.Lock()
	defer hc.rwm.Unlock()
	hc.serviceWatchers[serviceName] = append(hc.serviceWatchers[serviceName], onStatusChanged)
}
