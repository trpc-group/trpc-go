//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package trpc_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

// TestStreamCodecInit tests stream Init frame codec.
func TestStreamCodecInit(t *testing.T) {

	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)

	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	serverCodec := codec.GetServer("trpc")
	clientCodec := codec.GetClient("trpc")

	// Client encode
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT),
		StreamID:        100,
	}
	initResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x53, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0,
		0xa, 0x41, 0xa, 0x17, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x73, 0x65, 0x72, 0x76,
		0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f, 0x74, 0x72, 0x70, 0x63,
		0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x2e,
		0x47, 0x72, 0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f}
	msg.WithFrameHead(frameHead)
	msg.WithStreamID(100)
	msg.WithCallerServiceName("trpc.app.server.service")
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	msg.WithLocalAddr(laddr)
	msg.WithRemoteAddr(raddr)
	initBuf, err := clientCodec.Encode(msg, nil)
	assert.Nil(t, err)
	assert.Equal(t, initResult, initBuf)

	// Client Encode With MetaData
	initMetaResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x63, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0,
		0x0, 0xa, 0x51, 0xa, 0x17, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x73, 0x65, 0x72,
		0x76, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f, 0x74, 0x72,
		0x70, 0x63, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72,
		0x6c, 0x64, 0x2e, 0x47, 0x72, 0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65,
		0x6c, 0x6c, 0x6f, 0x2a, 0xe, 0xa, 0x5, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x12, 0x5, 0x77, 0x6f, 0x72,
		0x6c, 0x64}
	msg.WithFrameHead(frameHead)
	msg.WithStreamID(100)
	meta := codec.MetaData{"hello": []byte("world")}
	msg.WithClientMetaData(meta)
	msg.WithCallerServiceName("trpc.app.server.service")
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	initBuf, err = clientCodec.Encode(msg, nil)
	assert.Nil(t, err)
	assert.Equal(t, initMetaResult, initBuf)

	// server Decode
	serverCtx := context.Background()
	_, serverMsg := codec.WithNewMessage(serverCtx)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	init, err := serverCodec.Decode(serverMsg, initResult)
	assert.Nil(t, err)
	assert.Nil(t, init)
	assert.Equal(t, uint32(100), serverMsg.StreamID())
	assert.NotNil(t, serverMsg.CallerService())

	// server encode
	ctx = context.Background()
	_, encodeMsg := codec.WithNewMessage(ctx)
	serverFrameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT),
		StreamID:        100,
	}
	encodeMsg.WithFrameHead(serverFrameHead)
	encodeMsg.WithStreamID(100)
	errRsp := errs.NewFrameError(errs.RetServerEncodeFail, "server test encode fail")
	encodeMsg.WithServerRspErr(errRsp)

	serverEncodeData := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x2d, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64,
		0x0, 0x0, 0x12, 0x1b, 0x8, 0x2, 0x12, 0x17, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x20, 0x74, 0x65,
		0x73, 0x74, 0x20, 0x65, 0x6e, 0x63,
		0x6f, 0x64, 0x65, 0x20, 0x66, 0x61, 0x69, 0x6c}
	rspBuf, err := serverCodec.Encode(encodeMsg, nil)
	assert.Nil(t, err)
	assert.Equal(t, serverEncodeData, rspBuf)

	// Client decode
	clientCtx := context.Background()
	_, clientMsg := codec.WithNewMessage(clientCtx)
	initRsp, err := clientCodec.Decode(clientMsg, serverEncodeData)
	assert.Nil(t, err)
	assert.Nil(t, initRsp)
	assert.Equal(t, uint32(100), clientMsg.StreamID())
	assert.NotNil(t, clientMsg.ClientRspErr())

}

