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

package registry

import (
	"fmt"
	"net"
	"time"
)

// Node is the information of a node.
type Node struct {
	// ServiceName is the service name of the node.
	ServiceName string
	// ContainerName is the container name of the node.
	ContainerName string
	// Address is the target address ip:port.
	Address string
	// Network is the network layer protocol, such as tcp or udp.
	Network string
	// Protocol is the business protocol, such as trpc or http.
	Protocol string
	// SetName is the set name of the node.
	SetName string
	// Weight is the weight of the node.
	Weight int
	// CostTime is the request duration of the node.
	CostTime time.Duration
	// EnvKey is the environment information for passthrough.
	EnvKey string
	// Metadata is the metadata of the node.
	Metadata map[string]interface{}
	// ParseAddr should be used to convert Node to net.Addr if it's not nil.
	// See test case TestSelectorRemoteAddrUseUserProvidedParser in client package.
	ParseAddr func(network, address string) net.Addr
}

// String returns an abbreviation information of node.
func (n *Node) String() string {
	return fmt.Sprintf("service: %s, addr: %s, cost: %s", n.ServiceName, n.Address, n.CostTime)
}
