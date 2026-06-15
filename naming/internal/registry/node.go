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

// Package registry contains helpers shared by naming implementations.
package registry

import publicregistry "trpc.group/trpc-go/trpc-go/naming/registry"

// DeepCopyNode returns a copy of node so selectors can attach per-call metadata
// without mutating discovery or load-balance snapshots.
func DeepCopyNode(node *publicregistry.Node) *publicregistry.Node {
	if node == nil {
		return nil
	}
	nodeCopy := *node
	if node.Metadata != nil {
		nodeCopy.Metadata = make(map[string]interface{}, len(node.Metadata))
		for k, v := range node.Metadata {
			nodeCopy.Metadata[k] = v
		}
	}
	return &nodeCopy
}