// TestStreamCodecData tests stream Data frame codec.
func TestStreamCodecData(t *testing.T) {

	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)

	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)

	serverCodec := codec.GetServer("trpc")
	clientCodec := codec.GetClient("trpc")

	// init first, ensuring that init frame is saved
	initResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x59, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0xa,
		0x47, 0xa, 0x1d, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x74, 0x72, 0x70, 0x63, 0x2d, 0x67,
		0x6f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x73, 0x65,
		0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f,
		0x74, 0x72, 0x70, 0x63, 0x2e, 0x74, 0x65, 0x73, 0x74,
		0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x2e, 0x47, 0x72,
		0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f}
	// server Decode
	serverCtx := context.Background()
	_, serverMsg := codec.WithNewMessage(serverCtx)
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	init, err := serverCodec.Decode(serverMsg, initResult)
	assert.Nil(t, err)
	assert.Nil(t, init)
	assert.Equal(t, uint32(100), serverMsg.StreamID())
	assert.NotNil(t, serverMsg.CallerService())

	// client Encode
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA),
		StreamID:        100,
	}
	msg.WithFrameHead(frameHead)
	msg.WithStreamID(100)
	dataResult := []byte{0x9, 0x30, 0x1, 0x2, 0x0, 0x0, 0x0, 0x1b, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x64, 0x0, 0x0, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64}
	data := []byte("hello world")
	dataBuf, err := clientCodec.Encode(msg, data)
	assert.Nil(t, err)
	assert.Equal(t, dataBuf, dataResult)

	// Server Decode
	serverCtx = context.Background()
	_, serverMsg = codec.WithNewMessage(serverCtx)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	dataDecode, err := serverCodec.Decode(serverMsg, dataResult)
	assert.Nil(t, err)
	assert.Equal(t, dataDecode, data)
	assert.Equal(t, uint32(100), serverMsg.StreamID())

	// server Encode
	ctx = context.Background()
	_, encodeMsg := codec.WithNewMessage(ctx)
	encodeMsg.WithLocalAddr(laddr)
	encodeMsg.WithRemoteAddr(raddr)
	serverFrameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA),
		StreamID:        100,
	}
	encodeMsg.WithFrameHead(serverFrameHead)
	encodeMsg.WithStreamID(100)
	encodeData := []byte("hi there")
	serverEncodeData := []byte{0x9, 0x30, 0x1, 0x2, 0x0, 0x0, 0x0, 0x18, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x64, 0x0, 0x0, 0x68, 0x69, 0x20, 0x74, 0x68, 0x65, 0x72, 0x65}
	rspBuf, err := serverCodec.Encode(encodeMsg, encodeData)
	assert.Nil(t, err)
	assert.Equal(t, serverEncodeData, rspBuf)

	// Client Decode
	clientCtx := context.Background()
	_, clientMsg := codec.WithNewMessage(clientCtx)
	rspData, err := clientCodec.Decode(clientMsg, serverEncodeData)
	assert.Nil(t, err)
	assert.Equal(t, uint32(100), clientMsg.StreamID())
	assert.Equal(t, string(rspData), "hi there")

	// server Encode with ServerRspErr
	encodeMsg.WithServerRspErr(errs.NewFrameError(1, "test error"))
	rspBuf, err = serverCodec.Encode(encodeMsg, encodeData)
	serverEncodeData = []byte{0x9, 0x30, 0x1, 0x4, 0x0, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64,
		0x0, 0x0, 0x8, 0x1, 0x10, 0x1, 0x1a, 0xa, 0x74, 0x65, 0x73, 0x74, 0x20, 0x65, 0x72, 0x72, 0x6f, 0x72}
	assert.Nil(t, err)
	assert.Equal(t, serverEncodeData, rspBuf)
}

