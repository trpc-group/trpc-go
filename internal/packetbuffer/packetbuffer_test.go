// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package packetbuffer_test

import (
	"context"
	"io"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/internal/packetbuffer"
)

type udpServer struct {
	cancel context.CancelFunc
	conn   net.PacketConn
}

func (s *udpServer) start(ctx context.Context) error {
	var err error
	s.conn, err = net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	ctx, s.cancel = context.WithCancel(ctx)
	go func() {
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			n, addr, err := s.conn.ReadFrom(buf)
			if err != nil {
				log.Println("l.ReadFrom err: ", err)
				return
			}
			s.conn.WriteTo(buf[:n], addr)
		}
	}()
	return nil
}

func (s *udpServer) stop() {
	s.cancel()
	s.conn.Close()
}

func TestPacketReaderSucceed(t *testing.T) {
	s := &udpServer{}
	s.start(context.Background())
	t.Cleanup(s.stop)

	p, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.Nil(t, err)
	_, err = p.WriteTo([]byte("helloworldA"), s.conn.LocalAddr())
	require.Nil(t, err)
	buf := packetbuffer.New(p, 65535)
	defer buf.Close()
	result := make([]byte, 20)
	n, err := buf.Read(result)
	require.Nil(t, err)
	require.Equal(t, []byte("helloworldA"), result[:n])
	require.Equal(t, s.conn.LocalAddr(), buf.CurrentPacketAddr())
	_, err = buf.Read(result)
	require.Equal(t, io.EOF, err)
	require.Nil(t, buf.Next())

	_, err = p.WriteTo([]byte("helloworldB"), s.conn.LocalAddr())
	require.Nil(t, err)
	n, err = buf.Read(result)
	require.Nil(t, err)
	require.Equal(t, []byte("helloworldB"), result[:n])
}

func TestPacketReaderFailed(t *testing.T) {
	s := &udpServer{}
	s.start(context.Background())
	t.Cleanup(s.stop)

	p, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.Nil(t, err)
	_, err = p.WriteTo([]byte("helloworld"), s.conn.LocalAddr())
	require.Nil(t, err)
	buf := packetbuffer.New(p, 65535)
	defer buf.Close()
	n, err := buf.Read(nil)
	require.Nil(t, err)
	require.Equal(t, 0, n)
	result := make([]byte, 5)
	_, err = buf.Read(result)
	require.Nil(t, err)
	// There are some remaining data in the buf that have not been read.
	require.NotNil(t, buf.Next())
}
