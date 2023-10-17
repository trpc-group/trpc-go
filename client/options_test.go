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

package client_test

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestSelectOptions(t *testing.T) {
	opts := &client.Options{}

	var selectOptionNum int
	var callOptionNum int

	// WithCallerServiceName sets service name of the service itself
	o := client.WithCallerServiceName("trpc.test.helloworld1")
	selectOptionNum++
	o(opts)
	require.Equal(t, "trpc.test.helloworld1", opts.CallerServiceName)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithCallerNamespace("Production")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithCallerEnvName("test")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithCalleeEnvName("test")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithCallerSetName("set")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithCalleeSetName("set")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithCalleeMethod("func")
	o(opts)
	require.Equal(t, "func", opts.CalleeMethod)

	o = client.WithCallerMetadata("tag", "data")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithCalleeMetadata("tag", "data")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithPassword("passwd")
	callOptionNum++
	o(opts)
	require.Equal(t, callOptionNum, len(opts.CallOptions))

	o = client.WithConnectionMode(transport.Connected)
	callOptionNum++
	o(opts)
	require.Equal(t, callOptionNum, len(opts.CallOptions))

	o = client.WithSendOnly()
	callOptionNum++
	o(opts)
	require.Equal(t, opts.CallType, codec.SendOnly)
	require.Equal(t, callOptionNum, len(opts.CallOptions))

	o = client.WithTLS("client.cert", "client.key", "ca.pem", "servername")
	callOptionNum++
	o(opts)
	require.Equal(t, callOptionNum, len(opts.CallOptions))

	o = client.WithDisableConnectionPool()
	callOptionNum++
	o(opts)
	require.Equal(t, callOptionNum, len(opts.CallOptions))

	o = client.WithDiscoveryName("polaris")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithServiceRouterName("polaris")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	o = client.WithBalancerName("polaris")
	selectOptionNum += 2
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

}

// TestSelectOptionsOther tests other SelectOptions.
func TestSelectOptionsOther(t *testing.T) {
	opts := &client.Options{}

	var selectOptionNum int

	o := client.WithCircuitBreakerName("polaris")
	selectOptionNum++
	o(opts)
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	client.WithNamespace("development")(opts)
	selectOptionNum++
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	client.WithEnvKey("env-key")(opts)
	selectOptionNum++
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	client.WithKey("hash key")(opts)
	selectOptionNum++
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	client.WithReplicas(100)(opts)
	selectOptionNum++
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))

	client.WithDisableServiceRouter()(opts)
	selectOptionNum++
	require.Equal(t, selectOptionNum, len(opts.SelectOptions))
	require.True(t, opts.DisableServiceRouter)

	client.WithDisableFilter()(opts)
	require.Equal(t, true, opts.DisableFilter)

}

func TestOptions(t *testing.T) {

	opts := &client.Options{}
	transportOpts := &transport.RoundTripOptions{}

	// WithServiceName sets service name of backend service
	o := client.WithServiceName("trpc.test.helloworld")
	o(opts)
	require.Equal(t, "trpc.test.helloworld", opts.ServiceName)

	// WithTarget sets target address
	o = client.WithTarget("cl5://111:222")
	o(opts)
	require.Equal(t, "cl5://111:222", opts.Target)

	// WithNetwork sets network of backend service: tcp or udp, tcp by default
	o = client.WithNetwork("tcp")
	o(opts)
	for _, o := range opts.CallOptions {
		o(transportOpts)
	}
	require.Equal(t, "tcp", transportOpts.Network)

	// WithTimeout sets timeout of dialing backend, 1s by default.
	o = client.WithTimeout(time.Second)
	o(opts)
	for _, o := range opts.CallOptions {
		o(transportOpts)
	}
	require.Equal(t, time.Second, opts.Timeout)

	// WithTransport replaces client transport plugin
	o = client.WithTransport(transport.DefaultClientTransport)
	o(opts)
	require.Equal(t, transport.DefaultClientTransport, opts.Transport)

	// WithStreamTransport replaces client stream transport plugin
	o = client.WithStreamTransport(transport.DefaultClientStreamTransport)
	o(opts)
	require.Equal(t, transport.DefaultClientStreamTransport, opts.StreamTransport)

	// WithProtocol sets protocol of backend service like trpc
	o = client.WithProtocol("trpc")
	o(opts)
	for _, o := range opts.CallOptions {
		o(transportOpts)
	}
	require.Equal(t, trpc.DefaultClientCodec, opts.Codec)
	require.Equal(t, transport.DefaultClientTransport, opts.Transport)

	o = client.WithProtocol("http")
	o(opts)
	for _, o := range opts.CallOptions {
		o(transportOpts)
	}
	require.Equal(t, http.DefaultClientCodec, opts.Codec)
	require.Equal(t, http.DefaultClientTransport, opts.Transport)

	o = client.WithSerializationType(codec.SerializationTypePB)
	o(opts)
	require.Equal(t, codec.SerializationTypePB, opts.SerializationType)

	o = client.WithCompressType(codec.CompressTypeGzip)
	o(opts)
	require.Equal(t, codec.CompressTypeGzip, opts.CompressType)

	o = client.WithClientStreamQueueSize(1024)
	o(opts)
	require.Equal(t, 1024, opts.ClientStreamQueueSize)

	o = client.WithMaxWindowSize(1024)
	o(opts)
	require.Equal(t, uint32(1024), opts.MaxWindowSize)

	o = client.WithMultiplexed(true)
	o(opts)
	require.Equal(t, true, opts.EnableMultiplexed)

	// WithFilter appends a client filter to client filter chain.
	o = client.WithFilter(filter.NoopClientFilter)
	o(opts)
	require.Equal(t, 1, len(opts.Filters))

	// WithFilters appends multiple client filters to client filter chain.
	o = client.WithFilters([]filter.ClientFilter{filter.NoopClientFilter})
	o(opts)
	require.Equal(t, 2, len(opts.Filters))

	// WithPool sets custom conn pool
	opt := []connpool.Option{
		connpool.WithIdleTimeout(time.Duration(10) * time.Second),
	}
	pool := connpool.NewConnectionPool(opt...)
	o = client.WithPool(pool)
	o(opts)
	for _, o := range opts.CallOptions {
		o(transportOpts)
	}
	require.Equal(t, pool, transportOpts.Pool)
}