// TestStreamCodecClose tests stream Close frame codec.
func TestStreamCodecClose(t *testing.T) {

	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)

	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	serverCodec := codec.GetServer("trpc")
	clientCodec := codec.GetClient("trpc")

	// init first, ensuring that init frame is saved
	initResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x59, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0xa,
		0x47, 0xa, 0x1d, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x74, 0x72, 0x70, 0x63, 0x2d, 0x67,
		0x6f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x73, 0x65,
		0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f,
		0x74, 0x72, 0x70, 0x63, 0x2e, 0x74, 0x65, 0x73, 0x74,
		0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x2e, 0x47, 0x72,
		0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f}
	// server Decode
	serverCtx := context.Background()
	_, serverMsg := codec.WithNewMessage(serverCtx)
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	init, err := serverCodec.Decode(serverMsg, initResult)
	assert.Nil(t, err)
	assert.Nil(t, init)
	assert.Equal(t, uint32(100), serverMsg.StreamID())
	assert.NotNil(t, serverMsg.CallerService())

	// client encode Close
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE),
		StreamID:        100,
	}
	msg.WithFrameHead(frameHead)
	msg.WithStreamID(100)
	close := &trpcpb.TrpcStreamCloseMeta{}
	close.CloseType = int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_CLOSE)
	close.Ret = int32(0)
	msg.WithStreamFrame(close)
	closeResult := []byte{0x9, 0x30, 0x1, 0x4, 0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0}
	closeBuf, err := clientCodec.Encode(msg, nil)
	assert.Nil(t, err)
	assert.Equal(t, closeResult, closeBuf)

	// server Decode Close
	serverCtx = context.Background()
	_, serverMsg = codec.WithNewMessage(serverCtx)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	closeDecode, err := serverCodec.Decode(serverMsg, closeResult)
	assert.Nil(t, err)
	assert.Nil(t, closeDecode)
	assert.Equal(t, uint32(100), serverMsg.StreamID())

	// server encode Close
	ctx = context.Background()
	_, encodeMsg := codec.WithNewMessage(ctx)
	encodeMsg.WithLocalAddr(laddr)
	encodeMsg.WithRemoteAddr(raddr)
	serverFrameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE),
		StreamID:        100,
	}
	close = &trpcpb.TrpcStreamCloseMeta{}
	close.CloseType = int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_CLOSE)
	close.Ret = int32(0)
	encodeMsg.WithFrameHead(serverFrameHead)
	encodeMsg.WithStreamFrame(close)
	encodeMsg.WithStreamID(100)
	serverEncodeData := []byte{0x9, 0x30, 0x1, 0x4, 0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0}
	rspBuf, err := serverCodec.Encode(encodeMsg, nil)
	assert.Equal(t, serverEncodeData, rspBuf)
	assert.Nil(t, err)
	assert.Equal(t, msg.StreamID(), uint32(100))

	// Server decode error after encode close
	serverCtx = context.Background()
	_, serverMsg = codec.WithNewMessage(serverCtx)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	closeDecode, err = serverCodec.Decode(serverMsg, closeResult)
	assert.NotNil(t, err)
	assert.Nil(t, closeDecode)

	// Client decode close
	clientCtx := context.Background()
	_, clientMsg := codec.WithNewMessage(clientCtx)
	clientMsg.WithLocalAddr(laddr)
	clientMsg.WithRemoteAddr(raddr)
	CloseRsp, err := clientCodec.Decode(clientMsg, serverEncodeData)
	assert.Nil(t, err)
	assert.Nil(t, CloseRsp)
	assert.Equal(t, uint32(100), clientMsg.StreamID())
	assert.Nil(t, clientMsg.ClientRspErr())

}

