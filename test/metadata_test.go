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
	"fmt"
	"strings"

	"github.com/stretchr/testify/require"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/server"
)

func (s *TestSuite) TestClientWithMetaDataOption() {
	s.Run("ServerReturnOK", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), func() {
				s.startServer(&TRPCService{})
				defer s.closeServer(nil)
				require.Nil(s.T(), s.testClientWithMetaDataOption())
			})
		}
	})
	s.Run("ServerReturnError", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), func() {
				s.startServer(&TRPCService{}, server.WithFilter(
					func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
						return nil, fmt.Errorf("unknow error")
					}))
				defer s.closeServer(nil)
				require.NotNil(s.T(), s.testClientWithMetaDataOption())
			})
		}
	})
	s.Run("ClientSetVeryLargeMetaData", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), s.testClientWithVeryLargeMetaData)
		}
	})
}

func (s *TestSuite) testClientWithVeryLargeMetaData() {
	c := s.newTRPCClient()
	head := &trpcpb.ResponseProtocol{}
	_, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithMetaData("invalid-key", make([]byte, 65536)),
		client.WithRspHead(head),
	)
	require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err))
	require.Contains(s.T(), err.Error(), "head len overflows uint16")
}

func (s *TestSuite) testClientWithMetaDataOption() error {
	c := s.newTRPCClient()
	head := &trpcpb.ResponseProtocol{}
	_, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithMetaData("invalid-key", []byte("value")),
		client.WithRspHead(head),
	)
	require.Len(s.T(), head.TransInfo, 1)
	require.Equal(
		s.T(),
		[]byte("value"),
		head.TransInfo["invalid-key"],
		"metadata set by client.WithMetaData option will automatic postback",
	)
	return err
}

func (s *TestSuite) TestServerSetMetaData() {
	s.Run("ServerSetVeryLargeMetaData", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), s.testServerSetVeryLargeMetaData)
		}
	})
	s.Run("MultipleSetMetaDataReturnOk", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), func() {
				s.startServer(
					&TRPCService{},
					server.WithFilter(
						func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
							if value := trpc.GetMetaData(ctx, "repeat-value"); len(value) != 0 {
								trpc.SetMetaData(ctx, "repeat-value", append(value, value...))
							}
							rsp, err := next(ctx, req)
							if value := trpc.GetMetaData(ctx, "repeat-value"); len(value) != 0 {
								trpc.SetMetaData(ctx, "repeat-value", append(value, value...))
							}
							return rsp, err
						}),
				)
				defer s.closeServer(nil)
				require.Nil(s.T(), s.testMultipleSetMetaData())
			})
		}
	})
	s.Run("MultipleSetMetaDataReturnError", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), func() {
				s.startServer(
					&TRPCService{},
					server.WithFilter(
						func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
							if value := trpc.GetMetaData(ctx, "repeat-value"); len(value) != 0 {
								trpc.SetMetaData(ctx, "repeat-value", append(value, value...))
							}
							next(ctx, req)
							if value := trpc.GetMetaData(ctx, "repeat-value"); len(value) != 0 {
								trpc.SetMetaData(ctx, "repeat-value", append(value, value...))
							}
							return nil, fmt.Errorf("unknow error")
						}),
				)
				defer s.closeServer(nil)
				require.NotNil(s.T(), s.testMultipleSetMetaData())
			})
		}
	})
}

func (s *TestSuite) testServerSetVeryLargeMetaData() {
	s.startServer(
		&TRPCService{},
		server.WithFilter(
			func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
				trpc.SetMetaData(ctx, "repeat-value", make([]byte, 65536))
				rsp, err := next(ctx, req)
				return rsp, err
			}),
	)
	defer s.closeServer(nil)

	c := s.newTRPCClient()
	head := &trpcpb.ResponseProtocol{}
	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithRspHead(head))
	require.Equal(s.T(), errs.RetServerEncodeFail, errs.Code(err))
	require.Contains(s.T(), err.Error(), "head len overflows uint16")
	require.Nil(s.T(), head.TransInfo)
}

func (s *TestSuite) testMultipleSetMetaData() error {
	c := s.newTRPCClient()
	head := &trpcpb.ResponseProtocol{}
	_, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithMetaData("repeat-value", []byte("value")),
		client.WithRspHead(head),
	)
	require.Len(s.T(), head.TransInfo, 1)
	require.Equal(s.T(), []byte(strings.Repeat("value", 8)), head.TransInfo["repeat-value"])
	return err
}

