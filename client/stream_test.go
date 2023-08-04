package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/naming/registry"

	_ "trpc.group/trpc-go/trpc-go"
)

// TestStream tests client stream.
func TestStream(t *testing.T) {
	ctx := context.Background()
	reqBody := &codec.Body{}
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})
	codec.Register("fake-nil", nil, nil)

	// calling without error
	streamCli := client.NewStream()
	require.NotNil(t, streamCli)
	opts, err := streamCli.Init(ctx, client.WithTarget("ip://127.0.0.1:8000"),
		client.WithTimeout(time.Second), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithStreamTransport(&fakeTransport{}), client.WithProtocol("fake"))
	require.Nil(t, err)
	require.NotNil(t, opts)
	err = streamCli.Invoke(ctx)
	require.Nil(t, err)
	err = streamCli.Send(ctx, reqBody)
	require.Nil(t, err)
	rsp, err := streamCli.Recv(ctx)
	require.Nil(t, err)
	require.Equal(t, []byte("body"), rsp)
	err = streamCli.Close(ctx)
	require.Nil(t, err)

	// test nil Codec
	opts, err = streamCli.Init(ctx,
		client.WithTarget("ip://127.0.0.1:8080"),
		client.WithTimeout(time.Second),
		client.WithProtocol("fake-nil"),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithStreamTransport(&fakeTransport{}))
	require.NotNil(t, err)
	require.Nil(t, opts)
	err = streamCli.Invoke(ctx)
	require.Nil(t, err)

	// test selectNode with error
	opts, err = streamCli.Init(ctx, client.WithTarget("ip/:/127.0.0.1:8080"),
		client.WithProtocol("fake"))
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "invalid")
	require.Nil(t, opts)

	// test stream recv failure
	ctx = context.WithValue(ctx, "recv-error", "recv failed")
	opts, err = streamCli.Init(ctx, client.WithTarget("ip://127.0.0.1:8000"),
		client.WithTimeout(time.Second), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithStreamTransport(&fakeTransport{}), client.WithProtocol("fake"))
	require.Nil(t, err)
	require.NotNil(t, opts)
	err = streamCli.Invoke(ctx)
	require.Nil(t, err)
	rsp, err = streamCli.Recv(ctx)
	require.Nil(t, rsp)
	require.NotNil(t, err)

	// test decode failure
	ctx = context.WithValue(ctx, "recv-decode-error", "businessfail")
	rsp, err = streamCli.Recv(ctx)
	require.Nil(t, rsp)
	require.NotNil(t, err)

	// test compress failure
	ctx = context.Background()
	opts, err = streamCli.Init(ctx, client.WithTarget("ip://127.0.0.1:8000"),
		client.WithTimeout(time.Second), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithStreamTransport(&fakeTransport{}), client.WithCurrentCompressType(codec.CompressTypeGzip),
		client.WithProtocol("fake"))
	require.Nil(t, err)
	require.NotNil(t, opts)
	err = streamCli.Invoke(ctx)
	require.Nil(t, err)
	rsp, err = streamCli.Recv(ctx)
	require.NotNil(t, err)

	// test compress without error
	opts, err = streamCli.Init(ctx, client.WithTarget("ip://127.0.0.1:8000"),
		client.WithTimeout(time.Second), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithStreamTransport(&fakeTransport{}), client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithProtocol("fake"))
	require.Nil(t, err)
	require.NotNil(t, opts)
	err = streamCli.Invoke(ctx)
	require.Nil(t, err)
	rsp, err = streamCli.Recv(ctx)
	require.Nil(t, err)
	require.NotNil(t, rsp)
}

func TestGetStreamFilter(t *testing.T) {
	type noopClientStream struct {
		client.ClientStream
	}
	testClientStream := &noopClientStream{}
	testFilter := func(ctx context.Context, desc *client.ClientStreamDesc,
		streamer client.Streamer) (client.ClientStream, error) {
		return testClientStream, nil
	}
	client.RegisterStreamFilter("testFilter", testFilter)
	filter := client.GetStreamFilter("testFilter")
	cs, err := filter(context.Background(), &client.ClientStreamDesc{}, nil)
	require.Nil(t, err)
	require.Equal(t, testClientStream, cs)
}

func TestStreamGetAddress(t *testing.T) {
	s := client.NewStream()
	require.NotNil(t, s)
	ctx, msg := codec.EnsureMessage(context.Background())
	node := &registry.Node{}
	const addr = "127.0.0.1:8000"
	opts, err := s.Init(ctx,
		client.WithTarget("ip://"+addr),
		client.WithTimeout(time.Second),
		client.WithSelectorNode(node),
	)
	require.Nil(t, err)
	require.NotNil(t, opts)
	require.Equal(t, addr, node.Address)
	require.NotNil(t, msg.RemoteAddr())
	require.Equal(t, addr, msg.RemoteAddr().String())
}
