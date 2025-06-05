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
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/transport"
	"trpc.group/trpc-go/trpc-go/transport/internal/dialer"
	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

func TestClientUDP(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(addr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			rsp, err := tnetRequest(ctx, helloWorld,
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(addr),
				transport.WithDialTimeout(500*time.Millisecond),
			)
			assert.Equal(t, helloWorld, rsp)
			assert.Nil(t, err)
		},
	)
}

func TestClientUDP_ReadTimeout(t *testing.T) {
	startClientTest(
		t,
		func(ctx context.Context, req []byte) ([]byte, error) {
			time.Sleep(time.Hour)
			return nil, nil
		},
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(addr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, err := tnetRequest(
				ctx,
				helloWorld,
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(addr),
			)
			assert.Equal(t, errs.RetClientTimeout, errs.Code(err))
		},
	)
}

func TestClientUDP_RPCZ(t *testing.T) {
	oldRPCZEnable := rpczenable.Enabled
	rpczenable.Enabled = true
	defer func() { rpczenable.Enabled = oldRPCZEnable }()
	startClientTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(addr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			rsp, err := tnetRequest(ctx, helloWorld,
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(addr),
				transport.WithDialTimeout(500*time.Millisecond),
			)
			assert.Equal(t, helloWorld, rsp)
			assert.Nil(t, err)
		},
	)
}

func TestClientUDP_ReadFrameErr(t *testing.T) {
	t.Run("FakeFrameBuilder", func(t *testing.T) {
		startClientTest(
			t,
			defaultServerHandle,
			[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
			func(addr string) {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				_, err := tnetRequest(ctx, helloWorld,
					transport.WithDialNetwork("udp"),
					transport.WithDialAddress(addr),
					transport.WithDialTimeout(500*time.Millisecond),
					transport.WithClientFramerBuilder(&fakeFrameBuilder{}),
				)
				assert.Equal(t, errs.RetClientReadFrameErr, errs.Code(err))
			},
		)
	})
	t.Run("SendOnly", func(t *testing.T) {
		startClientTest(
			t,
			defaultServerHandle,
			[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
			func(addr string) {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				_, err := tnetRequest(ctx, helloWorld,
					transport.WithDialNetwork("udp"),
					transport.WithDialAddress(addr),
					transport.WithDialTimeout(500*time.Millisecond),
					transport.WithReqType(transport.SendOnly),
				)
				assert.Equal(t, errs.ErrClientNoResponse, err)
			},
		)
	})
}

func TestClientUDP_InvalidAddress(t *testing.T) {
	startClientTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
		func(_ string) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, err := tnetRequest(ctx, helloWorld,
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress("invalid address"),
				transport.WithDialTimeout(500*time.Millisecond),
			)
			assert.NotNil(t, err)
		},
	)
}

