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
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestClientCancelAfterSend() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), s.testClientCancelAfterSend)
	}
}

func (s *TestSuite) testClientCancelAfterSend() {
	called := make(chan struct{})
	canceled := make(chan struct{})
	s.startServer(&TRPCService{
		EmptyCallF: func(ctx context.Context, _ *testpb.Empty) (*testpb.Empty, error) {
			close(called)
			<-canceled
			time.Sleep(2 * time.Second)
			return &testpb.Empty{}, nil
		},
	})
	defer s.closeServer(nil)

	ctx, cancel := context.WithTimeout(trpc.BackgroundContext(), 10*time.Second)

	errChan := make(chan error)
	go func() {
		c := s.newTRPCClient()
		_, err := c.EmptyCall(ctx, &testpb.Empty{})
		errChan <- err
	}()

	select {
	case <-called:
		cancel()
		close(canceled)
	case <-time.After(5 * time.Second):
		s.T().Fatalf("failed to perform EmptyCall after 5s.")
	}

	err := <-errChan
	if s.tRPCEnv.client.multiplexed {
		require.Equal(s.T(), errs.RetClientCanceled, errs.Code(err))
	} else {
		require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err))
	}

}

func (s *TestSuite) TestClientCancelBeforeSend() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), s.testClientCancelBeforeSend)
	}
}

func (s *TestSuite) testClientCancelBeforeSend() {
	s.startServer(&TRPCService{unaryCallSleepTime: time.Second})
	defer s.closeServer(nil)

	ctx, cancelBeforeSend := context.WithCancel(trpc.BackgroundContext())
	c := s.newTRPCClient()
	cancelBeforeSend()
	_, err := c.UnaryCall(ctx, s.defaultSimpleRequest)
	require.Contains(s.T(), err.Error(), "context canceled")
	require.Conditionf(
		s.T(),
		func() bool {
			code := errs.Code(err)
			return (100 < code && code < 200) || code == errs.RetUnknown
		},
		"client'server error code range is (100, 200).",
	)
}

func (s *TestSuite) TestClientDoesntDeadlockWhileWritingLargeMessages() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), s.testClientDoesntDeadlockWhileWritingLargeMessages)
	}
}

func (s *TestSuite) testClientDoesntDeadlockWhileWritingLargeMessages() {
	s.startServer(&TRPCService{})
	defer s.closeServer(nil)

	largeFrameSize := trpc.DefaultMaxFrameSize - 1000
	payload, err := newPayload(testpb.PayloadType_COMPRESSIBLE, int32(largeFrameSize))
	require.Nil(s.T(), err)
	req := &testpb.SimpleRequest{
		ResponseType: testpb.PayloadType_COMPRESSIBLE,
		Payload:      payload,
	}

	c := s.newTRPCClient()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				func() {
					ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
					defer cancel()
					_, err := c.UnaryCall(ctx, req)
					require.Nil(s.T(), err)
				}()
			}
		}()
	}
	wg.Wait()
}

func (s *TestSuite) TestRequestPacketOverClientMaxFrameSize() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), s.testRequestPacketOverClientMaxFrameSize)
	}
}

func (s *TestSuite) testRequestPacketOverClientMaxFrameSize() {
	s.startServer(&TRPCService{})
	defer s.closeServer(nil)

	oldDefaultMaxFrameSize := trpc.DefaultMaxFrameSize
	defer func() {
		trpc.DefaultMaxFrameSize = oldDefaultMaxFrameSize
	}()
	trpc.DefaultMaxFrameSize = 100

	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err))
	require.Contains(s.T(), err.Error(), "frame len is larger than MaxFrameSize")
}

func (s *TestSuite) TestRequestPacketOverServerMaxFrameSize() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), s.testRequestPacketOverServerMaxFrameSize)
	}
}

func (s *TestSuite) testRequestPacketOverServerMaxFrameSize() {
	oldDefaultMaxFrameSize := trpc.DefaultMaxFrameSize
	defer func() {
		trpc.DefaultMaxFrameSize = oldDefaultMaxFrameSize
	}()

	s.startServer(
		&TRPCService{},
		server.WithFilter(
			func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
				trpc.DefaultMaxFrameSize = 100
				return next(ctx, req)
			}),
	)
	defer s.closeServer(nil)

	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Equal(s.T(), errs.RetServerEncodeFail, errs.Code(err))
	require.Contains(s.T(), err.Error(), "frame len is larger than MaxFrameSize")
}

