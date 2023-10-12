// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package trpc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"sync"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"google.golang.org/protobuf/proto"
)

var (
	// error for unknown stream frame type
	errUnknownFrameType error = errors.New("unknown stream frame type")
	// error for invalid total length of client decoding
	errClientDecodeTotalLength error = errors.New("client decode total length invalid")
	// error for failing to encode Close frame
	errEncodeCloseFrame error = errors.New("encode close frame error")
	// error for failing to encode Feedback frame
	errEncodeFeedbackFrame error = errors.New("encode feedback error")
	// error for init metadata not found
	errUninitializedMeta error = errors.New("uninitialized meta")
	// error for invalid trpc framehead
	errFrameHeadTypeInvalid error = errors.New("framehead type invalid")
)

// NewServerStreamCodec initializes and returns a ServerStreamCodec.
func NewServerStreamCodec() *ServerStreamCodec {
	return &ServerStreamCodec{initMetas: make(map[net.Addr]map[uint32]*trpcpb.TrpcStreamInitMeta), m: &sync.RWMutex{}}
}

// NewClientStreamCodec initializes and returns a ClientStreamCodec.
func NewClientStreamCodec() *ClientStreamCodec {
	return &ClientStreamCodec{}
}

// ServerStreamCodec is an implementation of codec.Codec.
// Used for trpc server streaming codec.
type ServerStreamCodec struct {
	m         *sync.RWMutex
	initMetas map[net.Addr]map[uint32]*trpcpb.TrpcStreamInitMeta // addr->streamID->TrpcStreamInitMeta
}

// ClientStreamCodec is an implementation of codec.Codec.
// Used for trpc client streaming codec.
type ClientStreamCodec struct {
}