// TestStreamCodecReset tests stream Reset/Close frame codec.
func TestStreamCodecReset(t *testing.T) {
	// initMeta will be deleted from cache after TestStreamCodecClose.
	// TestStreamCodecInit should be called again to recreate initMeta.
	TestStreamCodecInit(t)

	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)

	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	serverCodec := codec.GetServer("trpc")
	clientCodec := codec.GetClient("trpc")

	// Client encode Reset
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE),
		StreamID:        100,
	}
	msg.WithFrameHead(frameHead)
	msg.WithStreamID(100)
	reset := &trpcpb.TrpcStreamCloseMeta{}
	reset.CloseType = int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_RESET)
	reset.Ret = int32(1)
	reset.Msg = []byte("reset after error")
	msg.WithStreamFrame(reset)
	resetResult := []byte{0x9, 0x30, 0x1, 0x4, 0x0, 0x0, 0x0, 0x27, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0,
		0x8, 0x1, 0x10, 0x1, 0x1a, 0x11, 0x72, 0x65, 0x73, 0x65, 0x74, 0x20, 0x61, 0x66, 0x74, 0x65, 0x72, 0x20,
		0x65, 0x72, 0x72, 0x6f, 0x72}
	resetBuf, err := clientCodec.Encode(msg, nil)
	assert.Nil(t, err)
	assert.Equal(t, resetResult, resetBuf)

	// Server decode Reset
	serverCtx := context.Background()
	_, serverMsg := codec.WithNewMessage(serverCtx)
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	resetDecode, err := serverCodec.Decode(serverMsg, resetResult)
	assert.Nil(t, err)
	assert.Nil(t, resetDecode)
	assert.Equal(t, uint32(100), serverMsg.StreamID())
	assert.Nil(t, err)
	assert.NotNil(t, serverMsg.ServerRspErr())

	// server encode Close
	ctx = context.Background()
	_, encodeMsg := codec.WithNewMessage(ctx)
	encodeMsg.WithLocalAddr(laddr)
	encodeMsg.WithRemoteAddr(raddr)
	serverFrameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE),
		StreamID:        100,
	}
	reset = &trpcpb.TrpcStreamCloseMeta{}
	reset.CloseType = int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_RESET)
	reset.Ret = int32(1)
	reset.Msg = []byte("Server Side Close error")
	encodeMsg.WithFrameHead(serverFrameHead)
	encodeMsg.WithStreamFrame(reset)
	encodeMsg.WithStreamID(100)
	serverEncodeData := []byte{0x9, 0x30, 0x1, 0x4, 0x0, 0x0, 0x0, 0x2d, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x64, 0x0, 0x0, 0x8, 0x1, 0x10, 0x1, 0x1a, 0x17, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x20, 0x53, 0x69, 0x64,
		0x65, 0x20, 0x43, 0x6c, 0x6f, 0x73, 0x65, 0x20, 0x65, 0x72, 0x72, 0x6f, 0x72}
	rspBuf, err := serverCodec.Encode(encodeMsg, nil)
	assert.Equal(t, serverEncodeData, rspBuf)
	assert.Nil(t, err)

	// client Decode reset
	clientCtx := context.Background()
	_, clientMsg := codec.WithNewMessage(clientCtx)
	encodeMsg.WithLocalAddr(laddr)
	encodeMsg.WithRemoteAddr(raddr)
	resetRsp, err := clientCodec.Decode(clientMsg, serverEncodeData)
	assert.Nil(t, err)
	assert.Nil(t, resetRsp)
	assert.Equal(t, uint32(100), clientMsg.StreamID())
	assert.NotNil(t, clientMsg.ClientRspErr())

}

// TestUnknownFrameType tests stream frame with unknown type.
func TestUnknownFrameType(t *testing.T) {
	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)
	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	clientCodec := codec.GetClient("trpc")
	serverCodec := codec.GetServer("trpc")

	serverEncodeData := []byte{0x9, 0x30, 0x1, 0x7, 0x0, 0x0, 0x0, 0x2d, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x64, 0x0, 0x0, 0x12, 0x1b, 0x8, 0x2,
		0x12, 0x17, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74, 0x20, 0x65, 0x6e, 0x63,
		0x6f, 0x64, 0x65, 0x20, 0x66, 0x61, 0x69, 0x6c}
	// Client decode
	clientCtx := context.Background()
	_, clientMsg := codec.WithNewMessage(clientCtx)
	initRsp, err := clientCodec.Decode(clientMsg, serverEncodeData)
	assert.NotNil(t, err)
	assert.Nil(t, initRsp)

	// client Encode unknown frame type
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(8),
		StreamID:        100,
	}
	msg.WithFrameHead(frameHead)
	msg.WithStreamID(100)
	data := []byte("hello world")
	dataBuf, err := clientCodec.Encode(msg, data)
	// Unknown stream frame type
	assert.NotNil(t, err)
	assert.Nil(t, dataBuf)

	// server Decode  unknown Frame Type
	dataResult := []byte{0x9, 0x30, 0x1, 0x8, 0x0, 0x0, 0x0, 0x1b, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x64, 0x0, 0x0, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64}
	serverCtx := context.Background()
	_, serverMsg := codec.WithNewMessage(serverCtx)
	dataDecode, err := serverCodec.Decode(serverMsg, dataResult)
	assert.NotNil(t, err)
	assert.Nil(t, dataDecode)

	// server Encode unknown frame type
	ctx = context.Background()
	_, encodeMsg := codec.WithNewMessage(ctx)
	serverFrameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(8),
		StreamID:        100,
	}
	encodeMsg.WithFrameHead(serverFrameHead)
	encodeMsg.WithStreamID(100)
	encodeData := []byte("hi there")
	rspBuf, err := serverCodec.Encode(encodeMsg, encodeData)
	assert.NotNil(t, err)
	assert.Nil(t, rspBuf)
}

