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

package graceful_test

import (
	"net"
	"testing"
	"time"

	. "trpc.group/trpc-go/trpc-go/internal/graceful/internal"
	"github.com/stretchr/testify/assert"
)

func TestListenPacket(t *testing.T) {
	addr := "127.0.0.1:8080"
	req := []byte("hello")
	conn, err := ListenPacket("udp", addr, false)
	assert.Nil(t, err)
	// Close
	defer conn.Close()

	client, err := net.Dial("udp", addr)
	assert.Nil(t, err)
	_, err = client.Write(req)
	assert.Nil(t, err)

	// ReadFrom
	buf := make([]byte, 1024)
	n, raddr, err := conn.ReadFrom(buf)
	assert.Nil(t, err)
	assert.Equal(t, req, buf[:n])
	assert.Equal(t, client.LocalAddr(), raddr)

	// WriteTo
	_, err = conn.WriteTo(req, client.LocalAddr())
	assert.Nil(t, err)

	// LocalAddr
	assert.Equal(t, addr, conn.LocalAddr().String())

	// SetDeadline
	assert.Nil(t, conn.SetDeadline(time.Now().Add(time.Second)))

	// SetReadDeadline
	assert.Nil(t, conn.SetReadDeadline(time.Now().Add(time.Second)))

	// SetWriteDeadline
	assert.Nil(t, conn.SetWriteDeadline(time.Now().Add(time.Second)))

	gracefulConn, ok := conn.(*PacketConn)
	assert.True(t, ok)

	// Unwrap
	assert.NotNil(t, gracefulConn.Unwrap())
}

func TestListenPacketInvalidNetwork(t *testing.T) {
	_, err := ListenPacket("tcp", "127.0.0.1:8080", false)
	assert.NotNil(t, err)
}
