// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package server_test

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// go test -v
type fakeTransport struct {
}

func (s *fakeTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) error {
	lsopts := &transport.ListenServeOptions{}
	for _, opt := range opts {
		opt(lsopts)
	}

	go func() {
		lsopts.Handler.Handle(ctx, []byte("normal-request"))
		lsopts.Handler.Handle(ctx, []byte("stream"))
		lsopts.Handler.Handle(ctx, []byte("no-rpc-name"))
		lsopts.Handler.Handle(ctx, []byte("decode-error"))
		lsopts.Handler.Handle(ctx, []byte("encode-error"))
		lsopts.Handler.Handle(ctx, []byte("handle-timeout"))
		lsopts.Handler.Handle(ctx, []byte("no-response"))
		lsopts.Handler.Handle(ctx, []byte("business-fail"))
		lsopts.Handler.Handle(ctx, []byte("handle-panic"))
		lsopts.Handler.Handle(ctx, []byte("compress-error"))
		lsopts.Handler.Handle(ctx, []byte("decompress-error"))
		lsopts.Handler.Handle(ctx, []byte("unmarshal-error"))
		lsopts.Handler.Handle(ctx, []byte("marshal-error"))
		ctx := context.Background()
		ctx, msg := codec.WithNewMessage(ctx)
		msg.WithServerRspErr(errors.New("connection is tryClose "))
		lsopts.Handler.Handle(ctx, nil)

	}()

	return nil
}

type fakeCodec struct {
}

func (c *fakeCodec) Decode(msg codec.Msg, reqBuf []byte) (reqBody []byte, err error) {
	req := string(reqBuf)

	if req == "stream" {
		msg.WithServerRPCName("/trpc.test.helloworld.Greeter/SayHi")
		return reqBuf, nil
	}
	if req != "no-rpc-name" {
		msg.WithServerRPCName("/trpc.test.helloworld.Greeter/SayHello")
	}
	if req == "decode-error" {
		return nil, errors.New("server decode request fail")
	}
	msg.WithRequestTimeout(time.Second)
	msg.WithSerializationType(codec.SerializationTypeNoop)
	log.Infof("fakeCodec ==> req[%v]", req)
	return reqBuf, nil
}

func (c *fakeCodec) Encode(msg codec.Msg, rspBody []byte) (rspBuf []byte, err error) {
	rsp := string(rspBody)
	if rsp == "encode-error" {
		return nil, errors.New("server encode response fail")
	}
	return rspBody, nil
}

func (c *fakeCodec) Compress(in []byte) (out []byte, err error) {
	rsp := string(in)
	if rsp == "compress-error" {
		return nil, errors.New("server compress fail")
	}
	return in, nil
}

func (c *fakeCodec) Decompress(in []byte) (out []byte, err error) {
	req := string(in)
	if req == "decompress-error" {
		return nil, errors.New("server decompress fail")
	}
	return in, nil
}

func (c *fakeCodec) Unmarshal(reqBuf []byte, reqBody interface{}) error {
	req := string(reqBuf)
	if req == "unmarshal-error" {
		return errors.New("server unmarshal fail")
	}
	return codec.Unmarshal(codec.SerializationTypeNoop, reqBuf, reqBody)
}

func (c *fakeCodec) Marshal(rspBody interface{}) (rspBuf []byte, err error) {
	if rsp, ok := rspBody.(*codec.Body); ok {
		if string(rsp.Data) == "marshal-error" {
			return nil, errors.New("server marshal fail")
		}
	}
	return codec.Marshal(codec.SerializationTypeNoop, rspBody)
}

type fakeRegistry struct {
}

func (r *fakeRegistry) Register(service string, opt ...registry.Option) error {
	return nil
}
func (r *fakeRegistry) Deregister(service string) error {
	return nil
}

