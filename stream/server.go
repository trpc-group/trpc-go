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

package stream

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"go.uber.org/atomic"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	"trpc.group/trpc-go/trpc-go/internal/queue"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"
)

// serverStream is a structure provided to the service implementation logic,
// and users use the API of this structure to send and receive streaming messages.
type serverStream struct {
	ctx       context.Context
	streamID  uint32
	opts      *server.Options
	recvQueue *queue.Queue[*response]
	done      chan struct{}
	err       atomic.Error // Carry the server tcp failure information.
	once      sync.Once
	rControl  *receiveControl // Receiver flow control.
	sControl  *sendControl    // Sender flow control.
}

// SendMsg is the API that users use to send streaming messages.
func (s *serverStream) SendMsg(m interface{}) error {
	if err := s.err.Load(); err != nil {
		return errs.WrapFrameError(err, errs.Code(err), "stream sending error")
	}
	msg := codec.Message(s.ctx)
	ctx, newMsg := codec.WithCloneContextAndMessage(s.ctx)
	defer codec.PutBackMessage(newMsg)
	newMsg.WithLocalAddr(msg.LocalAddr())
	newMsg.WithRemoteAddr(msg.RemoteAddr())
	newMsg.WithCompressType(msg.CompressType())
	newMsg.WithStreamID(s.streamID)
	// Refer to the pb code generated by trpc.proto, common to each language, automatically generated code.
	newMsg.WithFrameHead(newFrameHead(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA, s.streamID))

	var (
		err           error
		reqBodyBuffer []byte
	)
	serializationType, compressType := s.serializationAndCompressType(newMsg)
	if icodec.IsValidSerializationType(serializationType) {
		reqBodyBuffer, err = codec.Marshal(serializationType, m)
		if err != nil {
			return errs.NewFrameError(errs.RetServerEncodeFail, "server codec Marshal: "+err.Error())
		}
	}

	// compress
	if icodec.IsValidCompressType(compressType) && compressType != codec.CompressTypeNoop {
		reqBodyBuffer, err = codec.Compress(compressType, reqBodyBuffer)
		if err != nil {
			return errs.NewFrameError(errs.RetServerEncodeFail, "server codec Compress: "+err.Error())
		}
	}

	// Flow control only controls the payload of data.
	if s.sControl != nil {
		if err := s.sControl.GetWindow(uint32(len(reqBodyBuffer))); err != nil {
			return err
		}
	}

	// encode the entire request.
	reqBuffer, err := s.opts.Codec.Encode(newMsg, reqBodyBuffer)
	if err != nil {
		return errs.NewFrameError(errs.RetServerEncodeFail, "server codec Encode: "+err.Error())
	}

	// initiate a backend network request.
	return s.opts.StreamTransport.Send(ctx, reqBuffer)
}

func (s *serverStream) newFrameHead(streamFrameType trpcpb.TrpcStreamFrameType) *trpc.FrameHead {
	return &trpc.FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME),
		StreamFrameType: uint8(streamFrameType),
		StreamID:        s.streamID,
	}
}

func (s *serverStream) serializationAndCompressType(msg codec.Msg) (int, int) {
	serializationType := msg.SerializationType()
	compressType := msg.CompressType()
	if icodec.IsValidSerializationType(s.opts.CurrentSerializationType) {
		serializationType = s.opts.CurrentSerializationType
	}
	if icodec.IsValidCompressType(s.opts.CurrentCompressType) {
		compressType = s.opts.CurrentCompressType
	}
	return serializationType, compressType
}

// RecvMsg receives streaming messages, passes in the structure that needs to receive messages,
// and returns the serialized structure.
func (s *serverStream) RecvMsg(m interface{}) error {
	resp, ok := s.recvQueue.Get()
	if !ok {
		if err := s.err.Load(); err != nil {
			return err
		}
		return errs.NewFrameError(errs.RetServerSystemErr, streamClosed)
	}
	if resp.err != nil {
		return resp.err
	}
	if s.rControl != nil {
		if err := s.rControl.OnRecv(uint32(len(resp.data))); err != nil {
			return err
		}
	}
	// Decompress and deserialize the data frame into a structure.
	return s.decompressAndUnmarshal(resp.data, m)

}

// decompressAndUnmarshal decompresses the data frame and deserializes it.
func (s *serverStream) decompressAndUnmarshal(data []byte, m interface{}) error {
	msg := codec.Message(s.ctx)
	var err error
	serializationType, compressType := s.serializationAndCompressType(msg)
	if icodec.IsValidCompressType(compressType) && compressType != codec.CompressTypeNoop {
		data, err = codec.Decompress(compressType, data)
		if err != nil {
			return errs.NewFrameError(errs.RetClientDecodeFail, "server codec Decompress: "+err.Error())
		}
	}

	// Deserialize the binary body to a specific body structure.
	if icodec.IsValidSerializationType(serializationType) {
		if err := codec.Unmarshal(serializationType, data, m); err != nil {
			return errs.NewFrameError(errs.RetClientDecodeFail, "server codec Unmarshal: "+err.Error())
		}
	}
	return nil
}

