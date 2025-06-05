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

package multiplexed

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/codec"
)

func TestGetOptions(t *testing.T) {

	opts := NewGetOptions()
	fb := &emptyFramerBuilder{}
	msg := codec.Message(context.Background())
	caFile := "caFile"
	keyFile := "keyFile"
	serverName := "serverName"
	certFile := "certFile"
	localAddr := "127.0.0.1:8080"

	opts.WithFramerBuilder(fb)
	opts.WithMsg(msg)
	opts.WithDialTLS(certFile, keyFile, caFile, serverName)
	opts.WithLocalAddr(localAddr)

	assert.Equal(t, opts.FramerBuilder, fb)
	assert.Equal(t, opts.Msg, msg)
	assert.Equal(t, opts.CACertFile, caFile)
	assert.Equal(t, opts.TLSKeyFile, keyFile)
	assert.Equal(t, opts.TLSServerName, serverName)
	assert.Equal(t, opts.TLSCertFile, certFile)
	assert.Equal(t, opts.LocalAddr, localAddr)
}

type emptyFramerBuilder struct{}

func (*emptyFramerBuilder) New(io.Reader) codec.Framer {
	return &emptyFramer{}
}

type emptyFramer struct{}

func (*emptyFramer) ReadFrame() ([]byte, error) {
	return nil, nil
}
