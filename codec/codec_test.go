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

package codec_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/codec"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

// Fake is a fake codec for test
type Fake struct {
}

func (c *Fake) Encode(message codec.Msg, inbody []byte) (outbuf []byte, err error) {
	return nil, nil
}

func (c *Fake) Decode(message codec.Msg, inbuf []byte) (outbody []byte, err error) {
	return nil, nil
}

// TestCodec is unit test for the register logic of codec.
func TestCodec(t *testing.T) {
	f := &Fake{}

	codec.Register("fake", f, f)

	serverCodec := codec.GetServer("NoExists")
	assert.Nil(t, serverCodec)

	clientCodec := codec.GetClient("NoExists")
	assert.Nil(t, clientCodec)

	serverCodec = codec.GetServer("fake")
	assert.Equal(t, f, serverCodec)

	clientCodec = codec.GetClient("fake")
	assert.Equal(t, f, clientCodec)
}

// GOMAXPROCS=1 go test -bench=WithNewMessage -benchmem -benchtime=10s
// -memprofile mem.out -cpuprofile cpu.out codec_test.go

// BenchmarkWithNewMessage is the benchmark test of codec
func BenchmarkWithNewMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		codec.WithNewMessage(context.Background())
	}
}
