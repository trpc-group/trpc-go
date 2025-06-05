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

package tnet_test

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/tnet"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/transport"
	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

func TestClientTCP(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			rsp, err := tnetRequest(ctx, helloWorld,
				transport.WithDialAddress(addr),
				transport.WithDialTimeout(500*time.Millisecond),
			)
			assert.Equal(t, helloWorld, rsp)
			assert.Nil(t, err)
		},
	)
}

func TestClientTCP_NoFrameBuilder(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, err := tnetRequest(ctx, helloWorld,
				transport.WithDialAddress(addr),
				transport.WithClientFramerBuilder(nil),
			)
			assert.Equal(t, errs.RetClientConnectFail, errs.Code(err))
		},
	)
}

func TestClientTCP_CtxErr(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			// canceled context error
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := tnetRequest(ctx, helloWorld,
				transport.WithDialAddress(addr),
			)
			assert.Equal(t, errs.RetClientCanceled, errs.Code(err))

			// timeout context error
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(time.Nanosecond))
			defer cancel()
			time.Sleep(time.Nanosecond)
			_, err = tnetRequest(ctx, helloWorld,
				transport.WithDialAddress(addr),
			)
			assert.Equal(t, errs.RetClientTimeout, errs.Code(err))
		},
	)
}

func TestClientTCP_DisableConnPool(t *testing.T) {
	// success case
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			rsp, err := tnetRequest(
				context.Background(),
				helloWorld,
				transport.WithDialAddress(addr),
				transport.WithDisableConnectionPool(),
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
	// dial wrong address
	_, err := tnetRequest(
		context.Background(),
		helloWorld,
		transport.WithDialAddress("0"),
		transport.WithDisableConnectionPool(),
	)
	assert.Equal(t, errs.RetClientConnectFail, errs.Code(err))
}

func TestClientTCP_ReadTimeout(t *testing.T) {
	startClientTest(
		t,
		func(ctx context.Context, req []byte) ([]byte, error) {
			time.Sleep(time.Hour)
			return nil, nil
		},
		nil,
		func(addr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, err := tnetRequest(
				ctx,
				helloWorld,
				transport.WithDialAddress(addr),
			)
			assert.Equal(t, errs.RetClientTimeout, errs.Code(err))
		},
	)
}

func TestClientTCP_IdleTimeout(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			p := tnettrans.NewConnectionPool(
				// Limit the number of connections to 1 to test the idle timeout.
				connpool.WithMaxActive(1),
				// Set the idle timeout to 1 second. If the timeout is too small,
				// it might result in an error due to a short delay time.
				connpool.WithIdleTimeout(time.Second),
			)
			// If the only idle connection reaches the timeout, we should not be able
			// to obtain any connection from the pool.
			assert.NotNil(t, p)

			// Get a connection from the pool. The third parameter timeout is not used
			// in the pool's implementation, so we can pass any values.
			pc, err := p.Get("tcp", addr, 0)
			assert.Nil(t, err)

			// In the wrong version of the code, the connection that has already been obtained
			// will be closed as well if it is not used for more than the idle timeout. However,
			// in the fixed version, the connection should still be able to write data to the server
			// even if we have slept for a time longer than the idle timeout.
			time.Sleep(2 * time.Second)
			n, err := pc.Write(helloWorld)
			assert.Nil(t, err)
			assert.Equal(t, len(helloWorld), n)

			// Close the connection pool.
			err = pc.Close()
			assert.Nil(t, err)
		},
	)
}

func TestClientTCP_CustomPool(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			rsp, err := tnetRequest(
				ctx,
				helloWorld,
				transport.WithDialAddress(addr),
				transport.WithDialPool(&customPool{}),
			)
			assert.Equal(t, helloWorld, rsp)
			assert.Nil(t, err)
		},
	)
}

func TestClientUnix(t *testing.T) {
	// Unix socket is not supported, but it will switch to gonet default transport to roundtrip.
	unixAddr := "/tmp/server.sock"
	os.Remove(unixAddr)
	startClientTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{
			transport.WithListenAddress(unixAddr),
			transport.WithListenNetwork("unix"),
		},
		func(addr string) {
			rsp, err := tnetRequest(
				context.Background(),
				helloWorld,
				transport.WithDialAddress(unixAddr),
				transport.WithDialNetwork("unix"))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)

}

