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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"

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
	head := &trpcpb.RequestProtocol{}
	ctx, msg = codec.WithNewMessage(ctx)
	require.Nil(t, cli.Invoke(ctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"),
		client.WithProtocol("fake"),
		client.WithSendOnly(),
		client.WithReqHead(head),
	))
	require.Equal(t, msg.CallType(), codec.SendOnly)
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
	_ = cli.Invoke(nctx, reqBody, rspBody, client.WithTarget("ip://www.qq.com:8080"), client.WithProtocol("fake"))
	assert.Nil(t, codec.Message(nctx).RemoteAddr())

	// test calling target with ip schema failure
	nctx, msg := codec.WithNewMessage(ctx)
	reqBody = &codec.Body{Data: []byte("callfail")}
	err := cli.Invoke(nctx, reqBody, rspBody, client.WithTarget("ip://127.0.0.1:8080"), client.WithProtocol("fake"))
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
	require.Equal(t, errs.RetClientTimeout, e.Code)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err = cli.Invoke(ctx,
		&codec.Body{Data: []byte("timeout")}, rspBody,
		client.WithTarget(target),
		client.WithProtocol(protocol))
	require.NotNil(t, err)
	e, ok = err.(*errs.Error)
	require.True(t, ok)
	require.Equal(t, errs.RetClientFullLinkTimeout, e.Code)
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

	return nil, errors.New("unknown servicename")
}

func (c *fakeSelector) Report(node *registry.Node, cost time.Duration, err error) error {
	return nil
}
