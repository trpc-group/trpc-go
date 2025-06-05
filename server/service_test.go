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

package server_test

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	pb "trpc.group/trpc-go/trpc-go/testdata"
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

type fakeFrameHead struct {
	isStream bool
}

func (fh *fakeFrameHead) IsStream() bool {
	return fh.isStream
}

type fakeCodec struct{}

func (c *fakeCodec) Decode(msg codec.Msg, reqBuf []byte) (reqBody []byte, err error) {
	req := string(reqBuf)

	if req == "stream" {
		msg.WithServerRPCName("/trpc.test.helloworld.Greeter/SayHi")
		msg.WithFrameHead(&fakeFrameHead{true})
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
		addr, stop := startService(t, &Greeter{},
			server.WithTimeout(time.Second),
			server.WithFilter(
				func(ctx context.Context, req, rsp interface{}, next filter.HandleFunc) error {
					return errs.NewFrameError(errs.RetServerTimeout, "")
				}))
		defer stop()

		c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
		_, err := c.SayHello(context.Background(), &pb.HelloRequest{})
		require.NotNil(t, err)
		e, ok := err.(*errs.Error)
		require.True(t, ok)
		require.Equal(t, int32(errs.RetServerTimeout), e.Code)
	})
	t.Run("client full link timeout is converted to server timeout",
		func(t *testing.T) {
			addr, stop := startService(t,
				&Greeter{
					sayHello: func(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
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
			require.Equal(t, int32(errs.RetServerTimeout), e.Code)
		})
	t.Run("client full link timeout is converted to server full link timeout, and then dropped",
		func(t *testing.T) {
			addr, stop := startService(t,
				&Greeter{
					sayHello: func(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
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
			require.Equal(t, int32(errs.RetClientFullLinkTimeout), e.Code,
				"server full link timeout is dropped, and client should receive a client timeout error")
		})
}

func TestServiceMethodTimeout(t *testing.T) {
	t.Run("method_timeout_has_higher_priority_than_service_timeout", func(t *testing.T) {
		addr, stop := startService(t,
			&Greeter{sayHello: func(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
				select {
				case <-ctx.Done():
					return &pb.HelloReply{}, nil
				case <-time.After(time.Second):
					return nil, errors.New("wait ctx done timeout")
				}
			}},
			server.WithTimeout(time.Millisecond*50),
			server.WithMethodTimeout("SayHello", time.Millisecond*100))
		defer stop()

		c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
		start := time.Now()
		_, err := c.SayHello(context.Background(), &pb.HelloRequest{})
		require.Error(t, err)
		require.InDelta(t, time.Millisecond*100, time.Since(start), float64(time.Millisecond*30))
	})
}

func TestServiceOverload(t *testing.T) {
	require.Nil(t, os.Setenv(transport.EnvGraceRestart, ""))
	addr, stop := startService(t, &Greeter{}, server.WithOverloadCtrl(&overloadControllerAlwaysFail{}))
	defer stop()

	c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
	_, err := c.SayHello(context.Background(), &pb.HelloRequest{})
	require.NotNil(t, err)
	trpcErr, ok := err.(*errs.Error)
	require.True(t, ok)
	require.Equal(t, errs.RetServerOverload, int(trpcErr.Code))
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

func TestCloseWaitTime(t *testing.T) {
	startService := func(opts ...server.Option) (chan struct{}, func()) {
		received, done := make(chan struct{}), make(chan struct{})
		addr, stop := startService(t, &Greeter{}, append([]server.Option{server.WithFilter(
			func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
				received <- struct{}{}
				<-done
				return nil, errors.New("must fail")
			}), server.WithServerAsync(true)}, opts...)...)
		go func() {
			_, _ = pb.NewGreeterClientProxy(client.WithTarget("ip://"+addr)).
				SayHello(context.Background(), &pb.HelloRequest{})
		}()
		<-received
		return done, stop
	}
	t.Run(": active requests feature is not enabled on missing MaxCloseWaitTime", func(t *testing.T) {
		t.Parallel()
		done, stop := startService()
		defer close(done)
		start := time.Now()
		stop()
		require.Less(t, time.Since(start), time.Millisecond*200)
	})
	t.Run(": total wait time should not significantly greater than MaxCloseWaitTime", func(t *testing.T) {
		t.Parallel()
		const closeWaitTime, maxCloseWaitTime = time.Millisecond * 500, time.Second
		done, stop := startService(
			server.WithMaxCloseWaitTime(maxCloseWaitTime),
			server.WithCloseWaitTime(closeWaitTime))
		defer close(done)
		start := time.Now()
		stop()
		require.WithinRange(t, time.Now(),
			start.Add(maxCloseWaitTime),
			start.Add(maxCloseWaitTime).Add(time.Millisecond*200))
	})
	t.Run(": total wait time is at least CloseWaitTime", func(t *testing.T) {
		t.Parallel()
		const closeWaitTime, maxCloseWaitTime = time.Millisecond * 500, time.Second
		done, stop := startService(
			server.WithMaxCloseWaitTime(maxCloseWaitTime),
			server.WithCloseWaitTime(closeWaitTime))
		start := time.Now()
		time.AfterFunc(closeWaitTime/2, func() { close(done) })
		stop()
		require.WithinRange(t, time.Now(), start.Add(closeWaitTime), start.Add(closeWaitTime+time.Millisecond*200))
	})
	t.Run(": no active request before MaxCloseWaitTime", func(t *testing.T) {
		t.Parallel()
		const closeWaitTime, maxCloseWaitTime = time.Millisecond * 500, time.Second
		done, stop := startService(
			server.WithMaxCloseWaitTime(maxCloseWaitTime),
			server.WithCloseWaitTime(closeWaitTime))
		start := time.Now()
		time.AfterFunc((closeWaitTime+maxCloseWaitTime)/2, func() { close(done) })
		stop()
		require.WithinRange(t, time.Now(), start.Add(closeWaitTime), start.Add(maxCloseWaitTime))
	})
	t.Run(": no active request before service timeout", func(t *testing.T) {
		t.Parallel()
		const closeWaitTime, maxCloseWaitTime, timeout = time.Millisecond * 500, time.Second, time.Second * 2
		done, stop := startService(
			server.WithMaxCloseWaitTime(maxCloseWaitTime),
			server.WithCloseWaitTime(closeWaitTime),
			server.WithTimeout(timeout))
		start := time.Now()
		time.AfterFunc(maxCloseWaitTime+time.Second, func() { close(done) })
		stop()
		require.WithinRange(t, time.Now(),
			start.Add(maxCloseWaitTime+time.Second),
			start.Add(maxCloseWaitTime+timeout+time.Millisecond*200))
	})
}

func TestServicePreDecode(t *testing.T) {
	old := rpczenable.Enabled
	defer func() {
		rpczenable.Enabled = old
	}()
	rpczenable.Enabled = true
	codec.Register("fake", &fakeCodec{}, nil)

	// Test cases for various scenarios.
	tests := []struct {
		name              string
		input             []byte
		protocol          string
		expectError       bool
		expectHandleError bool
	}{
		{
			name:              "with invalid codec",
			input:             []byte("test-input"),
			protocol:          "not-exist",
			expectError:       true,
			expectHandleError: true,
		},
		{
			name:              "with valid codec",
			input:             []byte("test-input"),
			protocol:          "fake",
			expectError:       false,
			expectHandleError: false,
		},
		{
			name:              "with failing codec",
			input:             []byte("decode-error"),
			protocol:          "fake",
			expectError:       true,
			expectHandleError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := server.New(server.WithProtocol(tc.protocol))

			// Assert that the server implements the PreDecodeHandler interface.
			pdh, ok := s.(keeporder.PreDecodeHandler)
			require.True(t, ok, "server must implement keeporder.PreDecodeHandler")
			// Call PreDecode and capture the output.
			output, err := pdh.PreDecode(context.Background(), tc.input)

			if tc.expectError {
				require.Error(t, err, "expected an error")
			} else {
				require.NoError(t, err, "did not expect an error")
				require.NotNil(t, output, "expected non-nil output")
			}
			// Test Handle with pre-unmarshal value embedded.
			h, ok := s.(transport.Handler)
			require.True(t, ok)
			ctx = keeporder.NewContextWithPreDecode(ctx, &keeporder.PreDecodeInfo{
				ReqBodyBuf: output,
			})
			_, err = h.Handle(ctx, tc.input)
			if tc.expectHandleError {
				require.Error(t, err, "expected an error")
			} else {
				require.NoError(t, err, "expected no error")
			}
		})
	}
}

func TestServicePreUnmarshal(t *testing.T) {
	old := rpczenable.Enabled
	defer func() {
		rpczenable.Enabled = old
	}()
	rpczenable.Enabled = true
	s := server.New(
		server.WithProtocol("trpc"),
		server.WithNetwork("tcp"),
	)
	// Assert that the server implements the PreUnmarshalHandler interface.
	puh, ok := s.(keeporder.PreUnmarshalHandler)
	require.True(t, ok, "server must implement keeporder.PreUnmarshalHandler")
	ctx, msg := codec.EnsureMessage(trpc.BackgroundContext())
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	info := &keeporder.PreUnmarshalInfo{}
	ctx = keeporder.NewContextWithPreUnmarshal(ctx, info)
	req := &pb.HelloRequest{Msg: "hello"}
	reqBodyBytes, err := codec.Marshal(codec.SerializationTypePB, req)
	require.NoError(t, err)
	reqBuf, err := trpc.DefaultClientCodec.Encode(msg, reqBodyBytes)
	require.NoError(t, err)

	// Before pb register, there will be error.
	reqInterface, err := puh.PreUnmarshal(ctx, reqBuf)
	require.Error(t, err)
	pb.RegisterGreeterService(s, &Greeter{})

	// After pb register, there will be no error.
	reqInterface, err = puh.PreUnmarshal(ctx, reqBuf)
	require.NoError(t, err)
	unmarshaledReq, ok := reqInterface.(*pb.HelloRequest)
	require.True(t, ok)
	require.EqualValues(t, req.Msg, unmarshaledReq.Msg)

	// Test Handle with pre-unmarshal value embedded.
	h, ok := s.(transport.Handler)
	require.True(t, ok)
	_, err = h.Handle(ctx, reqBuf)
	require.NoError(t, err)
}

func startService(t *testing.T, gs pb.GreeterService, opts ...server.Option) (addr string, stop func()) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	require.Nil(t, err)

	s := server.New(append(append(
		[]server.Option{
			server.WithNetwork("tcp"),
			server.WithProtocol("trpc"),
		}, opts...),
		server.WithListener(l),
	)...)
	pb.RegisterGreeterService(s, gs)

	errCh := make(chan error)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(time.Millisecond * 200):
	}
	time.Sleep(200 * time.Millisecond)
	return l.Addr().String(), func() { s.Close(nil) }
}

type overloadControllerAlwaysFail struct{}

func (overloadControllerAlwaysFail) Acquire(context.Context, string) (overloadctrl.Token, error) {
	return nil, errors.New("always limited")
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
	sayHello func(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error)
}

func (g *Greeter) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	if g.sayHello != nil {
		return g.sayHello(ctx, req)
	}
	return &pb.HelloReply{}, nil
}

func (g *Greeter) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{}, nil
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

func TestServerTimeoutNormal(t *testing.T) {
	addr, stop := startService(t, &Greeter{
		sayHello: func(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
			// Wait until timeout.
			<-ctx.Done()
			// But do not return the timeout error.
			return &pb.HelloReply{}, nil
		},
	}, server.WithTimeout(10*time.Millisecond))
	defer stop()
	p := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
	_, err := p.SayHello(context.Background(), &pb.HelloRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "server context deadline exceeded")
}

func TestServerTimeoutFullLink(t *testing.T) {
	const (
		// Make sure client timeout is shorter than server timeout to
		// trigger full link timeout.
		clientTimeout = 5 * time.Millisecond
		serverTimeout = 10 * time.Millisecond
	)
	ch := make(chan error, 1)
	addr, stop := startService(t, &Greeter{
		sayHello: func(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
			// Wait until timeout.
			<-ctx.Done()
			// But do not return the timeout error.
			return &pb.HelloReply{}, nil
		},
	},
		server.WithTimeout(serverTimeout),
		server.WithNamedFilter("error_getter",
			func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
				rsp, err = next(ctx, req)
				ch <- err
				return
			}))
	defer stop()
	p := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
	_, err := p.SayHello(context.Background(), &pb.HelloRequest{}, client.WithTimeout(clientTimeout))
	require.Error(t, err)
	err = <-ch
	require.Error(t, err)
	require.Equal(t, errs.RetServerFullLinkTimeout, errs.Code(err))
	require.Contains(t, err.Error(), "server context deadline exceeded")
}

