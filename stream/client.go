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

// Package stream contains streaming client and server APIs.
package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	iatomic "trpc.group/trpc-go/trpc-go/internal/atomic"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	"trpc.group/trpc-go/trpc-go/internal/queue"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
)

// Client is the Streaming client interface, NewStream is its only method.
type Client interface {
	// NewStream returns a client stream, which the user uses to call Recv and Send to send,
	// receive and send streaming messages.
	NewStream(ctx context.Context, desc *client.ClientStreamDesc,
		method string, opt ...client.Option) (client.ClientStream, error)
}

// DefaultStreamClient is the default streaming client.
var DefaultStreamClient = NewStreamClient()

// NewStreamClient returns a streaming client.
func NewStreamClient() Client {
	// Streaming ID from 0-99 is reserved ID, used as control ID.
	return &streamClient{streamID: uint32(99)}
}

// an implementation of streamClient Client.
type streamClient struct {
	streamID uint32
}

// ClientStreamDesc client stream description.
//
// Deprecated: The architecture is adjusted to the client's package
type ClientStreamDesc = client.ClientStreamDesc

// ClientStream client streaming interface.
//
// Deprecated: The architecture is adjusted to the client's package
type ClientStream = client.ClientStream

// The specific implementation of ClientStream.
type clientStream struct {
	desc           *client.ClientStreamDesc
	method         string
	sc             *streamClient
	ctx            context.Context
	opts           *client.Options
	streamID       uint32
	stream         client.Stream
	recvQueue      *queue.Queue[*response]
	closed         uint32
	closeCh        chan struct{}
	closeOnce      sync.Once
	isServerClosed iatomic.Bool
}

// NewStream creates a new stream through which users send and receive messages.
func (c *streamClient) NewStream(ctx context.Context, desc *client.ClientStreamDesc,
	method string, opt ...client.Option) (client.ClientStream, error) {
	stream, err := c.newStream(ctx, desc, method, opt...)
	if err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientStreamInitErr, "new stream")
	}
	return stream, nil
}

// newStream creates a new stream through which users send and receive messages.
func (c *streamClient) newStream(
	ctx context.Context,
	desc *client.ClientStreamDesc,
	method string,
	opt ...client.Option,
) (_ client.ClientStream, err error) {
	ctx, msg := codec.EnsureMessage(ctx)
	// Note: This span only records the creation of a client stream.
	// It does not capture any subsequent events of sending/receiving messages.
	var (
		span  rpcz.Span
		ender rpcz.Ender
	)
	if rpczenable.Enabled {
		span, ender, ctx = rpcz.NewSpanContext(ctx, "new client stream")
		defer func() {
			if err == nil {
				span.SetAttribute(rpcz.TRPCAttributeError, msg.ClientRspErr())
			} else {
				span.SetAttribute(rpcz.TRPCAttributeError, err)
			}
			ender.End()
		}()
	}
	cs := &clientStream{
		desc:      desc,
		method:    method,
		sc:        c,
		streamID:  atomic.AddUint32(&c.streamID, 1),
		ctx:       ctx,
		closeCh:   make(chan struct{}, 1),
		recvQueue: queue.New[*response](ctx.Done()),
		stream:    client.NewStream(),
	}
	if rpczenable.Enabled {
		span.SetAttribute(rpcz.TRPCAttributeRPCName, method)
		span.SetAttribute(rpcz.TRPCAttributeStreamID, cs.streamID)
	}
	if err := cs.prepare(opt...); err != nil {
		return nil, fmt.Errorf("client stream (method = %s, streamID = %d) prepare error: %w",
			method, cs.streamID, err)
	}
	if rpczenable.Enabled {
		span.SetAttribute(rpcz.TRPCAttributeFilterNames, cs.opts.FilterNames)
	}
	if cs.opts.StreamFilters != nil {
		return cs.opts.StreamFilters.Filter(cs.ctx, cs.desc, cs.invoke)
	}
	return cs.invoke(cs.ctx, cs.desc)
}

