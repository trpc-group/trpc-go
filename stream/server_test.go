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

package stream_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/stream"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"

	"github.com/stretchr/testify/assert"
)

type fakeStreamHandle struct {
}

// StreamHandleFunc Mock StreamHandleFunc method
func (fs *fakeStreamHandle) StreamHandleFunc(ctx context.Context, sh server.StreamHandler, req []byte) ([]byte, error) {
	return nil, nil
}

// Init Mock Init method
func (fs *fakeStreamHandle) Init(opts *server.Options) {
	return
}

type fakeServerTransport struct{}

type fakeServerCodec struct{}

// Send Mock Send method
func (s *fakeServerTransport) Send(ctx context.Context, rspBuf []byte) error {
	if string(rspBuf) == "init-error" {
		return errors.New("init-error")
	}
	return nil
}

// Close Mock Close method
func (s *fakeServerTransport) Close(ctx context.Context) {
	return
}

// ListenAndServe Mock ListenAndServe method
func (s *fakeServerTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) error {

	return nil
}

// Decode Mock codec Decode method
func (c *fakeServerCodec) Decode(msg codec.Msg, reqBuf []byte) (reqBody []byte, err error) {
	return reqBuf, nil
}

// Encode Mock codec Encode method
func (c *fakeServerCodec) Encode(msg codec.Msg, rspBody []byte) (rspBuf []byte, err error) {
	rsp := string(rspBody)
	if rsp == "encode-error" {
		return nil, errors.New("server encode response fail")
	}
	if msg.StreamID() < uint32(100) {
		return nil, errors.New("streamID less than 100")
	}
	if msg.StreamID() == uint32(101) {
		return []byte("init-error"), nil
	}
	return rspBody, nil
}

func streamHandler(stream server.Stream) error {
	time.Sleep(time.Second)
	return nil
}

func errorStreamHandler(stream server.Stream) error {
	return errors.New("handle fail")
}

type fakeAddr struct {
}

// Network method of Network Mock net.Addr interface
func (f *fakeAddr) Network() string {
	return "tcp"
}

// String method of String Mock net.Addr interface
func (f *fakeAddr) String() string {
	return "127.0.0.01:67891"
}

// TestStreamDispatcherHandleInit Test Stream Dispatcher
func TestStreamDispatcherHandleInit(t *testing.T) {
	codec.Register("fake", &fakeServerCodec{}, nil)

	si := &server.StreamServerInfo{}
	dispatcher := stream.NewStreamDispatcher()
	assert.Equal(t, dispatcher, stream.DefaultStreamDispatcher)

	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)
	// StreamHandleFunc msg not nil
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	rsp, err := dispatcher.StreamHandleFunc(ctx, streamHandler, si, nil)
	assert.Nil(t, rsp)
	assert.Contains(t, err.Error(), "frameHead is not contained in msg")
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{})
	// StreamHandleFunc handle init
	fh := &trpc.FrameHead{}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	msg.WithRemoteAddr(&fakeAddr{})
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	// StreamHandleFunc handle init with codec encode error
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(99))
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err.Error(), "streamID less than 100")

	// StreamHandleFunc handle init send error
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(101))
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("init-error"))
	assert.Nil(t, rsp)
	assert.Contains(t, err.Error(), "init-error")

	// StreamHandleFun handle data to validate streamID was stored
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("data"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	// StreamHandleFunc handle error
	msg.WithFrameHead(fh)
	msg.WithStreamID(100)
	rsp, err = dispatcher.StreamHandleFunc(ctx, errorStreamHandler, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)
	time.Sleep(100 * time.Millisecond)
}

