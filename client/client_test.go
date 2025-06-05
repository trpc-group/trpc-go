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
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	iserver "trpc.group/trpc-go/trpc-go/internal/local/server"
	inaming "trpc.group/trpc-go/trpc-go/internal/naming"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	pb "trpc.group/trpc-go/trpc-go/testdata"
	"trpc.group/trpc-go/trpc-go/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "trpc.group/trpc-go/trpc-go"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

func TestMain(m *testing.M) {
	transport.DefaultClientTransport = &fakeTransport{}
	selector.Register("fake", &fakeSelector{}) // fake://{endpoint}
	transport.RegisterClientTransport("fake", &fakeTransport{})
	m.Run()
}

func TestClient(t *testing.T) {
	ctx := context.Background()
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})

	cli := client.New()
	require.Equal(t, cli, client.DefaultClient)

	// test if response is valid
	reqBody := &codec.Body{Data: []byte("body")}
	rspBody := &codec.Body{}
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second), client.WithProtocol("fake")))
	require.Equal(t, []byte("body"), rspBody.Data)

	// test setting req/resp head
	reqhead := &registry.Node{}
	rsphead := &registry.Node{}
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithReqHead(reqhead), client.WithRspHead(rsphead), client.WithProtocol("fake")))

	// test client options
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second),
		client.WithServiceName("trpc.app.callee.service"),
		client.WithCallerServiceName("trpc.app.caller.service"),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCompressType(codec.CompressTypeGzip),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithMetaData("key", []byte("value")),
		client.WithProtocol("fake")))

	// test selecting node with network: udp
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("fake://udpnetwork"),
		client.WithTimeout(time.Second), client.WithProtocol("fake")))

	// test selecting node with network: unknown, which will use tcp by default
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("fake://unknownnetwork"),
		client.WithTimeout(time.Second), client.WithProtocol("fake")))

	// test setting namespace in msg
	ctx = context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithNamespace("Development") // getServiceInfoOptions will set env info according to the namespace
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second), client.WithProtocol("fake")))
	require.Equal(t, []byte("body"), rspBody.Data)

	// test that env info from upstream service has higher priority
	ctx = context.Background()
	ctx, msg = codec.WithNewMessage(ctx)
	msg.WithEnvTransfer("faketransfer") // env info from upstream service exists
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second), client.WithProtocol("fake")))
	require.Equal(t, []byte("body"), rspBody.Data)

	// test disabling service router, which will clear env info from msg
	ctx = context.Background()
	ctx, msg = codec.WithNewMessage(ctx)
	msg.WithEnvTransfer("faketransfer") // env info from upstream service exists
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second), client.WithProtocol("fake"),
		client.WithDisableServiceRouter())) // opts that disables service router
	require.Equal(t, []byte("body"), rspBody.Data)
	require.Equal(t, msg.EnvTransfer(), "") // env info from upstream service was cleared

	// test setting CalleeMethod in opts
	// updateMsg will then update CalleeMethod in msg
	ctx = context.Background()
	ctx, msg = codec.WithNewMessage(ctx)
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second), client.WithProtocol("fake"),
		client.WithCalleeMethod("fakemethod"))) // opts 中指定了 CalleeMethod
	require.Equal(t, msg.CalleeMethod(), "fakemethod") // msg 中的 CalleeMethod 被更新

	// test that the parameters can be extracted from msg in the prev filter
	ctx = context.Background()
	ctx, msg = codec.WithNewMessage(ctx)
	rid := uint32(100000)
	msg.WithRequestID(uint32(rid))

	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second), client.WithProtocol("fake"),
		client.WithFilter(func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) (err error) {
			msg := trpc.Message(ctx)
			require.Equal(t, rid, msg.RequestID())
			return f(ctx, req, rsp)
		})))

	// test setting CallType in opts
	// updateMsg will then update CallType in msg
	ctx = context.Background()
	head := &trpc.RequestProtocol{}
	ctx, msg = codec.WithNewMessage(ctx)
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"),
		client.WithSendOnly(),
		client.WithReqHead(head),
	))
	require.Equal(t, msg.CallType(), codec.SendOnly)

	// test setting invalid tag in opts
	ctx = context.Background()
	head = &trpc.RequestProtocol{}
	ctx, _ = codec.WithNewMessage(ctx)
	require.NotNil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"),
		client.WithReqHead(head),
		client.WithTag("Non-existed"),
	))
}

