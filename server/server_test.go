// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package server_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"
)

// Greeter defines service
type GreeterServer interface {
	SayHello(ctx context.Context, req *codec.Body) (rsp *codec.Body, err error)
	SayHi(Greeter_SayHiServer) error
}

// Greeter_SayHiServer defines server stream
type Greeter_SayHiServer interface {
	Send(*codec.Body) error
	Recv() (*codec.Body, error)
	server.Stream
}

// greeterSayHiServer server stream impl
type greeterSayHiServer struct {
	server.Stream
}

func (x *greeterSayHiServer) Send(m *codec.Body) error {
	return nil
}

func (x *greeterSayHiServer) Recv() (*codec.Body, error) {
	return nil, nil
}

func GreeterServerSayHelloHandler(svr interface{}, ctx context.Context,
	f server.FilterFunc) (rspBody interface{}, err error) {
	req := &codec.Body{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, reqBody interface{}) (interface{}, error) {
		return svr.(GreeterServer).SayHello(ctx, reqBody.(*codec.Body))
	}
	return filters.Filter(ctx, req, handleFunc)
}

type GreeterServerImpl struct{}
type FailServerImpl struct{}

func (s *GreeterServerImpl) SayHello(ctx context.Context, req *codec.Body) (rsp *codec.Body, err error) {
	rsp = &codec.Body{}
	rsp.Data = req.Data
	if string(req.Data) == "handle-timeout" {
		time.Sleep(time.Second * 2)
	}
	if string(req.Data) == "no-response" {
		return nil, errs.ErrServerNoResponse
	}
	if string(req.Data) == "business-fail" {
		return nil, errs.New(1000, "inner db fail")
	}
	return rsp, nil
}

func (s *GreeterServerImpl) SayHi(gs Greeter_SayHiServer) error {
	return nil
}

type fakeStreamHandle struct {
}

func (fs *fakeStreamHandle) StreamHandleFunc(ctx context.Context, sh server.StreamHandler, si *server.StreamServerInfo, req []byte) ([]byte, error) {
	return nil, nil
}

func (fs *fakeStreamHandle) Init(opts *server.Options) error {
	return nil
}

func GreeterService_SayHi_Handler(srv interface{}, stream server.Stream) error {
	return srv.(GreeterServer).SayHi(&greeterSayHiServer{stream})
}

// GreeterServer_ServiceDesc descriptor for server.RegisterService
var GreeterServerServiceDesc = server.ServiceDesc{
	ServiceName:  "trpc.test.helloworld.Greeter",
	HandlerType:  (*GreeterServer)(nil),
	StreamHandle: &fakeStreamHandle{},
	Methods: []server.Method{
		{
			Name: "/trpc.test.helloworld.Greeter/SayHello",
			Func: GreeterServerSayHelloHandler,
			Bindings: []*restful.Binding{
				{
					Name:    "/trpc.test.helloworld.Greeter/SayHello",
					Pattern: restful.Enforce("/v1/foobar"),
				},
			},
		},
	},
	Streams: []server.StreamDesc{
		{
			StreamName:    "/trpc.test.helloworld.Greeter/SayHi",
			Handler:       GreeterService_SayHi_Handler,
			ServerStreams: true,
		},
	},
}

// GreeterServer_ServiceDesc descriptor for server.RegisterService
var GreeterServerServiceDescFail = server.ServiceDesc{
	ServiceName:  "trpc.test.helloworld.Greeter",
	HandlerType:  (*GreeterServer)(nil),
	StreamHandle: nil,
	Methods: []server.Method{
		{
			Name: "/trpc.test.helloworld.Greeter/SayHello",
			Func: GreeterServerSayHelloHandler,
		},
	},
	Streams: []server.StreamDesc{
		{
			StreamName:    "/trpc.test.helloworld.Greeter/SayHi",
			Handler:       GreeterService_SayHi_Handler,
			ServerStreams: true,
		},
	},
}

