// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package healthcheck

// Opt modifies HealthCheck.
type Opt func(*HealthCheck)

// WithUnregisteredServiceStatus changes the default status of unregistered service to status.
func WithUnregisteredServiceStatus(status Status) Opt {
	return func(hc *HealthCheck) {
		hc.unregisteredServiceStatus = status
	}
}

// WithStatusWatchers returns an Option which set watchers for HealthCheck.
func WithStatusWatchers(watchers map[string][]func(status Status)) Opt {
	return func(hc *HealthCheck) {
		hc.serviceWatchers = watchers
	}
}