// TestStreamDispatcherHandleData test StreamDispatcher Handle data
func TestStreamDispatcherHandleData(t *testing.T) {
	codec.Register("fake", &fakeServerCodec{}, nil)

	si := &server.StreamServerInfo{}
	dispatcher := stream.NewStreamDispatcher()
	assert.Equal(t, dispatcher, stream.DefaultStreamDispatcher)

	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)

	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	fh := &trpc.FrameHead{}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{})
	addr := &fakeAddr{}
	msg.WithRemoteAddr(addr)
	rsp, err := dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	// handleData normal
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("data"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	// handleData error no such addr
	msg.WithRemoteAddr(nil)
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("data"))
	assert.Nil(t, rsp)
	assert.Contains(t, err.Error(), "no such addr")

	// handle data error no such stream id
	msg.WithRemoteAddr(addr)
	msg.WithStreamID(uint32(101))
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("data"))
	assert.Nil(t, rsp)
	assert.Contains(t, err.Error(), "no such stream ID")
}

// TestStreamDispatcherHandleClose test handles Close frame
func TestStreamDispatcherHandleClose(t *testing.T) {

	codec.Register("fake", &fakeServerCodec{}, nil)

	si := &server.StreamServerInfo{}
	dispatcher := stream.NewStreamDispatcher()
	assert.Equal(t, dispatcher, stream.DefaultStreamDispatcher)

	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)

	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	fh := &trpc.FrameHead{}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{})

	addr := &fakeAddr{}
	msg.WithRemoteAddr(addr)
	msg.WithFrameHead(fh)
	rsp, err := dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	// handle close normal
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("close"))
	assert.Nil(t, rsp)
	assert.Equal(t, errs.ErrServerNoResponse, err)

	// handle close no such addr
	msg.WithFrameHead(fh)
	msg.WithRemoteAddr(nil)
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("close"))
	assert.Nil(t, rsp)
	assert.Equal(t, errs.ErrServerNoResponse, err)

	// handle close server rsp err
	msg.WithRemoteAddr(addr)
	msg.WithFrameHead(fh)
	msg.WithServerRspErr(errors.New("server rsp error"))
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("close"))
	assert.Nil(t, rsp)
	assert.Equal(t, errs.ErrServerNoResponse, err)

	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK)
	msg.WithFrameHead(fh)
	msg.WithStreamFrame(&trpcpb.TrpcStreamFeedBackMeta{})
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("feedback"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	fh.StreamFrameType = uint8(8)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, streamHandler, si, []byte("unknown"))
	assert.Nil(t, rsp)
	assert.Contains(t, err.Error(), "unknown frame type")
}

// TestServerStreamSendMsg test server receives messages
func TestServerStreamSendMsg(t *testing.T) {
	codec.Register("fake", &fakeServerCodec{}, nil)

	si := &server.StreamServerInfo{}
	dispatcher := stream.NewStreamDispatcher()
	assert.Equal(t, dispatcher, stream.DefaultStreamDispatcher)

	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)

	// StreamHandleFunc msg not nil
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	fh := &trpc.FrameHead{}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	msg.WithRemoteAddr(&fakeAddr{})
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{})

	opts.CurrentCompressType = codec.CompressTypeNoop
	opts.CurrentSerializationType = codec.SerializationTypeNoop

	sh := func(ss server.Stream) error {
		ctx = ss.Context()
		assert.NotNil(t, ctx)
		err := ss.SendMsg(&codec.Body{Data: []byte("init")})
		assert.Nil(t, err)
		return err
	}
	rsp, err := dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)
	time.Sleep(100 * time.Millisecond)

	opts.CurrentCompressType = 5
	opts.CurrentSerializationType = codec.SerializationTypeNoop
	sh = func(ss server.Stream) error {
		ctx = ss.Context()
		assert.NotNil(t, ctx)
		err := ss.SendMsg(&codec.Body{Data: []byte("init")})
		assert.NotNil(t, err)
		return err
	}
	dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	time.Sleep(200 * time.Millisecond)

	opts.CurrentCompressType = codec.CompressTypeNoop
	opts.CurrentSerializationType = codec.SerializationTypeNoop
	sh = func(ss server.Stream) error {
		ctx = ss.Context()
		assert.NotNil(t, ctx)
		err := ss.SendMsg(&codec.Body{Data: []byte("encode-error")})
		assert.Contains(t, err.Error(), "server codec Encode")
		return err
	}
	dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	time.Sleep(200 * time.Millisecond)

	opts.CurrentCompressType = codec.CompressTypeNoop
	opts.CurrentSerializationType = codec.SerializationTypeNoop
	sh = func(ss server.Stream) error {
		ctx = ss.Context()
		assert.NotNil(t, ctx)
		err := ss.SendMsg(&codec.Body{Data: []byte("init-error")})
		return err
	}
	dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	time.Sleep(200 * time.Millisecond)
}

