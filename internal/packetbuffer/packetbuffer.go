// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package packetbuffer implements functions for the manipulation
// of byte slices.
package packetbuffer

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/hashicorp/go-multierror"
	"trpc.group/trpc-go/trpc-go/internal/allocator"
)

// ErrReadFrom indicates packet connection Readfrom failed.
var ErrReadFrom = errors.New("packet connection ReadFrom failed")

// New creates a packet buffer with specific packet connection and size.
func New(conn net.PacketConn, size int) *PacketBuffer {
	buf, i := allocator.Malloc(size)
	return &PacketBuffer{
		buf:      buf,
		conn:     conn,
		toBeFree: i,
	}
}

// PacketBuffer encapsulates a packet connection and implements the io.Reader interface.
type PacketBuffer struct {
	buf      []byte
	toBeFree interface{}
	conn     net.PacketConn
	raddr    net.Addr
	r, w     int
}

// Read reads data from the packet. Continuous reads cannot cross between multiple packet only if Close is called.
func (pb *PacketBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if pb.w == 0 {
		n, raddr, err := pb.conn.ReadFrom(pb.buf)
		if err != nil {
			return 0, multierror.Append(ErrReadFrom, err).ErrorOrNil()
		}
		pb.w = n
		pb.raddr = raddr
	}
	n = copy(p, pb.buf[pb.r:pb.w])
	if n == 0 {
		return 0, io.EOF
	}
	pb.r += n
	return n, nil
}

// Next is used to distinguish continuous logic reads. It indicates that the reading on current packet has finished.
// If there remains data unconsumed, Next returns an error and discards the remaining data.
func (pb *PacketBuffer) Next() error {
	if pb.w == 0 {
		return nil
	}
	var err error
	if remain := pb.w - pb.r; remain != 0 {
		err = fmt.Errorf("packet data is not drained, the remaining %d will be dropped", remain)
	}
	pb.r, pb.w = 0, 0
	pb.raddr = nil
	return err
}

// CurrentPacketAddr returns current packet's remote address.
func (pb *PacketBuffer) CurrentPacketAddr() net.Addr {
	return pb.raddr
}

// Close closes this buffer and releases resource.
func (pb *PacketBuffer) Close() {
	allocator.Free(pb.toBeFree)
}