func TestBroadcastClient(t *testing.T) {
	ctx := context.Background()
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})

	bc := client.NewBroadcastClient[codec.Body]()

	reqBody := &codec.Body{Data: []byte("body")}
	rsps, err := bc.BroadcastInvoke(ctx, reqBody,
		client.WithTarget("fake://broadcast.service"),
		client.WithProtocol("fake"),
	)
	require.Nil(t, err)
	expectedRsps := [3]client.BroadcastRsp[codec.Body]{
		{
			Node: &registry.Node{Address: "127.0.0.1:8080"},
			Rsp:  &codec.Body{Data: []byte("body")},
			Err:  nil,
		},
		{
			Node: &registry.Node{Address: "127.0.0.1:8081"},
			Rsp:  &codec.Body{Data: []byte("body")},
			Err:  nil,
		},
		{
			Node: &registry.Node{Address: "127.0.0.1:8082"},
			Rsp:  &codec.Body{Data: []byte("body")},
			Err:  nil,
		},
	}
	for i := 0; i < len(rsps); i++ {
		require.Equal(t, expectedRsps[i].Node.Address, rsps[i].Node.Address, "Address mismatch at index %d", i)
		require.Equal(t, expectedRsps[i].Rsp, rsps[i].Rsp, "Rsp mismatch at index %d", i)
		require.Equal(t, expectedRsps[i].Err, rsps[i].Err, "Err mismatch at index %d", i)
	}
}

func TestClientFail(t *testing.T) {
	ctx := context.Background()
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})

	cli := client.New()
	require.Equal(t, cli, client.DefaultClient)

	reqBody := &codec.Body{Data: []byte("body")}
	rspBody := &codec.Body{}
	// test code failure
	require.NotNil(t, cli.Invoke(ctx, reqBody, rspBody,
		client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second),
		client.WithSerializationType(codec.SerializationTypeNoop)))

	// test invalid target
	err := cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip/:/127.0.0.1:8080"),
		client.WithProtocol("fake"))
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "invalid")

	// test target selector that not exists
	err = cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("cl6://127.0.0.1:8080"),
		client.WithProtocol("fake"))
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "not exist")

	// test recording selected node
	node := &registry.Node{}
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithSelectorNode(node), client.WithProtocol("fake")))
	require.Equal(t, node.Address, "127.0.0.1:8080")
	require.Equal(t, node.ServiceName, "127.0.0.1:8080")
	require.Empty(t, node.Network)

	// test encode failure
	reqBody = &codec.Body{Data: []byte("failbody")}
	require.NotNil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop)))

	// test network failure
	reqBody = &codec.Body{Data: []byte("callfail")}
	err = cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop))
	assert.NotNil(t, err)

	// test response failure
	reqBody = &codec.Body{Data: []byte("businessfail")}
	err = cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop))
	assert.NotNil(t, err)

	reqBody = &codec.Body{Data: []byte("msgfail")}
	err = cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop))
	assert.NotNil(t, err)

	// test nil rsp
	reqBody = &codec.Body{Data: []byte("nilrsp")}
	err = cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop))
	assert.Nil(t, err)

	// test timeout
	reqBody = &codec.Body{Data: []byte("timeout")}
	err = cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop))
	assert.NotNil(t, err)

	// test select node failure
	reqBody = &codec.Body{Data: []byte("body")}
	err = cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("fake://selectfail"),
		client.WithTimeout(time.Second), client.WithProtocol("fake"))
	assert.NotNil(t, err)

	// test selecting the node with empty addr
	err = cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("fake://emptynode"),
		client.WithTimeout(time.Second), client.WithProtocol("fake"))
	assert.NotNil(t, err)

}

