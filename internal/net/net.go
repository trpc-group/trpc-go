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

// Package net provides networking utilities.
package net

import "net"

// ResolveAddress is a utility function that quickly constructs a net.Addr from a given network type and address.
// It is intended to provide a more performant alternative to net.ResolveTCPAddr and net.ResolveUDPAddr by avoiding
// the overhead of error handling within the function. This function assumes that the provided address is valid
// and does not perform any sanity checks or error handling. It is the caller's responsibility to ensure that
// the address is valid before calling this function.
//
// Parameters:
// - network: A string representing the network type, e.g., "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6".
// - address: A string representing the network address.
//
// Returns:
// A net.Addr representing the resolved address.
func ResolveAddress(network, address string) net.Addr {
	return addr{network, address}
}

type addr struct {
	network string
	address string
}

// Network implements net.Addr, it is the name of the network (for example, "tcp", "udp").
func (a addr) Network() string {
	return a.network
}

// String implements net.Addr, it is the string form of address
// (for example, "192.0.2.1:25", "[2001:db8::1]:80").
func (a addr) String() string {
	return a.address
}