func TestServeFail(t *testing.T) {
	t.Run("test empty service", func(t *testing.T) {
		s := &server.Server{}
		assert.Panics(t, func() { s.Serve() }, "service empty")
	})
	t.Run("network mismatching", func(t *testing.T) {
		s := &server.Server{}
		s.AddService("trpc.test.helloworld.Greeter1", server.New(
			server.WithNetwork("tcp9"),
			server.WithAddress("127.0.0.1:8080"),
			server.WithProtocol("trpc"),
			server.WithServiceName("trpc.test.helloworld.Greeter1")))
		assert.NotNil(t, s.Register(&GreeterServerServiceDesc, &FailServerImpl{}))
		assert.NotNil(t, s.Serve())
	})
	t.Run("registry failure", func(t *testing.T) {
		s := &server.Server{}
		s.AddService("trpc.test.helloworld.Greeter", server.New(
			server.WithAddress("127.0.0.1:8081"),
			server.WithRegistry(&registry.NoopRegistry{})))
		assert.NotNil(t, s.Register(&GreeterServerServiceDesc, &FailServerImpl{}))
		assert.NotNil(t, s.Serve())
	})
}

func TestServer(t *testing.T) {
	// If the process is started by graceful restart,
	// exit here in case of infinite loop.
	if len(os.Getenv(transport.EnvGraceRestart)) > 0 {
		t.SkipNow()
	}
	s := &server.Server{}

	// 1. try to get service that not exists.
	assert.Nil(t, s.Service("empty"))

	service1 := server.New(server.WithAddress("127.0.0.1:12345"),
		server.WithNetwork("tcp"),
		server.WithProtocol("trpc"),
		server.WithServiceName("trpc.test.helloworld.Greeter1"))

	service2 := server.New(server.WithAddress("127.0.0.1:12346"),
		server.WithNetwork("tcp"),
		server.WithProtocol("trpc"),
		server.WithServiceName("trpc.test.helloworld.Greeter2"))

	s.AddService("trpc.test.helloworld.Greeter1", service1)
	s.AddService("trpc.test.helloworld.Greeter2", service2)

	assert.Equal(t, service1, s.Service("trpc.test.helloworld.Greeter1"))
	assert.Equal(t, service2, s.Service("trpc.test.helloworld.Greeter2"))
	assert.Nil(t, s.Service("empty"))

	// 2. test registering empty proto service.
	err := s.Register(nil, nil)
	assert.NotNil(t, err)

	impl := &GreeterServerImpl{}
	err = s.Register(&GreeterServerServiceDesc, impl)
	assert.Nil(t, err)

	// 3. valid serving.
	go func() {
		err = os.Setenv(transport.EnvGraceRestart, "")
		assert.Nil(t, err)

		err = s.Serve()
		assert.Nil(t, err)
	}()

	time.Sleep(time.Second * 1)
	err = s.Close(nil)
	assert.Nil(t, err)
}

func TestServerClose(t *testing.T) {
	const schTime = 10 * time.Millisecond
	cases := []struct {
		maxCloseWaitTime time.Duration
	}{
		{},
		{
			maxCloseWaitTime: server.MaxCloseWaitTime / 2,
		},
		{
			maxCloseWaitTime: server.MaxCloseWaitTime,
		},
		{
			maxCloseWaitTime: server.MaxCloseWaitTime * 2,
		},
	}
	for _, c := range cases {
		s := &server.Server{
			MaxCloseWaitTime: c.maxCloseWaitTime,
		}
		start := time.Now()
		s.Close(nil)
		et := time.Since(start)
		assert.Less(t, et, schTime)
	}
}

// TestServer_AtExit tests whether order of execution of shutdown hook functions matches
// order of registration of shutdown hook functions.
func TestServer_AtExit_ExecuteOrder(t *testing.T) {
	s := &server.Server{}
	const num = 3
	ch := make(chan int, num)
	for i := 0; i < num; i++ {
		// temporary variable j helps capture the iteration variable i.
		j := i
		s.RegisterOnShutdown(func() { ch <- j })
	}
	s.RegisterOnShutdown(func() { close(ch) })

	require.Nil(t, s.Close(nil))

	for i := 0; i < num; i++ {
		require.Equal(t, i, <-ch)
	}
	_, ok := <-ch
	require.False(t, ok)
}