// Encode implements codec.Codec.
func (c *ClientStreamCodec) Encode(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok || !frameHead.isStream() {
		return nil, errUnknownFrameType
	}
	switch trpcpb.TrpcStreamFrameType(frameHead.StreamFrameType) {
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		return c.encodeInitFrame(frameHead, msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		return c.encodeDataFrame(frameHead, msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		return c.encodeCloseFrame(frameHead, msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		return c.encodeFeedbackFrame(frameHead, msg, reqBuf)
	default:
		return nil, errUnknownFrameType
	}
}

// Decode implements codec.Codec.
func (c *ClientStreamCodec) Decode(msg codec.Msg, rspBuf []byte) ([]byte, error) {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok || !frameHead.isStream() {
		return nil, errUnknownFrameType
	}

	msg.WithStreamID(frameHead.StreamID)
	switch trpcpb.TrpcStreamFrameType(frameHead.StreamFrameType) {
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		return c.decodeInitFrame(msg, rspBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		return c.decodeDataFrame(msg, rspBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		return c.decodeCloseFrame(msg, rspBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		return c.decodeFeedbackFrame(msg, rspBuf)
	default:
		return nil, errUnknownFrameType
	}
}

// decodeCloseFrame decodes the Close frame.
func (c *ClientStreamCodec) decodeCloseFrame(msg codec.Msg, rspBuf []byte) ([]byte, error) {
	// unmarshal Close frame
	close := &trpcpb.TrpcStreamCloseMeta{}
	if err := proto.Unmarshal(rspBuf[frameHeadLen:], close); err != nil {
		return nil, err
	}

	// It is considered an exception and an error should be returned to the client if:
	// 1. the CloseType is Reset
	// 2. ret code != 0
	if close.GetCloseType() == int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_RESET) || close.GetRet() != 0 {
		e := &errs.Error{
			Type: errs.ErrorTypeCalleeFramework,
			Code: trpcpb.TrpcRetCode(close.GetRet()),
			Desc: "trpc",
			Msg:  string(close.GetMsg()),
		}
		msg.WithClientRspErr(e)
	}
	msg.WithStreamFrame(close)
	return nil, nil
}

// decodeFeedbackFrame decodes the Feedback frame.
func (c *ClientStreamCodec) decodeFeedbackFrame(msg codec.Msg, rspBuf []byte) ([]byte, error) {
	feedback := &trpcpb.TrpcStreamFeedBackMeta{}
	if err := proto.Unmarshal(rspBuf[frameHeadLen:], feedback); err != nil {
		return nil, err
	}
	msg.WithStreamFrame(feedback)
	return nil, nil
}

// decodeInitFrame decodes the Init frame.
func (c *ClientStreamCodec) decodeInitFrame(msg codec.Msg, rspBuf []byte) ([]byte, error) {
	initMeta := &trpcpb.TrpcStreamInitMeta{}
	if err := proto.Unmarshal(rspBuf[frameHeadLen:], initMeta); err != nil {
		return nil, err
	}

	msg.WithCompressType(int(initMeta.GetContentEncoding()))
	msg.WithSerializationType(int(initMeta.GetContentType()))

	// if ret code is not 0, an error should be set and returned
	if initMeta.GetResponseMeta().GetRet() != 0 {
		e := &errs.Error{
			Type: errs.ErrorTypeCalleeFramework,
			Code: trpcpb.TrpcRetCode(initMeta.GetResponseMeta().GetRet()),
			Desc: "trpc",
			Msg:  string(initMeta.GetResponseMeta().GetErrorMsg()),
		}
		msg.WithClientRspErr(e)
	}
	msg.WithStreamFrame(initMeta)
	return nil, nil

}

// decodeDataFrame decodes the Data frame.
func (c *ClientStreamCodec) decodeDataFrame(msg codec.Msg, rspBuf []byte) ([]byte, error) {
	// decoding Data frame is straightforward,
	// as it just returns all data following the frame head
	return rspBuf[frameHeadLen:], nil
}

// encodeInitFrame encodes the Init frame.
func (c *ClientStreamCodec) encodeInitFrame(frameHead *FrameHead, msg codec.Msg, reqBuf []byte) ([]byte, error) {
	initMeta, ok := msg.StreamFrame().(*trpcpb.TrpcStreamInitMeta)
	if !ok {
		initMeta = &trpcpb.TrpcStreamInitMeta{}
		initMeta.RequestMeta = &trpcpb.TrpcStreamInitRequestMeta{}
	}
	req := initMeta.RequestMeta
	// set caller service name
	// if nil, use the name of the process
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.app.%s.service", path.Base(os.Args[0])))
	}
	req.Caller = []byte(msg.CallerServiceName())
	// set callee service name
	req.Callee = []byte(msg.CalleeServiceName())
	// set backend rpc name, ClientRPCName already set by upper layer of client stub
	req.Func = []byte(msg.ClientRPCName())
	// set backend serialization type
	initMeta.ContentType = uint32(msg.SerializationType())
	// set backend compression type
	initMeta.ContentEncoding = uint32(msg.CompressType())
	// set dyeing info
	if msg.Dyeing() {
		req.MessageType = req.MessageType | uint32(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)
	}
	// set client transinfo
	req.TransInfo = setClientTransInfo(msg, req.TransInfo)
	streamBuf, err := proto.Marshal(initMeta)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// encodeDataFrame encodes the Data frame.
func (c *ClientStreamCodec) encodeDataFrame(frameHead *FrameHead, msg codec.Msg, reqBuf []byte) ([]byte, error) {
	return frameWrite(frameHead, reqBuf)
}

// encodeCloseFrame encodes the Close frame.
func (c *ClientStreamCodec) encodeCloseFrame(frameHead *FrameHead, msg codec.Msg,
	reqBuf []byte) (rspbuf []byte, err error) {
	closeFrame, ok := msg.StreamFrame().(*trpcpb.TrpcStreamCloseMeta)
	if !ok {
		return nil, errEncodeCloseFrame
	}
	streamBuf, err := proto.Marshal(closeFrame)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// encodeFeedbackFrame encodes the Feedback frame.
func (c *ClientStreamCodec) encodeFeedbackFrame(frameHead *FrameHead, msg codec.Msg, reqBuf []byte) ([]byte, error) {
	feedbackFrame, ok := msg.StreamFrame().(*trpcpb.TrpcStreamFeedBackMeta)
	if !ok {
		return nil, errEncodeFeedbackFrame
	}
	streamBuf, err := proto.Marshal(feedbackFrame)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// frameWrite converts FrameHead to binary frame.
func frameWrite(frameHead *FrameHead, streamBuf []byte) ([]byte, error) {
	// no pb header for streaming rpc
	return frameHead.construct(nil, streamBuf, nil)
}

// encodeCloseFrame encodes the Close frame.
func (s *ServerStreamCodec) encodeCloseFrame(frameHead *FrameHead, msg codec.Msg, reqBuf []byte) ([]byte, error) {
	defer s.deleteInitMeta(msg)
	closeFrame, ok := msg.StreamFrame().(*trpcpb.TrpcStreamCloseMeta)
	if !ok {
		return nil, errEncodeCloseFrame
	}
	msg.WithStreamID(frameHead.StreamID)
	streamBuf, err := proto.Marshal(closeFrame)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// encodeDataFrame encodes the Data frame.
func (s *ServerStreamCodec) encodeDataFrame(frameHead *FrameHead, msg codec.Msg, reqBuf []byte) ([]byte, error) {
	// If there is an error when processing the Data frame,
	// then return the Close frame and close the current stream.
	if err := msg.ServerRspErr(); err != nil {
		s.buildResetFrame(msg, frameHead, err)
		return s.encodeCloseFrame(frameHead, msg, reqBuf)
	}
	return frameWrite(frameHead, reqBuf)
}

// encodeInitFrame encodes the Init frame.
func (s *ServerStreamCodec) encodeInitFrame(frameHead *FrameHead, msg codec.Msg, reqBuf []byte) ([]byte, error) {
	rsp := getStreamInitMeta(msg)
	rsp.ContentType = uint32(msg.SerializationType())
	rsp.ContentEncoding = uint32(msg.CompressType())
	rspMeta := &trpcpb.TrpcStreamInitResponseMeta{}
	if e := msg.ServerRspErr(); e != nil {
		rspMeta.Ret = int32(e.Code)
		rspMeta.ErrorMsg = []byte(e.Msg)
	}
	rsp.ResponseMeta = rspMeta
	streamBuf, err := proto.Marshal(rsp)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// encodeFeedbackFrame encodes the Feedback frame.
func (s *ServerStreamCodec) encodeFeedbackFrame(frameHead *FrameHead, msg codec.Msg, reqBuf []byte) ([]byte, error) {
	feedback, ok := msg.StreamFrame().(*trpcpb.TrpcStreamFeedBackMeta)
	if !ok {
		return nil, errEncodeFeedbackFrame
	}
	streamBuf, err := proto.Marshal(feedback)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// getStreamInitMeta returns TrpcStreamInitMeta from msg.
// If not found, a new TrpcStreamInitMeta will be created and returned.
func getStreamInitMeta(msg codec.Msg) *trpcpb.TrpcStreamInitMeta {
	rsp, ok := msg.StreamFrame().(*trpcpb.TrpcStreamInitMeta)
	if !ok {
		rsp = &trpcpb.TrpcStreamInitMeta{ResponseMeta: &trpcpb.TrpcStreamInitResponseMeta{}}
	}
	return rsp
}

// Encode implements codec.Codec.
func (s *ServerStreamCodec) Encode(msg codec.Msg, reqBuf []byte) (rspbuf []byte, err error) {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok || !frameHead.isStream() {
		return nil, errUnknownFrameType
	}
	switch trpcpb.TrpcStreamFrameType(frameHead.StreamFrameType) {
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		return s.encodeInitFrame(frameHead, msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		return s.encodeDataFrame(frameHead, msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		return s.encodeCloseFrame(frameHead, msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		return s.encodeFeedbackFrame(frameHead, msg, reqBuf)
	default:
		return nil, errUnknownFrameType
	}
}

// Decode implements codec.Codec.
// It decodes the head and the stream frame data.
func (s *ServerStreamCodec) Decode(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok || !frameHead.isStream() {
		return nil, errUnknownFrameType
	}
	msg.WithStreamID(frameHead.StreamID)
	switch trpcpb.TrpcStreamFrameType(frameHead.StreamFrameType) {
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		return s.decodeInitFrame(msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		return s.decodeDataFrame(msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		return s.decodeCloseFrame(msg, reqBuf)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		return s.decodeFeedbackFrame(msg, reqBuf)
	default:
		return nil, errUnknownFrameType
	}
}

// decodeFeedbackFrame decodes the Feedback frame.
func (s *ServerStreamCodec) decodeFeedbackFrame(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	if err := s.setInitMeta(msg); err != nil {
		return nil, err
	}
	feedback := &trpcpb.TrpcStreamFeedBackMeta{}
	if err := proto.Unmarshal(reqBuf[frameHeadLen:], feedback); err != nil {
		return nil, err
	}
	msg.WithStreamFrame(feedback)
	return nil, nil
}

// setInitMeta finds the InitMeta and sets the ServerRPCName by the server handler in the InitMeta.
func (s *ServerStreamCodec) setInitMeta(msg codec.Msg) error {
	streamID := msg.StreamID()
	addr := msg.RemoteAddr()
	s.m.RLock()
	defer s.m.RUnlock()
	if streamIDToInitMeta, ok := s.initMetas[addr]; ok {
		if initMeta, ok := streamIDToInitMeta[streamID]; ok {
			rpcName := string(initMeta.GetRequestMeta().GetFunc())
			msg.WithServerRPCName(rpcName)
			msg.WithCalleeMethod(icodec.MethodFromRPCName(rpcName))
			return nil
		}
	}
	return errUninitializedMeta
}

// deleteInitMeta deletes the cached info by msg.
func (s *ServerStreamCodec) deleteInitMeta(msg codec.Msg) {
	addr := msg.RemoteAddr()
	streamID := msg.StreamID()
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.initMetas[addr], streamID)
	if len(s.initMetas[addr]) == 0 {
		delete(s.initMetas, addr)
	}
}

// decodeCloseFrame decodes the Close frame.
func (s *ServerStreamCodec) decodeCloseFrame(msg codec.Msg, rspBuf []byte) ([]byte, error) {
	if err := s.setInitMeta(msg); err != nil {
		return nil, err
	}
	close := &trpcpb.TrpcStreamCloseMeta{}
	if err := proto.Unmarshal(rspBuf[frameHeadLen:], close); err != nil {
		return nil, err
	}
	// It is considered an exception and an error should be returned to the client if:
	// 1. the CloseType is Reset
	// 2. ret code != 0
	if close.GetCloseType() == int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_RESET) || close.GetRet() != 0 {
		e := &errs.Error{
			Type: errs.ErrorTypeCalleeFramework,
			Code: trpcpb.TrpcRetCode(close.GetRet()),
			Desc: "trpc",
			Msg:  string(close.GetMsg()),
		}
		msg.WithServerRspErr(e)
	}
	msg.WithStreamFrame(close)
	return nil, nil
}

// decodeDataFrame decodes the Data frame.
func (s *ServerStreamCodec) decodeDataFrame(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	if err := s.setInitMeta(msg); err != nil {
		return nil, err
	}
	reqBody := reqBuf[frameHeadLen:]
	return reqBody, nil
}

// decodeInitFrame decodes the Init frame.
func (s *ServerStreamCodec) decodeInitFrame(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	initMeta := &trpcpb.TrpcStreamInitMeta{}
	if err := proto.Unmarshal(reqBuf[frameHeadLen:], initMeta); err != nil {
		return nil, err
	}
	s.updateMsg(msg, initMeta)
	s.storeInitMeta(msg, initMeta)
	msg.WithStreamFrame(initMeta)
	return nil, nil
}

// storeInitMeta stores the InitMeta every time when a new frame is received.
func (s *ServerStreamCodec) storeInitMeta(msg codec.Msg, initMeta *trpcpb.TrpcStreamInitMeta) {
	streamID := msg.StreamID()
	addr := msg.RemoteAddr()
	s.m.Lock()
	defer s.m.Unlock()
	if _, ok := s.initMetas[addr]; ok {
		s.initMetas[addr][streamID] = initMeta
	} else {
		t := make(map[uint32]*trpcpb.TrpcStreamInitMeta)
		t[streamID] = initMeta
		s.initMetas[addr] = t
	}
}

// updateMsg updates the Msg by InitMeta.
func (s *ServerStreamCodec) updateMsg(msg codec.Msg, initMeta *trpcpb.TrpcStreamInitMeta) {
	// get request meta
	req := initMeta.GetRequestMeta()

	// set caller service name
	msg.WithCallerServiceName(string(req.GetCaller()))
	msg.WithCalleeServiceName(string(req.GetCallee()))
	// set server handler method name
	rpcName := string(req.GetFunc())
	msg.WithServerRPCName(rpcName)
	msg.WithCalleeMethod(icodec.MethodFromRPCName(rpcName))
	// set body serialization type
	msg.WithSerializationType(int(initMeta.GetContentType()))
	// set body compression type
	msg.WithCompressType(int(initMeta.GetContentEncoding()))
	msg.WithDyeing((req.GetMessageType() & uint32(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)

	if len(req.TransInfo) > 0 {
		msg.WithServerMetaData(req.GetTransInfo())
		// set dyeing key
		if bs, ok := req.TransInfo[DyeingKey]; ok {
			msg.WithDyeingKey(string(bs))
		}
		// set environment message for transfer
		if envs, ok := req.TransInfo[EnvTransfer]; ok {
			msg.WithEnvTransfer(string(envs))
		}
	}
}

func (s *ServerStreamCodec) buildResetFrame(msg codec.Msg, frameHead *FrameHead, err error) {
	frameHead.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE)
	closeMeta := &trpcpb.TrpcStreamCloseMeta{
		CloseType: int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_RESET),
		Ret:       int32(errs.Code(err)),
		Msg:       []byte(errs.Msg(err)),
	}
	msg.WithStreamFrame(closeMeta)
}
