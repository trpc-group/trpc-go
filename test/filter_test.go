package test

import (
	"context"
	"time"

	"github.com/stretchr/testify/require"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

var filterTestError = errs.New(555, "filter test error")

func (s *TestSuite) TestUnaryServerFilter() {
	s.startServer(&TRPCService{}, server.WithFilter(errInjector))
	c := s.newTRPCClient()
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Equal(s.T(), filterTestError, err)
}

func errInjector(
	ctx context.Context,
	req interface{},
	next filter.ServerHandleFunc,
) (rsp interface{}, err error) {
	next(ctx, req)
	return nil, filterTestError
}

func (s *TestSuite) TestUnaryClientFilter() {
	s.startServer(&TRPCService{})
	c := s.newTRPCClient()
	_, err := c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithFilter(failOkayRPC))
	require.Equal(s.T(), filterTestError, err)
}

func failOkayRPC(
	ctx context.Context,
	req, rsp interface{},
	next filter.ClientHandleFunc,
) error {
	err := next(ctx, req, rsp)
	if err == nil {
		return filterTestError
	}
	return nil
}

func (s *TestSuite) TestStreamClientFilter() {
	s.startServer(&StreamingService{})

	respParam := []*testpb.ResponseParameters{
		{
			Size: int32(1),
		},
	}
	payload, err := newPayload(testpb.PayloadType_COMPRESSIBLE, int32(1))
	require.Nil(s.T(), err)
	req := &testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSIBLE,
		ResponseParameters: respParam,
		Payload:            payload,
	}
	c := s.newStreamingClient()

	_, err = c.StreamingOutputCall(
		trpc.BackgroundContext(),
		req,
		client.WithStreamFilter(failOkayStream))
	require.Equal(s.T(), filterTestError, err)
}

func failOkayStream(
	ctx context.Context,
	desc *client.ClientStreamDesc,
	streamer client.Streamer,
) (client.ClientStream, error) {
	s, err := streamer(ctx, desc)
	if err == nil {
		return nil, filterTestError
	}
	return s, nil
}

func (s *TestSuite) TestFilterOrderOfExecution() {
	const testKey = "TestFilterOrderOfExecution"
	addSquare := func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
		value := trpc.GetMetaData(ctx, testKey)
		trpc.SetMetaData(ctx, testKey, append(value, byte('[')))

		rsp, err := next(ctx, req)

		value = trpc.GetMetaData(ctx, testKey)
		trpc.SetMetaData(ctx, testKey, append(value, byte(']')))
		return rsp, err
	}

	appendParentheses := func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
		value := trpc.GetMetaData(ctx, testKey)
		trpc.SetMetaData(ctx, testKey, append(value, byte('(')))

		rsp, err := next(ctx, req)

		value = trpc.GetMetaData(ctx, testKey)
		trpc.SetMetaData(ctx, testKey, append(value, byte(')')))
		return rsp, err
	}

	s.startServer(
		&TRPCService{},
		server.WithFilter(addSquare),
		server.WithFilter(appendParentheses),
	)

	c := s.newTRPCClient()
	head := &trpcpb.ResponseProtocol{}
	_, err := c.EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
		client.WithMetaData(testKey, []byte("{")),
		client.WithFilter(
			func(ctx context.Context,
				req, rsp interface{},
				next filter.ClientHandleFunc,
			) error {
				msg := trpc.Message(ctx)
				msg.WithClientMetaData(codec.MetaData{testKey: append(msg.ClientMetaData()[testKey], byte('}'))})
				err := next(ctx, req, rsp)

				msg = trpc.Message(ctx)
				require.Equal(s.T(), []byte("{}[()]"), msg.ClientMetaData()[testKey])
				return err
			},
		),
		client.WithRspHead(head),
	)
	require.Nil(s.T(), err)
	require.Equal(s.T(), []byte("{}[()]"), head.TransInfo[testKey])
}

func (s *TestSuite) TestStreamServerFilter() {
	s.startServer(&StreamingService{}, server.WithStreamFilter(
		func(ss server.Stream, info *server.StreamServerInfo, handler server.StreamHandler) error {
			err := handler(ss)
			if info.FullMethod == "/trpc.testing.end2end.TestStreaming/FullDuplexCall" {
				return err
			}
			return filterTestError
		}))

	respParam := []*testpb.ResponseParameters{
		{
			Size: int32(1),
		},
	}
	payload, err := newPayload(testpb.PayloadType_COMPRESSIBLE, int32(1))
	require.Nil(s.T(), err)
	req := &testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSIBLE,
		ResponseParameters: respParam,
		Payload:            payload,
	}

	c1 := s.newStreamingClient()
	s1, err := c1.StreamingInputCall(trpc.BackgroundContext())
	s1.Send(&testpb.StreamingInputCallRequest{})
	require.Nil(s.T(), err)
	_, err = s1.CloseAndRecv()
	require.Equal(s.T(), errs.Code(filterTestError), errs.Code(err))
	require.Equal(s.T(), errs.Msg(filterTestError), errs.Msg(err))

	c2 := s.newStreamingClient()
	s2, err := c2.FullDuplexCall(trpc.BackgroundContext())
	require.Nil(s.T(), err)
	require.Nil(s.T(), s2.Send(req))
	_, err = s2.Recv()
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestCancelAtServerFilter() {
	cancelFilter := func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		rsp, err := next(ctx, req)
		return rsp, err
	}
	s.startServer(&TRPCService{}, server.WithFilter(cancelFilter))

	c := s.newTRPCClient()
	rsp, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Nil(s.T(), err, "framework won't check whether caller's ctx is canceled")
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))
}

func (s *TestSuite) TestTimeoutAtServerFilter() {
	timeoutFilter := func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
		time.Sleep(time.Second)
		rsp, err := next(ctx, req)
		return rsp, err
	}
	s.startServer(&TRPCService{}, server.WithFilter(timeoutFilter))

	c := s.newTRPCClient()
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(100*time.Millisecond))
	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err))
}
