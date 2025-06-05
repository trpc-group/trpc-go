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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package multiplex_test

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
	"trpc.group/trpc-go/trpc-go/codec"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport/tnet"
	tnetmultiplexed "trpc.group/trpc-go/trpc-go/transport/tnet/multiplex"
)

var (
	helloworld = []byte("hello world")
	reqID      uint32
)

var (
	_ (codec.FramerBuilder) = (*simpleFramer)(nil)
	_ (codec.Framer)        = (*simpleFramer)(nil)
	_ (codec.Decoder)       = (*simpleFramer)(nil)
)

/*
|   4 byte  |  4 byte  | bodyLen byte |
|  bodyLen  |    id    |      body    |
*/
type simpleFramer struct {
	reader       io.Reader
	isDecodeFail bool
	safe         bool
}

func (fr *simpleFramer) New(reader io.Reader) codec.Framer {
	return &simpleFramer{
		reader:       reader,
		isDecodeFail: fr.isDecodeFail,
		safe:         fr.safe,
	}
}

func (fr *simpleFramer) ReadFrame() ([]byte, error) {
	return nil, errors.New("not implements")
}

func (fr *simpleFramer) IsSafe() bool {
	return fr.safe
}

func (fr *simpleFramer) Decode() (codec.TransportResponseFrame, error) {
	head := make([]byte, 8)
	n, err := io.ReadFull(fr.reader, head)
	if err != nil {
		return nil, err
	}

	if fr.isDecodeFail {
		return nil, errors.New("decode fail")
	}

	if n != 8 {
		return nil, errors.New("invalid read full num")
	}

	bodyLen := binary.BigEndian.Uint32(head[:4])
	id := binary.BigEndian.Uint32(head[4:8])
	body := make([]byte, int(bodyLen))

	n, err = io.ReadFull(fr.reader, body)
	if err != nil {
		return nil, err
	}

	if n != int(bodyLen) {
		return nil, errors.New("invalid read full body")
	}

	return &simpleFrame{
		id:   id,
		body: body,
	}, nil
}

func (fr *simpleFramer) UpdateMsg(interface{}, codec.Msg) error {
	return nil
}

var _ (codec.TransportResponseFrame) = (*simpleFrame)(nil)

type simpleFrame struct {
	id   uint32
	body []byte
}

func (f *simpleFrame) GetRequestID() uint32 {
	return f.id
}

