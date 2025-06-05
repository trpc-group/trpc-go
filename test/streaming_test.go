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
	"fmt"
	"io"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
	"trpc.group/trpc-go/trpc-go/transport"
)

func (s *TestSuite) TestBidirectionalStreamingServerCrashWhenReceivingMessage() {
	// Given a trpc streaming server.
	s.startServer(&StreamingService{})

	// And a trpc streaming client.
	c := s.newStreamingClient()
	req := &testpb.StreamingOutputCallRequest{
		ResponseType: testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: []*testpb.ResponseParameters{
			{
				Size:     2,
				Interval: durationpb.New(time.Microsecond),
			},
		},
		Payload: &testpb.Payload{
			Type: testpb.PayloadType_COMPRESSABLE,
			Body: make([]byte, 1),
		},
	}

	// When the client makes a full duplex stream.
	cs, err := c.FullDuplexCall(trpc.BackgroundContext())

	// Then the stream should be created successfully.
	require.Nil(s.T(), err)

	// When the stream sends one request on the stream.
	err = cs.Send(req)

	// Then the request should be sent successfully.
	require.Nil(s.T(), err)

	// When the streaming server crashes for some reason.
	s.closeServer(nil)

	// And the client still send requests continuously.
	for {
		if err = cs.Send(req); err != nil {
			break
		}
	}

	require.NotNil(s.T(), err, "err: %+v", err)
}

func (s *TestSuite) TestBidirectionalStreaming() {
	for _, e := range allStreamEnvs {
		s.Run(e.String(), func() {
			s.testBidirectionalStreaming(e)
		})
	}
}

func (s *TestSuite) testBidirectionalStreaming(e streamEnv) {
	s.startServer(&StreamingService{},
		server.WithStreamTransport(transport.GetServerStreamTransport(e.server.transport)))
	s.Run("CallSequentiallyOk", func() {
		_, err := s.testBidirectionalStreamingCallSequentiallyOk(e)
		require.Nil(s.T(), err)
	})
	s.Run("CallConcurrentlyOk", func() {
		s.testBidirectionalStreamingCallConcurrentlyOk(e)
	})
	s.Run("CallSequentiallyFailed", func() {
		s.testBidirectionalStreamingCallSequentiallyFailed(e)
	})
	s.Run("ClientSendDataAfterCloseSend", func() {
		s.testBidirectionalStreamingClientSendDataAfterCloseSend(e)
	})
	s.Run("ContinueReceivingDataAfterReceiveEOF", func() {
		s.testBidirectionalStreamingContinueReceivingDataAfterReceiveEOF(e)
	})
	s.Run("CallCloseAndRecvTwice", func() {
		s.testBidirectionalStreamingCallCloseAndReceiveTwice(e)
	})
	s.Run("DontSendDataAfterCreatingStreaming", func() {
		s.testBidirectionalStreamingDontSendDataAfterCreatingStreaming(e)
	})
}

func (s *TestSuite) testBidirectionalStreamingCallSequentiallyOk(e streamEnv) (testpb.TestStreaming_FullDuplexCallClient, error) {
	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.FullDuplexCall(trpc.BackgroundContext())
	if err != nil {
		return nil, err
	}

	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
	if err != nil {
		return nil, err
	}

	const (
		itemSize  = 2
		sendNum   = 10
		totalSize = itemSize * sendNum
	)
	req := &testpb.StreamingOutputCallRequest{
		ResponseType: testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: []*testpb.ResponseParameters{
			{
				Size:     int32(itemSize),
				Interval: durationpb.New(time.Microsecond),
			},
		},
		Payload: payload,
	}
	for i := 0; i < sendNum; i++ {
		if err := cs.Send(req); err != nil {
			return nil, err
		}
	}
	if err := cs.CloseSend(); err != nil {
		return nil, err
	}

	rspSize := 0
	for {
		rsp, err := cs.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		rspSize += len(rsp.GetPayload().GetBody())
	}
	if rspSize != totalSize {
		return nil, fmt.Errorf("send size doesn't equal received size(want: %d != got: %d)", totalSize, rspSize)
	}
	return cs, nil
}