// TestServerStreamRecvMsg test receive message
func TestServerStreamRecvMsg(t *testing.T) {
	codec.Register("fake", &fakeServerCodec{}, nil)

	si := &server.StreamServerInfo{}
	dispatcher := stream.NewStreamDispatcher()
	assert.Equal(t, dispatcher, stream.DefaultStreamDispatcher)

	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)

	// StreamHandleFunc msg not nil
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	fh := &trpc.FrameHead{}
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	msg.WithRemoteAddr(&fakeAddr{})
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{})
	opts.CurrentCompressType = codec.CompressTypeNoop
	opts.CurrentSerializationType = codec.SerializationTypeNoop

	sh := func(ss server.Stream) error {
		ctx := ss.Context()
		assert.NotNil(t, ctx)
		body := &codec.Body{}
		err := ss.RecvMsg(body)
		assert.Nil(t, err)
		assert.Equal(t, string(body.Data), "data")
		err = ss.RecvMsg(body)
		assert.Equal(t, err, io.EOF)

		err = ss.RecvMsg(body)
		assert.Equal(t, err, io.EOF)
		return err
	}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	rsp, err := dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)
	// handleData normal
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, sh, si, []byte("data"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, sh, si, []byte("close"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	time.Sleep(300 * time.Millisecond)
}

// TestServerStreamRecvMsgFail test for failure to receive data
func TestServerStreamRecvMsgFail(t *testing.T) {
	codec.Register("fake", &fakeServerCodec{}, nil)
	si := &server.StreamServerInfo{}
	dispatcher := stream.NewStreamDispatcher()
	assert.Equal(t, dispatcher, stream.DefaultStreamDispatcher)
	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)

	// StreamHandleFunc msg not nil
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	fh := &trpc.FrameHead{}
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	msg.WithRemoteAddr(&fakeAddr{})
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{})

	opts.CurrentCompressType = codec.CompressTypeGzip
	opts.CurrentSerializationType = codec.SerializationTypeNoop

	sh := func(ss server.Stream) error {
		ctx := ss.Context()
		assert.NotNil(t, ctx)
		body := &codec.Body{}
		err := ss.RecvMsg(body)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "server codec Decompress")

		err = ss.RecvMsg(body)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "server codec Unmarshal")
		return err
	}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	msg.WithFrameHead(fh)
	rsp, err := dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)
	// handleData normal
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, sh, si, []byte("data"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)
}

// TesthandleError test server error condition
func TestHandleError(t *testing.T) {
	codec.Register("fake", &fakeServerCodec{}, nil)
	si := &server.StreamServerInfo{}
	dispatcher := stream.NewStreamDispatcher()
	assert.Equal(t, dispatcher, stream.DefaultStreamDispatcher)
	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)

	// StreamHandleFunc msg not nil
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	fh := &trpc.FrameHead{}
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	msg.WithRemoteAddr(&fakeAddr{})
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{})

	opts.CurrentCompressType = codec.CompressTypeGzip
	opts.CurrentSerializationType = codec.SerializationTypeNoop

	sh := func(ss server.Stream) error {
		ctx := ss.Context()
		assert.NotNil(t, ctx)
		body := &codec.Body{}
		err := ss.RecvMsg(body)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "Connection is closed")
		return err
	}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	rsp, err := dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)
	// handleError
	msg.WithFrameHead(nil)
	msg.WithServerRspErr(errors.New("Connection is closed"))

	noopSh := func(ss server.Stream) error {
		return nil
	}
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, noopSh, si, nil)
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)
	time.Sleep(100 * time.Millisecond)
}

