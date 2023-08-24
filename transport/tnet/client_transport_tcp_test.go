// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/tnet"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
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

func TestClientUDP(t *testing.T) {
	// UDP is not supported, but it will switch to gonet default transport to roundtrip.
	startClientTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(addr string) {
			rsp, err := tnetRequest(
				context.Background(),
				helloWorld,
				transport.WithDialAddress(addr),
				transport.WithDialNetwork("udp"))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
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

func TestClientTCP_Multiplex(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			req := helloWorld
			ctx, msg := codec.EnsureMessage(context.Background())
			reqFrame, err := trpc.DefaultClientCodec.Encode(codec.Message(ctx), req)
			assert.Nil(t, err)

			cliOpts := getRoundTripOption(
				transport.WithDialAddress(addr),
				transport.WithMultiplexed(true),
				transport.WithMsg(msg),
			)
			clientTrans := tnettrans.NewClientTransport()
			rspFrame, err := clientTrans.RoundTrip(ctx, reqFrame, cliOpts...)
			assert.Nil(t, err)
			rsp, err := trpc.DefaultClientCodec.Decode(msg, rspFrame)
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

func TestClientTCP_HealthCheck(t *testing.T) {
	addr := getAddr()
	s := transport.NewServerTransport()
	serveOpts := getListenServeOption(transport.WithListenAddress(addr))
	err := s.ListenAndServe(context.Background(), serveOpts...)
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
	serveOpts := getListenServeOption(
		transport.WithListenAddress(addr),
		transport.WithHandler(handler),
	)
	serveOpts = append(serveOpts, svrCustomOpts...)
	err := s.ListenAndServe(context.Background(), serveOpts...)
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

func (p *customPool) Get(network string, address string, opts connpool.GetOptions) (net.Conn, error) {
	c, err := tnet.DialTCP(network, address, opts.DialTimeout)
	if err != nil {
		return nil, err
	}
	return &customConn{Conn: c, framer: opts.FramerBuilder.New(c)}, nil
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
