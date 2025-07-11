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

package connpool

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/codec"
)

func TestWithGetOptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	fb := &noopFramerBuilder{}
	opts := &GetOptions{CustomReader: codec.NewReader,
		FramerBuilder: fb,
		Ctx:           ctx,
	}

	localAddr := "127.0.0.1:8080"
	opts.WithLocalAddr(localAddr)
	protocol := "xxx-protocol"
	opts.WithProtocol(protocol)
	opts.WithCustomReader(codec.NewReader)

	assert.Equal(t, opts.FramerBuilder, fb)
	assert.Equal(t, opts.Ctx, ctx)
	assert.Equal(t, opts.LocalAddr, localAddr)
	assert.Equal(t, protocol, opts.Protocol)
	assert.NotNil(t, opts.CustomReader)
}

type emptyFramerBuilder struct{}

// New creates a new Framer.
func (*emptyFramerBuilder) New(io.Reader) codec.Framer {
	return &emptyFramer{}
}

type emptyFramer struct{}

// ReadFrame reads the frame.
func (*emptyFramer) ReadFrame() ([]byte, error) {
	return nil, nil
}

type safeFramerBuilder struct{}

// New creates a new Framer.
func (*safeFramerBuilder) New(io.Reader) codec.Framer {
	return &safeFramer{}
}

type safeFramer struct{}

// ReadFrame reads the frame.
func (*safeFramer) ReadFrame() ([]byte, error) {
	in := []byte("hello world!")
	out := make([]byte, len(in))
	copy(out, in)
	return out, nil
}

func (*safeFramer) IsSafe() bool {
	return true
}

func TestGetDialCtx(t *testing.T) {
	opts := &GetOptions{CustomReader: codec.NewReader}
	ctx, cancel := opts.getDialCtx(0)
	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)

	ctx, cancel = opts.getDialCtx(defaultDialTimeout)
	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)

	backgroundCtx := context.Background()
	opts.WithContext(backgroundCtx)
	ctx, cancel = opts.getDialCtx(defaultDialTimeout)
	assert.NotEqual(t, backgroundCtx, ctx)
	assert.NotNil(t, cancel)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opts.WithContext(timeoutCtx)
	ctx, cancel = opts.getDialCtx(defaultDialTimeout)
	assert.Equal(t, timeoutCtx, ctx)
	assert.Nil(t, cancel)

	opts.WithContext(timeoutCtx)
	opts.WithDialTimeout(200 * time.Millisecond)
	ctx, cancel = opts.getDialCtx(defaultDialTimeout)
	assert.NotEqual(t, timeoutCtx, ctx)
	assert.NotNil(t, cancel)

	opts.WithContext(timeoutCtx)
	opts.WithDialTimeout(2 * time.Second)
	ctx, cancel = opts.getDialCtx(defaultDialTimeout)
	assert.Equal(t, timeoutCtx, ctx)
	assert.Nil(t, cancel)
}