// The CloseSend server closes the stream, where ret represents the close type,
// which is divided into TRPC_STREAM_CLOSE and TRPC_STREAM_RESET.
// message represents the returned message, where error messages can be logged.
func (s *serverStream) CloseSend(closeType, ret int32, message string) error {
	oldMsg := codec.Message(s.ctx)
	ctx, msg := codec.WithCloneContextAndMessage(s.ctx)
	defer codec.PutBackMessage(msg)
	msg.WithLocalAddr(oldMsg.LocalAddr())
	msg.WithRemoteAddr(oldMsg.RemoteAddr())
	msg.WithFrameHead(newFrameHead(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE, s.streamID))
	msg.WithStreamFrame(&trpcpb.TrpcStreamCloseMeta{
		CloseType: closeType,
		Ret:       ret,
		Msg:       []byte(message),
	})

	rspBuffer, err := s.opts.Codec.Encode(msg, nil)
	if err != nil {
		return err
	}
	return s.opts.StreamTransport.Send(ctx, rspBuffer)
}

// newServerStream creates a new server stream, which can send and receive streaming messages.
func newServerStream(ctx context.Context, streamID uint32, opts *server.Options) *serverStream {
	s := &serverStream{
		ctx:      ctx,
		opts:     opts,
		streamID: streamID,
		done:     make(chan struct{}, 1),
	}
	s.recvQueue = queue.New[*response](s.done)
	return s
}

func (s *serverStream) feedback(w uint32) error {
	oldMsg := codec.Message(s.ctx)
	ctx, msg := codec.WithCloneContextAndMessage(s.ctx)
	defer codec.PutBackMessage(msg)
	msg.WithLocalAddr(oldMsg.LocalAddr())
	msg.WithRemoteAddr(oldMsg.RemoteAddr())
	msg.WithStreamID(s.streamID)
	msg.WithFrameHead(newFrameHead(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK, s.streamID))
	msg.WithStreamFrame(&trpcpb.TrpcStreamFeedBackMeta{WindowSizeIncrement: w})

	feedbackBuf, err := s.opts.Codec.Encode(msg, nil)
	if err != nil {
		return err
	}
	return s.opts.StreamTransport.Send(ctx, feedbackBuf)
}

// Context returns the context of the serverStream structure.
func (s *serverStream) Context() context.Context {
	return s.ctx
}

// The structure of streamDispatcher is used to distribute streaming data.
type streamDispatcher struct {
	m                      sync.RWMutex
	streamIDToServerStream map[net.Addr]map[uint32]*serverStream
	opts                   *server.Options
}

// DefaultStreamDispatcher is the default implementation of the trpc dispatcher,
// supports the data distribution of the trpc streaming protocol.
var DefaultStreamDispatcher = NewStreamDispatcher()

// NewStreamDispatcher returns a new dispatcher.
func NewStreamDispatcher() server.StreamHandle {
	return &streamDispatcher{
		streamIDToServerStream: make(map[net.Addr]map[uint32]*serverStream),
	}
}

// storeServerStream msg contains the socket address of the client connection,
// there are multiple streams under each socket address, and map it to serverStream
// again according to the id of the stream.
func (sd *streamDispatcher) storeServerStream(addr net.Addr, streamID uint32, ss *serverStream) {
	sd.m.Lock()
	defer sd.m.Unlock()
	if addrToStreamID, ok := sd.streamIDToServerStream[addr]; !ok {
		// Does not exist, indicating that a new connection is coming, re-create the structure.
		sd.streamIDToServerStream[addr] = map[uint32]*serverStream{streamID: ss}
	} else {
		addrToStreamID[streamID] = ss
	}
}

// deleteServerStream deletes the serverStream from cache.
func (sd *streamDispatcher) deleteServerStream(addr net.Addr, streamID uint32) {
	sd.m.Lock()
	defer sd.m.Unlock()
	if addrToStreamID, ok := sd.streamIDToServerStream[addr]; ok {
		if _, ok = addrToStreamID[streamID]; ok {
			delete(addrToStreamID, streamID)
		}
		if len(addrToStreamID) == 0 {
			delete(sd.streamIDToServerStream, addr)
		}
	}
}

