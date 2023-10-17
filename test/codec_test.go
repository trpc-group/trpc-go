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
	"github.com/stretchr/testify/require"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
)

func (s *TestSuite) TestCompressOkSetByConfig() {
	trpc.ServerConfigPath = "trpc_go_trpc_server_with_compress.yaml"
	s.startTRPCServerWithListener(&TRPCService{})

	c := s.newTRPCClient()
	rsp, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))

	_, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCurrentCompressType(codec.CompressTypeGzip),
	)
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err))
}

func (s *TestSuite) TestCurrentCompressTypeOption() {
	s.startServer(&TRPCService{}, server.WithCurrentCompressType(codec.CompressTypeGzip))

	c := s.newTRPCClient()
	rsp, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCurrentCompressType(codec.CompressTypeGzip),
	)
	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))

	rsp, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCurrentCompressType(codec.CompressTypeZlib),
	)
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err))

	rsp, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCompressType(codec.CompressTypeGzip),
	)
	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))

	rsp, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCompressType(codec.CompressTypeZlib),
	)
	require.Equal(
		s.T(),
		errs.RetServerDecodeFail,
		errs.Code(err),
		"failed because of server.WithCurrentCompressType",
	)
}

func (s *TestSuite) TestWithCompressTypeOption() {
	s.startServer(&TRPCService{})

	supportedCompressTypes := []int{
		codec.CompressTypeNoop,
		codec.CompressTypeGzip,
		codec.CompressTypeSnappy,
		codec.CompressTypeZlib,
		codec.CompressTypeStreamSnappy,
		codec.CompressTypeBlockSnappy,
	}
	c := s.newTRPCClient()
	for _, ct := range supportedCompressTypes {
		rsp, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithCompressType(ct))
		require.Nil(s.T(), err)
		require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))
	}
}

func (s *TestSuite) TestClientCompressorNotRegistered() {
	s.startServer(&TRPCService{})
	s.Run("PositiveCompressType", func() {
		c := s.newTRPCClient()
		_, err := c.UnaryCall(
			trpc.BackgroundContext(),
			s.defaultSimpleRequest,
			client.WithCurrentCompressType(100),
		)
		require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err))
		require.Contains(s.T(), err.Error(), "compressor not registered")
	})
	s.Run("NegativeCompressType", func() {
		c := s.newTRPCClient()
		_, err := c.UnaryCall(
			trpc.BackgroundContext(),
			s.defaultSimpleRequest,
			client.WithCurrentCompressType(-100),
		)
		require.Nil(s.T(), err)
	})
}

func (s *TestSuite) TestRegisterNewCompressorOk() {
	s.startServer(&TRPCService{})
	for _, newCompressType := range []int{-100, 0, 100} {
		s.testRegisterNewCompressorOk(newCompressType)
	}
}

func (s *TestSuite) testRegisterNewCompressorOk(compressType int) {
	codec.RegisterCompressor(compressType, &codec.NoopCompress{})
	c := s.newTRPCClient()
	rsp, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCurrentCompressType(compressType),
	)
	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))
}

func (s *TestSuite) TestSerializationOkSetByConfig() {
	s.writeTRPCConfig(`
global:
  namespace: Development
  env_name: test
server:
  app: testing
  server: end2end
  service:
    - name: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      serialization: 2 # json
client:
  service:
    - callee: trpc.testing.end2end.TestTRPC
      protocol: trpc
      network: tcp
      serialization: 2
`)
	s.startTRPCServerWithListener(&TRPCService{})

	c := s.newTRPCClient()
	rsp, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)
	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))

	_, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCurrentSerializationType(codec.SerializationTypePB),
	)
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err))
}

func (s *TestSuite) TestCurrentSerializationTypeOption() {
	s.startServer(&TRPCService{}, server.WithCurrentSerializationType(codec.SerializationTypeJSON))

	c := s.newTRPCClient()
	rsp, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCurrentSerializationType(codec.SerializationTypeJSON),
	)
	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))

	rsp, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCurrentCompressType(codec.SerializationTypePB),
	)
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err))

	rsp, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithSerializationType(codec.SerializationTypeJSON),
	)
	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))

	rsp, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithSerializationType(codec.SerializationTypePB),
	)
	require.Equal(
		s.T(),
		errs.RetServerDecodeFail,
		errs.Code(err),
		"failed because of server.WithCurrentSerializationType",
	)
}

func (s *TestSuite) TestWithSerializationTypeOption() {
	supportedSerializationTypes := []int{
		codec.SerializationTypePB,
		codec.SerializationTypeJSON,
	}
	s.startServer(&TRPCService{})

	c := s.newTRPCClient()
	for _, st := range supportedSerializationTypes {
		rsp, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithSerializationType(st))
		require.Nil(s.T(), err)
		require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))
	}
}

func (s *TestSuite) TestClientSerializerNotRegistered() {
	s.startServer(&TRPCService{})
	s.Run("PositiveSerializationType", func() {
		c := s.newTRPCClient()
		_, err := c.UnaryCall(
			trpc.BackgroundContext(),
			s.defaultSimpleRequest,
			client.WithSerializationType(100),
		)
		require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err))
		require.Contains(s.T(), err.Error(), "serializer not registered")
	})
	s.Run("NegativeSerializationType", func() {
		c := s.newTRPCClient()
		rsp, err := c.UnaryCall(
			trpc.BackgroundContext(),
			s.defaultSimpleRequest,
			client.WithSerializationType(-100),
		)
		require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))
		require.Nil(s.T(), err)
	})
	s.Run("UnsupportedSerializationType", func() {
		c := s.newTRPCClient()
		rsp, err := c.UnaryCall(
			trpc.BackgroundContext(),
			s.defaultSimpleRequest,
			client.WithSerializationType(codec.SerializationTypeUnsupported),
		)
		require.Nil(s.T(), err)
		require.Nil(s.T(), rsp.Payload)
	})
}

func (s *TestSuite) TestRegisterUnsupportedSerializationType() {
	codec.RegisterSerializer(codec.SerializationTypeUnsupported, &codec.PBSerialization{})
	s.startServer(&TRPCService{}, server.WithCurrentSerializationType(codec.SerializationTypeUnsupported))
	c := s.newTRPCClient()
	rsp, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithCurrentSerializationType(codec.SerializationTypeUnsupported),
	)
	require.Nil(s.T(), err)
	require.Nil(s.T(), rsp.Payload)
}
func (s *TestSuite) TestRegisterNewSerializerOk() {
	for _, newSerializerType := range []int{-100, 100, 0} {
		s.testRegisterNewSerializerOk(newSerializerType)
	}
}

func (s *TestSuite) testRegisterNewSerializerOk(serializationType int) {
	codec.RegisterSerializer(serializationType, &codec.PBSerialization{})
	s.startServer(&TRPCService{})
	defer s.closeServer(nil)

	c := s.newTRPCClient()
	rsp, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithSerializationType(serializationType),
	)
	require.Nil(s.T(), err)
	require.Len(s.T(), rsp.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))
}
