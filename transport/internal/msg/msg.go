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

// Package msg provides utility functions for handling messages.
package msg

import (
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/net"
)

// WithLocalAddr is a function that sets the local address of a given message.
// If the provided address is empty, it returns the original message without any modifications.
// Otherwise, it resolves the address using the provided network and sets it on the message.
// It then returns the modified message.
func WithLocalAddr(msg codec.Msg, network, addr string) codec.Msg {
	if addr == "" {
		return msg
	}
	msg.WithLocalAddr(net.ResolveAddress(network, addr))
	return msg
}