// TestFeedbackFrameType tests feedback frame type.
func TestFeedbackFrameType(t *testing.T) {
	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)
	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	clientCodec := codec.GetClient("trpc")
	serverCodec := codec.GetServer("trpc")

	// init first, ensuring that init frame is saved
	initResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x59, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0xa,
		0x47, 0xa, 0x1d, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x74, 0x72, 0x70, 0x63, 0x2d, 0x67,
		0x6f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x73, 0x65,
		0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f,
		0x74, 0x72, 0x70, 0x63, 0x2e, 0x74, 0x65, 0x73, 0x74,
		0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x2e, 0x47, 0x72,
		0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f}
	// server Decode
	serverCtx := context.Background()
	_, serverMsg := codec.WithNewMessage(serverCtx)
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	init, err := serverCodec.Decode(serverMsg, initResult)
	assert.Nil(t, err)
	assert.Nil(t, init)
	assert.Equal(t, uint32(100), serverMsg.StreamID())
	assert.NotNil(t, serverMsg.CallerService())

	encodeData := []byte{0x9, 0x30, 0x1, 0x3, 0x0, 0x0, 0x0, 0x13, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0x8, 0x90, 0x4e}
	// Client decode feedback
	clientCtx := context.Background()
	_, clientMsg := codec.WithNewMessage(clientCtx)
	res, err := clientCodec.Decode(clientMsg, encodeData)
	assert.Nil(t, err)
	assert.Nil(t, res)
	feedback, ok := clientMsg.StreamFrame().(*trpcpb.TrpcStreamFeedBackMeta)
	assert.True(t, ok)
	assert.Equal(t, uint32(10000), feedback.WindowSizeIncrement)

	// client Encode feedback type
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK),
		StreamID:        100,
	}
	msg.WithFrameHead(frameHead)
	msg.WithStreamID(100)
	var data []byte
	feedbackMeta := &trpcpb.TrpcStreamFeedBackMeta{}
	msg.WithStreamFrame(feedbackMeta)
	feedbackMeta.WindowSizeIncrement = 10000
	dataBuf, err := clientCodec.Encode(msg, data)
	assert.Nil(t, err)
	assert.Equal(t, encodeData, dataBuf)

	// server Decode  feedback frame
	serverCtx = context.Background()
	_, serverMsg = codec.WithNewMessage(serverCtx)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	dataDecode, err := serverCodec.Decode(serverMsg, encodeData)
	assert.Nil(t, dataDecode)
	assert.Nil(t, err)
	feedback, ok = clientMsg.StreamFrame().(*trpcpb.TrpcStreamFeedBackMeta)
	assert.True(t, ok)
	assert.Equal(t, uint32(10000), feedback.WindowSizeIncrement)

	// server Encode feedbackframe
	ctx = context.Background()
	_, encodeMsg := codec.WithNewMessage(ctx)
	serverFrameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK),
		StreamID:        100,
	}
	encodeMsg.WithFrameHead(serverFrameHead)
	encodeMsg.WithStreamID(100)
	feedbackMeta = &trpcpb.TrpcStreamFeedBackMeta{}
	encodeMsg.WithStreamFrame(feedbackMeta)
	feedbackMeta.WindowSizeIncrement = 10000
	rspBuf, err := serverCodec.Encode(encodeMsg, nil)
	assert.Nil(t, err)
	assert.Equal(t, rspBuf, encodeData)

}