func (s *TestSuite) testBidirectionalStreamingCallSequentiallyFailed(e streamEnv) {
	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.FullDuplexCall(trpc.BackgroundContext())
	require.Nil(s.T(), err)

	const (
		invalidSize = -1
		itemSize    = 2
		sendNum     = 10
	)
	respParams := []*testpb.ResponseParameters{
		{
			Size:     int32(itemSize),
			Interval: durationpb.New(time.Microsecond),
		},
	}
	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
	require.Nil(s.T(), err)
	req := &testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: respParams,
		Payload:            payload,
	}
	for i := 0; i < sendNum; i++ {
		require.Nil(s.T(), cs.Send(req))
	}
	req.ResponseParameters[0].Size = invalidSize
	require.Nil(s.T(), cs.Send(req))
	require.Nil(s.T(), cs.CloseSend())

	var rspSize int
	for {
		rsp, err := cs.Recv()
		if err == io.EOF {
			break
		}
		require.Nil(s.T(), err)
		rspSize += len(rsp.GetPayload().GetBody())
		if rspSize == itemSize*sendNum {
			break
		}
	}
	_, err = cs.Recv()
	require.NotNil(s.T(), err)
}

func (s *TestSuite) testBidirectionalStreamingCallConcurrentlyOk(e streamEnv) {
	var g errgroup.Group
	for i := 0; i < 20; i++ {
		g.Go(func() error {
			_, err := s.testBidirectionalStreamingCallSequentiallyOk(e)
			return err
		})
	}
	require.Nil(s.T(), g.Wait())
}

func (s *TestSuite) testBidirectionalStreamingClientSendDataAfterCloseSend(e streamEnv) {
	cs, err := s.testBidirectionalStreamingCallSequentiallyOk(e)
	require.Nil(s.T(), err)

	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
	require.Nil(s.T(), err)
	err = cs.Send(&testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: []*testpb.ResponseParameters{{Size: 1}},
		Payload:            payload,
	})

	require.Equal(s.T(), errs.RetServerSystemErr, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "Connection is Closed")
}

func (s *TestSuite) testBidirectionalStreamingContinueReceivingDataAfterReceiveEOF(e streamEnv) {
	cs, err := s.testBidirectionalStreamingCallSequentiallyOk(e)
	require.Nil(s.T(), err)
	errChan := make(chan error)
	go func() {
		_, err = cs.Recv()
		errChan <- err
	}()

	select {
	case <-time.After(3 * time.Second):
	case <-errChan:
		s.T().Fatal("Recv should be blocked forever")
	}
}

func (s *TestSuite) testBidirectionalStreamingCallCloseAndReceiveTwice(e streamEnv) {
	cs, err := s.testBidirectionalStreamingCallSequentiallyOk(e)
	require.Nil(s.T(), err)
	err = cs.CloseSend()
	require.Equal(s.T(), errs.RetServerSystemErr, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "Connection is Closed")
}

func (s *TestSuite) testBidirectionalStreamingDontSendDataAfterCreatingStreaming(e streamEnv) {
	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.FullDuplexCall(trpc.BackgroundContext())
	require.Nil(s.T(), err)
	require.Nil(s.T(), cs.CloseSend())
}

func (s *TestSuite) TestBidirectionalStreamingClientReceiveDataWhileServerCloseStream() {
	s.startServer(&StreamingService{
		FullDuplexCallF: func(stream testpb.TestStreaming_FullDuplexCallServer) error {
			return nil
			// server has closed stream when this function return.
		},
	})

	c := s.newStreamingClient()
	cs, err := c.FullDuplexCall(trpc.BackgroundContext())
	require.Nil(s.T(), err)

	_, err = cs.Recv()
	require.ErrorIs(s.T(), err, io.EOF)
}

func (s *TestSuite) TestFlowControlWindowSizeUpdateOk() {
	const InitWindowSize = 65535
	s.startServer(&StreamingService{}, server.WithMaxWindowSize(InitWindowSize))

	c := s.newStreamingClient(client.WithMaxWindowSize(InitWindowSize))
	cs, err := c.FullDuplexCall(trpc.BackgroundContext())
	require.Nil(s.T(), err)

	doFullDuplexCall := func(messageSize int) {
		payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(messageSize))
		require.Nil(s.T(), err)
		respParams := []*testpb.ResponseParameters{
			{
				Size: int32(messageSize),
			},
		}
		req := &testpb.StreamingOutputCallRequest{
			ResponseType:       testpb.PayloadType_COMPRESSABLE,
			ResponseParameters: respParams,
			Payload:            payload,
		}
		require.Nil(s.T(), cs.Send(req))
		rsp, err := cs.Recv()
		require.Nil(s.T(), err)
		require.Len(s.T(), rsp.GetPayload().GetBody(), int(respParams[0].Size))
	}

	for i := 1; i <= 20; i++ {
		doFullDuplexCall(InitWindowSize * i)
	}
	require.Nil(s.T(), cs.CloseSend())
}

