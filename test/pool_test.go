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

package test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
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
	// Note: above bug is fixed by internal merge_requests/1695 ^_^
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

func (s *TestSuite) TestMultiplexedPool_ClientReconnect() {
	logDir := s.T().TempDir()
	defaultLogger := log.DefaultLogger
	defer log.SetLogger(defaultLogger)
	logger := log.NewZapLog(log.Config{
		{
			Writer: log.OutputFile,
			WriteConfig: log.WriteConfig{
				LogPath:   logDir,
				Filename:  "trpc.log",
				WriteMode: log.WriteSync,
			},
			Level: "debug",
		},
	})
	log.SetLogger(logger)

	s.startServer(
		&TRPCService{UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
			time.Sleep(10 * time.Microsecond)
			return &testpb.SimpleResponse{}, nil
		}})

	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		select {
		case <-ctx.Done():
			s.closeServer(nil)
			done <- struct{}{}
		}
	}()

	c := s.newTRPCClient(client.WithMultiplexed(true))
Loop:
	for {
		select {
		case <-done:
			break Loop
		default:
		}
		c.UnaryCall(context.Background(), s.defaultSimpleRequest, client.WithTimeout(100*time.Millisecond))
	}
	time.Sleep(10 * time.Millisecond)

	// read log from file
	fp := filepath.Join(logDir, "trpc.log")
	buf, err := os.ReadFile(fp)
	assert.Nil(s.T(), err)
	// should not reconnect when client read EOF.
	assert.NotContains(s.T(), string(buf), "reconnect fail")
}
