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

package discovery

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

func TestIpDiscovery(t *testing.T) {
	const serviceName = "ipDiscovery.ip.62:8989"
	d := &IPDiscovery{}
	nodes, err := d.List(serviceName, nil)
	require.Nil(t, err)
	require.Equal(t, []*registry.Node{{ServiceName: serviceName, Address: serviceName}}, nodes)
}
