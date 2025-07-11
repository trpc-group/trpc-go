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
	"testing"

	"github.com/stretchr/testify/assert"
)

func moreCaseNewReusablePortPacketConn(t *testing.T) {
	listenerFour, err := NewReusablePortListener("udp6", ":10081")
	assert.Nil(t, err)
	defer listenerFour.Close()

	listenerFive, err := NewReusablePortListener("udp4", ":10081")
	assert.Nil(t, err)
	defer listenerFive.Close()

	listenerSix, err := NewReusablePortListener("udp", ":10081")
	assert.Nil(t, err)
	defer listenerSix.Close()
}

func TestNewReusablePortPacketConn(t *testing.T) {
	listenerOne, err := NewReusablePortPacketConn("udp4", "localhost:10082")
	assert.Nil(t, err)
	defer listenerOne.Close()

	listenerTwo, err := NewReusablePortPacketConn("udp", "127.0.0.1:10082")
	assert.Nil(t, err)
	defer listenerTwo.Close()

	listenerThree, err := NewReusablePortPacketConn("udp6", ":10082")
	assert.Nil(t, err)
	defer listenerThree.Close()

	moreCaseNewReusablePortPacketConn(t)
}

func TestListenPacket(t *testing.T) {
	type args struct {
		proto string
		addr  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "case1",
			args: args{
				proto: "udp4",
				addr:  "localhost:10082",
			},
			wantErr: false,
		},
		{
			name: "case2",
			args: args{
				proto: "udp",
				addr:  "localhost:10082",
			},
			wantErr: false,
		},
		{
			name: "case3",
			args: args{
				proto: "udp6",
				addr:  ":10082",
			},
			wantErr: false,
		},
		{
			name: "case4",
			args: args{
				proto: "udp4",
				addr:  ":10081",
			},
			wantErr: false,
		},
		{
			name: "case5",
			args: args{
				proto: "udp6",
				addr:  ":10081",
			},
			wantErr: false,
		},
		{
			name: "case6",
			args: args{
				proto: "udp",
				addr:  ":10081",
			},
			wantErr: false,
		},
		{
			name: "case7",
			args: args{
				proto: "udp6_no_ipv_device",
				addr:  "[::1]:10081",
			},
			wantErr: true,
		},
		{
			name: "case8_not_support_proto",
			args: args{
				proto: "xxx",
				addr:  "[::1]:10081",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotL, err := ListenPacket(tt.args.proto, tt.args.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListenPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotL != nil {
				_ = gotL.Close()
			}
		})
	}
}

func BenchmarkNewReusableUDPPortListener(b *testing.B) {
	for i := 0; i < b.N; i++ {
		listener, err := NewReusablePortPacketConn("udp4", "localhost:10082")

		if err != nil {
			b.Error(err)
		} else {
			listener.Close()
		}
	}
}

// TestNewReusablePortPacketConn2 一些边界覆盖
func TestNewReusablePortPacketConn2(t *testing.T) {
	// new socket fd failed, unsupported protocol
	_, err := NewReusablePortPacketConn("udp4xx", "localhost:10082")
	assert.NotNil(t, err)

	// reusePort failed
	oldReusePort := reusePort
	defer func() {
		reusePort = oldReusePort
	}()
	reusePort = 0
	_, err = NewReusablePortPacketConn("udp4", "localhost:10082")
	assert.NotNil(t, err)
}
