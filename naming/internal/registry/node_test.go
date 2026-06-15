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

package registry

import (
	"testing"

	publicregistry "trpc.group/trpc-go/trpc-go/naming/registry"

	"github.com/stretchr/testify/require"
)

func TestDeepCopyNode(t *testing.T) {
	node := &publicregistry.Node{
		ServiceName: "trpc.test.helloworld.Greeter",
		Address:     "127.0.0.1:8000",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	nodeCopy := DeepCopyNode(node)
	require.NotSame(t, node, nodeCopy)
	require.Equal(t, node, nodeCopy)
	require.NotSame(t, node.Metadata, nodeCopy.Metadata)

	nodeCopy.Metadata["key"] = "changed"
	require.Equal(t, "value", node.Metadata["key"])
}

func TestDeepCopyNodeNil(t *testing.T) {
	require.Nil(t, DeepCopyNode(nil))
}

func TestDeepCopyNodeCopiesEmptyMetadata(t *testing.T) {
	node := &publicregistry.Node{
		Address:  "127.0.0.1:8000",
		Metadata: map[string]interface{}{},
	}

	nodeCopy := DeepCopyNode(node)
	require.NotSame(t, node.Metadata, nodeCopy.Metadata)

	nodeCopy.Metadata["key"] = "value"
	require.Empty(t, node.Metadata)
}