func (s *TestSuite) TestMessageWithServerMetaDataOption() {
	s.Run("WithServerVeryLargeMetaData", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), s.testMessageWithServerVeryLargeMetaData)
		}
	})
	s.Run("ServerReturnError", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), func() {
				s.startServer(&TRPCService{}, server.WithFilter(
					func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
						return nil, fmt.Errorf("unknow error")
					}))
				defer s.closeServer(nil)
				require.NotNil(s.T(), s.testMessageWithServerMetaDataOption())
			})
		}
	})
	s.Run("ServerReturnOk", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), func() {
				s.startServer(&TRPCService{})
				defer s.closeServer(nil)
				require.Nil(s.T(), s.testMessageWithServerMetaDataOption())
			})
		}
	})
}

func (s *TestSuite) testMessageWithServerVeryLargeMetaData() {
	ctx := trpc.BackgroundContext()
	msg := trpc.Message(ctx)
	testMetadata := codec.MetaData{
		"key1":     []byte("value1"),
		"key2":     []byte("value2"),
		"key3-bin": make([]byte, 65536),
	}
	msg.WithServerMetaData(testMetadata)
	head := &trpcpb.ResponseProtocol{}
	c := s.newTRPCClient()
	_, err := c.UnaryCall(ctx, s.defaultSimpleRequest, client.WithRspHead(head))
	require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err))
	require.Contains(s.T(), err.Error(), "head len overflows uint16")
}

func (s *TestSuite) testMessageWithServerMetaDataOption() error {
	ctx := trpc.BackgroundContext()
	msg := trpc.Message(ctx)
	testMetadata := codec.MetaData{
		"key1":     []byte("value1"),
		"key2":     []byte("value2"),
		"key3-bin": []byte{1, 2, 3},
	}
	msg.WithServerMetaData(testMetadata)
	head := &trpcpb.ResponseProtocol{}
	c := s.newTRPCClient()
	_, err := c.UnaryCall(ctx, s.defaultSimpleRequest, client.WithRspHead(head))
	require.Equal(s.T(), testMetadata, codec.MetaData(head.TransInfo))
	return err
}

func (s *TestSuite) TestMessageWithClientMetaDataOption() {
	s.Run("ClientSetVeryLargeMetaData", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), s.testMessageWithClientVeryLargeMetaData)
		}
	})
	s.Run("ServerReturnOk", func() {
		for _, e := range allTRPCEnvs {
			s.tRPCEnv = e
			s.Run(e.String(), s.testMessageWithClientMetaDataOption)
		}
	})
}

func (s *TestSuite) testMessageWithClientVeryLargeMetaData() {
	ctx := trpc.BackgroundContext()
	testMetadata := codec.MetaData{
		"repeat-value": make([]byte, 65536),
	}
	head := &trpcpb.ResponseProtocol{}
	c := s.newTRPCClient()
	_, err := c.UnaryCall(
		ctx,
		s.defaultSimpleRequest,
		client.WithRspHead(head),
		client.WithFilter(func(
			ctx context.Context,
			req, rsp interface{},
			next filter.ClientHandleFunc) error {
			msg := trpc.Message(ctx)
			msg.WithClientMetaData(testMetadata)
			return next(ctx, req, rsp)
		}),
	)
	require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err))
	require.Contains(s.T(), err.Error(), "head len overflows uint16")
}

func (s *TestSuite) testMessageWithClientMetaDataOption() {
	s.startServer(&TRPCService{})
	defer s.closeServer(nil)
	ctx := trpc.BackgroundContext()
	testMetadata := codec.MetaData{
		"repeat-value": []byte("value1"),
		"key2":         []byte("value2"),
		"key3-bin":     []byte{1, 2, 3},
	}
	head := &trpcpb.ResponseProtocol{}
	c := s.newTRPCClient()
	_, err := c.UnaryCall(
		ctx,
		s.defaultSimpleRequest,
		client.WithRspHead(head),
		client.WithFilter(func(
			ctx context.Context,
			req, rsp interface{},
			next filter.ClientHandleFunc) error {
			msg := trpc.Message(ctx)
			msg.WithClientMetaData(testMetadata)
			return next(ctx, req, rsp)
		}),
	)
	require.Nil(s.T(), err)
	testMetadata["repeat-value"] = []byte(strings.Repeat("value1", 2))
	require.Equal(s.T(), testMetadata, codec.MetaData(head.TransInfo))
}

func (s *TestSuite) TestServerGetMetaDataOk() {
	for _, e := range allTRPCEnvs {
		s.tRPCEnv = e
		s.Run(e.String(), s.testServerGetMetaDataOk)
	}
}

func (s *TestSuite) testServerGetMetaDataOk() {
	s.startServer(&TRPCService{})
	defer s.closeServer(nil)

	c := s.newTRPCClient()
	head := &trpcpb.ResponseProtocol{}
	_, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithMetaData("repeat-value", []byte("value")),
		client.WithRspHead(head),
	)
	require.Nil(s.T(), err)
	require.Len(s.T(), head.TransInfo, 1)
	require.Equal(s.T(), []byte(strings.Repeat("value", 2)), head.TransInfo["repeat-value"])
}
