// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	. "trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport/tnet"
)

var (
	helloworld = []byte("hello world")
	reqID      uint32
)

var (
	_ (FrameParser) = (*simpleFrameParser)(nil)
)

/*
|   4 byte  |  4 byte  | bodyLen byte |
|  bodyLen  |    id    |      body    |
*/
type simpleFrameParser struct {
	isParseFail bool
}

func (fr *simpleFrameParser) Parse(reader io.Reader) (uint32, []byte, error) {
	head := make([]byte, 8)
	n, err := io.ReadFull(reader, head)
	if err != nil {
		return 0, nil, err
	}

	if fr.isParseFail {
		return 0, nil, errors.New("decode fail")
	}

	if n != 8 {
		return 0, nil, errors.New("invalid read full num")
	}

	bodyLen := binary.BigEndian.Uint32(head[:4])
	id := binary.BigEndian.Uint32(head[4:8])
	body := make([]byte, int(bodyLen))

	n, err = io.ReadFull(reader, body)
	if err != nil {
		return 0, nil, err
	}

	if n != int(bodyLen) {
		return 0, nil, errors.New("invalid read full body")
	}

	return id, body, nil
}

func encodeFrame(id uint32, body []byte) []byte {
	bodyLen := len(body)
	buf := bytes.NewBuffer(make([]byte, 0, 8+bodyLen))
	if err := binary.Write(buf, binary.BigEndian, uint32(bodyLen)); err != nil {
		panic(err)
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(id)); err != nil {
		panic(err)
	}

	if _, err := buf.Write(body); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func getReqID() uint32 {
	return atomic.AddUint32(&reqID, 1)
}

func echo(c net.Conn) {
	io.Copy(c, c)
}

func beginTCPServer(t *testing.T, handle func(net.Conn), tlsConfig *tls.Config) (net.Addr, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	addrCh := make(chan net.Addr, 1)
	go func() {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.Nil(t, err)
		if tlsConfig != nil {
			l = tls.NewListener(l, tlsConfig)
		}
		addrCh <- l.Addr()
		go func() {
			for {
				conn, err := l.Accept()
				if err != nil {
					require.NotNil(t, ctx.Err())
					return
				}
				go handle(conn)
			}
		}()
		<-ctx.Done()
		l.Close()
	}()
	addr := <-addrCh
	return addr, cancel
}

func TestMultiplexedBasic(t *testing.T) {
	testcases := []struct {
		name     string
		begin    func(*testing.T, func(net.Conn), *tls.Config) (net.Addr, context.CancelFunc)
		dialFunc DialFunc
	}{
		{name: "TCP_Tnet", begin: beginTCPServer, dialFunc: tnet.DialMultiplexConn},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			addr, cancel := tt.begin(t, echo, nil)
			t.Cleanup(cancel)

			getOpts := func() (uint32, GetOptions) {
				id := getReqID()
				opts := NewGetOptions()
				opts.WithFrameParser(&simpleFrameParser{})
				opts.WithVID(id)
				return id, opts
			}
			t.Run("Multiple Conns Concurrent Read Write", func(t *testing.T) {
				pool := NewPool(
					tt.dialFunc,
					WithEnableMetrics(),
					WithMaxConcurrentVirtualConnsPerConn(500),
				)
				var wg sync.WaitGroup
				for i := 0; i < 100; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for i := 0; i < 100; i++ {
							id, opts := getOpts()
							conn, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
							require.Nil(t, err)

							err = conn.Write(encodeFrame(id, append(helloworld, byte(id))))
							require.Nil(t, err)
							b, err := conn.Read()
							require.Nil(t, err)
							require.Equal(t, append(helloworld, byte(id)), b)
							conn.Close()
						}
					}()
				}
				wait := make(chan struct{}, 1)
				go func() {
					wg.Wait()
					wait <- struct{}{}
				}()
				select {
				case <-time.After(time.Second):
					require.FailNow(t, "Some responses are missed")
				case <-wait:
				}
			})
		})
	}
}