func TestDataOptions(t *testing.T) {
	opts := &client.Options{}

	// WithReqHead sets req head
	o := client.WithReqHead(nil)
	o(opts)
	require.Equal(t, nil, opts.ReqHead)

	// WithRspHead sets rsp head
	o = client.WithRspHead(nil)
	o(opts)
	require.Equal(t, nil, opts.RspHead)

	// WithSelectorNode records selected node
	node := &registry.Node{}
	o = client.WithSelectorNode(node)
	o(opts)
	require.Equal(t, node, opts.Node.Node)

	o = client.WithCurrentSerializationType(1)
	o(opts)
	require.Equal(t, 1, opts.CurrentSerializationType)

	o = client.WithCurrentCompressType(1)
	o(opts)
	require.Equal(t, 1, opts.CurrentCompressType)

	o = client.WithMetaData("key", []byte("value"))
	o(opts)
	require.Equal(t, []byte("value"), opts.MetaData["key"])
}

// TestWithMultiplexedPool tests WithMultiplexedPool.
func TestWithMultiplexedPool(t *testing.T) {
	opts := &client.Options{}
	roundTripOptions := &transport.RoundTripOptions{}
	m := multiplexed.New(multiplexed.WithConnectNumber(8))
	o := client.WithMultiplexedPool(m)
	o(opts)
	require.True(t, opts.EnableMultiplexed)
	for _, o := range opts.CallOptions {
		o(roundTripOptions)
	}
	require.Equal(t, m, roundTripOptions.Multiplexed)
}

func TestWithOptionsImmutable(t *testing.T) {
	ctx := context.Background()
	require.False(t, client.IsOptionsImmutable(ctx))

	newCtx := client.WithOptionsImmutable(ctx)
	require.True(t, client.IsOptionsImmutable(newCtx))
}

func TestWithLocalAddrOption(t *testing.T) {
	opts := &client.Options{}
	localAddr := "127.0.0.1:8080"
	o := client.WithLocalAddr("127.0.0.1:8080")
	o(opts)
	roundTripOptions := &transport.RoundTripOptions{}
	for _, o := range opts.CallOptions {
		o(roundTripOptions)
	}
	require.Equal(t, roundTripOptions.LocalAddr, localAddr)
}

func TestWithDialTimeoutOption(t *testing.T) {
	opts := &client.Options{}
	timeout := time.Second
	o := client.WithDialTimeout(timeout)
	o(opts)
	roundTripOptions := &transport.RoundTripOptions{}
	for _, o := range opts.CallOptions {
		o(roundTripOptions)
	}
	require.Equal(t, roundTripOptions.DialTimeout, timeout)
}

func TestWithNamedFilter(t *testing.T) {
	var (
		filterNames []string
		filters     filter.ClientChain

		cf = func(
			ctx context.Context,
			req, rsp interface{},
			next filter.ClientHandleFunc) error {
			return next(ctx, req, rsp)
		}
	)
	for i := 0; i < 10; i++ {
		filterNames = append(filterNames, fmt.Sprintf("filter-%d", i))
		filters = append(filters, cf)
	}

	var os []client.Option
	for i := range filters {
		os = append(os, client.WithNamedFilter(filterNames[i], filters[i]))
	}

	options := &client.Options{}
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
