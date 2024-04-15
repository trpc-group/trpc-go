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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

func TestDial(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	assert.Nil(t, err)
	defer l.Close()

	tests := []struct {
		name    string
		opts    *connpool.DialOptions
		want    net.Conn
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty CACertFile and Network",
			opts: &connpool.DialOptions{
				CACertFile: "",
				Network:    "",
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				return assert.Contains(t, err.Error(), "unknown network")
			},
		},
		{
			name: "invalid idle timeout",
			opts: &connpool.DialOptions{
				CACertFile:  "",
				Network:     "tcp",
				Address:     l.Addr().String(),
				IdleTimeout: -1,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				return assert.NoError(t, err, "idletimeout of -1 means no idle timeout, the error should be nil")
			},
		},
		{
			name: "wrong CACertFile and TLSServerName ",
			opts: &connpool.DialOptions{
				CACertFile:    "xxx",
				TLSServerName: "xxx",
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				return assert.Equal(t, errs.RetClientDecodeFail, errs.Code(err)) &&
					assert.Contains(t, err.Error(), "client dial tnet tls fail")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tnettrans.Dial(tt.opts)
			assert.True(t, tt.wantErr(t, err, fmt.Sprintf("Dial(%v)", tt.opts)))
			assert.Equalf(t, tt.want, got, "Dial(%v)", tt.opts)
		})
	}
}