func (f *simpleFrame) GetResponseBuf() []byte {
	return f.body
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

func beginServer(t *testing.T, handle func(net.Conn), tlsConfig *tls.Config) (net.Addr, context.CancelFunc) {
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

func TestBasic(t *testing.T) {
	addr, cancel := beginServer(t, echo, nil)
	defer cancel()

	getOpts := func() (context.Context, uint32, multiplexed.GetOptions) {
		ctx, msg := codec.EnsureMessage(context.Background())
		id := getReqID()
		msg.WithRequestID(id)
		opts := multiplexed.NewGetOptions()
		opts.WithFramerBuilder(&simpleFramer{})
		opts.WithMsg(msg)
		return ctx, id, opts
	}

	t.Run("Mutiple Conns Concurrent Read Write", func(t *testing.T) {
		pool := tnetmultiplexed.NewPool(
			tnet.Dial,
			tnetmultiplexed.WithEnableMetrics(),
			tnetmultiplexed.WithMaxConcurrentVirtualConnsPerConn(500),
		)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					ctx, id, opts := getOpts()
					conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
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
		wg.Wait()
	})
	time.Sleep(time.Second * 3)
}

func TestTLS(t *testing.T) {
	tlsConf, err := itls.GetServerConfig(
		"../../../testdata/ca.pem",
		"../../../testdata/server.crt",
		"../../../testdata/server.key",
	)
	require.Nil(t, err)
	addr, cancel := beginServer(t, echo, tlsConf)
	defer cancel()

	getOpts := func() (context.Context, uint32, multiplexed.GetOptions) {
		ctx, msg := codec.EnsureMessage(context.Background())
		id := getReqID()
		msg.WithRequestID(id)
		opts := multiplexed.NewGetOptions()
		opts.WithFramerBuilder(&simpleFramer{})
		opts.WithMsg(msg)
		opts.WithDialTLS(
			"../../../testdata/client.crt",
			"../../../testdata/client.key",
			"../../../testdata/ca.pem",
			"localhost",
		)
		return ctx, id, opts
	}
	t.Run("Mutiple Conns Concurrent Read Write", func(t *testing.T) {
		pool := tnetmultiplexed.NewPool(
			tnet.Dial,
			tnetmultiplexed.WithEnableMetrics(),
			tnetmultiplexed.WithMaxConcurrentVirtualConnsPerConn(500),
		)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					ctx, id, opts := getOpts()
					conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
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
		require.Eventually(t,
			func() bool {
				wg.Wait()
				return true
			}, time.Second, 200*time.Millisecond,
			"Some responses are missed",
		)
	})
}

func TestGetConnection(t *testing.T) {
	addr, cancel := beginServer(t, echo, nil)
	defer cancel()
	pool := tnetmultiplexed.NewPool(tnet.Dial)

	getOpts := func() (context.Context, multiplexed.GetOptions) {
		ctx, msg := codec.EnsureMessage(context.Background())
		msg.WithRequestID(getReqID())
		opts := multiplexed.NewGetOptions()
		opts.WithFramerBuilder(&simpleFramer{})
		opts.WithMsg(msg)
		return ctx, opts
	}

	t.Run("Get Once", func(t *testing.T) {
		ctx, opts := getOpts()
		conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		conn.Close()
	})
	t.Run("Get Multiple Succeed", func(t *testing.T) {
		ctx, opts := getOpts()
		conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		conn.Close()
		localAddr := conn.LocalAddr()
		for i := 0; i < 9; i++ {
			ctx, opts := getOpts()
			conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
			require.Nil(t, err)
			require.Equal(t, localAddr, conn.LocalAddr())
			conn.Close()
		}
	})
	t.Run("Exceed MaxConcurrentVirConns", func(t *testing.T) {
		pool := tnetmultiplexed.NewPool(tnet.Dial, tnetmultiplexed.WithMaxConcurrentVirtualConnsPerConn(1))

		ctx, opts := getOpts()
		c1, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		defer c1.Close()

		ctx, opts = getOpts()
		c2, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		require.NotEqual(t, c1.LocalAddr(), c2.LocalAddr())
		defer c2.Close()
	})
	t.Run("Request ID Already Exist", func(t *testing.T) {
		ctx, opts := getOpts()
		c1, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		defer c1.Close()

		_, err = pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Equal(t, tnetmultiplexed.ErrDuplicateID, err)
	})
	t.Run("Empty FramerBuilder", func(t *testing.T) {
		ctx, opts := getOpts()
		opts.WithFramerBuilder(nil)
		_, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Contains(t, "framer builder is not provided", err.Error())
	})
}

func TestDial(t *testing.T) {
	addr, cancel := beginServer(t, echo, nil)
	defer cancel()

	getOpts := func() (context.Context, context.CancelFunc, multiplexed.GetOptions) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(200*time.Millisecond))
		ctx, msg := codec.EnsureMessage(ctx)
		msg.WithRequestID(getReqID())
		opts := multiplexed.NewGetOptions()
		opts.WithFramerBuilder(&simpleFramer{})
		opts.WithMsg(msg)
		return ctx, cancel, opts
	}

	t.Run("Dial Succeed", func(t *testing.T) {
		pool := tnetmultiplexed.NewPool(tnet.Dial)
		ctx, cancel, opts := getOpts()
		defer cancel()
		conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Nil(t, err)
		conn.Close()
	})
	t.Run("Dial Timeout", func(t *testing.T) {
		pool := tnetmultiplexed.NewPool(func(opts *connpool.DialOptions) (net.Conn, error) {
			time.Sleep(time.Second)
			return nil, errors.New("dial fail")
		})
		ctx, cancel, opts := getOpts()
		defer cancel()
		_, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Equal(t, context.DeadlineExceeded, err)
	})
	t.Run("Dial Error", func(t *testing.T) {
		pool := tnetmultiplexed.NewPool(func(opts *connpool.DialOptions) (net.Conn, error) {
			return nil, errors.New("dial error")
		})
		ctx, cancel, opts := getOpts()
		defer cancel()
		_, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Equal(t, errors.New("dial error"), err)
	})
	t.Run("Dial Gonet", func(t *testing.T) {
		pool := tnetmultiplexed.NewPool(func(opts *connpool.DialOptions) (net.Conn, error) {
			return net.Dial(opts.Network, opts.Address)
		})
		ctx, cancel, opts := getOpts()
		defer cancel()
		_, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
		require.Contains(t, err.Error(), "does't implements tnet.Conn or tnet/tls.Conn")
	})
}

func TestClose(t *testing.T) {
	pool := tnetmultiplexed.NewPool(tnet.Dial)
	getOpts := func() (context.Context, uint32, multiplexed.GetOptions) {
		ctx, msg := codec.EnsureMessage(context.Background())
		id := getReqID()
		msg.WithRequestID(id)
		opt := multiplexed.NewGetOptions()
		opt.WithFramerBuilder(&simpleFramer{})
		opt.WithMsg(msg)
		return ctx, id, opt
	}

	t.Run("Server Close Conn After Accept", func(t *testing.T) {
		addr, cancel := beginServer(t, func(c net.Conn) {
			c.Close()
		}, nil)
		defer cancel()
		var wg sync.WaitGroup
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			ctx, _, opts := getOpts()
			go func() {
				defer wg.Done()
				conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
				if err != nil {
					return
				}
				_, err = conn.Read()
				require.Contains(t, err.Error(), tnetmultiplexed.ErrConnClosed.Error())
				err = conn.Write(nil)
				require.Contains(t, err.Error(), tnetmultiplexed.ErrConnClosed.Error())
				conn.Close()
			}()
		}
		wg.Wait()
	})

	t.Run("Decode Fail", func(t *testing.T) {
		addr, cancel := beginServer(t, echo, nil)
		defer cancel()
		// return error when decode fail.
		for i := 0; i < 5; i++ {
			ctx, id, opts := getOpts()
			opts.WithFramerBuilder(&simpleFramer{isDecodeFail: true})
			conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
			require.Nil(t, err)

			err = conn.Write(encodeFrame(id, helloworld))
			require.Nil(t, err)
			_, err = conn.Read()
			require.Contains(t, err.Error(), "decode fail")
			conn.Close()
		}
		// return nil when decode succeed.
		for i := 0; i < 5; i++ {
			ctx, id, opts := getOpts()
			connpool.WithFramerBuilder(&simpleFramer{})
			conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
			require.Nil(t, err)

			err = conn.Write(encodeFrame(id, helloworld))
			require.Nil(t, err)
			_, err = conn.Read()
			require.Nil(t, err)
			conn.Close()
		}
	})
}
