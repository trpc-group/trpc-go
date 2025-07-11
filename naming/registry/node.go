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
	"fmt"
	"net"
	"time"
)

// Node is the information of a node.
type Node struct {
	ServiceName   string        // 服务名
	ContainerName string        // 容器名
	Address       string        // 目标地址 ip:port
	Network       string        // 网络层协议 tcp/udp
	Protocol      string        // 业务协议 trpc/http
	SetName       string        // 节点 Set 名
	Weight        int           // 权重
	CostTime      time.Duration // 当次请求耗时
	EnvKey        string        // 透传的环境信息
	Metadata      map[string]interface{}
	// ParseAddr should be used to convert Node to net.Addr if it's not nil.
	// See test case TestSelectorRemoteAddrUseUserProvidedParser in client package.
	ParseAddr func(network, address string) net.Addr
}

// String returns an abbreviation information of node.
func (n *Node) String() string {
	return fmt.Sprintf("service:%s, addr:%s, cost:%s", n.ServiceName, n.Address, n.CostTime)
}
