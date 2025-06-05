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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	pb "trpc.group/trpc-go/trpc-go/testdata"
	"trpc.group/trpc-go/trpc-go/transport"

	_ "trpc.group/trpc-go/trpc-go"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

var ctx = context.Background()

type fakeHandler struct {
}

func (s *fakeHandler) Handle(ctx context.Context, req []byte) (rsp []byte, err error) {
	return req, nil
}

func TestOptions(t *testing.T) {

	opts := &server.Options{}
	transportOpts := &transport.ListenServeOptions{}

	// WithServiceName
	o := server.WithServiceName("trpc.test.helloworld")
	o(opts)
	assert.Equal(t, opts.ServiceName, "trpc.test.helloworld")

	o = server.WithNamespace("Development")
	o(opts)
	assert.Equal(t, opts.Namespace, "Development")

	o = server.WithEnvName("formal")
	o(opts)
	assert.Equal(t, opts.EnvName, "formal")

	o = server.WithSetName("a.b.c")
	o(opts)
	assert.Equal(t, opts.SetName, "a.b.c")

	// WithDisableRequestTimeout
	assert.Equal(t, opts.DisableRequestTimeout, false) // false by default
	o = server.WithDisableRequestTimeout(true)
	o(opts)
	assert.Equal(t, opts.DisableRequestTimeout, true)

	assert.Nil(t, opts.OverloadCtrl)
	o = server.WithOverloadCtrl(overloadctrl.NoopOC{})
	o(opts)
	assert.NotNil(t, opts.OverloadCtrl)

	// WithAddress
	o = server.WithAddress("127.0.0.1:8080")
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.Address, "127.0.0.1:8080")
	assert.Equal(t, opts.Address, "127.0.0.1:8080")

	// WithNetwork
	o = server.WithNetwork("tcp")
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.Network, "tcp")

	lis, _ := net.Listen("tcp", "127.0.0.1:8080")
	o = server.WithListener(lis)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.Listener, lis)
	if lis != nil {
		lis.Close()
	}

	o = server.WithTLS("server.crt", "server.key", "ca.pem")
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.TLSCertFile, "server.crt")
	assert.Equal(t, transportOpts.TLSKeyFile, "server.key")
}

