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

package tnet

// ClientTransportOption is client transport option.
type ClientTransportOption func(o *ClientTransportOptions)

// ClientTransportOptions is client transport options struct.
type ClientTransportOptions struct {
	ExactUDPBufferSizeEnabled bool
}

// WithClientExactUDPBufferSizeEnabled sets whether to allocate an exact-sized buffer for UDP packets, false in default.
// If set to true, an exact-sized buffer is allocated for each UDP packet, requiring two system calls.
// If set to false, a fixed buffer size of maxUDPPacketSize is used, 65536 in default, requiring only one system call.
func WithClientExactUDPBufferSizeEnabled(enable bool) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.ExactUDPBufferSizeEnabled = enable
	}
}