// TestStreamDispatcherHandleFeedback test handles feedback frame
func TestStreamDispatcherHandleFeedback(t *testing.T) {

	codec.Register("fake", &fakeServerCodec{}, nil)
	si := &server.StreamServerInfo{}

	dispatcher := stream.NewStreamDispatcher()
	assert.Equal(t, dispatcher, stream.DefaultStreamDispatcher)

	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)

	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	fh := &trpc.FrameHead{}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{InitWindowSize: 10})

	sh := func(ss server.Stream) error {
		time.Sleep(time.Second)
		return nil
	}

	addr := &fakeAddr{}
	msg.WithRemoteAddr(addr)
	rsp, err := dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	// handle feedback get server stream fail
	msg.WithRemoteAddr(nil)
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, nil, si, []byte("feedback"))
	assert.Nil(t, rsp)
	assert.NotNil(t, err)

	// handle feedback invalid stream
	msg.WithRemoteAddr(addr)
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK)
	msg.WithFrameHead(fh)
	rsp, err = dispatcher.StreamHandleFunc(ctx, nil, si, []byte("feedback"))
	assert.Nil(t, rsp)
	assert.NotNil(t, err)

	// normal feedback
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK)
	msg.WithFrameHead(fh)
	msg.WithStreamFrame(&trpcpb.TrpcStreamFeedBackMeta{WindowSizeIncrement: 1000})
	rsp, err = dispatcher.StreamHandleFunc(ctx, nil, si, []byte("feedback"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)
}

// TestServerFlowControl tests the situation of server-side flow control
func TestServerFlowControl(t *testing.T) {
	codec.Register("fake", &fakeServerCodec{}, nil)
	si := &server.StreamServerInfo{}
	dispatcher := stream.NewStreamDispatcher()
	// Init test
	opts := &server.Options{}
	ft := &fakeServerTransport{}
	opts.Transport = ft
	opts.Codec = codec.GetServer("fake")
	err := dispatcher.Init(opts)
	assert.Nil(t, err)
	assert.Equal(t, opts.Transport, opts.StreamTransport)
	// StreamHandleFunc msg not nil
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	fh := &trpc.FrameHead{}
	msg.WithFrameHead(fh)
	msg.WithStreamID(uint32(100))
	addr := &fakeAddr{}
	msg.WithRemoteAddr(addr)
	msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{InitWindowSize: 65535})
	opts.CurrentCompressType = codec.CompressTypeNoop
	opts.CurrentSerializationType = codec.SerializationTypeNoop
	var wg sync.WaitGroup
	wg.Add(1)
	sh := func(ss server.Stream) error {
		defer wg.Done()
		for i := 0; i < 20000; i++ {
			body := &codec.Body{}
			err := ss.RecvMsg(body)
			assert.Nil(t, err)
			assert.Equal(t, string(body.Data), "data")
		}
		return nil
	}
	fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
	rsp, err := dispatcher.StreamHandleFunc(ctx, sh, si, []byte("init"))
	assert.Nil(t, rsp)
	assert.Equal(t, err, errs.ErrServerNoResponse)

	// handleData normal
	for i := 0; i < 20000; i++ {
		newCtx := context.Background()
		newCtx, newMsg := codec.WithNewMessage(newCtx)
		newMsg.WithStreamID(uint32(100))
		newMsg.WithRemoteAddr(addr)
		newFh := &trpc.FrameHead{}
		newFh.StreamID = uint32(100)
		newFh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
		newMsg.WithFrameHead(newFh)
		rsp, err := dispatcher.StreamHandleFunc(newCtx, sh, si, []byte("data"))
		assert.Nil(t, rsp)
		assert.Equal(t, err, errs.ErrServerNoResponse)
	}
	wg.Wait()
}

