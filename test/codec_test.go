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
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestSimpleRPCClientCodecFailed() {
	s.startServer(&TRPCService{})
	s.T().Run("decompress failed", func(t *testing.T) {
		const testCompressType = 20230426
		codec.RegisterCompressor(testCompressType, newFakeCompressor(nil, errors.New("decompress failed")))
		c := s.newTRPCClient()
		_, err := c.UnaryCall(
			trpc.BackgroundContext(),
			s.defaultSimpleRequest,
			client.WithCurrentCompressType(testCompressType),
		)
		require.Equal(s.T(), errs.RetClientDecodeFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "decompress failed")
	})
	s.T().Run("unmarshal failed", func(t *testing.T) {
		const testSerializationType = 20230426
		codec.RegisterSerializer(testSerializationType, newFakeSerializer(nil, errors.New("unmarshal failed")))
		c := s.newTRPCClient()
		_, err := c.UnaryCall(
			trpc.BackgroundContext(),
			s.defaultSimpleRequest,
			client.WithSerializationType(codec.SerializationTypePB),
			client.WithCurrentSerializationType(testSerializationType),
		)
		require.Equal(t, errs.RetClientDecodeFail, errs.Code(err))
		require.Contains(t, errs.Msg(err), "unmarshal failed")
	})
}

func (s *TestSuite) TestStreamingRPCClientCodecFailed() {
	s.startServer(&StreamingService{})
	s.T().Run("encode failed", func(t *testing.T) {
		codec.Register("test", trpc.DefaultServerCodec, newFakeCodec(trpc.DefaultClientCodec, errors.New("encode failed"), nil))
		c := s.newStreamingClient(client.WithProtocol("test"))
		_, err := c.FullDuplexCall(trpc.BackgroundContext())
		require.Equal(t, errs.RetClientStreamInitErr, errs.Code(err))
		require.Contains(t, errs.Msg(err), "encode failed")
	})
	s.T().Run("unmarshal failed", func(t *testing.T) {
		const testSerializationType = 20230428
		codec.RegisterSerializer(testSerializationType, newFakeSerializer(nil, errors.New("unmarshal failed")))
		c := s.newStreamingClient(client.WithCurrentSerializationType(testSerializationType))
		cs, err := c.FullDuplexCall(trpc.BackgroundContext())
		require.Nil(t, err)
		payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
		req := &testpb.StreamingOutputCallRequest{
			ResponseType: testpb.PayloadType_COMPRESSABLE,
			ResponseParameters: []*testpb.ResponseParameters{
				{
					Size:     2,
					Interval: durationpb.New(time.Microsecond),
				},
			},
			Payload: payload,
		}
		require.Nil(t, err)
		require.Nil(t, cs.Send(req))
		require.Nil(t, cs.CloseSend())

		_, err = cs.Recv()
		require.Equal(t, errs.RetClientDecodeFail, errs.Code(err))
		require.Contains(t, errs.Msg(err), "unmarshal failed")
	})
}

func (s *TestSuite) TestStreamingRPCServerCodecFailed() {
	s.T().Run("marshal failed", func(t *testing.T) {
		const testSerializationType = 20230428
		codec.RegisterSerializer(testSerializationType, newFakeSerializer(errors.New("marshal failed"), nil))
		s.startServer(&StreamingService{}, server.WithCurrentSerializationType(testSerializationType))
		t.Cleanup(func() {
			s.closeServer(nil)
		})

		cs := mustNewDuplexCallAndSendData(t, s.newStreamingClient())
		_, err := cs.Recv()

		require.Equal(t, errs.RetServerEncodeFail, errs.Code(err))
		require.Contains(t, errs.Msg(err), "server codec Marshal: marshal failed")
	})
	s.T().Run("compress failed", func(t *testing.T) {
		const testCompressType = 20230428
		codec.RegisterCompressor(testCompressType, newFakeCompressor(errors.New("compress failed"), nil))
		s.startServer(&StreamingService{}, server.WithCurrentCompressType(testCompressType))
		t.Cleanup(func() {
			s.closeServer(nil)
		})

		cs := mustNewDuplexCallAndSendData(t, s.newStreamingClient())
		_, err := cs.Recv()

		require.Equal(t, errs.RetServerEncodeFail, errs.Code(err))
		require.Contains(t, errs.Msg(err), "server codec Compress: compress failed")
	})
	s.T().Run("encode failed", func(t *testing.T) {
		codec.Register("test-20230428", newFakeCodec(trpc.DefaultServerCodec, errors.New("encode failed"), nil), trpc.DefaultClientCodec)
		s.startServer(&StreamingService{}, server.WithProtocol("test-20230428"))
		t.Cleanup(func() {
			s.closeServer(nil)
		})
		c := s.newStreamingClient(client.WithProtocol("test-20230428"))
		cs, err := c.FullDuplexCall(trpc.BackgroundContext())

		// TODO: should return a new error wrapped io.EOF
		require.True(t, errors.Is(err, io.EOF), "encode fail:encode failed, err: %+v", err)
		require.Nil(t, cs)
	})
}

