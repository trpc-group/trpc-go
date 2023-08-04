package server_test

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
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

	var os []server.Option
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