func TestMoreOptions(t *testing.T) {
	// WithHandler
	h := &fakeHandler{}
	o := server.WithHandler(h)
	opts := &server.Options{}
	transportOpts := &transport.ListenServeOptions{}
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.Handler, h)

	// WithTimeout
	o = server.WithTimeout(time.Second)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, opts.Timeout, time.Second)

	// WithReadTimeout
	o = server.WithReadTimeout(time.Second)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, opts.Timeout, time.Second)

	// WithTransport
	o = server.WithTransport(transport.DefaultServerTransport)
	o(opts)
	assert.Equal(t, opts.Transport, transport.DefaultServerTransport)

	// register ServerTransport
	transport.RegisterServerTransport("trpc", transport.DefaultServerTransport)
	// WithProtocol
	o = server.WithProtocol("trpc")
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.NotEqual(t, opts.Codec, nil)

	o = server.WithProtocol("fake")
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}

	o = server.WithCurrentSerializationType(codec.SerializationTypeNoop)
	o(opts)
	assert.Equal(t, opts.CurrentSerializationType, codec.SerializationTypeNoop)

	o = server.WithCurrentCompressType(codec.CompressTypeSnappy)
	o(opts)
	assert.Equal(t, opts.CurrentCompressType, codec.CompressTypeSnappy)

	// WithFilter
	o = server.WithFilter(filter.NoopServerFilter)
	o(opts)
	assert.Equal(t, len(opts.Filters), 1)

	// WithFilters
	o = server.WithFilters([]filter.ServerFilter{filter.NoopServerFilter})
	o(opts)
	assert.Equal(t, len(opts.Filters), 2)

	// WithStreamFilter
	sf1 := func(ss server.Stream, info *server.StreamServerInfo, handler server.StreamHandler) error {
		return nil
	}
	o = server.WithStreamFilter(sf1)
	o(opts)
	assert.Equal(t, 1, len(opts.StreamFilters))

	// WithStreamFilters
	sf2 := func(ss server.Stream, info *server.StreamServerInfo, handler server.StreamHandler) error {
		return nil
	}
	o = server.WithStreamFilters(sf1, sf2)
	o(opts)
	assert.Equal(t, 3, len(opts.StreamFilters))

	// WithRegistry
	o = server.WithRegistry(registry.DefaultRegistry)
	o(opts)
	assert.Equal(t, registry.DefaultRegistry, opts.Registry)

	// WithServerAsync
	o = server.WithServerAsync(true)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.ServerAsync, true)

	// WithMaxRoutines
	o = server.WithMaxRoutines(100)
	// WithWritev
	o = server.WithWritev(true)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.Writev, true)

	// WithMaxRoutines
	o = server.WithMaxRoutines(100)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.Routines, 100)

	// WithStreamTransport
	o = server.WithStreamTransport(transport.DefaultServerStreamTransport)
	o(opts)
	assert.Equal(t, opts.StreamTransport, transport.DefaultServerStreamTransport)

	// WithCloseWaitTime
	o = server.WithCloseWaitTime(0 * time.Millisecond)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, opts.CloseWaitTime, 0*time.Millisecond)

	// WithMaxCloseWaitTime
	o = server.WithMaxCloseWaitTime(100 * time.Millisecond)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, opts.MaxCloseWaitTime, 100*time.Millisecond)

	// WithDisableGracefulRestart
	o = server.WithDisableGracefulRestart(true)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, opts.DisableGracefulRestart, true)

	// WithRESTOptions
	o1 := server.WithRESTOptions(restful.WithServiceName("name a"))
	o2 := server.WithRESTOptions(restful.WithServiceName("name b"))
	o1(opts)
	o2(opts)
	restOptions := &restful.Options{}
	for _, o := range opts.RESTOptions {
		o(restOptions)
	}
	assert.Equal(t, 2, len(opts.RESTOptions))
	assert.Equal(t, "name b", restOptions.ServiceName)

	// WithIdleTimeout
	idleTimeout := time.Second
	o = server.WithIdleTimeout(idleTimeout)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, transportOpts.IdleTimeout, idleTimeout)

	// WithDisableKeepAlives
	disableKeepAlives := true
	o = server.WithDisableKeepAlives(disableKeepAlives)
	o(opts)
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}
	assert.Equal(t, disableKeepAlives, transportOpts.DisableKeepAlives)

	// WithMaxWindowSize
	var maxWindowSize uint32 = 100
	o = server.WithMaxWindowSize(maxWindowSize)
	o(opts)
	assert.Equal(t, maxWindowSize, opts.MaxWindowSize)

	// WithProfilerTagger
	var profilerTagger server.ProfilerTagger = &noopTagger{}
	expectedFilterLength := len(opts.Filters) + 1
	o = server.WithProfilerTagger(profilerTagger)
	o(opts)
	assert.Contains(t, opts.FilterNames, "profiler_tagger_filter")
	assert.Equal(t, expectedFilterLength, len(opts.Filters))
	assert.Equal(t, expectedFilterLength, len(opts.FilterNames))

	// WithStreamProfilerTagger
	var streamProfilerTagger server.StreamProfilerTagger = &noopStreamTagger{}
	expectedStreamFilterLength := len(opts.StreamFilters) + 1
	o = server.WithStreamProfilerTagger(streamProfilerTagger)
	o(opts)
	assert.Equal(t, expectedStreamFilterLength, len(opts.StreamFilters))
}