func TestBroadcastClientFail(t *testing.T) {
	ctx := context.Background()
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})

	bc := client.NewBroadcastClient[codec.Body]()

	reqBody := &codec.Body{Data: []byte("body")}
	rsps, err := bc.BroadcastInvoke(ctx, reqBody,
		client.WithTarget("fake://broadcast.emptyList.service"),
		client.WithProtocol("fake"),
	)
	require.Nil(t, rsps)
	require.Error(t, err)

	reqBody = &codec.Body{Data: []byte("body")}
	rsps, err = bc.BroadcastInvoke(ctx, reqBody,
		client.WithTarget("fake://broadcast.noList.service"),
		client.WithProtocol("fake"),
	)
	require.Nil(t, rsps)
	require.Error(t, err)

	reqBody = &codec.Body{Data: []byte("body")}
	rsps, err = bc.BroadcastInvoke(ctx, reqBody,
		client.WithTarget("fake://broadcast.wrongList.service"),
		client.WithProtocol("fake"),
	)
	require.Nil(t, rsps)
	require.Error(t, err)

	reqFialedBody := &codec.Body{Data: []byte("nilrsp")}
	rsps, err = bc.BroadcastInvoke(ctx, reqFialedBody,
		client.WithTarget("fake://broadcast.service"),
		client.WithProtocol("fake"),
	)
	require.NotNil(t, rsps)
	require.Nil(t, err)
	for i := 0; i < len(rsps); i++ {
		require.Equal(t, &codec.Body{Data: []byte(nil)}, rsps[i].Rsp, "Rsp mismatch at index %d", i)
		require.Nil(t, err)
	}

	reqFialedBody = &codec.Body{Data: []byte("callfail")}
	rsps, err = bc.BroadcastInvoke(ctx, reqFialedBody,
		client.WithTarget("fake://broadcast.service"),
		client.WithProtocol("fake"),
	)
	require.NotNil(t, rsps)
	require.Error(t, err)
	for i := 0; i < len(rsps); i++ {
		require.Equal(t, &codec.Body{Data: []byte(nil)}, rsps[i].Rsp, "Rsp mismatch at index %d", i)
		require.NotNil(t, err, rsps[i].Err, "Err mismatch at index %d", i)
	}

	reqFialedBody = &codec.Body{Data: []byte("one_fail")}
	rsps, err = bc.BroadcastInvoke(ctx, reqFialedBody,
		client.WithTarget("fake://broadcast.service"),
		client.WithProtocol("fake"),
	)
	require.NotNil(t, rsps)
	require.Error(t, err)
	expectedRsps := [3]client.BroadcastRsp[codec.Body]{
		{
			Node: &registry.Node{Address: "127.0.0.1:8080"},
			Rsp:  &codec.Body{Data: []byte(nil)},
			Err:  errors.New("transport call fail"),
		},
		{
			Node: &registry.Node{Address: "127.0.0.1:8081"},
			Rsp:  &codec.Body{Data: []byte("one_fail")},
			Err:  nil,
		},
		{
			Node: &registry.Node{Address: "127.0.0.1:8082"},
			Rsp:  &codec.Body{Data: []byte("one_fail")},
			Err:  nil,
		},
	}
	for i := 0; i < len(rsps); i++ {
		require.Equal(t, expectedRsps[i].Node.Address, rsps[i].Node.Address, "Address mismatch at index %d", i)
		require.Equal(t, expectedRsps[i].Rsp, rsps[i].Rsp, "Rsp mismatch at index %d", i)
		require.Equal(t, expectedRsps[i].Err, rsps[i].Err, "Err mismatch at index %d", i)
	}
}

