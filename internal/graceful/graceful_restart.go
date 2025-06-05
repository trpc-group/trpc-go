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

//go:build !windows

package graceful

import (
	"net"

	igr "trpc.group/trpc-go/trpc-go/internal/graceful/internal"
)

// Restart attempts to perform a graceful restart.
var Restart = igr.Restart

// Listen creates a net.Listener on network address and supports port reuse.
var Listen = igr.Listen

// ListenPacket creates a net.PacketConn on network address and supports port reuse.
var ListenPacket = igr.ListenPacket

var UnwrapListener = igr.Unwrap[net.Listener]

var UnwrapPacketConn = igr.Unwrap[net.PacketConn]