func TestClientUDP_NotConnect(t *testing.T) {
	t.Run("without localaddr", func(t *testing.T) {
		startClientTest(
			t,
			defaultServerHandle,
			[]transport.ListenServeOption{transport.WithListenNetwork("udp4")},
			func(addr string) {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				rsp, err := tnetRequest(ctx, helloWorld,
					transport.WithDialNetwork("udp4"),
					transport.WithDialAddress(addr),
					transport.WithDialTimeout(500*time.Millisecond),
					transport.WithConnectionMode(dialer.NotConnected),
				)
				assert.Equal(t, helloWorld, rsp)
				assert.Nil(t, err)
			},
		)
	})
	t.Run("with localaddr", func(t *testing.T) {
		startClientTest(
			t,
			defaultServerHandle,
			[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
			func(addr string) {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				rsp, err := tnetRequest(ctx, helloWorld,
					transport.WithDialNetwork("udp"),
					transport.WithDialAddress(addr),
					transport.WithDialTimeout(500*time.Millisecond),
					transport.WithConnectionMode(dialer.NotConnected),
					transport.WithLocalAddr("127.0.0.1:"),
				)
				assert.Equal(t, helloWorld, rsp)
				assert.Nil(t, err)
			},
		)
	})
	t.Run("with mismatch localaddr", func(t *testing.T) {
		startClientTest(
			t,
			defaultServerHandle,
			[]transport.ListenServeOption{transport.WithListenNetwork("udp")},
			func(addr string) {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				_, err := tnetRequest(ctx, helloWorld,
					transport.WithDialNetwork("udp"),
					transport.WithDialAddress(addr),
					transport.WithDialTimeout(500*time.Millisecond),
					transport.WithConnectionMode(dialer.NotConnected),
					transport.WithLocalAddr("[::1]:8080"),
				)
				assert.NotNil(t, err)
			},
		)
	})
}

func TestClientUDP_WithClientExactUDPBufferSizeEnabled(t *testing.T) {
	t.Run("tnet transport", func(t *testing.T) {
		helloworld := []byte("helloworld")
		addr := getAddr()
		handler := newUserDefineHandler(func(ctx context.Context, req []byte) ([]byte, error) {
			return defaultServerHandle(ctx, req)
		})
		err := transport.ListenAndServe(transport.WithListenNetwork("udp"),
			transport.WithListenAddress(addr),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithHandler(handler))
		assert.Nil(t, err)

		c := tnettrans.NewClientTransport(tnettrans.WithClientExactUDPBufferSizeEnabled(true))
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		reqbytes, err := trpc.DefaultClientCodec.Encode(
			codec.Message(ctx),
			helloworld,
		)
		assert.Nil(t, err)
		rspbytes, err := c.RoundTrip(ctx,
			reqbytes,
			transport.WithDialNetwork("udp"),
			transport.WithDialAddress(addr),
			transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder))
		assert.Nil(t, err)
		rsp, err := trpc.DefaultServerCodec.Decode(
			codec.Message(ctx),
			rspbytes,
		)
		assert.Nil(t, err)
		assert.Equal(t, helloworld, rsp)
	})
	t.Run("transportWithoutReadFrame enable exactUDPBufferSize", func(t *testing.T) {
		helloworld := []byte("helloworld")
		addr := getAddr()
		go func() {
			conn, err := net.ListenPacket("udp", addr)
			assert.Nil(t, err)
			defer conn.Close()
			buf := make([]byte, 1024)
			n, raddr, err := conn.ReadFrom(buf)
			assert.Nil(t, err)
			_, err = conn.WriteTo(buf[:n], raddr)
			assert.Nil(t, err)
		}()
		time.Sleep(10 * time.Millisecond)

		c := &transportWithoutReadFrame{exactUDPBufferSizeEnabled: true}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		rspbytes, err := c.RoundTrip(ctx,
			helloworld,
			transport.WithDialNetwork("udp"),
			transport.WithDialAddress(addr),
			transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder))
		assert.Nil(t, err)
		assert.Equal(t, helloworld, rspbytes)
		sockaddrSize := unix.SizeofSockaddrInet6
		// tnet will allocate buffer of size sockaddrSize+len(helloworld) memory,
		// so the cap of buffer is nextPowerOf2(sockaddrSize+len(helloworld)),
		// the cap of data is nextPowerOf2(sockaddrSize+len(helloworld))-sockaddrSize.
		assert.Equal(t, nextPowerOf2(sockaddrSize+len(helloworld))-sockaddrSize, cap(rspbytes))
	})
	t.Run("transportWithoutReadFrame disable exactUDPBufferSize", func(t *testing.T) {
		helloworld := []byte("helloworld")
		addr := getAddr()
		go func() {
			conn, err := net.ListenPacket("udp", addr)
			assert.Nil(t, err)
			defer conn.Close()
			buf := make([]byte, 1024)
			n, raddr, err := conn.ReadFrom(buf)
			assert.Nil(t, err)
			_, err = conn.WriteTo(buf[:n], raddr)
			assert.Nil(t, err)
		}()
		time.Sleep(10 * time.Millisecond)

		c := &transportWithoutReadFrame{exactUDPBufferSizeEnabled: false}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		rspbytes, err := c.RoundTrip(ctx,
			helloworld,
			transport.WithDialNetwork("udp"),
			transport.WithDialAddress(addr),
			transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder))
		assert.Nil(t, err)
		assert.Equal(t, helloworld, rspbytes)
		assert.Less(t, 65536, cap(rspbytes))
	})
}

type fakeFrameBuilder struct {
}

func (f *fakeFrameBuilder) New(r io.Reader) codec.Framer {
	return &fakeFramer{}
}

type fakeFramer struct {
}

func (f *fakeFramer) ReadFrame() ([]byte, error) {
	return nil, errors.New("read frame error")
}

type transportWithoutReadFrame struct {
	exactUDPBufferSizeEnabled bool
}

func (t *transportWithoutReadFrame) RoundTrip(
	ctx context.Context,
	req []byte,
	opts ...transport.RoundTripOption,
) ([]byte, error) {
	opt := transport.RoundTripOptions{}
	for _, o := range opts {
		o(&opt)
	}
	switch opt.Network {
	case "udp", "udp4", "udp6":
		break
	default:
		return nil, fmt.Errorf("network %v not supported", opt.Network)
	}
	conn, err := tnet.DialUDP(opt.Network, opt.Address, opt.DialTimeout)
	if err != nil {
		return nil, err
	}
	conn.SetExactUDPBufferSizeEnabled(t.exactUDPBufferSizeEnabled)
	_, err = conn.Write(req)
	if err != nil {
		return nil, err
	}
	packet, _, err := conn.ReadPacket()
	if err != nil {
		return nil, err
	}
	defer packet.Free()
	return packet.Data()
}

func nextPowerOf2(n int) int {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}
