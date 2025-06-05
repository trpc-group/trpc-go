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

//go:build windows

package graceful

import (
	"errors"
	"net"
)

// Restart is not available on Windows systems.
var Restart = func([]uintptr) error {
	return errors.New("graceful restart is not available for windows")
}

// Listen creates a net.Listener on network address.
func Listen(network, address string, reusePort bool) (net.Listener, error) {
	return net.Listen(network, address)
}

// ListenPacket creates a net.PacketConn on network address.
var ListenPacket = func(network string, address string, reusePort bool) (net.PacketConn, error) {
	return net.ListenPacket(network, address)
}

var UnwrapListener = func(ln any) net.Listener {
	if l, ok := ln.(net.Listener); ok {
		return l
	}
	panic("unreachable in normal, unexpected listener type")
}

var UnwrapPacketConn = func(udpconn any) net.PacketConn {
	if c, ok := udpconn.(net.PacketConn); ok {
		return c
	}
	panic("unreachable in normal, unexpected packetConn type")
}