func TestServiceProfilerTagger(t *testing.T) {
	tempDir := t.TempDir()
	profilePath := filepath.Join(tempDir, "cpuprofile.pb.gz")

	// generate profile in profilePath
	generateProfile(t, profilePath)

	// Parse profile
	ff, err := os.Open(profilePath)
	if err != nil {
		t.Fatal("could not open CPU profile: ", err)
	}
	defer ff.Close()
	p, err := profile.Parse(ff)
	if err != nil {
		t.Fatal(err)
	}

	// Find the corresponding labelValue in the label by labelKey
	labels := make(map[string]string)
	for _, sample := range p.Sample {
		if sample.Label == nil {
			continue
		}
		for k, v := range sample.Label {
			if len(v) > 0 {
				labels[k] = v[0]
			}
		}
	}
	assert.Equal(t, map[string]string{"serviceName": "EmptyService"}, labels)
}

func generateProfile(t *testing.T, profilePath string) {
	// Setup CPU profiling.
	f, err := os.Create(profilePath)
	if err != nil {
		t.Fatal("could not create CPU profile:", err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal("could not start CPU profile:", err)
	}
	defer pprof.StopCPUProfile()

	addr, stop := startService(t,
		&Greeter{
			sayHello: func(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
				// make cpu busy
				for i := 0; i < 100_000_000; i++ {
				}
				return &pb.HelloReply{}, nil
			},
		},
		server.WithProfilerTagger(&serviceNameTagger{}))
	defer stop()

	c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + addr))
	_, err = c.SayHello(ctx, &pb.HelloRequest{})
	assert.Nil(t, err)
}

type serviceNameTagger struct {
}

func (t *serviceNameTagger) Tag(ctx context.Context, req interface{}) (*server.ProfileLabel, error) {
	profileLabel := server.NewProfileLabel()
	profileLabel.Store("serviceName", "EmptyService")
	return profileLabel, nil
}
