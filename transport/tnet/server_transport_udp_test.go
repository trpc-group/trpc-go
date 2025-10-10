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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/transport"
	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

func TestServerUDP_ListenAndServe(t *testing.T) {
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(addr string) {
			rsp, err := gonetRequest(
				context.Background(),
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(addr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestServerUDP_UserDefineListener(t *testing.T) {
	serverAddr := getAddr()
	ln, err := net.ListenPacket("udp", serverAddr)
	assert.Nil(t, err)
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{
			transport.WithListenNetwork("udp"),
			transport.WithUDPListener(ln)},
		func(_ string) {
			rsp, err := gonetRequest(
				context.Background(),
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(serverAddr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestServerUDP_InvalidAddress(t *testing.T) {
	s := tnettrans.NewServerTransport()
	serOpts := getListenServeOption(
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress("invalid addr"),
	)
	err := s.ListenAndServe(context.Background(), serOpts...)
	assert.NotNil(t, err)
}

func TestServerUDP_HandleErr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	startServerTest(
		t,
		newUserDefineHandler(errServerHandle),
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(addr string) {
			_, err := gonetRequest(
				ctx,
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(addr),
				transport.WithDialTimeout(time.Millisecond))
			assert.NotNil(t, err)
		},
	)
}

func TestServerUDP_WriteFail(t *testing.T) {
	ch := make(chan struct{}, 1)
	var isHandled bool
	startServerTest(
		t,
		newUserDefineHandler(func(ctx context.Context, req []byte) ([]byte, error) {
			isHandled = true
			<-ch
			return nil, nil
		}),
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(addr string) {
			ctx, _ := codec.EnsureMessage(context.Background())
			req, err := trpc.DefaultClientCodec.Encode(codec.Message(ctx), helloWorld)
			assert.Nil(t, err)

			conn, err := tnet.DialUDP("udp", addr, 0)
			assert.Nil(t, err)
			_, err = conn.Write(req)
			assert.Nil(t, err)

			// sleep to make sure server received data
			time.Sleep(50 * time.Millisecond)
			conn.Close()
			// notify server write back data, but server will fail, because connection is closed
			ch <- struct{}{}
			_, err = conn.Read(make([]byte, 1))
			assert.NotNil(t, err)
			// make sure server run into handle
			assert.True(t, isHandled)
		},
	)
}

func TestServerUDP_ClientWrongReq(t *testing.T) {
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(addr string) {
			cliconn, err := tnet.DialUDP("udp", addr, 0)
			assert.Nil(t, err)
			_, err = cliconn.Write([]byte("1234567890123456"))
			assert.Nil(t, err)

			// sleep to make sure ListenAndServe run into onRequest()
			time.Sleep(50 * time.Millisecond)
			err = cliconn.Close()
			assert.Nil(t, err)
		},
	)
}

func TestServerUDP_Close(t *testing.T) {
	stopListening := make(chan struct{})
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{
			transport.WithListenNetwork("udp"),
			transport.WithStopListening(stopListening)},
		func(addr string) {
			rsp, err := gonetRequest(
				context.Background(),
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(addr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
	stopListening <- struct{}{}
	// sleep to make sure server close
	time.Sleep(50 * time.Millisecond)
}

func TestServerUDP_RPCZ(t *testing.T) {
	oldRPCZEnable := rpczenable.Enabled
	rpczenable.Enabled = true
	defer func() { rpczenable.Enabled = oldRPCZEnable }()
	t.Run("DefaultServerHandle", func(t *testing.T) {
		startServerTest(
			t,
			defaultUserDefineHandler,
			[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
			func(addr string) {
				rsp, err := gonetRequest(
					context.Background(),
					transport.WithDialNetwork("udp"),
					transport.WithDialAddress(addr))
				assert.Nil(t, err)
				assert.Equal(t, helloWorld, rsp)
			},
		)
	})
	t.Run("ErrServerHandle", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		startServerTest(
			t,
			newUserDefineHandler(errServerHandle),
			[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
			func(addr string) {
				_, err := gonetRequest(
					ctx,
					transport.WithDialNetwork("udp"),
					transport.WithDialAddress(addr))
				assert.NotNil(t, err)
			},
		)
	})
}

func TestServerUDP_WithMaxUDPPacketSize(t *testing.T) {
	addr := getAddr()
	s := tnettrans.NewServerTransport(
		tnettrans.WithKeepAlivePeriod(15*time.Second),
		tnettrans.WithReusePort(true),
		tnettrans.WithMaxUDPPacketSize(32767),
	)
	handler := newUserDefineHandler(func(ctx context.Context, req []byte) ([]byte, error) {
		return defaultServerHandle(ctx, req)
	})
	serOpts := getListenServeOption(
		transport.WithListenAddress(addr),
		transport.WithHandler(handler),
		transport.WithListenNetwork("udp"),
	)
	err := s.ListenAndServe(context.Background(), serOpts...)
	assert.Nil(t, err)
}

func TestServerUDP_WithServerExactUDPBufferSizeEnabled(t *testing.T) {
	addr := getAddr()
	s := tnettrans.NewServerTransport(
		tnettrans.WithKeepAlivePeriod(15*time.Second),
		tnettrans.WithReusePort(true),
		tnettrans.WithServerExactUDPBufferSizeEnabled(true),
	)
	handler := newUserDefineHandler(func(ctx context.Context, req []byte) ([]byte, error) {
		return defaultServerHandle(ctx, req)
	})
	serOpts := getListenServeOption(
		transport.WithListenAddress(addr),
		transport.WithHandler(handler),
		transport.WithListenNetwork("udp"),
	)
	err := s.ListenAndServe(context.Background(), serOpts...)
	assert.Nil(t, err)

	rsp, err := gonetRequest(
		context.Background(),
		transport.WithDialAddress(addr),
		transport.WithDialNetwork("udp"))
	assert.Nil(t, err)
	assert.Equal(t, helloWorld, rsp)
}

func TestServerUDP_HandleClose(t *testing.T) {
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithListenNetwork("udp"), transport.WithHandler(&fakeHandler{})},
		func(addr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, err := tnetRequest(ctx, helloWorld,
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(addr),
				transport.WithDialTimeout(500*time.Millisecond),
			)
			assert.NotNil(t, err)
		},
	)
}

type fakeHandler struct {
}

func (h *fakeHandler) Handle(ctx context.Context, req []byte) (rsp []byte, err error) {
	return nil, errors.New("fake handle close")
}

func (h *fakeHandler) HandleClose(ctx context.Context) error {
	return errors.New("fake handle close")
}
