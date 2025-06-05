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

// Package addrutil provides some utility functions for net address.
package addrutil

import (
	"net"
	"strings"
)

// AddrToKey combines local and remote address into a string.
func AddrToKey(local, remote net.Addr) string {
	return strings.Join([]string{local.Network(), local.String(), remote.String()}, "_")
}
