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

package graceful

import (
	"net"
)

var _ net.Conn = (*Conn)(nil)

// NewConn creates a new Conn which implements net.Conn. onClosed will be called when Conn is closed.
func NewConn(conn net.Conn, onClosed func(net.Conn)) *Conn {
	return &Conn{
		Conn:     conn,
		onClosed: onClosed,
	}
}

// Conn wraps a new.Conn with onClosed callback.
type Conn struct {
	net.Conn
	onClosed func(net.Conn)
}

// Unwrap gives the wrapping net.Conn.
func (c *Conn) Unwrap() net.Conn {
	return c.Conn
}

// Close calls onClosed, and then close the wrapped net.Conn.
func (c *Conn) Close() error {
	c.onClosed(c)
	return c.Conn.Close()
}
