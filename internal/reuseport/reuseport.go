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

//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd
// +build linux darwin dragonfly freebsd netbsd openbsd

// Package reuseport provides a function that returns a net.Listener powered
// by a net.FileListener with a SO_REUSEPORT option set to the socket.
package reuseport

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
)

const fileNameTemplate = "reuseport.%d.%s.%s"

var errUnsupportedProtocol = errors.New("only tcp, tcp4, tcp6, udp, udp4, udp6 are supported")

// getSockaddr parses protocol and address and returns implementor
// of syscall.Sockaddr: syscall.SockaddrInet4 or syscall.SockaddrInet6.
func getSockaddr(proto, addr string) (sa syscall.Sockaddr, soType int, err error) {
	switch proto {
	case "tcp", "tcp4", "tcp6":
		return getTCPSockaddr(proto, addr)
	case "udp", "udp4", "udp6":
		return getUDPSockaddr(proto, addr)
	default:
		return nil, -1, errUnsupportedProtocol
	}
}

func getSocketFileName(proto, addr string) string {
	return fmt.Sprintf(fileNameTemplate, os.Getpid(), proto, addr)
}

// Listen function is an alias for NewReusablePortListener.
func Listen(proto, addr string) (l net.Listener, err error) {
	return NewReusablePortListener(proto, addr)
}

// ListenPacket is an alias for NewReusablePortPacketConn.
func ListenPacket(proto, addr string) (l net.PacketConn, err error) {
	return NewReusablePortPacketConn(proto, addr)
}