func (s *TestSuite) TestSendRequestAfterServerClosed() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), s.testSendRequestAfterServerClosed)
	}
}

func (s *TestSuite) testSendRequestAfterServerClosed() {
	s.startServer(&TRPCService{})

	c := s.newTRPCClient()
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(s.T(), err)

	done := make(chan struct{})
	go s.closeServer(done)
	<-done
	for {
		if _, err = c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{}); err != nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	_, err = s.newTRPCClient().EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	if s.tRPCEnv.client.multiplexed {
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err))
	} else {
		require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err))
	}
}

func (s *TestSuite) TestServeCloseIsNotIdempotent() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.testServeCloseIsNotIdempotent()
	}
}

func (s *TestSuite) testServeCloseIsNotIdempotent() {
	s.startServer(&TRPCService{})

	c := s.newTRPCClient()
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(s.T(), err)

	done := make(chan struct{})
	go s.server.Close(done)
	<-done

	defer func() {
		p := recover()
		require.NotNil(s.T(), p)
		require.Contains(s.T(), fmt.Sprint(p), "close of closed channel")
		// avoid panic again in TearDownTest function
		s.server = nil
	}()
	s.server.Close(nil)
}

func (s *TestSuite) TestWithServerAsyncOption() {
	s.T().Skip("unstable test case.")
	for _, e := range allTRPCEnvs {
		if !e.server.async {
			continue
		}
		s.tRPCEnv = e
		s.Run(e.String(), s.testWithServerAsyncOption)
	}
	s.server = nil
}

func (s *TestSuite) testWithServerAsyncOption() {
	const (
		maxRoutines  = 2
		numUnaryCall = 3
	)
	serverChan := make(chan struct{})
	s.startServer(
		&TRPCService{UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
			<-serverChan
			return &testpb.SimpleResponse{}, nil
		}},
		server.WithMaxRoutines(maxRoutines),
		server.WithServerAsync(true),
	)

	closeServer := func() {
		done := make(chan struct{})
		go s.closeServer(done)
		<-done
		c := s.newTRPCClient()
		for {
			if _, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{}); err != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	defer closeServer()

	clientChan := s.concurrencyUnaryCall(numUnaryCall)
	for i := 0; i < maxRoutines; i++ {
		serverChan <- struct{}{}
	}
	require.Equal(s.T(), numUnaryCall-maxRoutines, <-clientChan)
	close(serverChan)
}

// concurrencyUnaryCall returns a channel to receive the number of failed UnaryCall.
func (s *TestSuite) concurrencyUnaryCall(num int) <-chan int {
	ch := make(chan int, 1)
	go func() {
		var (
			failedNum int32
			wg        sync.WaitGroup
		)
		c := s.newTRPCClient()
		for i := 0; i < num; i++ {
			wg.Add(1)
			go func(req *testpb.SimpleRequest) {
				defer wg.Done()
				if _, err := c.UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second)); err != nil {
					atomic.AddInt32(&failedNum, 1)
				}
			}(s.defaultSimpleRequest)
		}
		wg.Wait()
		ch <- int(failedNum)
		close(ch)
	}()
	return ch
}

func (s *TestSuite) TestClientTimeoutAtUnaryCall() {
	s.startServer(&TRPCService{unaryCallSleepTime: time.Second})

	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(100*time.Millisecond))
	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err))
}

func (s *TestSuite) TestServerWithLongMaxCloseWaitTimeAndHandleOverAllOldRequest() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), func() {
			require.Nil(s.T(), s.testServerWithCloseWaitTime(time.Second, 3*time.Second, time.Second))
		})
	}
	s.server = nil
}

func (s *TestSuite) TestServerWithLongMaxCloseWaitTimeAndStillHasOldRequestWhenClosed() {
	for _, e := range allTRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.tRPCEnv = e
		s.Run(e.String(), func() {
			require.NotNil(s.T(), s.testServerWithCloseWaitTime(time.Second, 3*time.Second, 5*time.Second))
		})
	}
	s.server = nil
}