func TestTLS(t *testing.T) {
	tlsConf, err := itls.GetServerConfig(
		"../../testdata/ca.pem",
		"../../testdata/server.crt",
		"../../testdata/server.key",
	)
	require.Nil(t, err)
	addr, cancel := beginTCPServer(t, echo, tlsConf)
	defer cancel()

	getOpts := func() (uint32, GetOptions) {
		id := getReqID()
		opts := NewGetOptions()
		opts.WithFrameParser(&simpleFrameParser{})
		opts.WithVID(id)
		opts.WithDialTLS(
			"../../testdata/client.crt",
			"../../testdata/client.key",
			"../../testdata/ca.pem",
			"localhost",
		)
		return id, opts
	}
	t.Run("Multiple Conns Concurrent Read Write", func(t *testing.T) {
		pool := NewPool(
			NewDialFunc(WithDropFull(), WithSendingQueueSize(100)),
			WithEnableMetrics(),
			WithMaxConcurrentVirtualConnsPerConn(500),
		)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					id, opts := getOpts()
					conn, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
					require.Nil(t, err)

					err = conn.Write(encodeFrame(id, append(helloworld, byte(id))))
					require.Nil(t, err)
					b, err := conn.Read()
					require.Nil(t, err)
					require.Equal(t, append(helloworld, byte(id)), b)
					conn.Close()
				}
			}()
		}
		wait := make(chan struct{}, 1)
		go func() {
			wg.Wait()
			wait <- struct{}{}
		}()
		select {
		case <-time.After(time.Second):
			require.FailNow(t, "Some responses are missed")
		case <-wait:
		}
	})
}

func TestMultiplexedGetConnection(t *testing.T) {
	testcases := []struct {
		name     string
		begin    func(*testing.T, func(net.Conn), *tls.Config) (net.Addr, context.CancelFunc)
		dialFunc DialFunc
	}{
		{name: "TCP_Tnet", begin: beginTCPServer, dialFunc: tnet.DialMultiplexConn},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			addr, cancel := tt.begin(t, echo, nil)
			t.Cleanup(cancel)

			pool := NewPool(tt.dialFunc)
			getOpts := func() GetOptions {
				opts := NewGetOptions()
				opts.WithFrameParser(&simpleFrameParser{})
				opts.WithVID(getReqID())
				return opts
			}

			t.Run("Get Once", func(t *testing.T) {
				opts := getOpts()
				conn, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
				require.Nil(t, err)
				conn.Close()
			})
			t.Run("Get Multiple Succeed", func(t *testing.T) {
				opts := getOpts()
				conn, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
				require.Nil(t, err)
				conn.Close()
				localAddr := conn.LocalAddr()
				for i := 0; i < 9; i++ {
					opts := getOpts()
					conn, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
					require.Nil(t, err)
					require.Equal(t, localAddr, conn.LocalAddr())
					conn.Close()
				}
			})
			t.Run("Exceed MaxConcurrentVirtualConns", func(t *testing.T) {
				pool := NewPool(tt.dialFunc, WithMaxConcurrentVirtualConnsPerConn(1))

				opts := getOpts()
				c1, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
				require.Nil(t, err)
				defer c1.Close()

				opts = getOpts()
				c2, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
				require.Nil(t, err)
				require.NotEqual(t, c1.LocalAddr(), c2.LocalAddr())
				defer c2.Close()
			})
			t.Run("Request ID Already Exist", func(t *testing.T) {
				opts := getOpts()
				c1, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
				require.Nil(t, err)
				defer c1.Close()

				_, err = pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
				require.Equal(t, ErrDuplicateID, err)
			})
			t.Run("Empty FrameParser", func(t *testing.T) {
				opts := getOpts()
				opts.WithFrameParser(nil)
				_, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
				require.ErrorIs(t, err, ErrFrameParserNil)
			})
		})
	}
}

