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

//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd
// +build linux darwin dragonfly freebsd netbsd openbsd

package reuseport

import (
	"errors"
	"net"
	"os"
	"syscall"
)

var errUnsupportedUDPProtocol = errors.New("only udp, udp4, udp6 are supported")

func getUDP4Sockaddr(udp *net.UDPAddr) (syscall.Sockaddr, int, error) {
	sa := &syscall.SockaddrInet4{Port: udp.Port}

	if udp.IP != nil {
		if len(udp.IP) == 16 {
			copy(sa.Addr[:], udp.IP[12:16]) // copy last 4 bytes of slice to array
		} else {
			copy(sa.Addr[:], udp.IP) // copy all bytes of slice to array
		}
	}

	return sa, syscall.AF_INET, nil
}

func getUDP6Sockaddr(udp *net.UDPAddr) (syscall.Sockaddr, int, error) {
	sa := &syscall.SockaddrInet6{Port: udp.Port}

	if udp.IP != nil {
		copy(sa.Addr[:], udp.IP) // copy all bytes of slice to array
	}

	if udp.Zone != "" {
		iface, err := net.InterfaceByName(udp.Zone)
		if err != nil {
			return nil, -1, err
		}

		sa.ZoneId = uint32(iface.Index)
	}

	return sa, syscall.AF_INET6, nil
}

func getUDPAddr(proto, addr string) (*net.UDPAddr, string, error) {

	var udp *net.UDPAddr

	udp, err := net.ResolveUDPAddr(proto, addr)
	if err != nil {
		return nil, "", err
	}

	udpVersion, err := determineUDPProto(proto, udp)
	if err != nil {
		return nil, "", err
	}

	return udp, udpVersion, nil
}

func getUDPSockaddr(proto, addr string) (sa syscall.Sockaddr, soType int, err error) {
	udp, udpVersion, err := getUDPAddr(proto, addr)
	if err != nil {
		return nil, -1, err
	}

	switch udpVersion {
	case "udp":
		return &syscall.SockaddrInet4{Port: udp.Port}, syscall.AF_INET, nil
	case "udp4":
		return getUDP4Sockaddr(udp)
	default:
		// must be "udp6"
		return getUDP6Sockaddr(udp)
	}
}

func determineUDPProto(proto string, ip *net.UDPAddr) (string, error) {
	// If the protocol is set to "udp", we try to determine the actual protocol
	// version from the size of the resolved IP address. Otherwise, we simple use
	// the protocol given to us by the caller.

	if ip.IP.To4() != nil {
		return "udp4", nil
	}

	if ip.IP.To16() != nil {
		return "udp6", nil
	}

	switch proto {
	case "udp", "udp4", "udp6":
		return proto, nil
	default:
		return "", errUnsupportedUDPProtocol
	}
}

// NewReusablePortPacketConn returns net.FilePacketConn that created from
// a file descriptor for a socket with SO_REUSEPORT option.
func NewReusablePortPacketConn(proto, addr string) (net.PacketConn, error) {
	sockaddr, soType, err := getSockaddr(proto, addr)
	if err != nil {
		return nil, err
	}

	syscall.ForkLock.RLock()
	fd, err := syscall.Socket(soType, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err == nil {
		syscall.CloseOnExec(fd)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}
	return createPacketConn(fd, sockaddr, getSocketFileName(proto, addr))
}

func createPacketConn(fd int, sockaddr syscall.Sockaddr, fdName string) (net.PacketConn, error) {
	if err := setPacketConnSockOpt(fd, sockaddr); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	file := os.NewFile(uintptr(fd), fdName)
	l, err := net.FilePacketConn(file)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if err = file.Close(); err != nil {
		syscall.Close(fd)
		return nil, err
	}
	return l, err
}

func setPacketConnSockOpt(fd int, sockaddr syscall.Sockaddr) error {
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}

	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, reusePort, 1); err != nil {
		return err
	}

	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1); err != nil {
		return err
	}

	return syscall.Bind(fd, sockaddr)
}
