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

package net_test

import (
	"net"
	"testing"

	inet "trpc.group/trpc-go/trpc-go/internal/net"
)

// BenchmarkResolveAddressVsResolveTCPAddr benchmarks both the custom ResolveAddress
// function and the standard library's net.ResolveTCPAddr function with different addresses.
//
// Possible results:
//
//	goos: linux
//	goarch: amd64
//	pkg: trpc.group/trpc-go/trpc-go/internal/net
//	cpu: Intel(R) Xeon(R) Platinum 8255C CPU @ 2.50GHz
//	BenchmarkResolveAddressVsResolveTCPAddr/CustomResolveAddress-10  522945207  2.310 ns/op    0 B/op  0 allocs/op
//	BenchmarkResolveAddressVsResolveTCPAddr/StdResolveTCPAddr-10       1690285  707.8 ns/op  260 B/op  9 allocs/op
//	PASS
//	ok  	trpc.group/trpc-go/trpc-go/internal/net	3.363s
func BenchmarkResolveAddressVsResolveTCPAddr(b *testing.B) {
	testCases := []struct {
		name    string
		network string
		address string
	}{
		{"IPv4", "tcp", "192.0.2.1:25"},
		{"IPv6", "tcp", "[2001:db8::1]:80"},
	}

	b.Run("CustomResolveAddress", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, tc := range testCases {
				_ = inet.ResolveAddress(tc.network, tc.address)
			}
		}
	})

	b.Run("StdResolveTCPAddr", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, tc := range testCases {
				_, _ = net.ResolveTCPAddr(tc.network, tc.address)
			}
		}
	})
}