func mustNewDuplexCallAndSendData(t *testing.T,
	c testpb.TestStreamingClientProxy) testpb.TestStreaming_FullDuplexCallClient {
	t.Helper()

	cs, err := c.FullDuplexCall(trpc.BackgroundContext())
	if err != nil {
		t.Fatal(err)
	}

	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
	if err != nil {
		t.Fatal(err)
	}

	req := &testpb.StreamingOutputCallRequest{
		ResponseType: testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: []*testpb.ResponseParameters{
			{
				Size:     2,
				Interval: durationpb.New(time.Microsecond),
			},
		},
		Payload: payload,
	}
	if err := cs.Send(req); err != nil {
		t.Fatal(err)
	}
	return cs
}

type fakeCodec struct {
	codec       codec.Codec
	encodeErr   error
	encodeCount int
	decodeErr   error
}

func newFakeCodec(codec codec.Codec, encodeErr, decodeErr error) *fakeCodec {
	return &fakeCodec{
		codec:     codec,
		encodeErr: encodeErr,
		decodeErr: decodeErr,
	}
}

func (c *fakeCodec) Encode(msg codec.Msg, reqBody []byte) ([]byte, error) {
	reqBuf, err := c.codec.Encode(msg, reqBody)
	if err != nil {
		return nil, err
	}
	if c.encodeErr != nil {
		return nil, c.encodeErr
	}
	return reqBuf, nil
}

func (c *fakeCodec) Decode(msg codec.Msg, rspBuf []byte) ([]byte, error) {
	reqBody, err := c.codec.Decode(msg, rspBuf)
	if err != nil {
		return nil, err
	}
	if c.decodeErr != nil {
		return nil, c.decodeErr
	}
	return reqBody, nil
}

type fakeCompressor struct {
	compressErr   error
	decompressErr error
}

func newFakeCompressor(compressErr, decompressErr error) *fakeCompressor {
	return &fakeCompressor{compressErr: compressErr, decompressErr: decompressErr}
}

func (c *fakeCompressor) Compress(in []byte) (out []byte, err error) {
	if c.compressErr != nil {
		return nil, c.compressErr
	}
	return in, nil
}

func (c *fakeCompressor) Decompress(in []byte) (out []byte, err error) {
	if c.decompressErr != nil {
		return nil, c.decompressErr
	}
	return in, nil
}

type fakeSerializer struct {
	s            codec.PBSerialization
	marshalErr   error
	unmarshalErr error
}

func newFakeSerializer(marshalErr, unmarshalErr error) *fakeSerializer {
	return &fakeSerializer{marshalErr: marshalErr, unmarshalErr: unmarshalErr}
}

func (s *fakeSerializer) Marshal(body interface{}) ([]byte, error) {
	bts, err := s.s.Marshal(body)
	if err != nil {
		return nil, err
	}
	if s.marshalErr != nil {
		return nil, s.marshalErr
	}
	return bts, nil
}

func (s *fakeSerializer) Unmarshal(in []byte, body interface{}) error {
	if err := s.s.Unmarshal(in, body); err != nil {
		return err
	}
	if s.unmarshalErr != nil {
		return s.unmarshalErr
	}
	return nil
}

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
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err), "full err: %+v", err)
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
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err), "full err: %+v", err)

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
		require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err), "full err: %+v", err)
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
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err), "full err: %+v", err)
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
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err), "full err: %+v", err)

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
		require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err), "full err: %+v", err)
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