func TestClientAddrResolve(t *testing.T) {
	ctx := context.Background()
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})
	cli := client.New()

	reqBody := &codec.Body{Data: []byte("body")}
	rspBody := &codec.Body{}
	// test target with ip schema
	nctx, _ := codec.WithNewMessage(ctx)
	_ = cli.Invoke(nctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"), client.WithProtocol("fake"))
	assert.Equal(t, "127.0.0.1:8080", codec.Message(nctx).RemoteAddr().String())

	// test target with ip schema and network: tcp
	nctx, _ = codec.WithNewMessage(ctx)
	_ = cli.Invoke(nctx, reqBody, rspBody,
		client.WithTarget("ip://127.0.0.1:8080"),
		client.WithNetwork("tcp"),
		client.WithProtocol("fake"),
	)
	require.Equal(t, "127.0.0.1:8080", codec.Message(nctx).RemoteAddr().String())

	// test target with hostname schema
	nctx, _ = codec.WithNewMessage(ctx)
	err := cli.Invoke(nctx, reqBody, rspBody, client.WithTarget("ip://www.qq.com:8080"), client.WithProtocol("fake"))
	require.Nil(t, err)
	assert.Nil(t, codec.Message(nctx).RemoteAddr())

	// test calling target with ip schema failure
	nctx, msg := codec.WithNewMessage(ctx)
	reqBody = &codec.Body{Data: []byte("callfail")}
	err = cli.Invoke(nctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"), client.WithProtocol("fake"))
	assert.NotNil(t, err)
	assert.Equal(t, "127.0.0.1:8080", msg.RemoteAddr().String())

	// test target with unix schema
	nctx, _ = codec.WithNewMessage(ctx)
	_ = cli.Invoke(nctx, reqBody, rspBody,
		client.WithTarget("unix://temp.sock"),
		client.WithNetwork("unix"),
		client.WithProtocol("fake"),
	)
	require.Equal(t, "temp.sock", codec.Message(nctx).RemoteAddr().String())
}

func TestTimeout(t *testing.T) {
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})
	target, protocol := "ip://127.0.0.1:8080", "fake"

	cli := client.New()
	rspBody := &codec.Body{}
	err := cli.Invoke(context.Background(),
		&codec.Body{Data: []byte("timeout")}, rspBody,
		client.WithTarget(target),
		client.WithProtocol(protocol))
	require.NotNil(t, err)
	e, ok := err.(*errs.Error)
	require.True(t, ok)
	require.Equal(t, int32(errs.RetClientTimeout), e.Code)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err = cli.Invoke(ctx,
		&codec.Body{Data: []byte("timeout")}, rspBody,
		client.WithTarget(target),
		client.WithProtocol(protocol))
	require.NotNil(t, err)
	e, ok = err.(*errs.Error)
	require.True(t, ok)
	require.Equal(t, int32(errs.RetClientFullLinkTimeout), e.Code)

}

func TestSameCalleeMultiServiceName(t *testing.T) {
	callee := "trpc.test.pbcallee"
	serviceNames := []string{
		"trpc.test.helloworld0",
		"trpc.test.helloworld1",
		"trpc.test.helloworld2",
		"trpc.test.helloworld3",
	}
	for i := range serviceNames {
		if i != 2 {
			require.Nil(t, client.RegisterClientConfig(callee, &client.BackendConfig{
				ServiceName: serviceNames[i],
				Compression: codec.CompressTypeSnappy,
			}))
			continue
		}
		require.Nil(t, client.RegisterClientConfig(callee, &client.BackendConfig{
			ServiceName: serviceNames[i],
			Compression: codec.CompressTypeBlockSnappy,
		}))
	}
	ctx, msg := codec.EnsureMessage(context.Background())
	msg.WithCalleeServiceName(callee)
	require.NotNil(t, client.DefaultClient.Invoke(ctx, nil, nil, client.WithServiceName(serviceNames[0])))
	require.Equal(t, codec.CompressTypeSnappy, msg.CompressType())

	ctx, msg = codec.EnsureMessage(context.Background())
	msg.WithCalleeServiceName(callee)
	require.NotNil(t, client.DefaultClient.Invoke(ctx, nil, nil, client.WithServiceName(serviceNames[2])))
	require.Equal(t, codec.CompressTypeBlockSnappy, msg.CompressType())
}

func TestMultiplexedUseLatestMsg(t *testing.T) {
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	const target = "ip://127.0.0.1:8080"

	rspBody := &codec.Body{}
	require.Nil(t, client.New().Invoke(context.Background(),
		&codec.Body{Data: []byte(t.Name())}, rspBody,
		client.WithTarget(target),
		client.WithTransport(&multiplexedTransport{
			require: func(_ context.Context, _ []byte, opts ...transport.RoundTripOption) {
				var o transport.RoundTripOptions
				for _, opt := range opts {
					opt(&o)
				}
				require.NotZero(t, o.Msg.RequestID())
			}}),
		client.WithMultiplexed(true),
		client.WithFilter(func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
			// make a copy of the msg, after next, copy the new msg back.
			oldMsg := codec.Message(ctx)
			ctx, msg := codec.WithNewMessage(ctx)
			codec.CopyMsg(msg, oldMsg)
			err := next(ctx, req, rsp)
			codec.CopyMsg(oldMsg, msg)
			return err
		}),
	))
}

