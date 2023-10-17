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
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// IPDiscovery discovers node by IPs.
type IPDiscovery struct{}

// List returns the original IP/Port.
func (*IPDiscovery) List(serviceName string, opt ...Option) ([]*registry.Node, error) {
	node := &registry.Node{ServiceName: serviceName, Address: serviceName}
	return []*registry.Node{node}, nil
}
