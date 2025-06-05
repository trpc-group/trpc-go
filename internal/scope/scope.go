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

// Package scope provides definitions for scope variables.
package scope

// Scope defines the current scope.
type Scope = string

// Definitions of common scopes.
const (
	// Local means that the caller of local scope can only access the server in the current process.
	Local Scope = "local"
	// Remote means that the caller of remote scope can only access the server in the remote server (normal RPC).
	Remote Scope = "remote"
	// All means that the caller of all scope can access the server in both local and remote server.
	// The caller will first try to access the server in the current process(local scope) and then try to access
	// servers in the remote server(remote scope).
	All Scope = "all"
)