func (s *TestSuite) TestServerWithShortMaxCloseWaitTime() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), func() {
			require.NotNil(s.T(), s.testServerWithCloseWaitTime(3*time.Second, time.Second, 2*time.Second))
		})
	}
	s.server = nil
}

func (s *TestSuite) testServerWithCloseWaitTime(
	minCloseWaitTime, maxCloseWaitTime, serviceHandleTime time.Duration,
) (err error) {
	startHandleFirstRequest := make(chan struct{})
	isFirstRequest := true
	s.startServer(
		&TRPCService{EmptyCallF: func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
			if isFirstRequest {
				isFirstRequest = false
				close(startHandleFirstRequest)
				time.Sleep(serviceHandleTime)
			}
			return &testpb.Empty{}, nil
		}},
		server.WithCloseWaitTime(minCloseWaitTime),
		server.WithMaxCloseWaitTime(maxCloseWaitTime),
	)

	go func() {
		c := s.newTRPCClient()
		_, err = c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{}, client.WithTimeout(maxCloseWaitTime))
	}()

	<-startHandleFirstRequest
	go func() {
		s.server.Close(nil)
	}()

	sendRequestPeriodicallyUntilServerHasClosed := func() int {
		const sleepTime = 100 * time.Millisecond
		succeededNum := 0
		for {
			c := s.newTRPCClient()
			if _, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{}); err != nil {
				break
			}
			succeededNum++
			time.Sleep(sleepTime)
		}
		return succeededNum
	}
	require.NotZero(s.T(), sendRequestPeriodicallyUntilServerHasClosed())

	return err
}

func (s *TestSuite) TestRegisterOnShutdown() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), s.testRegisterOnShutdown)
	}
}

func (s *TestSuite) testRegisterOnShutdown() {
	s.startServer(&TRPCService{})
	onShutdown := make(chan struct{})
	s.server.RegisterOnShutdown(func() { <-onShutdown })

	startClose := make(chan struct{})
	go func() {
		close(startClose)
		s.server.Close(nil)
	}()
	defer func() { s.server = nil }()

	<-startClose
	c := s.newTRPCClient()
	for i := 0; i < 10; i++ {
		_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
		require.Nil(s.T(), err)
	}

	close(onShutdown)
	for {
		if _, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{}); err != nil {
			break
		}
	}
}

func (s *TestSuite) TestTRPCProtocol() {
	s.startServer(&TRPCService{}, server.WithFilter(serverFilter))
	ctx := trpc.BackgroundContext()
	_, err := s.newTRPCClient().EmptyCall(ctx, &testpb.Empty{}, client.WithFilter(clientFilter))
	require.Nil(s.T(), err)
}

func serverFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
	requestID := trpc.Request(ctx).RequestId
	if requestID == 0 {
		return nil, fmt.Errorf("request id equals to zero, which is invalid")
	}
	rsp, err = next(ctx, req)
	if responseID := trpc.Response(ctx).RequestId; responseID != requestID {
		return nil, fmt.Errorf("ResponseProtocol' responseID(%d) doesn't not equal to "+
			"RequestProtocol's requestID(%d)", responseID, requestID)
	}
	return rsp, err
}

func clientFilter(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
	if requestProtocol := trpc.Request(ctx); !reflect.DeepEqual(requestProtocol, &trpcpb.RequestProtocol{}) {
		return fmt.Errorf("RequestProtocol(%v) is not empty", requestProtocol)
	}
	err := next(ctx, req, rsp)
	if responseProtocol := trpc.Response(ctx); !reflect.DeepEqual(responseProtocol, &trpcpb.ResponseProtocol{}) {
		return fmt.Errorf("ResponseProtocol(%v) is not empty", responseProtocol)
	}
	return err
}