// TestDecodeEncodeFail tests codec error.
func TestDecodeEncodeFail(t *testing.T) {
	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	clientCodec := codec.GetClient("trpc")
	serverCodec := codec.GetServer("trpc")

	// init first, ensuring that init frame is saved
	initResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x59, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0xa,
		0x47, 0xa, 0x1d, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x74, 0x72, 0x70, 0x63, 0x2d, 0x67,
		0x6f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x73, 0x65,
		0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f,
		0x74, 0x72, 0x70, 0x63, 0x2e, 0x74, 0x65, 0x73, 0x74,
		0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x2e, 0x47, 0x72,
		0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f}
	// server Decode
	serverCtx := context.Background()
	_, serverMsg := codec.WithNewMessage(serverCtx)
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	init, err := serverCodec.Decode(serverMsg, initResult)
	assert.Nil(t, err)
	assert.Nil(t, init)
	assert.Equal(t, uint32(100), serverMsg.StreamID())
	assert.NotNil(t, serverMsg.CallerService())

	encodeData := []byte{0x9, 0x30, 0x1, 0x3, 0x0, 0x0, 0x0, 0x01, 0x0, 0x10,
		0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0x8, 0x90, 0x4e}
	// client decode total length error
	clientCtx := context.Background()
	_, clientMsg := codec.WithNewMessage(clientCtx)
	rsp, err := clientCodec.Decode(clientMsg, encodeData)
	assert.Nil(t, rsp)
	assert.NotNil(t, err)

	// client init unmarshal error
	serverInitData := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x2d, 0x0, 0x0, 0x0, 0x0, 0x1, 0x64,
		0x0, 0x0, 0x12, 0x1b, 0x90, 0x2, 0x12, 0x17, 0x73, 0x65, 0x54, 0x76, 0x65, 0x72, 0x20, 0x74, 0x65,
		0x73, 0x74, 0x20, 0x61, 0x6e, 0x62,
		0x6f, 0x64, 0x65, 0x20, 0x66, 0x61, 0x69, 0x6c}
	initRsp, err := clientCodec.Decode(clientMsg, serverInitData)
	assert.Nil(t, initRsp)
	assert.NotNil(t, err)

	t.Run("ClientStreamCodec unknown stream frame type error from message", func(t *testing.T) {
		cc := trpc.NewClientStreamCodec()

		_, err := cc.Encode(trpc.Message(context.Background()), nil)
		require.Contains(t, err.Error(), "unknown stream frame type")

		_, err = cc.Decode(trpc.Message(context.Background()), nil)
		require.Contains(t, err.Error(), "unknown stream frame type")
	})
	t.Run("ServerStreamCodec unknown stream frame type error from message", func(t *testing.T) {
		cc := trpc.NewServerStreamCodec()

		_, err := cc.Encode(trpc.Message(context.Background()), nil)
		require.Contains(t, err.Error(), "unknown stream frame type")

		_, err = cc.Decode(trpc.Message(context.Background()), nil)
		require.Contains(t, err.Error(), "unknown stream frame type")
	})

}

// TestEncodeWithMetadata tests encode with metadata.
func TestEncodeWithMetadata(t *testing.T) {
	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)
	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	serverCodec := codec.GetServer("trpc")
	clientCodec := codec.GetClient("trpc")
	// Client encode
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT),
		StreamID:        100,
	}
	initResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x61, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0,
		0xa, 0x4f, 0xa, 0x17, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x73, 0x65, 0x72, 0x76,
		0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f, 0x74, 0x72, 0x70, 0x63,
		0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x2e,
		0x47, 0x72, 0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x2a,
		0xc, 0xa, 0x3, 0x6b, 0x65, 0x79, 0x12, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65}
	msg.WithFrameHead(frameHead)
	msg.WithClientMetaData(codec.MetaData{"key": []byte("value")})
	msg.WithStreamID(100)
	msg.WithCallerServiceName("trpc.app.server.service")
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	initBuf, err := clientCodec.Encode(msg, nil)
	assert.Nil(t, err)
	assert.Equal(t, initResult, initBuf)

	// Server Decode
	serverCtx, serverMsg := codec.WithNewMessage(context.Background())
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	initRsp, err := serverCodec.Decode(serverMsg, initResult)
	assert.Nil(t, err)
	assert.Nil(t, initRsp)
	assert.Equal(t, trpc.GetMetaData(serverCtx, "key"), []byte("value"))
}

