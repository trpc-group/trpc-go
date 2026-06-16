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

package transport

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTCPConnHandleKeepOrderPreDecodeRequiresHandlerInterface(t *testing.T) {
	c := &tcpconn{conn: &conn{handler: tcpKeepOrderPlainHandler{}}}
	require.PanicsWithValue(t,
		"bug: handler must implement pre-decode interface for keep-order requests",
		func() { c.handleKeepOrderPreDecode([]byte("request")) },
	)
}

func TestTCPConnHandleKeepOrderPreDecodeFallbackOnError(t *testing.T) {
	c := &tcpconn{conn: &conn{handler: tcpKeepOrderErrorHandler{
		preDecodeErr: errors.New("pre-decode failed"),
	}}}
	require.False(t, c.handleKeepOrderPreDecode([]byte("request")))
}

func TestTCPConnHandleKeepOrderPreUnmarshalRequiresHandlerInterface(t *testing.T) {
	c := &tcpconn{conn: &conn{handler: tcpKeepOrderPlainHandler{}}}
	require.PanicsWithValue(t,
		"bug: handler must implement pre-unmarshal interface for keep-order requests",
		func() { c.handleKeepOrderPreUnmarshal([]byte("request")) },
	)
}

func TestTCPConnHandleKeepOrderPreUnmarshalFallbackOnError(t *testing.T) {
	c := &tcpconn{conn: &conn{handler: tcpKeepOrderErrorHandler{
		preUnmarshalErr: errors.New("pre-unmarshal failed"),
	}}}
	require.False(t, c.handleKeepOrderPreUnmarshal([]byte("request")))
}

type tcpKeepOrderPlainHandler struct{}

func (tcpKeepOrderPlainHandler) Handle(context.Context, []byte) ([]byte, error) {
	return nil, nil
}

type tcpKeepOrderErrorHandler struct {
	preDecodeErr    error
	preUnmarshalErr error
}

func (h tcpKeepOrderErrorHandler) Handle(context.Context, []byte) ([]byte, error) {
	return nil, nil
}

func (h tcpKeepOrderErrorHandler) PreDecode(context.Context, []byte) ([]byte, error) {
	return nil, h.preDecodeErr
}

func (h tcpKeepOrderErrorHandler) PreUnmarshal(context.Context, []byte) (interface{}, error) {
	return nil, h.preUnmarshalErr
}
