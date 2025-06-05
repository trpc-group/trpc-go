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

//go:build linux && amd64
// +build linux,amd64

package client

import (
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
	"trpc.group/trpc-go/trpc-go/transport/tnet"
)

func attemptSwitchingTransport(o *Options) transport.ClientTransport {
	// If the user doesn't explicitly set the transport (which is usually the case for trpc protocol),
	// attempt to switch to the tnet transport.
	if o.Transport == nil {
		if check(o) {
			cheer(o)
			return tnet.DefaultClientTransport
		}
		sigh(o)
		return transport.DefaultClientTransport
	}
	return o.Transport
}

func check(o *Options) bool {
	// Only use tnet transport with TCP and trpc.
	return (o.Network == "tcp" ||
		o.Network == "tcp4" ||
		o.Network == "tcp6") &&
		o.protocol == "trpc"
}

func cheer(o *Options) {
	log.Infof("client %s is empowered with tnet! 🤩 ", o.ServiceName)
}

func sigh(o *Options) {
	log.Infof("client: %s, tnet is not enabled by default 🧐 ", o.ServiceName)
}
