// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package multiplex_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport/tnet"
	"trpc.group/trpc-go/trpc-go/transport/tnet/multiplex"
)

var (
	helloworld = []byte("hello world")
	reqID      uint32
)

var (
	_ (multiplexed.FrameParser) = (*simpleFrameParser)(nil)
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

func beginServer(t *testing.T, handle func(net.Conn)) (net.Addr, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	addrCh := make(chan net.Addr, 1)
	go func() {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.Nil(t, err)
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

func TestBasic(t *testing.T) {
	addr, cancel := beginServer(t, echo)
	defer cancel()

	getOpts := func() (uint32, multiplexed.GetOptions) {
		id := getReqID()
		opts := multiplexed.NewGetOptions()
		opts.WithFrameParser(&simpleFrameParser{})
		opts.WithVID(id)
		return id, opts
	}

	t.Run("Multiple Conns Concurrent Read Write", func(t *testing.T) {
		pool := multiplex.NewPool(
			tnet.Dial,
			multiplex.WithEnableMetrics(),
			multiplex.WithMaxConcurrentVirConnsPerConn(500),
		)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					id, opts := getOpts()
					conn, err := pool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
					require.Nil(t, err)

					err = conn.Write(encodeFrame(id, helloworld))
					require.Nil(t, err)
					b, err := conn.Read()
					require.Nil(t, err)
					require.Equal(t, helloworld, b)
					conn.Close()
				}
			}()
		}
		wg.Wait()
	})
}

func TestGetConnection(t *testing.T) {
	addr, cancel := beginServer(t, echo)
	defer cancel()
	muxPool := multiplex.NewPool(tnet.Dial)

	getOpts := func() multiplexed.GetOptions {
		opts := multiplexed.NewGetOptions()
		opts.WithFrameParser(&simpleFrameParser{})
		opts.WithVID(getReqID())
		return opts
	}

	t.Run("Get Once", func(t *testing.T) {
		opts := getOpts()
		conn, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		conn.Close()
	})
	t.Run("Get Multiple Succeed", func(t *testing.T) {
		opts := getOpts()
		conn, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		conn.Close()
		localAddr := conn.LocalAddr()
		for i := 0; i < 9; i++ {
			opts := getOpts()
			conn, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
			require.Nil(t, err)
			require.Equal(t, localAddr, conn.LocalAddr())
			conn.Close()
		}
	})
	t.Run("Exceed MaxConcurrentVirConns", func(t *testing.T) {
		muxPool := multiplex.NewPool(tnet.Dial, multiplex.WithMaxConcurrentVirConnsPerConn(1))

		opts := getOpts()
		c1, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		defer c1.Close()

		opts = getOpts()
		c2, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		require.NotEqual(t, c1.LocalAddr(), c2.LocalAddr())
		defer c2.Close()
	})
	t.Run("Request ID Already Exist", func(t *testing.T) {
		opts := getOpts()
		c1, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		defer c1.Close()

		_, err = muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
		require.Equal(t, multiplex.ErrDuplicateID, err)
	})
	t.Run("Empty FrameParser", func(t *testing.T) {
		opts := getOpts()
		opts.WithFrameParser(nil)
		_, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
		require.Contains(t, "frame parser is not provided", err.Error())
	})
}

func TestDial(t *testing.T) {
	addr, cancel := beginServer(t, echo)
	defer cancel()

	getOpts := func() (context.Context, context.CancelFunc, multiplexed.GetOptions) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(200*time.Millisecond))
		opts := multiplexed.NewGetOptions()
		opts.WithFrameParser(&simpleFrameParser{})
		opts.WithVID(getReqID())
		return ctx, cancel, opts
	}

	t.Run("Dial Succeed", func(t *testing.T) {
		muxPool := multiplex.NewPool(tnet.Dial)
		ctx, cancel, opts := getOpts()
		defer cancel()
		conn, err := muxPool.GetMuxConn(ctx, addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		conn.Close()
	})
	t.Run("Dial Timeout", func(t *testing.T) {
		muxPool := multiplex.NewPool(func(opts *connpool.DialOptions) (net.Conn, error) {
			time.Sleep(time.Second)
			return nil, errors.New("dial fail")
		})
		ctx, cancel, opts := getOpts()
		defer cancel()
		_, err := muxPool.GetMuxConn(ctx, addr.Network(), addr.String(), opts)
		require.Equal(t, context.DeadlineExceeded, err)
	})
	t.Run("Dial Error", func(t *testing.T) {
		muxPool := multiplex.NewPool(func(opts *connpool.DialOptions) (net.Conn, error) {
			return nil, errors.New("dial error")
		})
		ctx, cancel, opts := getOpts()
		defer cancel()
		_, err := muxPool.GetMuxConn(ctx, addr.Network(), addr.String(), opts)
		require.Equal(t, errors.New("dial error"), err)
	})
	t.Run("Dial Gonet", func(t *testing.T) {
		muxPool := multiplex.NewPool(func(opts *connpool.DialOptions) (net.Conn, error) {
			return net.Dial(opts.Network, opts.Address)
		})
		ctx, cancel, opts := getOpts()
		defer cancel()
		_, err := muxPool.GetMuxConn(ctx, addr.Network(), addr.String(), opts)
		require.Contains(t, "dialed connection must implements tnet.Conn", err.Error())
	})
}

func TestClose(t *testing.T) {
	muxPool := multiplex.NewPool(tnet.Dial)
	getOpts := func() (uint32, multiplexed.GetOptions) {
		id := getReqID()
		opts := multiplexed.NewGetOptions()
		opts.WithFrameParser(&simpleFrameParser{})
		opts.WithVID(id)
		return id, opts
	}

	t.Run("Server Close Conn After Accept", func(t *testing.T) {
		addr, cancel := beginServer(t, func(c net.Conn) {
			c.Close()
		})
		defer cancel()
		var wg sync.WaitGroup
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			_, opts := getOpts()
			go func() {
				defer wg.Done()
				conn, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
				if err != nil {
					return
				}
				_, err = conn.Read()
				require.Contains(t, err.Error(), multiplex.ErrConnClosed.Error())
				err = conn.Write(nil)
				require.Contains(t, err.Error(), multiplex.ErrConnClosed.Error())
				conn.Close()
			}()
		}
		wg.Wait()
	})

	t.Run("Decode Fail", func(t *testing.T) {
		addr, cancel := beginServer(t, echo)
		defer cancel()
		// return error when decode fail.
		for i := 0; i < 5; i++ {
			id, opts := getOpts()
			opts.WithFrameParser(&simpleFrameParser{isParseFail: true})
			conn, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
			require.Nil(t, err)

			err = conn.Write(encodeFrame(id, helloworld))
			require.Nil(t, err)
			_, err = conn.Read()
			require.Contains(t, err.Error(), "decode fail")
			conn.Close()
		}
		// return nil when decode succeed.
		for i := 0; i < 5; i++ {
			id, opts := getOpts()
			opts.WithFrameParser(&simpleFrameParser{})
			conn, err := muxPool.GetMuxConn(context.Background(), addr.Network(), addr.String(), opts)
			require.Nil(t, err)

			err = conn.Write(encodeFrame(id, helloworld))
			require.Nil(t, err)
			_, err = conn.Read()
			require.Nil(t, err)
			conn.Close()
		}
	})
}
