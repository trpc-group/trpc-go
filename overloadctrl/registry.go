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

package overloadctrl

// AnyMethod indicates any method.
const AnyMethod = "*"

var (
	clientBuilders = make(map[string]Builder)
	serverBuilders = make(map[string]Builder)
)

// Builder constructs an overload controller.
type Builder func(*ServiceMethodInfo) OverloadController

// ServiceMethodInfo describes the callee service and method.
type ServiceMethodInfo struct {
	ServiceName string
	MethodName  string
}

// RegisterClient registers a client-side overload controller builder.
func RegisterClient(name string, newOC Builder) {
	clientBuilders[name] = newOC
}

// RegisterServer registers a server-side overload controller builder.
func RegisterServer(name string, newOC Builder) {
	serverBuilders[name] = newOC
}

// GetClient returns a client-side overload controller builder.
func GetClient(name string) Builder {
	return clientBuilders[name]
}

// GetServer returns a server-side overload controller builder.
func GetServer(name string) Builder {
	return serverBuilders[name]
}