// TestEncodeWithDyeing tests encode with dyeing key.
func TestEncodeWithDyeing(t *testing.T) {
	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)
	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	serverCodec := codec.GetServer("trpc")
	clientCodec := codec.GetClient("trpc")
	// Client encode
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT),
		StreamID:        100,
	}
	initResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x74, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0,
		0xa, 0x62, 0xa, 0x17, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x73, 0x65, 0x72, 0x76,
		0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f, 0x74, 0x72, 0x70, 0x63,
		0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x2e,
		0x47, 0x72, 0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20,
		0x1, 0x2a, 0x1d, 0xa, 0xf, 0x74, 0x72, 0x70, 0x63, 0x2d, 0x64, 0x79, 0x65, 0x69, 0x6e, 0x67, 0x2d, 0x6b,
		0x65, 0x79, 0x12, 0xa, 0x64, 0x79, 0x65, 0x69, 0x6e, 0x67, 0x2d, 0x6b, 0x65, 0x79}
	msg.WithFrameHead(frameHead)
	msg.WithDyeing(true)
	msg.WithDyeingKey("dyeing-key")
	msg.WithStreamID(100)
	msg.WithCallerServiceName("trpc.app.server.service")
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	initBuf, err := clientCodec.Encode(msg, nil)
	assert.Nil(t, err)
	assert.Equal(t, initResult, initBuf)

	// Server Decode
	serverCtx, serverMsg := codec.WithNewMessage(context.Background())
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	initRsp, err := serverCodec.Decode(serverMsg, initResult)
	assert.Nil(t, err)
	assert.Nil(t, initRsp)
	assert.Equal(t, trpc.GetMetaData(serverCtx, trpc.DyeingKey), []byte("dyeing-key"))
}

// TestEncodeWithEnvTransfer tests encode with envtransfor.
func TestEncodeWithEnvTransfer(t *testing.T) {
	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)
	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)
	serverCodec := codec.GetServer("trpc")
	clientCodec := codec.GetClient("trpc")
	// Client encode
	frameHead := &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT),
		StreamID:        100,
	}
	initResult := []byte{0x9, 0x30, 0x1, 0x1, 0x0, 0x0, 0x0, 0x6a, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0,
		0xa, 0x58, 0xa, 0x17, 0x74, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x73, 0x65, 0x72, 0x76,
		0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x26, 0x2f, 0x74, 0x72, 0x70, 0x63,
		0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x2e,
		0x47, 0x72, 0x65, 0x65, 0x74, 0x65, 0x72, 0x2f, 0x53, 0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x2a,
		0x15, 0xa, 0x8, 0x74, 0x72, 0x70, 0x63, 0x2d, 0x65, 0x6e, 0x76, 0x12, 0x9, 0x65, 0x6e, 0x76, 0x2d,
		0x74, 0x72, 0x61, 0x6e, 0x73}
	msg.WithFrameHead(frameHead)
	msg.WithEnvTransfer("env-trans")
	msg.WithStreamID(100)
	msg.WithCallerServiceName("trpc.app.server.service")
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	initBuf, err := clientCodec.Encode(msg, nil)
	assert.Nil(t, err)
	assert.Equal(t, initResult, initBuf)

	// Server Decode
	serverCtx, serverMsg := codec.WithNewMessage(context.Background())
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	assert.Nil(t, err)
	raddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:10001")
	assert.Nil(t, err)
	serverMsg.WithLocalAddr(laddr)
	serverMsg.WithRemoteAddr(raddr)
	initRsp, err := serverCodec.Decode(serverMsg, initResult)
	assert.Nil(t, err)
	assert.Nil(t, initRsp)
	assert.Equal(t, trpc.GetMetaData(serverCtx, trpc.EnvTransfer), []byte("env-trans"))
}