func (s *TestSuite) TestTRPCGoAndWait() {
	s.startServer(&TRPCService{})
	s.T().Run("Clone Context", func(t *testing.T) {
		ctx := trpc.BackgroundContext()
		trpc.Message(ctx).WithServerMetaData(codec.MetaData{"request": []byte("0")})
		c := s.newTRPCClient()

		var requests []func() error
		for i := 0; i < 10; i++ {
			i := i
			requests = append(requests, func() error {
				ctx := trpc.CloneContext(ctx)
				smd := trpc.Message(ctx).ServerMetaData()
				if r := smd["request"]; !bytes.Equal(r, []byte("0")) {
					return fmt.Errorf("request field(%s) in ServerMetaData doesn't equal to 0", string(r))
				}
				smd["request"] = []byte(fmt.Sprint(i))
				_, err := c.EmptyCall(trpc.CloneContext(ctx), &testpb.Empty{})
				return err
			})
		}

		err := trpc.GoAndWait(requests...)
		require.Nil(s.T(), err)
	})
	s.T().Run("Not Clone Context", func(t *testing.T) {
		ctx := trpc.BackgroundContext()
		trpc.Message(ctx).WithServerMetaData(codec.MetaData{"request": []byte("0")})
		c := s.newTRPCClient()

		var requests []func() error
		for i := 0; i < 10; i++ {
			i := i
			requests = append(requests, func() error {
				smd := trpc.Message(ctx).ServerMetaData()
				if r := smd["request"]; !bytes.Equal(r, []byte("0")) {
					return fmt.Errorf("request field(%s) in ServerMetaData doesn't equal to 0", string(r))
				}
				smd["request"] = []byte(fmt.Sprint(i))
				_, err := c.EmptyCall(trpc.CloneContext(ctx), &testpb.Empty{})
				return err
			})
		}

		err := trpc.GoAndWait(requests...)
		require.Regexp(s.T(), "^request field\\([0-9]+\\) in ServerMetaData doesn't equal to 0$", err.Error())
	})
	s.T().Run("recover from panic", func(t *testing.T) {
		ctx := context.Background()
		c := s.newTRPCClient()

		var requests []func() error
		for i := 0; i < 10; i++ {
			requests = append(requests, func() error {
				_, _ = c.EmptyCall(trpc.CloneContext(ctx), &testpb.Empty{})
				panic("something wrong")
			})
		}
		err := trpc.GoAndWait(requests...)
		require.Equal(t, errs.RetServerSystemErr, errs.Code(err))
		require.Contains(t, errs.Msg(err), "panic")
	})
}

func (s *TestSuite) TestTRPCGoer() {
	s.startServer(&TRPCService{})
	s.T().Run("async goer should recover", func(t *testing.T) {
		g := trpc.NewAsyncGoer(3, 1024, true)
		c := s.newTRPCClient()
		var wg sync.WaitGroup

		wg.Add(1)
		err := g.Go(trpc.BackgroundContext(), time.Second, func(ctx context.Context) {
			defer wg.Done()
			_, err := c.EmptyCall(ctx, &testpb.Empty{})
			if err != nil {
				s.T().Log(err)
			}
			panic("something wrong")
		})
		require.Nil(t, err)
		wg.Wait()
	})
	s.T().Run("async goer shouldn't recover", func(t *testing.T) {
		g := trpc.NewAsyncGoer(3, 1024, false)
		c := s.newTRPCClient()
		var wg sync.WaitGroup
		wg.Add(1)
		err := g.Go(trpc.BackgroundContext(), time.Second, func(ctx context.Context) {
			defer wg.Done()
			_, err := c.EmptyCall(ctx, &testpb.Empty{})
			if err != nil {
				s.T().Log(err)
			}
		})
		require.Nil(t, err)

		wg.Add(1)
		err = g.Go(trpc.BackgroundContext(), time.Second, func(ctx context.Context) {
			defer wg.Done()
			_, err := c.EmptyCall(ctx, &testpb.Empty{})
			if err != nil {
				s.T().Log(err)
			}
			panic("something wrong")
		})
		require.Nil(t, err, "panic is recovered by ants package, but underlying pool worker will exit")
		wg.Wait()
	})
	s.T().Run("sync goer", func(t *testing.T) {
		g := trpc.NewSyncGoer()
		c := s.newTRPCClient()

		err := g.Go(trpc.BackgroundContext(), time.Second, func(ctx context.Context) {
			_, err := c.EmptyCall(ctx, &testpb.Empty{})
			if err != nil {
				s.T().Log(err)
			}
		})
		require.Nil(t, err)

		err = g.Go(trpc.BackgroundContext(), time.Second, func(ctx context.Context) {
			_, err := c.EmptyCall(ctx, &testpb.Empty{})
			if err != nil {
				s.T().Log(err)
			}
		})
		require.Nil(t, err)
	})

}
