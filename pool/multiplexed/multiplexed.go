// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package multiplexed provides multiplexed pool implementation.
package multiplexed

import (
	"context"
	"net"

	"trpc.group/trpc-go/trpc-go/pool/connpool"
)

// Pool is a virtual connection pool for multiplexing.
type Pool interface {
	// GetVirtualConn gets a virtual connection to the address on named network.
	GetVirtualConn(ctx context.Context, network string, address string, opts GetOptions) (VirtualConn, error)
}

// VirtualConn is virtual connection multiplexing on a real connection.
type VirtualConn interface {
	// Write writes data to the virtual connection.
	Write([]byte) error

	// Read reads a packet from virtual connection.
	Read() ([]byte, error)

	// LocalAddr returns the local network address, if known.
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address, if known.
	RemoteAddr() net.Addr

	// Close closes the virtual connection.
	// Any blocked Read or Write operations will be unblocked and return errors.
	Close()
}

// DialFunc represents a function type for establishing a connection.
type DialFunc func(fp FrameParser, opts *connpool.DialOptions) (Conn, error)

// Conn is a real connection within multiplexing.
type Conn interface {
	// Start initiates the connection by starting background read and write processes.
	// Notifier's Dispatch should be called when it parse vid and frame from data reading.
	// Notifier's Close should be called when the connection is closed.
	Start(Notifier) error
	// Write writes data to the connection and returns the number of bytes written.
	Write([]byte) (int, error)
	// Close closes the connection.
	Close() error
	// LocalAddr returns the local network address of the connection.
	LocalAddr() net.Addr
	// RemoteAddr returns the remote network address of the connection.
	RemoteAddr() net.Addr
	// IsActive returns true if the connection is active, otherwise false.
	IsActive() bool
}

// Notifier represents an interface for notifying the multiplexing pool about relevant events and actions.
type Notifier interface {
	// Dispatch notifies the multiplexing pool to pass the buffer to the appropriate virtual
	// connection based on the specified vid.
	Dispatch(vid uint32, buf []byte)

	// Close notifies the multiplexing pool that the associated connection is being closed,
	// allowing it to perform any necessary cleanup or termination procedures.
	Close(error)
}