// loadServerStream loads the stored serverStream through the socket address
// of the client connection and the id of the stream.
func (sd *streamDispatcher) loadServerStream(addr net.Addr, streamID uint32) (*serverStream, error) {
	sd.m.RLock()
	defer sd.m.RUnlock()
	addrToStream, ok := sd.streamIDToServerStream[addr]
	if !ok || addr == nil {
		return nil, errs.NewFrameError(errs.RetServerSystemErr, noSuchAddr)
	}

	var ss *serverStream
	if ss, ok = addrToStream[streamID]; !ok {
		return nil, errs.NewFrameError(errs.RetServerSystemErr, noSuchStreamID)
	}
	return ss, nil
}

// Init initializes some settings of dispatcher.
func (sd *streamDispatcher) Init(opts *server.Options) error {
	sd.opts = opts
	st, ok := sd.opts.Transport.(transport.ServerStreamTransport)
	if !ok {
		return errors.New(streamTransportUnimplemented)
	}
	sd.opts.StreamTransport = st
	sd.opts.ServeOptions = append(sd.opts.ServeOptions,
		transport.WithServerAsync(false), transport.WithCopyFrame(true))
	return nil
}

// startStreamHandler is used to start the goroutine, execute streamHandler,
// streamHandler is implemented for the specific streaming server.
func (sd *streamDispatcher) startStreamHandler(addr net.Addr, streamID uint32,
	ss *serverStream, si *server.StreamServerInfo, sh server.StreamHandler) {
	defer func() {
		sd.deleteServerStream(addr, streamID)
		ss.once.Do(func() { close(ss.done) })
	}()

	// Execute the implementation code of the server stream.
	var err error
	if ss.opts.StreamFilters != nil {
		err = ss.opts.StreamFilters.Filter(ss, si, sh)
	} else {
		err = sh(ss)
	}

	var frameworkError *errs.Error
	switch {
	case errors.As(err, &frameworkError):
		err = ss.CloseSend(int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_RESET), int32(frameworkError.Code), frameworkError.Msg)
	case err != nil:
		// return business error.
		err = ss.CloseSend(int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_RESET), 0, err.Error())
	default:
		// Stream is normally closed.
		err = ss.CloseSend(int32(trpcpb.TrpcStreamCloseType_TRPC_STREAM_CLOSE), 0, "")
	}
	if err != nil {
		ss.err.Store(err)
		log.Trace(closeSendFail, err)
	}
}

// setSendControl obtained from the init frame.
func (s *serverStream) setSendControl(msg codec.Msg) (uint32, error) {
	initMeta, ok := msg.StreamFrame().(*trpcpb.TrpcStreamInitMeta)
	if !ok {
		return 0, errors.New(streamFrameInvalid)
	}

	// This section of logic is compatible with framework implementations in other languages
	// that do not enable flow control, and will be deleted later.
	if initMeta.InitWindowSize == 0 {
		// Compatible with the client without flow control enabled.
		s.rControl = nil
		s.sControl = nil
		return initMeta.InitWindowSize, nil
	}
	s.sControl = newSendControl(initMeta.InitWindowSize, s.done)
	return initMeta.InitWindowSize, nil
}

// handleInit processes the sent init package.
func (sd *streamDispatcher) handleInit(ctx context.Context,
	sh server.StreamHandler, si *server.StreamServerInfo) ([]byte, error) {
	// The Msg in ctx is passed to us by the upper layer, and we can't make any assumptions about its life cycle.
	// Before creating ServerStream, make a complete copy of Msg.
	oldMsg := codec.Message(ctx)
	ctx, msg := codec.WithNewMessage(ctx)
	codec.CopyMsg(msg, oldMsg)

	streamID := msg.StreamID()
	ss := newServerStream(ctx, streamID, sd.opts)
	w := getWindowSize(sd.opts.MaxWindowSize)
	ss.rControl = newReceiveControl(w, ss.feedback)
	sd.storeServerStream(msg.RemoteAddr(), streamID, ss)

	cw, err := ss.setSendControl(msg)
	if err != nil {
		return nil, err
	}

	// send init response packet.
	newCtx, newMsg := codec.WithCloneContextAndMessage(ctx)
	defer codec.PutBackMessage(newMsg)
	newMsg.WithLocalAddr(msg.LocalAddr())
	newMsg.WithRemoteAddr(msg.RemoteAddr())
	newMsg.WithStreamID(streamID)
	newMsg.WithFrameHead(newFrameHead(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT, ss.streamID))

	initMeta := &trpcpb.TrpcStreamInitMeta{ResponseMeta: &trpcpb.TrpcStreamInitResponseMeta{}}
	// If the client does not set it, the server should not set it to prevent incompatibility.
	if cw == 0 {
		initMeta.InitWindowSize = 0
	} else {
		initMeta.InitWindowSize = w
	}
	newMsg.WithStreamFrame(initMeta)

	rspBuffer, err := ss.opts.Codec.Encode(newMsg, nil)
	if err != nil {
		return nil, err
	}
	if err := ss.opts.StreamTransport.Send(newCtx, rspBuffer); err != nil {
		return nil, err
	}

	// Initiate a goroutine to execute specific business logic.
	go sd.startStreamHandler(msg.RemoteAddr(), streamID, ss, si, sh)
	return nil, errs.ErrServerNoResponse
}