func TestFixTimeout(t *testing.T) {
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})
	target, protocol := "ip://127.0.0.1:8080", "fake"

	cli := client.New()

	rspBody := &codec.Body{}
	t.Run("RetClientCanceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := cli.Invoke(ctx,
			&codec.Body{Data: []byte("clientCanceled")}, rspBody,
			client.WithTarget(target),
			client.WithProtocol(protocol))
		require.Equal(t, errs.RetClientCanceled, errs.Code(err))
	})

	t.Run("RetClientFullLinkTimeout", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Millisecond))
		defer cancel()
		var d time.Duration
		deadline, ok := t.Deadline()
		if !ok {
			d = 5 * time.Second
		} else {
			const arbitraryCleanupMargin = 1 * time.Second
			d = time.Until(deadline) - arbitraryCleanupMargin
		}
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-timer.C:
			t.Fatalf(" context not timed out after %v", d)
		case <-ctx.Done():
		}
		if e := ctx.Err(); e != context.DeadlineExceeded {
			t.Errorf("c.Err() == %v; want %v", e, context.DeadlineExceeded)
		}
		err := cli.Invoke(ctx,
			&codec.Body{Data: []byte("fixTimeout")}, rspBody,
			client.WithTarget(target),
			client.WithProtocol(protocol))
		require.Equal(t, errs.RetClientFullLinkTimeout, errs.Code(err))
	})

	t.Run("RetClientTimeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := cli.Invoke(ctx,
			&codec.Body{Data: []byte("timeout")}, rspBody,
			client.WithTarget(target),
			client.WithTimeout(0),
			client.WithProtocol(protocol))
		require.NotNil(t, err)
		e, ok := err.(*errs.Error)
		require.True(t, ok)
		require.Equal(t, int32(errs.RetClientTimeout), e.Code)
	})
}

func TestMethodTimeout(t *testing.T) {
	newInt := func(i int) *int { return &i }
	require.Nil(t, client.RegisterClientConfig(t.Name(), &client.BackendConfig{
		Callee:      t.Name(),
		ServiceName: t.Name(),
		Timeout:     200,
		Method: map[string]*client.MethodConfig{
			"M1": {Timeout: newInt(100)},
			"M2": {Timeout: newInt(300)},
		},
	}))

	ctx, msg := codec.EnsureMessage(context.Background())
	msg.WithCalleeServiceName(t.Name())
	invoke := func(method string, opts ...client.Option) {
		msg.WithCalleeMethod(method)
		require.Nil(t, client.New().Invoke(ctx, nil, nil, append(opts, client.WithFilter(
			func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(time.Second):
					return errors.New("wait ctx done timeout")
				}
			}))...))
	}

	t.Run("method_timeout_not_configured", func(t *testing.T) {
		start := time.Now()
		invoke("M0")
		require.InDelta(t, time.Millisecond*200, time.Since(start), float64(time.Millisecond*170))
	})

	t.Run("method_timeout_is_less_than_service_timeout", func(t *testing.T) {
		start := time.Now()
		invoke("M1")
		require.InDelta(t, time.Millisecond*100, time.Since(start), float64(time.Millisecond*170))
	})

	t.Run("method_timeout_is_greater_than_service_timeout", func(t *testing.T) {
		start := time.Now()
		invoke("M2")
		require.InDelta(t, time.Millisecond*300, time.Since(start), float64(time.Millisecond*170))
	})

	t.Run("client_options_has_highest_priority", func(t *testing.T) {
		const timeout = time.Millisecond * 400
		start := time.Now()
		invoke("M2", client.WithTimeout(timeout))
		require.InDelta(t, timeout, time.Since(start), float64(time.Millisecond*170))
	})
}

