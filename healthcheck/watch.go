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

package healthcheck

var watchers = make(map[string][]func(Status))

// Watch registers a service status watcher.
// NOTE: No lock is used in this function, so it is not concurrency safe.
func Watch(serviceName string, onStatusChanged func(Status)) {
	watchers[serviceName] = append(watchers[serviceName], onStatusChanged)
}

// GetWatchers returns all registered watchers.
// NOTE: No lock is used in this function, so it is not concurrency safe.
// NOTE: The result is read-only, DO NOT modify.
func GetWatchers() map[string][]func(Status) {
	return watchers
}