func (s *TestSuite) TestWithMaxWindowSizeNotWorkWhenLessThanDefaultInitWindowSize() {
	const (
		defaultInitWindowSize = 65535
		windowSize            = 0
	)

	s.startServer(&StreamingService{}, server.WithMaxWindowSize(windowSize))

	c := s.newStreamingClient(client.WithMaxWindowSize(windowSize))
	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, defaultInitWindowSize)
	require.Nil(s.T(), err)
	cs, err := c.FullDuplexCall(
		trpc.BackgroundContext(),
		client.WithMaxWindowSize(windowSize),
	)
	require.Nil(s.T(), err)

	for i := 0; i < 3; i++ {
		require.Nil(
			s.T(),
			cs.Send(&testpb.StreamingOutputCallRequest{
				Payload:            payload,
				ResponseType:       testpb.PayloadType_COMPRESSABLE,
				ResponseParameters: []*testpb.ResponseParameters{{Size: int32(defaultInitWindowSize)}},
			}),
		)
	}

	for i := 0; i < 3; i++ {
		rsp, err := cs.Recv()
		require.Nil(s.T(), err)
		require.Equal(s.T(), defaultInitWindowSize, len(rsp.GetPayload().GetBody()))
	}

	require.Nil(s.T(), cs.CloseSend())
}

func (s *TestSuite) TestSendBlockWhenContinuousSendDataMoreThanReceivedWindowSize() {
	const defaultInitWindowSize = 65535
	send := make(chan struct{})
	s.startServer(
		&StreamingService{
			FullDuplexCallF: func(stream testpb.TestStreaming_FullDuplexCallServer) error {
				_, err := stream.Recv()
				if err != nil {
					return err
				}
				<-send
				_, err = stream.Recv()
				return err
			},
		},
		server.WithMaxWindowSize(defaultInitWindowSize),
	)

	c := s.newStreamingClient()
	cs, err := c.FullDuplexCall(trpc.BackgroundContext())
	require.Nil(s.T(), err)

	payloadSize := defaultInitWindowSize
	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(payloadSize))
	require.Nil(s.T(), err)
	require.Nil(s.T(), cs.Send(&testpb.StreamingOutputCallRequest{
		Payload:      payload,
		ResponseType: testpb.PayloadType_COMPRESSABLE,
	}))

	payload, err = newPayload(testpb.PayloadType_COMPRESSABLE, int32(payloadSize))
	require.Nil(s.T(), err)
	require.Nil(s.T(), cs.Send(&testpb.StreamingOutputCallRequest{
		Payload:      payload,
		ResponseType: testpb.PayloadType_COMPRESSABLE,
	}))

	received := make(chan struct{})
	go func() {
		payload, err = newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
		require.Nil(s.T(), err)
		cs.Send(&testpb.StreamingOutputCallRequest{
			Payload:      payload,
			ResponseType: testpb.PayloadType_COMPRESSABLE,
		})
		close(received)
	}()

	isBlock := true
	select {
	case <-received:
		isBlock = false
	case <-time.After(5 * time.Second):
		close(send)
	}
	require.True(s.T(), isBlock)
}

func (s *TestSuite) TestServerStreaming() {
	for _, e := range allStreamEnvs {
		s.Run(e.String(), func() {
			s.testServerStreaming(e)
		})
	}
}

func (s *TestSuite) testServerStreaming(e streamEnv) {
	s.startServer(&StreamingService{},
		server.WithStreamTransport(transport.GetServerStreamTransport(e.server.transport)))
	s.T().Cleanup(func() { s.closeServer(nil) })
	s.Run("CallSequentiallyOk", func() {
		_, err := s.testServerStreamingCallSequentiallyOk(e)
		require.Nil(s.T(), err)
	})
	s.Run("CallConcurrentlyOk", func() {
		s.testServerStreamingCallConcurrentlyOk(e)
	})
	s.Run("CallSequentiallyFailed", func() {
		s.testServerStreamingCallSequentiallyFailed(e)
	})
	s.Run("ClientSendDataAfterCloseSend", func() {
		s.testServerStreamingSendDataAfterCloseSend(e)
	})
	s.Run("ReceiveDataAfterReceiveEOF", func() {
		s.testServerStreamingReceiveDataAfterReceiveEOF(e)
	})
	s.Run("DontReceiveDataAfterCreatingStreaming", func() {
		s.testServerStreamingDontReceiveDataAfterCreatingStreaming(e)
	})
	s.Run("CallCloseAndRecvTwice", func() {
		s.testServerStreamingCallCloseAndReceiveTwice(e)
	})
}

