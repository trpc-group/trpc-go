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
	"errors"
	"fmt"
	"net"

	"trpc.group/trpc-go/trpc-go/internal/reuseport"
)

// [network][address] -> net.PacketConn
var inheritPacketConns = NewSafe[map[string]map[string]net.PacketConn](nil)
var packetConns = NewSafe[map[string]map[string]net.PacketConn](nil)

// ListenPacket creates a net.PacketConn on network address.
// If we have inherited a PacketConn from parent process, then return the inherited one.
// Otherwise, create a new net.PacketConn by net.PacketConn.
// In either case, the PacketConn is stored to a global variable packetConns and is ready
// to pass to child process on next graceful restart.
func ListenPacket(network, address string, reusePort bool) (net.PacketConn, error) {
	// Listen packet from inheritPacketConns
	inheritPacketConns.Lock()
	if addrs, ok := inheritPacketConns.T[network]; ok {
		if conn, ok := addrs[address]; ok {
			packetConns.Lock()
			packetConns.T = appendMap(packetConns.T, network, address, conn)
			packetConns.Unlock()
			deleteMap(inheritPacketConns.T, network, address)
			inheritPacketConns.Unlock()
			return conn, nil
		}
	}
	inheritPacketConns.Unlock()

	var (
		conn net.PacketConn
		err  error
	)
	if reusePort {
		conn, err = reuseport.ListenPacket(network, address)
		if err != nil {
			return nil, fmt.Errorf("reuseport listen packet: %w", err)
		}
	} else {
		conn, err = net.ListenPacket(network, address)
		if err != nil {
			return nil, fmt.Errorf("net listen packet: %w", err)
		}
	}
	conn, err = NewPacketConn(conn, network, address)
	if err != nil {
		return nil, fmt.Errorf("new packet conn: %w", err)
	}
	packetConns.Lock()
	packetConns.T = appendMap(packetConns.T, network, address, conn)
	packetConns.Unlock()
	return conn, nil
}

// NewPacketConn creates a new PacketConn based on net.PacketConn.
func NewPacketConn(conn net.PacketConn, network, address string) (net.PacketConn, error) {
	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		return nil, errors.New("conn is not a net.UDPConn")
	}
	return &PacketConn{
		network: network,
		address: address,
		UDPConn: udpConn,
	}, nil
}

// PacketConn is a wrap implementation of net.UDPConn.
type PacketConn struct {
	network string
	address string
	*net.UDPConn
}

// Close closes the underlying net.Listener.
func (conn *PacketConn) Close() error {
	packetConns.Lock()
	deleteMap(packetConns.T, conn.network, conn.address)
	packetConns.Unlock()
	return conn.UDPConn.Close()
}

// Unwrap unwraps and giving the underlying net.Listener.
func (conn *PacketConn) Unwrap() net.PacketConn {
	return conn.UDPConn
}