func TestSelectorRemoteAddrUseUserProvidedParser(t *testing.T) {
	selector.Register(t.Name(), &fSelector{
		selectNode: func(s string, option ...selector.Option) (*registry.Node, error) {
			return &registry.Node{
				Network: t.Name(),
				Address: t.Name(),
				ParseAddr: func(network, address string) net.Addr {
					return newUnresolvedAddr(network, address)
				}}, nil
		},
		report: func(node *registry.Node, duration time.Duration, err error) error { return nil },
	})
	fake := "fake"
	codec.Register(fake, nil, &fakeCodec{})
	ctx := trpc.BackgroundContext()
	require.NotNil(t, client.New().Invoke(ctx, "failbody", nil,
		client.WithServiceName(t.Name()),
		client.WithProtocol(fake),
		client.WithTarget(fmt.Sprintf("%s://xxx", t.Name()))))
	addr := trpc.Message(ctx).RemoteAddr()
	require.NotNil(t, addr)
	require.Equal(t, t.Name(), addr.Network())
	require.Equal(t, t.Name(), addr.String())
}

func TestClientLocalScope(t *testing.T) {
	ctx := context.Background()
	iserver.Register(
		pb.GreeterServer_ServiceDesc.ServiceName,
		pb.GreeterServer_ServiceDesc.Methods[0].Name,
		func(ctx context.Context, f iserver.FilterFunc) (interface{}, error) {
			return pb.GreeterServer_ServiceDesc.Methods[0].Func(&testServer{}, ctx, f)
		},
		iserver.Options{
			Protocol: "trpc",
			ServerCodecGetter: func() codec.Codec {
				return trpc.DefaultServerCodec
			},
		},
	)
	p := pb.NewGreeterClientProxy(
		client.WithScope("local"),
		client.WithServiceName(pb.GreeterServer_ServiceDesc.ServiceName),
		client.WithProtocol("trpc"),
	)
	msg := "hello world"
	// Test scope "local".
	rsp, err := p.SayHello(ctx, &pb.HelloRequest{Msg: msg})
	require.NoError(t, err)
	require.Equal(t, msg, rsp.Msg)

	// Test scope "all".
	rsp, err = p.SayHello(ctx, &pb.HelloRequest{Msg: msg}, client.WithScope("all"))
	require.NoError(t, err)
	require.Equal(t, msg, rsp.Msg)
}

type testServer struct{}

func (s *testServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Msg: req.Msg}, nil
}
func (s *testServer) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Msg: req.Msg}, nil
}

type multiplexedTransport struct {
	require func(context.Context, []byte, ...transport.RoundTripOption)
	fakeTransport
}

func (t *multiplexedTransport) RoundTrip(
	ctx context.Context,
	req []byte,
	opts ...transport.RoundTripOption,
) ([]byte, error) {
	t.require(ctx, req, opts...)
	return t.fakeTransport.RoundTrip(ctx, req, opts...)
}

type fakeTransport struct {
	send  func() error
	recv  func() ([]byte, error)
	close func()
}

func (c *fakeTransport) RoundTrip(ctx context.Context, req []byte,
	roundTripOpts ...transport.RoundTripOption) (rsp []byte, err error) {
	time.Sleep(time.Millisecond * 2)
	if string(req) == "callfail" {
		return nil, errors.New("transport call fail")
	}

	if string(req) == "timeout" {
		return nil, &errs.Error{
			Type: errs.ErrorTypeFramework,
			Code: errs.RetClientTimeout,
			Msg:  "transport call fail",
		}
	}

	if string(req) == "nilrsp" {
		return nil, nil
	}

	if string(req) == "one_fail" {
		opts := &transport.RoundTripOptions{}
		for _, o := range roundTripOpts {
			o(opts)
		}
		if opts.Address == "127.0.0.1:8080" {
			return nil, errors.New("transport call fail")
		}
		return req, nil
	}

	return req, nil
}

func (c *fakeTransport) Send(ctx context.Context, req []byte, opts ...transport.RoundTripOption) error {
	if c.send != nil {
		return c.send()
	}
	return nil
}

func (c *fakeTransport) Recv(ctx context.Context, opts ...transport.RoundTripOption) ([]byte, error) {
	if c.recv != nil {
		return c.recv()
	}
	return []byte("body"), nil
}