func (s *TestSuite) testServerStreamingCallSequentiallyOk(e streamEnv) (testpb.TestStreaming_StreamingOutputCallClient, error) {
	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
	if err != nil {
		return nil, err
	}

	respParams := make([]*testpb.ResponseParameters, 10)
	for i := int32(0); i < 10; i++ {
		respParams = append(respParams, &testpb.ResponseParameters{Size: i})
	}
	req := &testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: respParams,
		Payload:            payload,
	}

	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.StreamingOutputCall(trpc.BackgroundContext(), req)
	if err != nil {
		return nil, err
	}

	if err := cs.CloseSend(); err != nil {
		return nil, err
	}

	var (
		count       int
		sendSize    int32
		receiveSize int32
	)
	for {
		rsp, err := cs.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		sendSize = respParams[count].GetSize()
		receiveSize = int32(len(rsp.GetPayload().GetBody()))
		if sendSize != receiveSize {
			return nil, fmt.Errorf("receive size(want: %d != got: %d)", sendSize, receiveSize)
		}
		count++
	}
	if count != len(respParams) {
		return nil, fmt.Errorf("num of receiving (want: %d != got: %d)", len(respParams), count)
	}

	return cs, nil
}

func (s *TestSuite) testServerStreamingCallSequentiallyFailed(e streamEnv) {
	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
	require.Nil(s.T(), err)

	respParams := make([]*testpb.ResponseParameters, 10)
	for i := int32(0); i < 10; i++ {
		respParams = append(respParams, &testpb.ResponseParameters{Size: i})
	}
	invalidIndex := len(respParams)
	respParams = append(respParams, &testpb.ResponseParameters{Size: int32(-1)})

	req := &testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: respParams,
		Payload:            payload,
	}
	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.StreamingOutputCall(trpc.BackgroundContext(), req)
	require.Nil(s.T(), err)

	var index int
	for {
		rsp, err := cs.Recv()
		if err == io.EOF {
			break
		}
		if index == invalidIndex {
			require.NotNil(s.T(), err)
			break
		} else {
			require.Nil(s.T(), err)
			require.Equal(s.T(), respParams[index].GetSize(), int32(len(rsp.GetPayload().GetBody())))
		}
		index++
	}
	require.Equal(s.T(), invalidIndex, index)
}

func (s *TestSuite) testServerStreamingCallConcurrentlyOk(e streamEnv) {
	var g errgroup.Group
	for i := 0; i < 20; i++ {
		g.Go(func() error {
			_, err := s.testServerStreamingCallSequentiallyOk(e)
			return err
		})
	}
	require.Nil(s.T(), g.Wait())
}

func (s *TestSuite) testServerStreamingSendDataAfterCloseSend(e streamEnv) {
	cs, err := s.testServerStreamingCallSequentiallyOk(e)
	require.Nil(s.T(), err)

	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
	require.Nil(s.T(), err)

	req := &testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: []*testpb.ResponseParameters{{Size: 1}},
		Payload:            payload,
	}
	err = cs.SendMsg(req)
	require.Equal(s.T(), errs.RetServerSystemErr, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "Connection is Closed")
}

func (s *TestSuite) testServerStreamingReceiveDataAfterReceiveEOF(e streamEnv) {
	cs, err := s.testServerStreamingCallSequentiallyOk(e)
	require.Nil(s.T(), err)
	errChan := make(chan error)
	go func() {
		_, err = cs.Recv()
		errChan <- err
	}()

	select {
	case <-time.After(3 * time.Second):
	case <-errChan:
		s.T().Fatal("Recv should be blocked forever")
	}
}

func (s *TestSuite) testServerStreamingDontReceiveDataAfterCreatingStreaming(e streamEnv) {
	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.StreamingOutputCall(trpc.BackgroundContext(), &testpb.StreamingOutputCallRequest{})
	require.Nil(s.T(), err)
	require.Nil(s.T(), cs.CloseSend())
}

func (s *TestSuite) testServerStreamingCallCloseAndReceiveTwice(e streamEnv) {
	cs, err := s.testServerStreamingCallSequentiallyOk(e)
	require.Nil(s.T(), err)
	err = cs.CloseSend()
	require.Equal(s.T(), errs.RetServerSystemErr, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "Connection is Closed")
}

func (s *TestSuite) TestClientStreaming() {
	for _, e := range allStreamEnvs {
		s.Run(e.String(), func() {
			s.testClientStreaming(e)
		})
	}
}

