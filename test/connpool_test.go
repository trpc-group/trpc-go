// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestConnectionPool_ClientTimeoutDueToSeverOverload() {
	// Given a trpc server that handling request is very slow for some reason.
	var requestCount int32
	s.startServer(
		&TRPCService{UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
			time.Sleep(time.Duration(atomic.AddInt32(&requestCount, 1)) * 100 * time.Millisecond)
			return &testpb.SimpleResponse{}, nil
		}})

	// And a trpc client with ConnectionPool.
	pool := connpool.NewConnectionPool(
		connpool.WithMaxIdle(9),
		connpool.WithMaxActive(9),
		connpool.WithIdleTimeout(-1),
		connpool.WithWait(true),
	)
	c := s.newTRPCClient(client.WithPool(pool))

	// When sending many request to the server, we expect to receive timeout error
	// But the client will be blocked, because internal token resources may be repeatedly released
	// due to incorrect connection management.
	require.Eventually(s.T(), func() bool {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(t *testing.T) {
				_, err := c.UnaryCall(context.Background(), s.defaultSimpleRequest, client.WithTimeout(100*time.Millisecond))
				if err != nil {
					t.Log(err)
				}
				wg.Done()
			}(s.T())
		}
		wg.Wait()
		return true
	}, 10*time.Second, 500*time.Millisecond)
}
