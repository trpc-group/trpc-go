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
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/tnet"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"
	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

var (
	port       uint64 = 9000
	helloWorld        = []byte("helloworld")
)

func TestServerTCP_ListenAndServe(t *testing.T) {
	startServerTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			rsp, err := gonetRequest(context.Background(), transport.WithDialAddress(addr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestServerTCP_Asyn(t *testing.T) {
	startServerTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithServerAsync(true)},
		func(addr string) {
			rsp, err := gonetRequest(context.Background(), transport.WithDialAddress(addr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestServerTCP_CustomizedFramerCopyFrame(t *testing.T) {
	startServerTest(
		t,
		func(ctx context.Context, req []byte) ([]byte, error) {
			return req, nil
		},
		[]transport.ListenServeOption{
			transport.WithServerFramerBuilder(&reuseBufferFramerBuilder{}),
			transport.WithServerAsync(true),
		},
		func(addr string) {
			req := helloWorld
			ctx, _ := codec.EnsureMessage(context.Background())
			reqbytes, err := (&emptyClientCodec{}).Encode(
				codec.Message(ctx),
				req,
			)
			assert.Nil(t, err)

			cliOpts := []transport.RoundTripOption{
				transport.WithDialAddress(addr),
				transport.WithDialNetwork("tcp"),
				transport.WithClientFramerBuilder(&reuseBufferFramerBuilder{}),
				transport.WithDialTimeout(5 * time.Second),
			}
			clientTrans := transport.NewClientTransport()
			rspbytes, err := clientTrans.RoundTrip(
				ctx,
				reqbytes,
				cliOpts...,
			)
			assert.Nil(t, err)

			rsp, err := (&emptyClientCodec{}).Decode(
				codec.Message(ctx),
				rspbytes,
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestServerTCP_UserDefineListener(t *testing.T) {
	serverAddr := getAddr()
	ln, err := tnet.Listen("tcp", serverAddr)
	assert.Nil(t, err)
	startServerTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithListener(ln)},
		func(_ string) {
			rsp, err := gonetRequest(context.Background(), transport.WithDialAddress(serverAddr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestServerTCP_ErrorCases(t *testing.T) {
	s := tnettrans.NewServerTransport()

	// Without framerBuilder
	serveOpts := getListenServeOption(
		transport.WithServerFramerBuilder(nil),
	)
	err := s.ListenAndServe(context.Background(), serveOpts...)
	assert.NotNil(t, err)

	// Unsupported network type
	serveOpts = getListenServeOption(
		transport.WithListenNetwork("ip"),
	)
	err = s.ListenAndServe(context.Background(), serveOpts...)
	assert.NotNil(t, err)
}

func TestServerTCP_HandleErr(t *testing.T) {
	startServerTest(
		t,
		errServerHandle,
		nil,
		func(addr string) {
			_, err := gonetRequest(context.Background(), transport.WithDialAddress(addr))
			fmt.Println(err)
			assert.NotNil(t, err)
		},
	)
}

func TestServerTCP_IdleTimeout(t *testing.T) {
	startServerTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithServerIdleTimeout(time.Second)},
		func(addr string) {
			cliconn, err := tnet.DialTCP("tcp", addr, 0)
			assert.Nil(t, err)
			_, err = cliconn.Write([]byte("0"))
			assert.Nil(t, err)

			// sleep to make sure ListenAndServe run into onRequest()
			time.Sleep(2 * time.Second)
			_, err = cliconn.Write([]byte("0"))
			assert.NotNil(t, err)
		},
	)

}

func TestServerTCP_WriteFail(t *testing.T) {
	ch := make(chan struct{}, 1)
	var isHandled bool
	startServerTest(
		t,
		func(ctx context.Context, req []byte) ([]byte, error) {
			isHandled = true
			<-ch
			return nil, nil
		},
		[]transport.ListenServeOption{transport.WithServerAsync(true)},
		func(addr string) {
			ctx, _ := codec.EnsureMessage(context.Background())
			req, err := trpc.DefaultClientCodec.Encode(codec.Message(ctx), helloWorld)
			assert.Nil(t, err)

			cliconn, err := tnet.DialTCP("tcp", addr, 0)
			assert.Nil(t, err)
			_, err = cliconn.Write(req)
			assert.Nil(t, err)

			// sleep to make sure server received data
			time.Sleep(50 * time.Millisecond)
			cliconn.Close()
			// notify server write back data, but server will fail, because connection is closed
			ch <- struct{}{}
			_, err = cliconn.ReadN(1)
			assert.NotNil(t, err)
			// make sure server run into handle
			assert.True(t, isHandled)
		},
	)
}

func TestServerTCP_PassedListener(t *testing.T) {
	serverAddr := getAddr()
	listener, err := net.Listen("tcp", serverAddr)
	assert.Nil(t, err)

	transport.SaveListener(listener)
	fds := transport.GetListenersFds()
	var fd int
	for _, f := range fds {
		if f.Address == serverAddr {
			fd = int(f.Fd)
		}
	}

	os.Setenv(transport.EnvGraceRestart, "1")
	os.Setenv(transport.EnvGraceFirstFd, strconv.Itoa(fd))
	os.Setenv(transport.EnvGraceRestartFdNum, "1")

	defer func() {
		os.Setenv(transport.EnvGraceRestart, "0")
		os.Setenv(transport.EnvGraceFirstFd, "0")
		os.Setenv(transport.EnvGraceRestartFdNum, "0")
	}()

	startServerTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithListenAddress(serverAddr)},
		func(_ string) {
			rsp, err := gonetRequest(context.Background(), transport.WithDialAddress(serverAddr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestServerTCP_ClientWrongReq(t *testing.T) {
	startServerTest(
		t,
		defaultServerHandle,
		nil,
		func(addr string) {
			cliconn, err := tnet.DialTCP("tcp", addr, 0)
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

func TestServerTCP_SendAndClose(t *testing.T) {
	addr := getAddr()
	s := tnettrans.NewServerTransport()
	serveOpts := getListenServeOption(
		transport.WithListenAddress(addr),
		transport.WithServerAsync(true),
	)
	err := s.ListenAndServe(context.Background(), serveOpts...)
	assert.Nil(t, err)

	cliconn, err := tnet.DialTCP("tcp", addr, 0)
	assert.Nil(t, err)
	cliAddr := cliconn.LocalAddr()

	time.Sleep(50 * time.Millisecond)
	streamTransport, ok := s.(transport.ServerStreamTransport)
	assert.True(t, ok)
	ctx, msg := codec.EnsureMessage(context.Background())
	msg.WithRemoteAddr(cliAddr)
	svrAddr, err := net.ResolveTCPAddr("tcp", addr)
	assert.Nil(t, err)
	msg.WithLocalAddr(svrAddr)
	err = streamTransport.Send(ctx, helloWorld)
	assert.Nil(t, err)

	b := make([]byte, len(helloWorld))
	cliconn.Read(b)
	assert.Equal(t, b, helloWorld)

	streamTransport.Close(ctx)
	err = streamTransport.Send(ctx, helloWorld)
	assert.NotNil(t, err)
}

func TestServerTCP_TLS(t *testing.T) {
	startServerTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithServeTLS("../../testdata/server.crt", "../../testdata/server.key", "../../testdata/ca.pem")},
		func(addr string) {
			rsp, err := gonetRequest(
				context.Background(),
				transport.WithDialAddress(addr),
				transport.WithDialTLS("../../testdata/client.crt", "../../testdata/client.key", "../../testdata/ca.pem", "localhost"),
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)

			rsp, err = gonetRequest(
				context.Background(),
				transport.WithDialAddress(addr),
				transport.WithDialTLS("../../testdata/client.crt", "../../testdata/client.key", "none", ""),
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestUDP(t *testing.T) {
	// UDP is not supported, but it will switch to gonet default transport to serve.
	startServerTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{transport.WithListenNetwork("tcp,udp")},
		func(addr string) {
			rsp, err := gonetRequest(
				context.Background(),
				transport.WithDialAddress(addr),
				transport.WithDialNetwork("udp"))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)

			rsp, err = gonetRequest(
				context.Background(),
				transport.WithDialAddress(addr),
				transport.WithDialNetwork("tcp"))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestUnix(t *testing.T) {
	// Unix socket is not supported, but it will switch to gonet default transport to serve.
	myAddr := "/tmp/server.sock"
	os.Remove(myAddr)
	startServerTest(
		t,
		defaultServerHandle,
		[]transport.ListenServeOption{
			transport.WithListenNetwork("unix"),
			transport.WithListenAddress(myAddr),
		},
		func(_ string) {
			rsp, err := gonetRequest(
				context.Background(),
				transport.WithDialAddress(myAddr),
				transport.WithDialNetwork("unix"))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func getListenServeOption(opts ...transport.ListenServeOption) []transport.ListenServeOption {
	lsopts := []transport.ListenServeOption{
		transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
		transport.WithListenNetwork("tcp"),
		transport.WithHandler(newUserDefineHandler(defaultServerHandle)),
		transport.WithServerIdleTimeout(5 * time.Second),
	}
	lsopts = append(lsopts, opts...)
	return lsopts
}

func defaultServerHandle(ctx context.Context, req []byte) (rsp []byte, err error) {
	msg := codec.Message(ctx)
	reqdata, err := trpc.DefaultServerCodec.Decode(msg, req)
	if err != nil {
		return nil, err
	}
	rspdata := make([]byte, len(reqdata))
	copy(rspdata, reqdata)
	rsp, err = trpc.DefaultServerCodec.Encode(msg, rspdata)
	return rsp, err
}

func errServerHandle(ctx context.Context, req []byte) (rsp []byte, err error) {
	return nil, errors.New("mock error")
}

type userDefineHandler struct {
	handleFunc func(context.Context, []byte) ([]byte, error)
}

func newUserDefineHandler(f func(context.Context, []byte) ([]byte, error)) *userDefineHandler {
	return &userDefineHandler{handleFunc: f}
}

func (uh *userDefineHandler) Handle(ctx context.Context, req []byte) (rsp []byte, err error) {
	return uh.handleFunc(ctx, req)
}

func startServerTest(
	t *testing.T,
	serverHandle func(ctx context.Context, req []byte) ([]byte, error),
	svrCustomOpts []transport.ListenServeOption,
	clientHandle func(addr string),
) {
	addr := getAddr()
	s := tnettrans.NewServerTransport(
		tnettrans.WithKeepAlivePeriod(15*time.Second),
		tnettrans.WithReusePort(true),
	)
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

func gonetRequest(ctx context.Context, opts ...transport.RoundTripOption) ([]byte, error) {
	req := helloWorld
	ctx, _ = codec.EnsureMessage(ctx)
	reqbytes, err := trpc.DefaultClientCodec.Encode(
		codec.Message(ctx),
		req,
	)
	if err != nil {
		return nil, err
	}

	cliOpts := getRoundTripOption(opts...)
	clientTrans := transport.NewClientTransport()
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

func getRoundTripOption(opts ...transport.RoundTripOption) []transport.RoundTripOption {
	rtopts := []transport.RoundTripOption{
		transport.WithDialNetwork("tcp"),
		transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder),
		transport.WithDialTimeout(5 * time.Second),
	}
	rtopts = append(rtopts, opts...)
	return rtopts
}

func getAddr() string {
	atomic.AddUint64(&port, 1)
	return "127.0.0.1:" + fmt.Sprint(port)
}

type reuseBufferFramerBuilder struct{}

func (*reuseBufferFramerBuilder) New(r io.Reader) codec.Framer {
	return &reuseBufferFramer{r: r, reuseBuffer: make([]byte, len(helloWorld))}
}

type reuseBufferFramer struct {
	r           io.Reader
	reuseBuffer []byte
}

func (f *reuseBufferFramer) ReadFrame() ([]byte, error) {
	_, err := io.ReadFull(f.r, f.reuseBuffer)
	if err != nil {
		return nil, fmt.Errorf("io.ReadFull err: %w", err)
	}
	return f.reuseBuffer, nil
}

type emptyServerCodec struct{}

func (s *emptyServerCodec) Decode(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	return reqBuf, nil
}

func (s *emptyServerCodec) Encode(msg codec.Msg, rspBody []byte) ([]byte, error) {
	return rspBody, nil
}

type emptyClientCodec struct{}

func (s *emptyClientCodec) Decode(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	return reqBuf, nil
}

func (s *emptyClientCodec) Encode(msg codec.Msg, rspBody []byte) ([]byte, error) {
	return rspBody, nil
}