func (s *TestSuite) testClientStreaming(e streamEnv) {
	s.startServer(&StreamingService{},
		server.WithStreamTransport(transport.GetServerStreamTransport(e.server.transport)))
	s.T().Cleanup(func() { s.closeServer(nil) })
	s.Run("CallSequentiallyOk", func() {
		_, err := s.testClientStreamingCallSequentiallyOk(e)
		require.Nil(s.T(), err)
	})
	s.Run("CallConcurrentlyOk", func() {
		s.testClientStreamingCallConcurrentlyOk(e)
	})
	s.Run("ClientSendDataAfterCloseSend", func() {
		s.testClientStreamingClientSendDataAfterCloseSend(e)
	})
	s.Run("ReceiveDataAfterCloseAndReceive", func() {
		s.testClientStreamingReceiveDataAfterCloseAndReceive(e)
	})
	s.Run("DontSendDataAfterCreatingStreaming", func() {
		s.testClientStreamingDontSendDataAfterCreatingStreaming(e)
	})
	s.Run("CallCloseAndRecvTwice", func() {
		s.testClientStreamingCallCloseAndReceiveTwice(e)
	})
}

func (s *TestSuite) testClientStreamingCallSequentiallyOk(e streamEnv) (testpb.TestStreaming_StreamingInputCallClient, error) {
	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.StreamingInputCall(trpc.BackgroundContext())
	if err != nil {
		return cs, err
	}

	var sendSize int
	for i := 1; i <= 10; i++ {
		sendSize += i
		payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(i))
		if err != nil {
			return cs, err
		}

		if err := cs.Send(&testpb.StreamingInputCallRequest{Payload: payload}); err != nil {
			return cs, err
		}
	}
	rsp, err := cs.CloseAndRecv()
	if err != nil {
		return cs, err
	}
	if int(rsp.AggregatedPayloadSize) != sendSize {
		return cs, fmt.Errorf("sendSize(%d) doest't not equal receiveSize(%d)", sendSize, rsp.AggregatedPayloadSize)
	}
	return cs, nil
}

func (s *TestSuite) testClientStreamingCallConcurrentlyOk(e streamEnv) {
	var g errgroup.Group
	for i := 0; i < 10; i++ {
		g.Go(func() error {
			_, err := s.testClientStreamingCallSequentiallyOk(e)
			return err
		})
	}
	require.Nil(s.T(), g.Wait())
}

func (s *TestSuite) testClientStreamingClientSendDataAfterCloseSend(e streamEnv) {
	cs, err := s.testClientStreamingCallSequentiallyOk(e)
	require.Nil(s.T(), err)

	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, 10)
	require.Nil(s.T(), err)
	err = cs.Send(&testpb.StreamingInputCallRequest{Payload: payload})
	require.Equal(s.T(), errs.RetServerSystemErr, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "Connection is Closed")
}

func (s *TestSuite) testClientStreamingReceiveDataAfterCloseAndReceive(e streamEnv) {
	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.StreamingInputCall(trpc.BackgroundContext())
	require.Nil(s.T(), err)
	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, 10)
	require.Nil(s.T(), err)
	require.Nil(s.T(), cs.Send(&testpb.StreamingInputCallRequest{Payload: payload}))
	require.Nil(s.T(), cs.Send(&testpb.StreamingInputCallRequest{Payload: payload}))

	rsp, err := cs.CloseAndRecv()
	require.Nil(s.T(), err)
	require.Equal(s.T(), 20, int(rsp.AggregatedPayloadSize))

	errChan := make(chan error)
	go func() {
		errChan <- cs.RecvMsg(rsp)
	}()

	select {
	case <-time.After(3 * time.Second):
	case <-errChan:
		s.T().Fatal("RecvMsg should be blocked forever")
	}
}

func (s *TestSuite) testClientStreamingCallCloseAndReceiveTwice(e streamEnv) {
	cs, err := s.testClientStreamingCallSequentiallyOk(e)
	require.Nil(s.T(), err)
	_, err = cs.CloseAndRecv()
	require.Equal(s.T(), errs.RetServerSystemErr, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "Connection is Closed")
}

func (s *TestSuite) testClientStreamingDontSendDataAfterCreatingStreaming(e streamEnv) {
	c := s.newStreamingClient(client.WithStreamTransport(transport.GetClientStreamTransport(e.client.transport)))
	cs, err := c.StreamingInputCall(trpc.BackgroundContext())
	require.Nil(s.T(), err)

	_, err = cs.CloseAndRecv()
	require.Nil(s.T(), err)
}
