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
	"errors"
	"fmt"
	"io"
	"time"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

// TRPCService to test tRPC service.
type TRPCService struct {
	// Customizable implementations of server handlers.
	EmptyCallF func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error)
	UnaryCallF func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error)

	unaryCallSleepTime time.Duration
}

// EmptyCall to test empty call.
func (s *TRPCService) EmptyCall(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
	if s.EmptyCallF != nil {
		return s.EmptyCallF(ctx, in)
	}
	return &testpb.Empty{}, nil
}

// UnaryCall to test unary call.
func (s *TRPCService) UnaryCall(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
	if s.UnaryCallF != nil {
		return s.UnaryCallF(ctx, in)
	}
	// Simulate some service delay.
	time.Sleep(s.unaryCallSleepTime)

	if err := in.Validate(); err != nil {
		return nil, errs.Wrap(err, errs.RetServerValidateFail, "validating request parameter failed!")
	}

	payload, err := newPayload(in.GetResponseType(), in.GetResponseSize())
	if err != nil {
		return nil, err
	}

	if value := trpc.GetMetaData(ctx, "repeat-value"); len(value) != 0 {
		trpc.SetMetaData(ctx, "repeat-value", append(value, value...))
	}

	rsp := &testpb.SimpleResponse{Payload: payload}
	if in.FillUsername {
		// Validate the user name in request.
		if in.Username != validUserNameForAuth {
			return nil, errs.NewFrameError(errs.RetServerAuthFail, "need valid user name!")
		}
		rsp.Username = in.Username
	}
	return rsp, nil
}

// StreamingService to test streaming service.
type StreamingService struct {
	StreamingOutputCallF func(
		args *testpb.StreamingOutputCallRequest,
		stream testpb.TestStreaming_StreamingOutputCallServer) error
	FullDuplexCallF func(stream testpb.TestStreaming_FullDuplexCallServer) error
}

// StreamingOutputCall to test streaming output call.
func (s *StreamingService) StreamingOutputCall(
	args *testpb.StreamingOutputCallRequest,
	stream testpb.TestStreaming_StreamingOutputCallServer) error {
	if s.StreamingOutputCallF != nil {
		return s.StreamingOutputCallF(args, stream)
	}

	cs := args.GetResponseParameters()
	for _, c := range cs {
		if dur := c.GetInterval().AsDuration(); dur > 0 {
			time.Sleep(dur)
		}

		payload, err := newPayload(args.GetResponseType(), c.GetSize())
		if err != nil {
			return err
		}

		if err := stream.Send(&testpb.StreamingOutputCallResponse{
			Payload: payload,
		}); err != nil {
			return err
		}
	}
	return nil
}

// StreamingInputCall to test streaming input call.
func (s *StreamingService) StreamingInputCall(stream testpb.TestStreaming_StreamingInputCallServer) error {
	var sum int
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&testpb.StreamingInputCallResponse{
				AggregatedPayloadSize: int32(sum),
			})
		}
		if err != nil {
			return err
		}
		sum += len(in.GetPayload().GetBody())
	}
}

// FullDuplexCall to test full duplex call.
func (s *StreamingService) FullDuplexCall(stream testpb.TestStreaming_FullDuplexCallServer) error {
	if s.FullDuplexCallF != nil {
		return s.FullDuplexCallF(stream)
	}

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		cs := in.GetResponseParameters()
		for _, c := range cs {
			if dur := c.GetInterval().AsDuration(); dur > 0 {
				time.Sleep(dur * time.Microsecond)
			}

			payload, err := newPayload(in.GetResponseType(), c.GetSize())
			if err != nil {
				return err
			}

			if err := stream.Send(&testpb.StreamingOutputCallResponse{
				Payload: payload,
			}); err != nil {
				return err
			}
		}
	}
}

// HalfDuplexCall to test half duplex call.
func (s *StreamingService) HalfDuplexCall(stream testpb.TestStreaming_HalfDuplexCallServer) error {
	return nil
}

type testHTTPService struct {
	TRPCService
}

type testRESTfulService struct {
	ts TRPCService
	// Customizable implementations of server handlers.
	UnaryCallF func(ctx context.Context, req *testpb.SimpleRequest) (*testpb.SimpleResponse, error)
}

func (s *testRESTfulService) UnaryCall(
	ctx context.Context,
	req *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
	if s.UnaryCallF != nil {
		return s.UnaryCallF(ctx, req)
	}
	return s.ts.UnaryCall(ctx, req)
}

func newPayload(t testpb.PayloadType, size int32) (*testpb.Payload, error) {
	if size < 0 {
		return nil, fmt.Errorf("requested a response with invalid length %d", size)
	}

	switch t {
	case testpb.PayloadType_COMPRESSIBLE:
		return &testpb.Payload{
			Type: t,
			Body: make([]byte, size),
		}, nil
	case testpb.PayloadType_UNCOMPRESSABLE:
		return nil, errors.New("PayloadType UNCOMPRESSABLE is not supported")
	default:
		return nil, errs.New(retUnsupportedPayload, fmt.Sprintf("unsupported payload type: %d", t))
	}
}
