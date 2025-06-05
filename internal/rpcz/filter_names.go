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

// Package rpcz provides internal utilities for rpcz.
package rpcz

import (
	"context"

	"trpc.group/trpc-go/trpc-go/rpcz"
)

// FilterNames retrieves filter names from context.
func FilterNames(ctx context.Context) ([]string, bool) {
	value, ok := rpcz.SpanFromContext(ctx).Attribute(rpcz.TRPCAttributeFilterNames)
	if !ok {
		return nil, false
	}
	names, ok := value.([]string)
	return names, ok
}

// FilterName returns the name at the given index.
// Return "unknown" for invalid index.
func FilterName(names []string, index int) string {
	if index >= len(names) || index < 0 {
		const unknownName = "unknown"
		return unknownName
	}
	return names[index]
}