func (c *fakeTransport) Init(ctx context.Context, opts ...transport.RoundTripOption) error {
	return nil
}
func (c *fakeTransport) Close(ctx context.Context) {
	if c.close != nil {
		c.close()
	}
}

type fakeCodec struct {
}

func (c *fakeCodec) Encode(msg codec.Msg, reqBody []byte) (reqBuf []byte, err error) {
	if string(reqBody) == "failbody" {
		return nil, errors.New("encode fail")
	}
	return reqBody, nil
}

func (c *fakeCodec) Decode(msg codec.Msg, rspBuf []byte) (rspBody []byte, err error) {
	if string(rspBuf) == "businessfail" {
		return nil, errors.New("businessfail")
	}

	if string(rspBuf) == "msgfail" {
		msg.WithClientRspErr(errors.New("msgfail"))
		return nil, nil
	}
	return rspBuf, nil
}

type fakeSelector struct {
}

func (c *fakeSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
	if serviceName == "selectfail" {
		return nil, errors.New("selectfail")
	}

	if serviceName == "emptynode" {
		return &registry.Node{}, nil
	}

	if serviceName == "udpnetwork" {
		return &registry.Node{
			Network: "udp",
			Address: "127.0.0.1:8080",
		}, nil
	}

	if serviceName == "unknownnetwork" {
		return &registry.Node{
			Network: "unknown",
			Address: "127.0.0.1:8080",
		}, nil
	}

	if serviceName == "broadcast.service" {
		list1 := make([]*registry.Node, 0, 3)
		list1 = append(list1, &registry.Node{
			Address: "127.0.0.1:8080",
		})
		list1 = append(list1, &registry.Node{
			Address: "127.0.0.1:8081",
		})
		list1 = append(list1, &registry.Node{
			Address: "127.0.0.1:8082",
		})

		return &registry.Node{
			Network: "unknown",
			Address: "127.0.0.1:8080",
			Metadata: map[string]interface{}{
				inaming.BroadcastNodeListKey: list1,
			},
		}, nil
	}

	if serviceName == "broadcast.emptyList.service" {
		return &registry.Node{
			Network: "unknown",
			Address: "127.0.0.1:8080",
			Metadata: map[string]interface{}{
				inaming.BroadcastNodeListKey: []registry.Node{},
			},
		}, nil
	}

	if serviceName == "broadcast.noList.service" {
		return &registry.Node{
			Network:  "unknown",
			Address:  "127.0.0.1:8080",
			Metadata: map[string]interface{}{},
		}, nil
	}

	if serviceName == "broadcast.wrongList.service" {
		return &registry.Node{
			Network: "unknown",
			Address: "127.0.0.1:8080",
			Metadata: map[string]interface{}{
				inaming.BroadcastNodeListKey: []string{
					"127.0.0.1:8080",
					"127.0.0.1:8081",
					"127.0.0.1:8082",
				},
			},
		}, nil
	}

	return nil, errors.New("unknown servicename")
}

func (c *fakeSelector) Report(node *registry.Node, cost time.Duration, err error) error {
	return nil
}

type fSelector struct {
	selectNode func(string, ...selector.Option) (*registry.Node, error)
	report     func(*registry.Node, time.Duration, error) error
}

func (s *fSelector) Select(serviceName string, opts ...selector.Option) (*registry.Node, error) {
	return s.selectNode(serviceName, opts...)
}

func (s *fSelector) Report(node *registry.Node, cost time.Duration, err error) error {
	return s.report(node, cost, err)
}

// newUnresolvedAddr returns a new unresolvedAddr.
func newUnresolvedAddr(network, address string) *unresolvedAddr {
	return &unresolvedAddr{network: network, address: address}
}

var _ net.Addr = (*unresolvedAddr)(nil)

// unresolvedAddr is a net.Addr which returns the original network or address.
type unresolvedAddr struct {
	network string
	address string
}

// Network returns the unresolved original network.
func (a *unresolvedAddr) Network() string {
	return a.network
}

// String returns the unresolved original address.
func (a *unresolvedAddr) String() string {
	return a.address
}