func TestService(t *testing.T) {
	codec.Register("fake", &fakeCodec{}, nil)
	// register the fake codec
	codec.RegisterCompressor(930, &fakeCodec{})
	codec.RegisterSerializer(1930, &fakeCodec{})

	// 1.codec not set，transport will cause error.
	service := server.New(server.WithServiceName("trpc.test.helloworld.Greeter"),
		server.WithTransport(&fakeTransport{}),
		server.WithRegistry(&registry.NoopRegistry{}))

	impl := &GreeterServerImpl{}
	err := service.Register(&GreeterServerServiceDesc, impl)
	assert.Nil(t, err)

	go func() {
		_ = service.Serve()
	}()
	// closing service will not return error even if registry fails.
	err = service.Close(nil)
	assert.Nil(t, err)

	// 2. valid service registration
	service = server.New(server.WithProtocol("fake"),
		server.WithServiceName("trpc.test.helloworld.Greeter"),
		server.WithTransport(&fakeTransport{}),
		server.WithRegistry(&fakeRegistry{}),
		server.WithCurrentSerializationType(1930),
		server.WithCurrentCompressType(930),
		server.WithCloseWaitTime(100*time.Millisecond),
		server.WithMaxCloseWaitTime(200*time.Millisecond))
	err = service.Register(&GreeterServerServiceDesc, impl)
	assert.Nil(t, err)

	// RESTful router should exist
	assert.NotNil(t, restful.GetRouter("trpc.test.helloworld.Greeter"))

	go func() {
		_ = service.Serve()
	}()
	time.Sleep(time.Second * 2)
	err = service.Close(nil)
	assert.Nil(t, err)
}

// TestServiceFail tests failures of request handling.
func TestServiceFail(t *testing.T) {

	codec.Register("fake", &fakeCodec{}, nil)
	service := server.New(server.WithProtocol("fake"),
		server.WithServiceName("trpc.test.helloworld.Greeter"),
		server.WithTransport(&fakeTransport{}),
		server.WithRegistry(&fakeRegistry{}),
	)

	impl := &GreeterServerImpl{}
	err := service.Register(&GreeterServerServiceDescFail, impl)
	assert.Nil(t, err)
	go func() {
		service.Serve()
	}()

	time.Sleep(time.Second * 2)
}

// TestServiceMethodNameUniqueness tests method name uniqueness
func TestServiceMethodNameUniqueness(t *testing.T) {
	codec.Register("fake", &fakeCodec{}, nil)
	service := server.New(server.WithProtocol("fake"),
		server.WithServiceName("trpc.test.helloworld.Greeter"),
		server.WithTransport(&fakeTransport{}),
		server.WithRegistry(&fakeRegistry{}),
	)

	impl := &GreeterServerImpl{}
	err := service.Register(&GreeterServerServiceDescFail, impl)
	assert.Nil(t, err)

	err = service.Register(&GreeterServerServiceDescFail, impl)
	assert.NotNil(t, err)
}

func TestServiceTimeout(t *testing.T) {
	require.Nil(t, os.Setenv(transport.EnvGraceRestart, ""))
	t.Run("server timeout", func(t *testing.T) {
		addr, stop := startService(t, &GreeterServerImpl{},
			server.WithTimeout(time.Second),
			server.WithFilter(
				func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
					return nil, errs.NewFrameError(errs.RetServerTimeout, "")
				}))
		defer stop()

		c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
		_, err := c.SayHello(context.Background(), &pb.HelloRequest{})
		require.NotNil(t, err)
		e, ok := err.(*errs.Error)
		require.True(t, ok)
		require.EqualValues(t, int32(errs.RetServerTimeout), e.Code)
	})
	t.Run("client full link timeout is converted to server timeout",
		func(t *testing.T) {
			addr, stop := startService(t,
				&Greeter{
					sayHello: func(ctx context.Context, req *codec.Body) (rsp *codec.Body, err error) {
						return nil, errs.NewFrameError(errs.RetClientFullLinkTimeout, "")
					}},
				server.WithTimeout(time.Second))
			defer stop()

			c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
			_, err := c.SayHello(ctx, &pb.HelloRequest{})
			require.NotNil(t, err)
			e, ok := err.(*errs.Error)
			require.True(t, ok)
			require.Equal(t, errs.ErrorTypeCalleeFramework, e.Type)
			require.EqualValues(t, int32(errs.RetServerTimeout), e.Code)
		})
	t.Run("client full link timeout is converted to server full link timeout, and then dropped",
		func(t *testing.T) {
			addr, stop := startService(t,
				&Greeter{
					sayHello: func(ctx context.Context, req *codec.Body) (rsp *codec.Body, err error) {
						return nil, errs.NewFrameError(errs.RetClientFullLinkTimeout, "")
					}},
				server.WithTimeout(time.Second*2))
			defer stop()

			c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, err := c.SayHello(ctx, &pb.HelloRequest{})
			require.NotNil(t, err)
			e, ok := err.(*errs.Error)
			require.True(t, ok)
			require.Equal(t, errs.ErrorTypeFramework, e.Type)
			require.EqualValues(t, int32(errs.RetClientFullLinkTimeout), e.Code,
				"server full link timeout is dropped, and client should receive a client timeout error")
		})
}

