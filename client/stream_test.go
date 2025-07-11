//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
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
	t.Run("calling without error", func(t *testing.T) {
		require.NotNil(t, streamCli)
		opts, err := streamCli.Init(ctx,
			client.WithTarget("ip://127.0.0.1:8000"),
			client.WithTimeout(time.Second),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithStreamTransport(&fakeTransport{}),
			client.WithProtocol("fake"),
		)
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
	})

	t.Run("test nil Codec", func(t *testing.T) {
		opts, err := streamCli.Init(ctx,
			client.WithTarget("ip://127.0.0.1:8080"),
			client.WithTimeout(time.Second),
			client.WithProtocol("fake-nil"),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithStreamTransport(&fakeTransport{}))
		require.NotNil(t, err)
		require.Nil(t, opts)
		err = streamCli.Invoke(ctx)
		require.Nil(t, err)
	})

	t.Run("test selectNode with error", func(t *testing.T) {
		opts, err := streamCli.Init(ctx,
			client.WithTarget("ip/:/127.0.0.1:8080"),
			client.WithProtocol("fake"),
		)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "invalid")
		require.Nil(t, opts)
	})

	t.Run("test stream recv failure", func(t *testing.T) {
		opts, err := streamCli.Init(ctx,
			client.WithTarget("ip://127.0.0.1:8000"),
			client.WithTimeout(time.Second),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithStreamTransport(&fakeTransport{
				recv: func() ([]byte, error) {
					return nil, errors.New("recv failed")
				},
			}),
			client.WithProtocol("fake"),
		)
		require.Nil(t, err)
		require.NotNil(t, opts)
		err = streamCli.Invoke(ctx)
		require.Nil(t, err)
		rsp, err := streamCli.Recv(ctx)
		require.Nil(t, rsp)
		require.NotNil(t, err)
	})

	t.Run("test decode failure", func(t *testing.T) {
		_, err := streamCli.Init(ctx,
			client.WithTarget("ip://127.0.0.1:8000"),
			client.WithTimeout(time.Second),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithStreamTransport(&fakeTransport{
				recv: func() ([]byte, error) {
					return []byte("businessfail"), nil
				},
			}),
			client.WithProtocol("fake"),
		)
		require.Nil(t, err)
		rsp, err := streamCli.Recv(ctx)
		require.Nil(t, rsp)
		require.NotNil(t, err)
	})

	t.Run("test compress failure", func(t *testing.T) {
		opts, err := streamCli.Init(context.Background(),
			client.WithTarget("ip://127.0.0.1:8000"),
			client.WithTimeout(time.Second),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithStreamTransport(&fakeTransport{}),
			client.WithCurrentCompressType(codec.CompressTypeGzip),
			client.WithProtocol("fake"))
		require.Nil(t, err)
		require.NotNil(t, opts)
		err = streamCli.Invoke(ctx)
		require.Nil(t, err)
		_, err = streamCli.Recv(ctx)
		require.NotNil(t, err)
	})

	t.Run("test compress without error", func(t *testing.T) {
		opts, err := streamCli.Init(ctx,
			client.WithTarget("ip://127.0.0.1:8000"),
			client.WithTimeout(time.Second),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithStreamTransport(&fakeTransport{}),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithProtocol("fake"),
		)
		require.Nil(t, err)
		require.NotNil(t, opts)
		err = streamCli.Invoke(ctx)
		require.Nil(t, err)
		rsp, err := streamCli.Recv(ctx)
		require.Nil(t, err)
		require.NotNil(t, rsp)
	})
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

func TestStreamCloseTransport(t *testing.T) {
	codec.Register("fake", nil, &fakeCodec{})
	t.Run("close transport when send fail", func(t *testing.T) {
		var isClose bool
		streamCli := client.NewStream()
		_, err := streamCli.Init(context.Background(),
			client.WithTarget("ip://127.0.0.1:8000"),
			client.WithStreamTransport(&fakeTransport{
				send: func() error {
					return errors.New("expected error")
				},
				close: func() {
					isClose = true
				},
			}),
			client.WithProtocol("fake"),
		)
		require.Nil(t, err)
		require.NotNil(t, streamCli.Send(context.Background(), nil))
		require.True(t, isClose)
	})
	t.Run("close transport when recv fail", func(t *testing.T) {
		var isClose bool
		streamCli := client.NewStream()
		_, err := streamCli.Init(context.Background(),
			client.WithTarget("ip://127.0.0.1:8000"),
			client.WithStreamTransport(&fakeTransport{
				recv: func() ([]byte, error) {
					return nil, errors.New("expected error")
				},
				close: func() {
					isClose = true
				},
			}),
			client.WithProtocol("fake"),
		)
		require.Nil(t, err)
		_, err = streamCli.Recv(context.Background())
		require.NotNil(t, err)
		require.True(t, isClose)
	})
}