func TestClientTCP_Multiplexed(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			req := helloWorld
			cliOpts := getRoundTripOption(
				transport.WithDialAddress(addr),
				transport.WithMultiplexed(true),
			)
			clientTrans := tnettrans.NewClientTransport()
			ctx, msg := codec.EnsureMessage(context.Background())
			msg.WithRequestID(0)
			reqbytes, err := trpc.DefaultClientCodec.Encode(msg, req)
			assert.Nil(t, err)

			rsp, err := clientTrans.RoundTrip(ctx, reqbytes, append(cliOpts, transport.WithMsg(msg))...)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestClientTCP_TLS(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithServeTLS("../../testdata/server.crt", "../../testdata/server.key", "../../testdata/ca.pem")},
		func(addr string) {
			rsp, err := tnetRequest(
				context.Background(),
				helloWorld,
				transport.WithDialAddress(addr),
				transport.WithDialTLS("../../testdata/client.crt", "../../testdata/client.key", "../../testdata/ca.pem", "localhost"),
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)

			rsp, err = tnetRequest(
				context.Background(),
				helloWorld,
				transport.WithDialAddress(addr),
				transport.WithDialTLS("../../testdata/client.crt", "../../testdata/client.key", "none", ""),
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestClientTCP_TLS_Multiplex(t *testing.T) {
	invokeTest := func(tlsOpt transport.RoundTripOption) {
		startClientTest(
			t,
			defaultServerHandle,
			[]transport.ListenServeOption{
				transport.WithServeTLS("../../testdata/server.crt", "../../testdata/server.key", "../../testdata/ca.pem")},
			func(addr string) {
				cliOpts := getRoundTripOption(
					transport.WithDialAddress(addr),
					transport.WithMultiplexed(true),
					tlsOpt,
				)
				clientTrans := tnettrans.NewClientTransport()
				ctx, msg := codec.EnsureMessage(context.Background())
				reqbytes, err := trpc.DefaultClientCodec.Encode(msg, helloWorld)
				assert.Nil(t, err)
				rsp, err := clientTrans.RoundTrip(ctx, reqbytes, append(cliOpts, transport.WithMsg(msg))...)
				assert.Nil(t, err)
				assert.Equal(t, helloWorld, rsp)
			},
		)
	}
	// Set CAFile and ServerName
	invokeTest(transport.WithDialTLS("../../testdata/client.crt", "../../testdata/client.key", "../../testdata/ca.pem", "localhost"))

	// None CAFile and no ServerName
	invokeTest(transport.WithDialTLS("../../testdata/client.crt", "../../testdata/client.key", "none", ""))
}

func TestClientTCP_HealthCheck(t *testing.T) {
	addr := getAddr()
	s := transport.NewServerTransport()
	serOpts := getListenServeOption(transport.WithListenAddress(addr))
	err := s.ListenAndServe(context.Background(), serOpts...)
	assert.Nil(t, err)

	c, err := net.Dial("tcp", addr)
	assert.Nil(t, err)
	assert.True(t, tnettrans.HealthChecker(&connpool.PoolConn{Conn: c}, true))

	c, err = tnet.DialTCP("tcp", addr, 0)
	assert.Nil(t, err)
	assert.True(t, tnettrans.HealthChecker(&connpool.PoolConn{Conn: c}, true))

	c.Close()
	assert.False(t, tnettrans.HealthChecker(&connpool.PoolConn{Conn: c}, true))
}

func TestNewConnectionPool(t *testing.T) {
	p := tnettrans.NewConnectionPool()
	assert.NotNil(t, p)

	customDialFuncErr := errors.New("custom dial func test")
	p = tnettrans.NewConnectionPool(
		connpool.WithDialFunc(
			func(opts *connpool.DialOptions) (net.Conn, error) {
				return nil, customDialFuncErr
			},
		),
	)
	assert.NotNil(t, p)
	_, err := p.Get("", "", 0)
	assert.NotNil(t, err)
	assert.True(t, errors.Is(err, customDialFuncErr))
}

func TestClientTCP_KeepOrderInvoke(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			sendError := make(chan error, 1)
			recvError := make(chan error, 1)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			ctx = keeporder.NewContextWithClientInfo(ctx, &keeporder.ClientInfo{SendError: sendError})
			ctx, msg := codec.EnsureMessage(ctx)
			defer cancel()
			reqbytes, err := trpc.DefaultClientCodec.Encode(msg, helloWorld)
			require.NoError(t, err)
			clientTrans := tnettrans.NewClientTransport()
			go func() {
				cliOpts := getRoundTripOption(
					transport.WithDialAddress(addr),
					transport.WithMultiplexed(true),
					transport.WithMsg(msg),
				)
				_, recvErr := clientTrans.RoundTrip(ctx, reqbytes, cliOpts...)
				recvError <- recvErr
			}()
			sendErr := <-sendError
			require.NoError(t, sendErr)
			recvErr := <-recvError
			require.NoError(t, recvErr)
		},
	)
}

func startClientTest(
	t *testing.T,
	serverHandle func(ctx context.Context, req []byte) ([]byte, error),
	svrCustomOpts []transport.ListenServeOption,
	clientHandle func(addr string),
) {
	addr := getAddr()
	s := transport.NewServerTransport()
	handler := newUserDefineHandler(func(ctx context.Context, req []byte) ([]byte, error) {
		return serverHandle(ctx, req)
	})
	serOpts := getListenServeOption(
		transport.WithListenAddress(addr),
		transport.WithHandler(handler),
	)
	serOpts = append(serOpts, svrCustomOpts...)
	err := s.ListenAndServe(context.Background(), serOpts...)
	assert.Nil(t, err)

	clientHandle(addr)
}

type customPool struct{}

type customConn struct {
	tnet.Conn
	framer codec.Framer
}

func (c *customConn) ReadFrame() ([]byte, error) {
	return c.framer.ReadFrame()
}

func (p *customPool) Get(network string, address string,
	timeout time.Duration, opts ...connpool.GetOption) (net.Conn, error) {
	option := &connpool.GetOptions{}
	for _, opt := range opts {
		opt(option)
	}
	c, err := tnet.DialTCP(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return &customConn{Conn: c, framer: option.FramerBuilder.New(c)}, nil
}

func tnetRequest(ctx context.Context, req []byte, opts ...transport.RoundTripOption) ([]byte, error) {
	ctx, _ = codec.EnsureMessage(ctx)
	reqbytes, err := trpc.DefaultClientCodec.Encode(
		codec.Message(ctx),
		req,
	)
	if err != nil {
		return nil, err
	}

	cliOpts := getRoundTripOption(opts...)
	clientTrans := tnettrans.NewClientTransport()
	rspbytes, err := clientTrans.RoundTrip(
		ctx,
		reqbytes,
		cliOpts...,
	)
	if err != nil {
		return nil, err
	}
	rsp, err := trpc.DefaultClientCodec.Decode(
		codec.Message(ctx),
		rspbytes,
	)
	return rsp, err
}
