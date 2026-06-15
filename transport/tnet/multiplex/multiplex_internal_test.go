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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package multiplex

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/trpc-go/internal/queue"
)

func TestOnRequestDropsClosedVirtualConnection(t *testing.T) {
	const vid = 1
	ctx, cancel := context.WithCancel(context.Background())
	vc := &virtualConnection{
		ctx:        ctx,
		cancelFunc: cancel,
		recvQueue:  queue.New[[]byte](ctx.Done()),
		deleteVirConnFromConn: func() {
		},
		id: vid,
	}
	conn := &connection{
		fp:          &staticFrameParser{id: vid, buf: []byte("late response")},
		idToVirConn: newShardMap(1),
	}
	conn.idToVirConn.store(vid, vc)

	vc.Close()

	require.NoError(t, conn.onRequest(&fakeTnetConn{}))
	_, ok := vc.recvQueue.Get()
	require.False(t, ok)
}

type staticFrameParser struct {
	id  uint32
	buf []byte
}

func (p *staticFrameParser) Parse(io.Reader) (uint32, []byte, error) {
	return p.id, p.buf, nil
}

type fakeTnetConn struct {
	meta any
}

func (*fakeTnetConn) Read([]byte) (int, error) { return 0, io.EOF }
func (*fakeTnetConn) Write(b []byte) (int, error) {
	return len(b), nil
}
func (*fakeTnetConn) Close() error                     { return nil }
func (*fakeTnetConn) LocalAddr() net.Addr              { return &net.IPAddr{} }
func (*fakeTnetConn) RemoteAddr() net.Addr             { return &net.IPAddr{} }
func (*fakeTnetConn) SetDeadline(time.Time) error      { return nil }
func (*fakeTnetConn) SetReadDeadline(time.Time) error  { return nil }
func (*fakeTnetConn) SetWriteDeadline(time.Time) error { return nil }
func (*fakeTnetConn) Len() int                         { return 0 }
func (*fakeTnetConn) IsActive() bool                   { return true }
func (*fakeTnetConn) SetNonBlocking(bool)              {}
func (*fakeTnetConn) SetFlushWrite(bool)               {}
func (c *fakeTnetConn) SetMetaData(m any)              { c.meta = m }
func (c *fakeTnetConn) GetMetaData() any               { return c.meta }
func (*fakeTnetConn) Peek(int) ([]byte, error)         { return nil, io.EOF }
func (*fakeTnetConn) Next(int) ([]byte, error)         { return nil, io.EOF }
func (*fakeTnetConn) Skip(int) error                   { return nil }
func (*fakeTnetConn) Release()                         {}
func (*fakeTnetConn) ReadN(int) ([]byte, error)        { return nil, io.EOF }
func (*fakeTnetConn) Writev(p ...[]byte) (int, error) {
	var total int
	for _, b := range p {
		total += len(b)
	}
	return total, nil
}
func (*fakeTnetConn) SetKeepAlive(time.Duration) error   { return nil }
func (*fakeTnetConn) SetOnRequest(tnet.TCPHandler) error { return nil }
func (*fakeTnetConn) SetOnClosed(tnet.OnTCPClosed) error { return nil }
func (*fakeTnetConn) SetIdleTimeout(time.Duration) error { return nil }
func (*fakeTnetConn) SetSafeWrite(bool)                  {}
