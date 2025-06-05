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

// Package reflection is used to avoid circular references in trpc package.
package reflection

import "trpc.group/trpc-go/trpc-go/server"

var (
	// Register Registers the reflection service and to the server.Service.
	// reflection service get ServiceInfo by calling *server.Server.GetServiceInfo.
	// Register is set when the user imports the reflection package,
	// and actually called by the trpc package when the service starts.
	Register = func(server.Service, *server.Server) {}
)
