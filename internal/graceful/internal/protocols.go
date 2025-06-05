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

import "encoding/gob"

// protocol is the interface to mark a struct is an rpc protocol.
type protocol interface {
	proto()
}

// ReqListeners is the request that parent send to child to inherit listeners.
type ReqListeners struct {
	protocol
	Listeners []ReqListener
	Continue  bool
}

// ReqListener is a single Listener.
type ReqListener struct {
	protocol
	Network string
	Address string
}

// AckListeners is the response that child sends to parent to indicate
// that it has received all listeners.
type AckListeners struct {
	protocol
	Cnt int
}

// ReqConn is the request that parent send to child to deliver a net.conn.
type ReqConn struct {
	protocol
	Network string
	Address string
}

func initProtocols() {
	gob.Register(ReqListeners{})
	gob.Register(ReqListener{})
	gob.Register(AckListeners{})
	gob.Register(ReqConn{})
}
