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
	"testing"

	"trpc.group/trpc-go/trpc-go/internal/net"
)

func TestConstructAddr(t *testing.T) {
	tests := []struct {
		network, address string
		wantNetwork      string
		wantAddress      string
	}{
		{"tcp", "192.0.2.1:25", "tcp", "192.0.2.1:25"},
		{"udp", "[2001:db8::1]:80", "udp", "[2001:db8::1]:80"},
	}

	for _, tt := range tests {
		t.Run(tt.network+"/"+tt.address, func(t *testing.T) {
			addr := net.ResolveAddress(tt.network, tt.address)

			if got := addr.Network(); got != tt.wantNetwork {
				t.Errorf("ResolveTCPAddr().Network() = %v, want %v", got, tt.wantNetwork)
			}

			if got := addr.String(); got != tt.wantAddress {
				t.Errorf("ResolveTCPAddr().String() = %v, want %v", got, tt.wantAddress)
			}
		})
	}
}