// handleData handles data messages.
func (sd *streamDispatcher) handleData(msg codec.Msg, req []byte) ([]byte, error) {
	ss, err := sd.loadServerStream(msg.RemoteAddr(), msg.StreamID())
	if err != nil {
		return nil, err
	}
	ss.recvQueue.Put(&response{data: req})
	return nil, errs.ErrServerNoResponse
}

// handleClose handles the Close message.
func (sd *streamDispatcher) handleClose(msg codec.Msg) ([]byte, error) {
	ss, err := sd.loadServerStream(msg.RemoteAddr(), msg.StreamID())
	if err != nil {
		// The server has sent the Close frame.
		// Since the timing of the Close frame is unpredictable, when the server receives the Close frame from the client,
		// the Close frame may have been sent, causing the resource to be released, no need to respond to this error.
		log.Trace("handleClose loadServerStream fail", err)
		return nil, errs.ErrServerNoResponse
	}
	// is Reset message.
	if msg.ServerRspErr() != nil {
		ss.recvQueue.Put(&response{err: msg.ServerRspErr()})
		return nil, errs.ErrServerNoResponse
	}
	// is a normal Close message
	ss.recvQueue.Put(&response{err: io.EOF})
	return nil, errs.ErrServerNoResponse
}

// handleError When the connection is wrong, handle the error.
func (sd *streamDispatcher) handleError(msg codec.Msg) ([]byte, error) {
	sd.m.Lock()
	defer sd.m.Unlock()

	addr := msg.RemoteAddr()
	addrToStream, ok := sd.streamIDToServerStream[addr]
	if !ok || addr == nil {
		return nil, errs.NewFrameError(errs.RetServerSystemErr, noSuchAddr)
	}
	for streamID, ss := range addrToStream {
		ss.err.Store(msg.ServerRspErr())
		ss.once.Do(func() { close(ss.done) })
		delete(addrToStream, streamID)
	}
	delete(sd.streamIDToServerStream, addr)
	return nil, errs.ErrServerNoResponse
}

// StreamHandleFunc The processing logic after a complete streaming frame received by the streaming transport.
func (sd *streamDispatcher) StreamHandleFunc(ctx context.Context,
	sh server.StreamHandler, si *server.StreamServerInfo, req []byte) ([]byte, error) {
	msg := codec.Message(ctx)
	frameHead, ok := msg.FrameHead().(*trpc.FrameHead)
	if !ok {
		// If there is no frame head and serverRspErr, the server connection is abnormal
		// and returns to the upper service.
		if msg.ServerRspErr() != nil {
			return sd.handleError(msg)
		}
		return nil, errs.NewFrameError(errs.RetServerSystemErr, frameHeadNotInMsg)
	}
	msg.WithFrameHead(nil)
	return sd.handleByStreamFrameType(ctx, trpcpb.TrpcStreamFrameType(frameHead.StreamFrameType), sh, si, req)
}

// handleFeedback handles the feedback frame.
func (sd *streamDispatcher) handleFeedback(msg codec.Msg) ([]byte, error) {
	ss, err := sd.loadServerStream(msg.RemoteAddr(), msg.StreamID())
	if err != nil {
		return nil, err
	}
	fb, ok := msg.StreamFrame().(*trpcpb.TrpcStreamFeedBackMeta)
	if !ok {
		return nil, errors.New(streamFrameInvalid)
	}
	if ss.sControl != nil {
		ss.sControl.UpdateWindow(fb.WindowSizeIncrement)
	}
	return nil, errs.ErrServerNoResponse
}

// handleByStreamFrameType performs different logic processing according to the type of stream frame.
func (sd *streamDispatcher) handleByStreamFrameType(ctx context.Context, streamFrameType trpcpb.TrpcStreamFrameType,
	sh server.StreamHandler, si *server.StreamServerInfo, req []byte) ([]byte, error) {
	msg := codec.Message(ctx)
	switch streamFrameType {
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		return sd.handleInit(ctx, sh, si)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		return sd.handleData(msg, req)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		return sd.handleClose(msg)
	case trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		return sd.handleFeedback(msg)
	default:
		return nil, errs.NewFrameError(errs.RetServerSystemErr, unknownFrameType)
	}
}