func TestClientStreamFlowControl(t *testing.T) {
	svrOpts := []server.Option{server.WithAddress("127.0.0.1:30210")}
	handle := func(s server.Stream) error {
		req := getBytes(1024)
		for i := 0; i < 1000; i++ {
			err := s.RecvMsg(req)
			assert.Nil(t, err)
		}
		err := s.RecvMsg(req)
		assert.Equal(t, io.EOF, err)

		rsp := getBytes(1024)
		copy(rsp.Data, req.Data)
		for i := 0; i < 1000; i++ {
			err = s.SendMsg(rsp)
			assert.Nil(t, err)
		}
		return nil
	}
	svr := startStreamServer(handle, svrOpts)
	defer closeStreamServer(svr)

	cliOpts := []client.Option{client.WithTarget("ip://127.0.0.1:30210")}
	cliStream, err := getClientStream(context.Background(), bidiDesc, cliOpts)
	assert.Nil(t, err)

	req := getBytes(1024)
	rand.Read(req.Data)
	for i := 0; i < 1000; i++ {
		err = cliStream.SendMsg(req)
		assert.Nil(t, err)
	}
	err = cliStream.CloseSend()
	assert.Nil(t, err)
	rsp := getBytes(1024)
	for i := 0; i < 1000; i++ {
		err = cliStream.RecvMsg(rsp)
		assert.Nil(t, err)
		assert.Equal(t, req, rsp)
	}
	err = cliStream.RecvMsg(rsp)
	assert.Equal(t, io.EOF, err)
}

func TestServerStreamFlowControl(t *testing.T) {
	svrOpts := []server.Option{server.WithAddress("127.0.0.1:30211")}
	handle := func(s server.Stream) error {
		req := getBytes(1024)
		err := s.RecvMsg(req)
		assert.Nil(t, err)

		rsp := getBytes(1024)
		copy(rsp.Data, req.Data)
		for i := 0; i < 1000; i++ {
			err := s.SendMsg(rsp)
			assert.Nil(t, err)
		}
		return nil
	}
	svr := startStreamServer(handle, svrOpts)
	defer closeStreamServer(svr)

	cliOpts := []client.Option{client.WithTarget("ip://127.0.0.1:30211")}
	cliStream, err := getClientStream(context.Background(), bidiDesc, cliOpts)
	assert.Nil(t, err)

	req := getBytes(1024)
	rand.Read(req.Data)
	err = cliStream.SendMsg(req)
	assert.Nil(t, err)
	err = cliStream.CloseSend()
	assert.Nil(t, err)
	rsp := getBytes(1024)
	for i := 0; i < 1000; i++ {
		err = cliStream.RecvMsg(rsp)
		assert.Nil(t, err)
		assert.Equal(t, req, rsp)
	}
	err = cliStream.RecvMsg(rsp)
	assert.Equal(t, err, io.EOF)
}

