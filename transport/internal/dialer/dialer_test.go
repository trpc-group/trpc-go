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

// dialer provides common function for transport to dial.
package dialer_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/transport/internal/dialer"
)

type conn struct{}

func (c *conn) Read(b []byte) (int, error) { return 0, nil }

func (c *conn) Write(b []byte) (int, error) { return 0, nil }

func (c *conn) Close() error { return nil }

func (c *conn) LocalAddr() net.Addr { return nil }

func (c *conn) RemoteAddr() net.Addr { return nil }

func (c *conn) SetDeadline(t time.Time) error { return nil }

func (c *conn) SetReadDeadline(t time.Time) error { return nil }

func (c *conn) SetWriteDeadline(t time.Time) error { return nil }

func TestDialTCP(t *testing.T) {
	t.Run("Context has a shorter timeout than Dial", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		dialer.DialTCP(ctx, dialer.DialOptions{
			DisableConnectionPool: true,
			DialTimeout:           500 * time.Millisecond,
			Dial: func(opts *connpool.DialOptions) (net.Conn, error) {
				require.LessOrEqual(t, opts.Timeout, 100*time.Millisecond)
				return &conn{}, nil
			},
		})
	})
	t.Run("Context has a longer timeout than Dial", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		dialer.DialTCP(ctx, dialer.DialOptions{
			DialTimeout: 100 * time.Millisecond,
			Pool: connpool.NewConnectionPool(
				connpool.WithDialFunc(
					func(opts *connpool.DialOptions) (net.Conn, error) {
						require.LessOrEqual(t, opts.Timeout, 100*time.Millisecond)
						return &conn{}, nil
					},
				),
			),
		})
	})
}

func TestValidateContext(t *testing.T) {
	bgCtx := context.Background()
	t.Run("Valid context", func(t *testing.T) {
		_, err := dialer.DialTCP(bgCtx, dialer.DialOptions{
			DisableConnectionPool: true,
			Dial: func(opts *connpool.DialOptions) (net.Conn, error) {
				return &conn{}, nil
			},
		})
		require.Nil(t, err)
	})

	t.Run("Canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(bgCtx)
		cancel()
		_, err := dialer.DialTCP(ctx, dialer.DialOptions{})
		require.Equal(t, errs.RetClientCanceled, errs.Code(err))
	})

	t.Run("Timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(bgCtx, time.Millisecond)
		defer cancel()
		time.Sleep(100 * time.Millisecond)
		_, err := dialer.DialTCP(ctx, dialer.DialOptions{})
		require.Equal(t, errs.RetClientTimeout, errs.Code(err))
	})
}

func TestDialUDP(t *testing.T) {
	const network = "udp"
	ln, err := net.ListenPacket(network, "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	go func() {
		const size = 1024
		buf := make([]byte, size)
		for {
			n, addr, err := ln.ReadFrom(buf)
			if err != nil {
				t.Logf("ln.ReadFrom err: %+v\n", err)
				return
			}
			_, err = ln.WriteTo(buf[:n], addr)
			if err != nil {
				t.Logf("ln.WriteTo err: %+v\n", err)
				return
			}
		}
	}()
	t.Run("normal: mode = connected", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _, err = dialer.DialUDP(ctx, dialer.DialOptions{
			Network:        network,
			Address:        ln.LocalAddr().String(),
			LocalAddr:      "127.0.0.1:0",
			DialUDP:        dialer.DefaultDialUDP,
			ConnectionMode: dialer.Connected,
		})
		require.Nil(t, err)
	})
	t.Run("normal: mode = not connected", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _, err = dialer.DialUDP(ctx, dialer.DialOptions{
			Network:        network,
			Address:        ln.LocalAddr().String(),
			LocalAddr:      "127.0.0.1:0",
			DialUDP:        dialer.DefaultDialUDP,
			ConnectionMode: dialer.NotConnected,
		})
		require.Nil(t, err)
	})
	t.Run("dial timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _, err = dialer.DialUDP(ctx, dialer.DialOptions{
			Network:        network,
			Address:        ln.LocalAddr().String(),
			LocalAddr:      "127.0.0.1:0",
			DialUDP:        dialer.DefaultDialUDP,
			DialTimeout:    time.Microsecond,
			ConnectionMode: dialer.Connected,
		})
		require.Equal(t, errs.RetClientConnectFail, errs.Code(err), "err: %+v", err)
	})
}
