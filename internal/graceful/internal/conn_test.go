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

package graceful_test

import (
	"errors"
	"net"
	"testing"

	. "trpc.group/trpc-go/trpc-go/internal/graceful/internal"
	"github.com/stretchr/testify/require"
)

func TestConn(t *testing.T) {
	var onClosedCalled bool
	c := NewConn(conn{nil}, func(n net.Conn) {
		onClosedCalled = true
	})
	err := c.Close()
	require.NotNil(t, err)
	require.Equal(t, "test Close", err.Error())
	require.True(t, onClosedCalled)
	_, ok := Unwrap(net.Conn(c)).(conn)
	require.True(t, ok)
}

type conn struct{ net.Conn }

func (conn) Close() error {
	return errors.New("test Close")
}
