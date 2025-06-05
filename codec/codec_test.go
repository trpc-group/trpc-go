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

package codec_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

type fakeCodec struct{}

func (c *fakeCodec) Encode(message codec.Msg, in []byte) (out []byte, err error) {
	return nil, nil
}

func (c *fakeCodec) Decode(message codec.Msg, in []byte) (out []byte, err error) {
	return nil, nil
}

// TestCodec is unit test for the register logic of codec.
func TestCodec_Register(t *testing.T) {
	serverCodec, clientCodec := &fakeCodec{}, &fakeCodec{}
	codec.Register("fakeCode", serverCodec, serverCodec)

	t.Run("no registered codec", func(t *testing.T) {
		require.Nil(t, codec.GetServer("no registered codec"))
		require.Nil(t, codec.GetClient("no registered codec"))
	})
	t.Run("registered codec", func(t *testing.T) {
		require.Equal(t, serverCodec, codec.GetServer("fakeCode"))
		require.Equal(t, clientCodec, codec.GetClient("fakeCode"))
	})
}

func TestCodec_MustRegister(t *testing.T) {
	serverCodec, clientCodec := &fakeCodec{}, &fakeCodec{}

	t.Run("no registered codec", func(t *testing.T) {
		require.Nil(t, codec.GetServer("fakeCodeMustRegister"))
		require.Nil(t, codec.GetClient("fakeCodeMustRegister"))
	})

	codec.MustRegister("fakeCodeMustRegister", serverCodec, serverCodec)

	t.Run("registered codec", func(t *testing.T) {
		require.Equal(t, serverCodec, codec.GetServer("fakeCodeMustRegister"))
		require.Equal(t, clientCodec, codec.GetClient("fakeCodeMustRegister"))
	})
	t.Run("repeat register", func(t *testing.T) {
		require.Panics(t, func() {
			codec.MustRegister("fakeCodeMustRegister", serverCodec, serverCodec)
		})
	})
}

// GOMAXPROCS=1 go test -bench=WithNewMessage -benchmem -benchtime=10s
// -memprofile mem.out -cpuprofile cpu.out codec_test.go

// BenchmarkWithNewMessage is the benchmark test of codec
func BenchmarkWithNewMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		codec.WithNewMessage(context.Background())
	}
}