func TestKeepOrderOptions(t *testing.T) {
	opts := &server.Options{}
	transportOpts := &transport.ListenServeOptions{}
	fn := func(ctx context.Context, reqBody []byte) (string, bool) {
		return "key", true
	}
	o := server.WithKeepOrderPreDecodeExtractor(fn)
	o(opts)

	// Apply the server options to the transport options.
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}

	// Check if the function is set and behaves as expected.
	if transportOpts.KeepOrderPreDecodeExtractor == nil {
		t.Fatalf("KeepOrderPreDecodeExtractor should not be nil")
	}

	// Invoke the function and check the output.
	key, valid := transportOpts.KeepOrderPreDecodeExtractor(context.Background(), []byte{})
	require.Equal(t, "key", key, "Expected function to return 'key'")
	require.True(t, valid, "Expected function to return true")
}

func TestWithKeepOrderPreUnmarshalExtractor(t *testing.T) {
	opts := &server.Options{}
	transportOpts := &transport.ListenServeOptions{}
	fn := func(ctx context.Context, req interface{}) (string, bool) {
		return "unmarshal_key", true
	}
	o := server.WithKeepOrderPreUnmarshalExtractor(fn)
	o(opts)

	// Apply the server options to the transport options.
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}

	// Check if the function is set and behaves as expected.
	if transportOpts.KeepOrderPreUnmarshalExtractor == nil {
		t.Fatalf("KeepOrderPreUnmarshalExtractor should not be nil")
	}

	// Invoke the function and check the output.
	key, valid := transportOpts.KeepOrderPreUnmarshalExtractor(context.Background(), "request")
	require.Equal(t, "unmarshal_key", key, "Expected function to return 'unmarshal_key'")
	require.True(t, valid, "Expected function to return true")
}

func TestWithOrderedGroups(t *testing.T) {
	opts := &server.Options{}
	transportOpts := &transport.ListenServeOptions{}
	groups := &simpleOrderedGroups{}

	o := server.WithOrderedGroups(groups)
	o(opts)

	// Apply the server options to the transport options.
	for _, o := range opts.ServeOptions {
		o(transportOpts)
	}

	// Check if the groups are set correctly.
	require.Equal(t, groups, transportOpts.OrderedGroups, "Expected groups to be set correctly in the transport options")

	// Test the behavior of the OrderedGroups through the manual implementation.
	groups.Add("testKey", func() {})
	groups.Remove("testKey")
	groups.Stop()

	// Verify that the methods were called as expected.
	require.True(t, groups.AddedKeys["testKey"], "Expected Add to be called with 'testKey'")
	require.True(t, groups.RemovedKeys["testKey"], "Expected Remove to be called with 'testKey'")
	require.True(t, groups.Stopped, "Expected Stop to be called")
}

// simpleOrderedGroups is a simple implementation of the OrderedGroups interface for testing.
type simpleOrderedGroups struct {
	AddedKeys   map[string]bool
	RemovedKeys map[string]bool
	Stopped     bool
}

func (s *simpleOrderedGroups) Add(key string, fn func()) {
	if s.AddedKeys == nil {
		s.AddedKeys = make(map[string]bool)
	}
	s.AddedKeys[key] = true
	fn() // Execute the function to simulate real behavior.
}

func (s *simpleOrderedGroups) Remove(key string) {
	if s.RemovedKeys == nil {
		s.RemovedKeys = make(map[string]bool)
	}
	s.RemovedKeys[key] = true
}

func (s *simpleOrderedGroups) Stop() {
	s.Stopped = true
}

func TestWithNamedFilter(t *testing.T) {
	var (
		filterNames []string
		filters     filter.ServerChain

		sf = func(
			ctx context.Context,
			req interface{},
			next filter.ServerHandleFunc,
		) (rsp interface{}, err error) {
			return next(ctx, req)
		}
	)
	for i := 0; i < 10; i++ {
		filterNames = append(filterNames, fmt.Sprintf("filter-%d", i))
		filters = append(filters, sf)
	}

	os := make([]server.Option, 0, len(filters))
	for i := range filters {
		os = append(os, server.WithNamedFilter(filterNames[i], filters[i]))
	}

	options := &server.Options{}
	for _, o := range os {
		o(options)
	}
	require.Equal(t, filterNames, options.FilterNames)
	require.Equal(t, len(filters), len(options.Filters))
	for i := range filters {
		require.Equal(
			t,
			runtime.FuncForPC(reflect.ValueOf(filters[i]).Pointer()).Name(),
			runtime.FuncForPC(reflect.ValueOf(options.Filters[i]).Pointer()).Name(),
		)
	}
}