// Context returns the Context of the current stream.
func (cs *clientStream) Context() context.Context {
	return cs.ctx
}

// RecvMsg receives the message, if there is no message it will get stuck.
// RecvMsg and SendMsg are concurrency safe, but two RecvMsg are not concurrency safe.
func (cs *clientStream) RecvMsg(m interface{}) error {
	if err := cs.recv(m); err != nil {
		return err
	}
	if cs.desc.ServerStreams {
		// Subsequent messages should be received by subsequent RecvMsg calls.
		return nil
	}
	// Special handling for non-server-stream rpcs.
	// This recv expects EOF or errors.
	err := cs.recv(m)
	if err == nil {
		return errs.NewFrameError(errs.RetClientStreamReadEnd,
			"client streaming protocol violation: get <nil>, want <EOF>")
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func (cs *clientStream) recv(m interface{}) error {
	resp, ok := cs.recvQueue.Get()
	if !ok {
		return cs.dealContextDone()
	}
	if resp.err != nil {
		return resp.err
	}
	// Gather flow control information.
	if err := cs.recvFlowCtl(len(resp.data)); err != nil {
		return err
	}

	serializationType := codec.Message(cs.ctx).SerializationType()
	if icodec.IsValidSerializationType(cs.opts.CurrentSerializationType) {
		serializationType = cs.opts.CurrentSerializationType
	}
	if err := codec.Unmarshal(serializationType, resp.data, m); err != nil {
		return errs.NewFrameError(errs.RetClientDecodeFail, "client codec Unmarshal: "+err.Error())
	}
	return nil
}

func (cs *clientStream) recvFlowCtl(n int) error {
	if cs.opts.RControl == nil {
		return nil
	}
	// If the bottom layer has received the Close frame, then no feedback frame will be returned.
	if atomic.LoadUint32(&cs.closed) == 1 {
		return nil
	}
	if err := cs.opts.RControl.OnRecv(uint32(n)); err != nil {
		// Avoid receiving the Close frame after entering OnRecv, and make another judgment.
		if atomic.LoadUint32(&cs.closed) == 1 {
			return nil
		}
		return err
	}
	return nil
}

// dealContextDone returns the final error message according to the Error type of the context.
func (cs *clientStream) dealContextDone() error {
	if cs.ctx.Err() == context.Canceled {
		return errs.NewFrameError(errs.RetClientCanceled, "tcp client stream canceled before recv: "+cs.ctx.Err().Error())
	}
	if cs.ctx.Err() == context.DeadlineExceeded {
		return errs.NewFrameError(errs.RetClientTimeout,
			"tcp client stream canceled timeout before recv: "+cs.ctx.Err().Error())
	}
	return nil
}

// SendMsg is the specific implementation of sending a message.
// RecvMsg and SendMsg are concurrency safe, but two SendMsg are not concurrency safe.
func (cs *clientStream) SendMsg(m interface{}) error {
	ctx, msg := codec.WithCloneContextAndMessage(cs.ctx)
	defer codec.PutBackMessage(msg)
	msg.WithFrameHead(newFrameHead(trpc.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA, cs.streamID))
	msg.WithStreamID(cs.streamID)
	msg.WithClientRPCName(cs.method)
	msg.WithCompressType(codec.Message(cs.ctx).CompressType())
	return cs.stream.Send(ctx, m)
}

func newFrameHead(t trpc.TrpcStreamFrameType, id uint32) *trpc.FrameHead {
	return &trpc.FrameHead{
		FrameType:       uint8(trpc.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(t),
		StreamID:        id,
	}
}

// CloseSend normally closes the sender, no longer sends messages, only accepts messages.
func (cs *clientStream) CloseSend() error {
	return cs.closeSend(&trpc.TrpcStreamCloseMeta{CloseType: int32(trpc.TrpcStreamCloseType_TRPC_STREAM_CLOSE)})
}

func (cs *clientStream) closeSend(closeMeta *trpc.TrpcStreamCloseMeta) error {
	ctx, msg := codec.WithCloneContextAndMessage(cs.ctx)
	defer codec.PutBackMessage(msg)
	msg.WithFrameHead(newFrameHead(trpc.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE, cs.streamID))
	msg.WithStreamID(cs.streamID)
	msg.WithStreamFrame(closeMeta)
	return cs.stream.Send(ctx, nil)
}

func (cs *clientStream) prepare(opt ...client.Option) error {
	msg := codec.Message(cs.ctx)
	msg.WithClientRPCName(cs.method)
	msg.WithStreamID(cs.streamID)

	opts, err := cs.stream.Init(cs.ctx, opt...)
	if err != nil {
		return err
	}
	cs.opts = opts
	return nil
}

func (cs *clientStream) invoke(ctx context.Context, _ *client.ClientStreamDesc) (_ client.ClientStream, err error) {
	var (
		span  rpcz.Span
		ender rpcz.Ender
	)
	if rpczenable.Enabled {
		span, ender, ctx = rpcz.NewSpanContext(ctx, "client stream invoke")
		defer func() {
			span.SetAttribute(rpcz.TRPCAttributeError, err)
			ender.End()
		}()
	}
	// Create the underlying connection with a new context to prevent the
	// connection from being closed directly when the context is canceled.
	if err := cs.stream.Invoke(trpc.CloneContext(ctx)); err != nil {
		return nil, fmt.Errorf("client stream (method = %s, streamID = %d) invoke error: %w",
			cs.method, cs.streamID, err)
	}
	w := getWindowSize(cs.opts.MaxWindowSize)
	newCtx, newMsg := codec.WithCloneContextAndMessage(ctx)
	defer codec.PutBackMessage(newMsg)
	copyMetaData(newMsg, codec.Message(cs.ctx))
	newMsg.WithFrameHead(newFrameHead(trpc.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT, cs.streamID))
	newMsg.WithClientRPCName(cs.method)
	newMsg.WithStreamID(cs.streamID)
	newMsg.WithCompressType(codec.Message(cs.ctx).CompressType())
	newMsg.WithStreamFrame(&trpc.TrpcStreamInitMeta{
		RequestMeta:    &trpc.TrpcStreamInitRequestMeta{},
		InitWindowSize: w,
	})
	cs.opts.RControl = newReceiveControl(w, cs.feedback)
	// Send the init message out.
	if err := cs.stream.Send(newCtx, nil); err != nil {
		return nil, fmt.Errorf("client stream (method = %s, streamID = %d) send error: %w",
			cs.method, cs.streamID, err)
	}
	// After init is sent, the server will return directly.
	if _, err := cs.stream.Recv(newCtx); err != nil {
		return nil, fmt.Errorf("client stream (method = %s, streamID = %d) recv error: %w",
			cs.method, cs.streamID, err)
	}
	initRspMeta, ok := newMsg.StreamFrame().(*trpc.TrpcStreamInitMeta)
	if !ok {
		return nil, fmt.Errorf("client stream (method = %s, streamID = %d) recv "+
			"unexpected frame type: %T, expected: %T",
			cs.method, cs.streamID, newMsg.StreamFrame(), (*trpc.TrpcStreamInitMeta)(nil))
	}
	initWindowSize := initRspMeta.GetInitWindowSize()
	cs.configSendControl(initWindowSize)

	// Start the dispatch goroutine loop to send packets.
	go cs.dispatch()
	return cs, nil
}

// configSendControl configs Send Control according to initWindowSize.
func (cs *clientStream) configSendControl(initWindowSize uint32) {
	if initWindowSize == 0 {
		// Disable flow control, compatible with the server without flow control enabled, delete this logic later.
		cs.opts.RControl = nil
		cs.opts.SControl = nil
		return
	}
	cs.opts.SControl = newSendControl(initWindowSize, cs.ctx.Done(), cs.closeCh)
}

// feedback send feedback frame.
func (cs *clientStream) feedback(i uint32) error {
	ctx, msg := codec.WithCloneContextAndMessage(cs.ctx)
	defer codec.PutBackMessage(msg)
	msg.WithFrameHead(newFrameHead(trpc.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK, cs.streamID))
	msg.WithStreamID(cs.streamID)
	msg.WithClientRPCName(cs.method)
	msg.WithStreamFrame(&trpc.TrpcStreamFeedBackMeta{WindowSizeIncrement: i})
	return cs.stream.Send(ctx, nil)
}

// handleFrame performs different logical processing according to the type of frame.
func (cs *clientStream) handleFrame(ctx context.Context, resp *response,
	respData []byte, frameHead *trpc.FrameHead) error {
	msg := codec.Message(ctx)
	switch trpc.TrpcStreamFrameType(frameHead.StreamFrameType) {
	case trpc.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		// Get the data and return it to the client.
		resp.data = respData
		resp.err = nil
		cs.recvQueue.Put(resp)
		return nil
	case trpc.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		// Close, it should be judged as Reset or Close.
		resp.data = nil
		var err error
		if msg.ClientRspErr() != nil {
			// Description is Reset.
			err = msg.ClientRspErr()
		} else {
			err = io.EOF
		}
		resp.err = err
		cs.recvQueue.Put(resp)
		cs.isServerClosed.Store(true)
		return err
	case trpc.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		cs.handleFeedback(msg)
		return nil
	default:
		return nil
	}
}

// handleFeedback handles the feedback frame.
func (cs *clientStream) handleFeedback(msg codec.Msg) {
	if feedbackFrame, ok := msg.StreamFrame().(*trpc.TrpcStreamFeedBackMeta); ok && cs.opts.SControl != nil {
		cs.opts.SControl.UpdateWindow(feedbackFrame.WindowSizeIncrement)
	}
}

// dispatch is used to distribute the received data packets, receive them in a loop,
// and then distribute the data packets according to different data types.
func (cs *clientStream) dispatch() {
	go func() {
		select {
		case <-cs.ctx.Done():
			cs.close()
		case <-cs.closeCh:
		}
	}()
	defer cs.close()
	for {
		ctx, msg := codec.WithCloneContextAndMessage(cs.ctx)
		msg.WithCompressType(codec.Message(cs.ctx).CompressType())
		msg.WithStreamID(cs.streamID)
		respData, err := cs.stream.Recv(ctx)
		if err != nil {
			// return to client on error.
			if err == io.EOF {
				err = errs.WrapFrameError(err, errs.RetClientStreamReadEnd, streamClosed)
			}
			cs.recvQueue.Put(&response{
				err: err,
			})
			return
		}

		frameHead, ok := msg.FrameHead().(*trpc.FrameHead)
		if !ok {
			cs.recvQueue.Put(&response{
				err: errors.New(frameHeadInvalid),
			})
			return
		}

		if err := cs.handleFrame(ctx, &response{}, respData, frameHead); err != nil {
			// If there is a Close frame, the dispatch goroutine ends.
			return
		}
	}
}

func (cs *clientStream) close() {
	cs.closeOnce.Do(func() {
		if !cs.isServerClosed.Load() {
			if err := cs.closeSend(&trpc.TrpcStreamCloseMeta{
				CloseType: int32(trpc.TrpcStreamCloseType_TRPC_STREAM_RESET),
				Ret:       int32(errs.RetClientCanceled),
				Msg:       []byte("client has already canceled"),
			}); err != nil {
				log.Error("client stream close send failed", err)
			}
		}
		cs.opts.StreamTransport.Close(cs.ctx)
		atomic.StoreUint32(&cs.closed, 1)
		close(cs.closeCh)
	})
}

func copyMetaData(dst codec.Msg, src codec.Msg) {
	if src.ClientMetaData() != nil {
		dst.WithClientMetaData(src.ClientMetaData().Clone())
	}
}