func TestMultiplexedDial(t *testing.T) {
	testcases := []struct {
		name     string
		dialFunc DialFunc
		wantErr  error
	}{
		{name: "DialSucceed_Tnet", dialFunc: tnet.DialMultiplexConn, wantErr: nil},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			addr, cancel := beginTCPServer(t, echo, nil)
			t.Cleanup(cancel)

			getOpts := func() (context.Context, context.CancelFunc, GetOptions) {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(200*time.Millisecond))
				opts := NewGetOptions()
				opts.WithFrameParser(&simpleFrameParser{})
				opts.WithVID(getReqID())
				return ctx, cancel, opts
			}

			pool := NewPool(tt.dialFunc)
			ctx, cancel, opts := getOpts()
			defer cancel()
			conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
			require.Equal(t, tt.wantErr, err)
			if conn != nil {
				conn.Close()
			}
		})
	}
}

func TestMultiplexedServerCloseConnAfterAccept(t *testing.T) {
	testcases := []struct {
		name     string
		dialFunc DialFunc
	}{
		{name: "TCP_Tnet", dialFunc: tnet.DialMultiplexConn},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			addr, cancel := beginTCPServer(t, func(c net.Conn) {
				c.Close()
			}, nil)
			t.Cleanup(cancel)

			pool := NewPool(tt.dialFunc)
			getOpts := func() (uint32, GetOptions) {
				id := getReqID()
				opts := NewGetOptions()
				opts.WithFrameParser(&simpleFrameParser{})
				opts.WithVID(id)
				return id, opts
			}

			var wg sync.WaitGroup
			for i := 0; i < 1; i++ {
				wg.Add(1)
				_, opts := getOpts()
				go func() {
					defer wg.Done()
					conn, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
					if err != nil {
						return
					}
					_, err = conn.Read()
					require.Contains(t, err.Error(), ErrConnClosed.Error())
					err = conn.Write(nil)
					require.Contains(t, err.Error(), ErrConnClosed.Error())
					conn.Close()
				}()
			}
			wg.Wait()
		})
	}

}

func TestMultiplexedDecodeFail(t *testing.T) {
	testcases := []struct {
		name       string
		begin      func(*testing.T, func(net.Conn), *tls.Config) (net.Addr, context.CancelFunc)
		dialFunc   DialFunc
		wantErrMsg string
	}{
		{name: "TCP_Tnet", begin: beginTCPServer, dialFunc: tnet.DialMultiplexConn, wantErrMsg: "decode fail"},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			addr, cancel := tt.begin(t, echo, nil)
			t.Cleanup(cancel)

			pool := NewPool(tt.dialFunc)
			getOpts := func() (uint32, GetOptions) {
				id := getReqID()
				opts := NewGetOptions()
				opts.WithFrameParser(&simpleFrameParser{})
				opts.WithVID(id)
				return id, opts
			}
			// return error when decode fail.
			fp := &simpleFrameParser{isParseFail: true}
			for i := 0; i < 5; i++ {
				id, opts := getOpts()
				opts.WithFrameParser(fp)
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
				require.Nil(t, err)

				err = conn.Write(encodeFrame(id, helloworld))
				require.Nil(t, err)
				_, err = conn.Read()
				require.Contains(t, err.Error(), tt.wantErrMsg)
				conn.Close()
				cancel()
			}
			// return nil when decode succeed.
			for i := 0; i < 5; i++ {
				id, opts := getOpts()
				fp.isParseFail = false
				opts.WithFrameParser(fp)
				conn, err := pool.GetVirtualConn(context.Background(), addr.Network(), addr.String(), opts)
				require.Nil(t, err)

				err = conn.Write(encodeFrame(id, helloworld))
				require.Nil(t, err)
				_, err = conn.Read()
				require.Nil(t, err)
				conn.Close()
			}
		})
	}
}