func TestWithOnResponseObsoleted(t *testing.T) {
	rspPool := &sync.Pool{
		New: func() interface{} {
			return &pb.HelloReply{}
		},
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	s := server.New(server.WithOnResponseObsoleted(func(_ context.Context, rsp interface{}) {
		rspPool.Put(rsp)
	}), server.WithListener(ln), server.WithProtocol("trpc"))
	pb.RegisterGreeterService(s, &impl{rspPool: rspPool})
	go s.Serve()
	proxy := pb.NewGreeterClientProxy(client.WithTarget(fmt.Sprintf("ip://%s", ln.Addr())))
	rsp, err := proxy.SayHello(trpc.BackgroundContext(), &pb.HelloRequest{Msg: "hello"})
	require.Nil(t, err)
	require.NotNil(t, rsp)
}

type impl struct {
	rspPool *sync.Pool
}

func (i *impl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	rsp := i.rspPool.Get().(*pb.HelloReply)
	rsp.Msg = req.Msg
	return rsp, nil
}

func (i *impl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return nil, nil
}

func TestWithServiceOption(t *testing.T) {
	var cfg trpc.Config
	require.Nil(t, yaml.Unmarshal([]byte(`
server:
  service:
    - name: trpc.test.helloworld.Greeter1
    - name: trpc.test.helloworld.Greeter2
`), &cfg))

	// set logger to file
	logDir := t.TempDir()
	logger := log.NewZapLog(log.Config{
		{
			Writer: log.OutputFile,
			WriteConfig: log.WriteConfig{
				LogPath:   logDir,
				Filename:  "trpc.log",
				WriteMode: log.WriteSync,
			},
			Level: "DEBUG",
		},
	})
	dftLogger := log.DefaultLogger
	log.SetLogger(logger)
	defer log.SetLogger(dftLogger)

	// new server with two services with different address
	serviceName1, address1 := "trpc.test.helloworld.Greeter1", "127.0.0.1"
	serviceName2, address2 := "trpc.test.helloworld.Greeter2", "127.0.0.2"
	var printAddress server.Option = func(o *server.Options) { log.Infof("%v address %v", o.ServiceName, o.Address) }
	s := trpc.NewServerWithConfig(&cfg,
		server.WithServiceOption(serviceName1, server.WithAddress(address1)),
		server.WithServiceOption(serviceName2, server.WithAddress(address2)),
		server.WithServiceOption(serviceName1, printAddress),
		server.WithServiceOption(serviceName2, printAddress))
	assert.NotNil(t, s)

	// read log from file
	fp := filepath.Join(logDir, "trpc.log")
	buf, err := os.ReadFile(fp)
	assert.Nil(t, err)

	// WithServiceOption set different address for different service
	assert.Contains(t, string(buf), fmt.Sprintf("%v address %v", serviceName1, address1))
	assert.Contains(t, string(buf), fmt.Sprintf("%v address %v", serviceName2, address2))
}

type noopTagger struct{}

func (t *noopTagger) Tag(ctx context.Context, req interface{}) (*server.ProfileLabel, error) {
	return nil, nil
}

type noopStreamTagger struct{}

func (t *noopStreamTagger) Tag(ctx context.Context, info *server.StreamServerInfo) (*server.ProfileLabel, error) {
	return nil, nil
}

func (t *noopStreamTagger) TagRecvMsg(ctx context.Context) (*server.ProfileLabel, error) {
	return nil, nil
}

func (t *noopStreamTagger) TagSendMsg(ctx context.Context, m interface{}) (*server.ProfileLabel, error) {
	return nil, nil
}