func TestServiceUDP(t *testing.T) {
	addr := "127.0.0.1:10000"
	s := server.New([]server.Option{
		server.WithNetwork("udp"),
		server.WithProtocol("trpc"),
		server.WithAddress(addr),
		server.WithCurrentSerializationType(codec.SerializationTypeNoop),
	}...)
	require.Nil(t, s.Register(&GreeterServerServiceDesc, &GreeterServerImpl{}))
	go s.Serve()
	time.Sleep(time.Millisecond * 200)

	c := pb.NewGreeterClientProxy(client.WithTarget("ip://"+addr), client.WithNetwork("udp"))
	_, err := c.SayHello(context.Background(), &pb.HelloRequest{})
	require.Nil(t, err)
}

func TestServiceCloseWait(t *testing.T) {
	const waitChildTime = 300 * time.Millisecond
	const schTime = 10 * time.Millisecond
	cases := []struct {
		closeWaitTime    time.Duration
		maxCloseWaitTime time.Duration
		waitTime         time.Duration
	}{
		{
			waitTime: waitChildTime,
		},
		{
			closeWaitTime: 50 * time.Millisecond,
			waitTime:      waitChildTime + 50*time.Millisecond,
		},
		{
			closeWaitTime:    50 * time.Millisecond,
			maxCloseWaitTime: 30 * time.Millisecond,
			waitTime:         waitChildTime + 50*time.Millisecond,
		},
		{
			closeWaitTime:    50 * time.Millisecond,
			maxCloseWaitTime: 100 * time.Millisecond,
			waitTime:         waitChildTime + 50*time.Millisecond,
		},
	}
	for _, c := range cases {
		service := server.New(
			server.WithRegistry(&fakeRegistry{}),
			server.WithCloseWaitTime(c.closeWaitTime),
			server.WithMaxCloseWaitTime(c.maxCloseWaitTime),
		)
		start := time.Now()
		err := service.Close(nil)
		assert.Nil(t, err)
		cost := time.Since(start)
		assert.GreaterOrEqual(t, cost, c.waitTime)
		assert.LessOrEqual(t, cost, c.waitTime+schTime)
	}
}

func startService(t *testing.T, gs GreeterServer, opts ...server.Option) (addr string, stop func()) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	require.Nil(t, err)

	s := server.New(append(append([]server.Option{
		server.WithNetwork("tcp"),
		server.WithProtocol("trpc"),
	}, opts...),
		server.WithListener(l),
	)...)
	require.Nil(t, s.Register(&GreeterServerServiceDesc, gs))

	errCh := make(chan error)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(time.Millisecond * 200):
	}
	return l.Addr().String(), func() { s.Close(nil) }
}

func TestGetStreamFilter(t *testing.T) {
	expectedErr := errors.New("expected error")
	testFilter := func(ss server.Stream, info *server.StreamServerInfo, handler server.StreamHandler) error {
		return expectedErr
	}
	server.RegisterStreamFilter("testFilter", testFilter)
	filter := server.GetStreamFilter("testFilter")
	err := filter(nil, &server.StreamServerInfo{}, nil)
	assert.Equal(t, expectedErr, err)
}

type Greeter struct {
	sayHello func(ctx context.Context, req *codec.Body) (rsp *codec.Body, err error)
}

func (g *Greeter) SayHello(ctx context.Context, req *codec.Body) (rsp *codec.Body, err error) {
	return g.sayHello(ctx, req)
}

func (*Greeter) SayHi(gs Greeter_SayHiServer) error {
	return nil
}

func TestStreamFilterChainFilter(t *testing.T) {
	ch := make(chan int, 10)
	sf1 := func(ss server.Stream, info *server.StreamServerInfo, handler server.StreamHandler) error {
		ch <- 1
		err := handler(ss)
		ch <- 5
		return err
	}
	sf2 := func(ss server.Stream, info *server.StreamServerInfo, handler server.StreamHandler) error {
		ch <- 2
		err := handler(ss)
		ch <- 4
		return err
	}
	option := server.WithStreamFilters(sf1, sf2)
	options := server.Options{}
	option(&options)
	_ = options.StreamFilters.Filter(nil, nil, func(stream server.Stream) error {
		ch <- 3
		return nil
	})
	assert.Equal(t, 1, <-ch)
	assert.Equal(t, 2, <-ch)
	assert.Equal(t, 3, <-ch)
	assert.Equal(t, 4, <-ch)
	assert.Equal(t, 5, <-ch)
}