func startStreamServer(handle func(server.Stream) error, opts []server.Option) server.Service {
	svrOpts := []server.Option{
		server.WithProtocol("trpc"),
		server.WithNetwork("tcp"),
		server.WithStreamTransport(transport.NewServerStreamTransport(transport.WithReusePort(true))),
		server.WithTransport(transport.NewServerStreamTransport(transport.WithReusePort(true))),
		// The server must actively set the serialization method
		server.WithCurrentSerializationType(codec.SerializationTypeNoop),
	}
	svrOpts = append(svrOpts, opts...)
	svr := server.New(svrOpts...)
	register(svr, handle)
	go func() {
		err := svr.Serve()
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(100 * time.Millisecond)
	return svr
}

func closeStreamServer(svr server.Service) {
	ch := make(chan struct{}, 1)
	svr.Close(ch)
	<-ch
}

var (
	clientDesc = &client.ClientStreamDesc{
		StreamName:    "streamTest",
		ClientStreams: true,
		ServerStreams: false,
	}
	serverDesc = &client.ClientStreamDesc{
		StreamName:    "streamTest",
		ClientStreams: false,
		ServerStreams: true,
	}
	bidiDesc = &client.ClientStreamDesc{
		StreamName:    "streamTest",
		ClientStreams: true,
		ServerStreams: true,
	}
)

func getClientStream(ctx context.Context, desc *client.ClientStreamDesc, opts []client.Option) (client.ClientStream, error) {
	cli := stream.NewStreamClient()
	method := "/trpc.test.stream.Greeter/StreamSayHello"
	cliOpts := []client.Option{
		client.WithProtocol("trpc"),
		client.WithTransport(transport.NewClientTransport()),
		client.WithStreamTransport(transport.NewClientStreamTransport()),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
	}
	cliOpts = append(cliOpts, opts...)
	return cli.NewStream(ctx, desc, method, cliOpts...)
}

func register(s server.Service, f func(server.Stream) error) {
	svr := &greeterServiceImpl{f: f}
	if err := s.Register(&GreeterServer_ServiceDesc, svr); err != nil {
		panic(fmt.Sprintf("Greeter register error: %v", err))
	}
}

type greeterServiceImpl struct {
	f func(server.Stream) error
}

func (s *greeterServiceImpl) BidiStreamSayHello(stream server.Stream) error {
	return s.f(stream)
}

func GreeterService_BidiStreamSayHello_Handler(srv interface{}, stream server.Stream) error {
	return srv.(GreeterService).BidiStreamSayHello(stream)
}

type GreeterService interface {
	// BidiStreamSayHello Bidi streaming
	BidiStreamSayHello(server.Stream) error
}

var GreeterServer_ServiceDesc = server.ServiceDesc{
	ServiceName:  "trpc.test.stream.Greeter",
	HandlerType:  (*GreeterService)(nil),
	StreamHandle: stream.NewStreamDispatcher(),
	Streams: []server.StreamDesc{
		{
			StreamName:    "/trpc.test.stream.Greeter/StreamSayHello",
			Handler:       GreeterService_BidiStreamSayHello_Handler,
			ServerStreams: true,
		},
	},
}

func getBytes(size int) *codec.Body {
	return &codec.Body{Data: make([]byte, size)}
}

/* --------------- Filter Unit Test -------------*/

type wrappedServerStream struct {
	server.Stream
}

func newWrappedServerStream(s server.Stream) server.Stream {
	return &wrappedServerStream{s}
}

func (w *wrappedServerStream) RecvMsg(m interface{}) error {
	err := w.Stream.RecvMsg(m)
	num := binary.LittleEndian.Uint64(m.(*codec.Body).Data[:8])
	binary.LittleEndian.PutUint64(m.(*codec.Body).Data[:8], num+1)
	return err
}

func (w *wrappedServerStream) SendMsg(m interface{}) error {
	num := binary.LittleEndian.Uint64(m.(*codec.Body).Data[:8])
	binary.LittleEndian.PutUint64(m.(*codec.Body).Data[:8], num+1)
	return w.Stream.SendMsg(m)
}

var (
	testKey1 = "hello"
	testKey2 = "ping"
	testData = map[string][]byte{
		testKey1: []byte("world"),
		testKey2: []byte("pong"),
	}
)

func serverFilterAdd1(ss server.Stream, si *server.StreamServerInfo,
	handler server.StreamHandler) error {
	msg := trpc.Message(ss.Context())
	meta := msg.ServerMetaData()
	if v, ok := meta[testKey1]; !ok {
		return errors.New("meta not exist")
	} else if !bytes.Equal(v, testData[testKey1]) {
		return errors.New("meta not match")
	}
	err := handler(newWrappedServerStream(ss))
	return err
}

func serverFilterAdd2(ss server.Stream, si *server.StreamServerInfo,
	handler server.StreamHandler) error {
	msg := trpc.Message(ss.Context())
	meta := msg.ServerMetaData()
	if v, ok := meta[testKey2]; !ok {
		return errors.New("meta not exist")
	} else if !bytes.Equal(v, testData[testKey2]) {
		return errors.New("meta not match")
	}
	err := handler(newWrappedServerStream(ss))
	return err
}
