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
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	. "trpc.group/trpc-go/trpc-go/internal/graceful/internal"
)

func TestListener(t *testing.T) {
	rawLis, err := net.Listen("tcp", "")
	require.Nil(t, err)
	conns := make(chan net.Conn)
	l := NewListener(rawLis, "tcp", "", conns)

	var wg sync.WaitGroup
	var success, failure int32
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := l.Accept()
			if err != nil {
				atomic.AddInt32(&failure, 1)
			} else {
				atomic.AddInt32(&success, 1)
			}
		}()
	}

	for i := 0; i < 5; i++ {
		_, err = net.Dial(rawLis.Addr().Network(), rawLis.Addr().String())
		require.Nil(t, err)
	}
	for i := 0; i < 5; i++ {
		conns <- nil
	}
	close(conns)

	wg.Wait()
	require.Equal(t, int32(0), failure)
	require.Equal(t, int32(10), success)

	// Even though all Accept has returned, next 5 Dials can also succeed.
	// The result will be cached for next accept.
	for i := 0; i < 5; i++ {
		_, err = net.Dial(rawLis.Addr().Network(), rawLis.Addr().String())
		require.Nil(t, err)
	}
	for i := 0; i < 5; i++ {
		_, err = l.Accept()
		require.Nil(t, err)
	}

	// Unix its own buffer for incoming connections.
	// Even though there is no Accept, net.Dial can also succeed.
	_, err = net.Dial(rawLis.Addr().Network(), rawLis.Addr().String())
	require.Nil(t, err)
	// This accepts the connection cached in runtime buffer.
	_, err = l.Accept()
	require.Nil(t, err)

	// repeat to coverage received recvState.
	_, err = net.Dial(rawLis.Addr().Network(), rawLis.Addr().String())
	require.Nil(t, err)
	_, err = l.Accept()
	require.Nil(t, err)
}
