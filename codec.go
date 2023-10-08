// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package trpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"sync/atomic"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	"trpc.group/trpc-go/trpc-go/transport"

	"google.golang.org/protobuf/proto"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

func init() {
	codec.Register(ProtocolName, DefaultServerCodec, DefaultClientCodec)
	transport.RegisterFramerBuilder(ProtocolName, DefaultFramerBuilder)
}

// default codec
var (
	DefaultServerCodec = &ServerCodec{
		streamCodec: NewServerStreamCodec(),
	}
	DefaultClientCodec = &ClientCodec{
		streamCodec:   NewClientStreamCodec(),
		defaultCaller: fmt.Sprintf("trpc.client.%s.service", path.Base(os.Args[0])),
	}
	DefaultFramerBuilder = &FramerBuilder{}

	// DefaultMaxFrameSize is the default max size of frame including attachment,
	// which can be modified if size of the packet is bigger than this.
	DefaultMaxFrameSize = 10 * 1024 * 1024
)

var (
	errHeadOverflowsUint16       = errors.New("head len overflows uint16")
	errHeadOverflowsUint32       = errors.New("total len overflows uint32")
	errAttachmentOverflowsUint32 = errors.New("attachment len overflows uint32")
)

type errFrameTooLarge struct {
	maxFrameSize int
}

// Error implements the error interface and returns the description of the errFrameTooLarge.
func (e *errFrameTooLarge) Error() string {
	return fmt.Sprintf("frame len is larger than MaxFrameSize(%d)", e.maxFrameSize)
}

// frequently used const variables
const (
	DyeingKey   = "trpc-dyeing-key" // dyeing key
	UserIP      = "trpc-user-ip"    // user ip
	EnvTransfer = "trpc-env"        // env info

	ProtocolName = "trpc" // protocol name
)

// trpc protocol codec
const (
	// frame head format：
	// v0：
	// 2 bytes magic + 1 byte frame type + 1 byte stream frame type + 4 bytes total len
	// + 2 bytes pb header len + 4 bytes stream id + 2 bytes reserved
	// v1：
	// 2 bytes magic + 1 byte frame type + 1 byte stream frame type + 4 bytes total len
	// + 2 bytes pb header len + 4 bytes stream id + 1 byte protocol version + 1 byte reserved
	frameHeadLen       = uint16(16)       // total length of frame head: 16 bytes
	protocolVersion0   = uint8(0)         // v0
	protocolVersion1   = uint8(1)         // v1
	curProtocolVersion = protocolVersion1 // current protocol version
)

// FrameHead is head of the trpc frame.
type FrameHead struct {
	FrameType       uint8  // type of the frame
	StreamFrameType uint8  // type of the stream frame
	TotalLen        uint32 // total length
	HeaderLen       uint16 // header's length
	StreamID        uint32 // stream id for streaming rpc, request id for unary rpc
	ProtocolVersion uint8  // version of protocol
	FrameReserved   uint8  // reserved bits for further development
}

func newDefaultUnaryFrameHead() *FrameHead {
	return &FrameHead{
		FrameType:       uint8(trpcpb.TrpcDataFrameType_TRPC_UNARY_FRAME), // default unary
		ProtocolVersion: curProtocolVersion,
	}
}

// extract extracts field values of the FrameHead from the buffer.
func (h *FrameHead) extract(buf []byte) {
	h.FrameType = buf[2]
	h.StreamFrameType = buf[3]
	h.TotalLen = binary.BigEndian.Uint32(buf[4:8])
	h.HeaderLen = binary.BigEndian.Uint16(buf[8:10])
	h.StreamID = binary.BigEndian.Uint32(buf[10:14])
	h.ProtocolVersion = buf[14]
	h.FrameReserved = buf[15]
}

// construct constructs bytes data for the whole frame.
func (h *FrameHead) construct(header, body, attachment []byte) ([]byte, error) {
	headerLen := len(header)
	if headerLen > math.MaxUint16 {
		return nil, errHeadOverflowsUint16
	}
	attachmentLen := int64(len(attachment))
	if attachmentLen > math.MaxUint32 {
		return nil, errAttachmentOverflowsUint32
	}
	totalLen := int64(frameHeadLen) + int64(headerLen) + int64(len(body)) + attachmentLen
	if totalLen > int64(DefaultMaxFrameSize) {
		return nil, &errFrameTooLarge{maxFrameSize: DefaultMaxFrameSize}
	}
	if totalLen > math.MaxUint32 {
		return nil, errHeadOverflowsUint32
	}

	// construct the buffer
	buf := make([]byte, totalLen)
	binary.BigEndian.PutUint16(buf[:2], uint16(trpcpb.TrpcMagic_TRPC_MAGIC_VALUE))
	buf[2] = h.FrameType
	buf[3] = h.StreamFrameType
	binary.BigEndian.PutUint32(buf[4:8], uint32(totalLen))
	binary.BigEndian.PutUint16(buf[8:10], uint16(headerLen))
	binary.BigEndian.PutUint32(buf[10:14], h.StreamID)
	buf[14] = h.ProtocolVersion
	buf[15] = h.FrameReserved

	frameHeadLen := int(frameHeadLen)
	copy(buf[frameHeadLen:frameHeadLen+headerLen], header)
	copy(buf[frameHeadLen+headerLen:frameHeadLen+headerLen+len(body)], body)
	copy(buf[frameHeadLen+headerLen+len(body):], attachment)
	return buf, nil
}

func (h *FrameHead) isStream() bool {
	return trpcpb.TrpcDataFrameType(h.FrameType) == trpcpb.TrpcDataFrameType_TRPC_STREAM_FRAME
}

func (h *FrameHead) isUnary() bool {
	return trpcpb.TrpcDataFrameType(h.FrameType) == trpcpb.TrpcDataFrameType_TRPC_UNARY_FRAME
}

// upgradeProtocol upgrades protocol and sets stream id and request id.
// For compatibility, server should respond the same protocol version as that of the request.
// and client should always send request with the latest protocol version.
func (h *FrameHead) upgradeProtocol(protocolVersion uint8, requestID uint32) {
	h.ProtocolVersion = protocolVersion
	h.StreamID = requestID
}

// FramerBuilder is an implementation of codec.FramerBuilder.
// Used for trpc protocol.
type FramerBuilder struct{}

// New implements codec.FramerBuilder.
func (fb *FramerBuilder) New(reader io.Reader) codec.Framer {
	return &framer{
		reader: reader,
	}
}

// Parse implement multiplexed.FrameParser interface.
func (fb *FramerBuilder) Parse(rc io.Reader) (vid uint32, buf []byte, err error) {
	buf, err = fb.New(rc).ReadFrame()
	if err != nil {
		return 0, nil, err
	}
	return binary.BigEndian.Uint32(buf[10:14]), buf, nil
}

// framer is an implementation of codec.Framer.
// Used for trpc protocol.
type framer struct {
	reader io.Reader
	header [frameHeadLen]byte
}

// ReadFrame implements codec.Framer.
func (f *framer) ReadFrame() ([]byte, error) {
	num, err := io.ReadFull(f.reader, f.header[:])
	if err != nil {
		return nil, err
	}
	if num != int(frameHeadLen) {
		return nil, fmt.Errorf("trpc framer: read frame header num %d != %d, invalid", num, int(frameHeadLen))
	}
	magic := binary.BigEndian.Uint16(f.header[:2])
	if magic != uint16(trpcpb.TrpcMagic_TRPC_MAGIC_VALUE) {
		return nil, fmt.Errorf(
			"trpc framer: read framer head magic %d != %d, not match", magic, uint16(trpcpb.TrpcMagic_TRPC_MAGIC_VALUE))
	}
	totalLen := binary.BigEndian.Uint32(f.header[4:8])
	if totalLen < uint32(frameHeadLen) {
		return nil, fmt.Errorf(
			"trpc framer: read frame header total len %d < %d, invalid", totalLen, uint32(frameHeadLen))
	}

	if totalLen > uint32(DefaultMaxFrameSize) {
		return nil, fmt.Errorf(
			"trpc framer: read frame header total len %d > %d, too large", totalLen, uint32(DefaultMaxFrameSize))
	}

	msg := make([]byte, totalLen)
	num, err = io.ReadFull(f.reader, msg[frameHeadLen:totalLen])
	if err != nil {
		return nil, err
	}
	if num != int(totalLen-uint32(frameHeadLen)) {
		return nil, fmt.Errorf(
			"trpc framer: read frame total num %d != %d, invalid", num, int(totalLen-uint32(frameHeadLen)))
	}
	copy(msg, f.header[:])
	return msg, nil
}

// IsSafe implements codec.SafeFramer.
// Used for compatibility.
func (f *framer) IsSafe() bool {
	return true
}

// ServerCodec is an implementation of codec.Codec.
// Used for trpc serverside codec.
type ServerCodec struct {
	streamCodec *ServerStreamCodec
}

// Decode implements codec.Codec.
// It decodes the reqBuf and updates the msg that already initialized by service handler.
func (s *ServerCodec) Decode(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	if len(reqBuf) < int(frameHeadLen) {
		return nil, errors.New("server decode req buf len invalid")
	}
	frameHead := newDefaultUnaryFrameHead()
	frameHead.extract(reqBuf)
	msg.WithFrameHead(frameHead)
	if frameHead.TotalLen != uint32(len(reqBuf)) {
		return nil, fmt.Errorf("total len %d is not actual buf len %d", frameHead.TotalLen, len(reqBuf))
	}
	if frameHead.FrameType != uint8(trpcpb.TrpcDataFrameType_TRPC_UNARY_FRAME) { // streaming rpc has its own decoding
		rspBody, err := s.streamCodec.Decode(msg, reqBuf)
		if err != nil {
			// if decoding fails, the Close frame with Reset type will be returned to the client
			err := errs.NewFrameError(errs.RetServerDecodeFail, err.Error())
			s.streamCodec.buildResetFrame(msg, frameHead, err)
			return nil, err
		}
		return rspBody, nil
	}
	if frameHead.HeaderLen == 0 { // header not allowed to be empty for unary rpc
		return nil, errors.New("server decode pb head len empty")
	}

	requestProtocolBegin := uint32(frameHeadLen)
	requestProtocolEnd := requestProtocolBegin + uint32(frameHead.HeaderLen)
	if requestProtocolEnd > uint32(len(reqBuf)) {
		return nil, errors.New("server decode pb head len invalid")
	}
	req := &trpcpb.RequestProtocol{}
	if err := proto.Unmarshal(reqBuf[requestProtocolBegin:requestProtocolEnd], req); err != nil {
		return nil, err
	}

	attachmentBegin := frameHead.TotalLen - req.AttachmentSize
	if s := uint32(len(reqBuf)) - attachmentBegin; s != req.AttachmentSize {
		return nil, fmt.Errorf("decoding attachment: len of attachment(%d) "+
			"isn't equal to expected AttachmentSize(%d) ", s, req.AttachmentSize)
	}

	msgWithRequestProtocol(msg, req, reqBuf[attachmentBegin:])

	requestBodyBegin, requestBodyEnd := requestProtocolEnd, attachmentBegin
	return reqBuf[requestBodyBegin:requestBodyEnd], nil
}

func msgWithRequestProtocol(msg codec.Msg, req *trpcpb.RequestProtocol, attm []byte) {
	// set server request head
	msg.WithServerReqHead(req)
	// construct response protocol head in advance
	rsp := newResponseProtocol(req)
	msg.WithServerRspHead(rsp)

	// ---------the following code is to set the essential info-----------//
	// set upstream timeout
	msg.WithRequestTimeout(time.Millisecond * time.Duration(req.GetTimeout()))
	// set upstream service name
	msg.WithCallerServiceName(string(req.GetCaller()))
	msg.WithCalleeServiceName(string(req.GetCallee()))
	// set server handler method name
	rpcName := string(req.GetFunc())
	msg.WithServerRPCName(rpcName)
	msg.WithCalleeMethod(icodec.MethodFromRPCName(rpcName))
	// set body serialization type
	msg.WithSerializationType(int(req.GetContentType()))
	// set body compression type
	msg.WithCompressType(int(req.GetContentEncoding()))
	// set dyeing mark
	msg.WithDyeing((req.GetMessageType() & uint32(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)
	// parse tracing MetaData, set MetaData into msg
	if len(req.TransInfo) > 0 {
		msg.WithServerMetaData(req.GetTransInfo())
		// mark with dyeing key
		if bs, ok := req.TransInfo[DyeingKey]; ok {
			msg.WithDyeingKey(string(bs))
		}
		// transmit env info
		if envs, ok := req.TransInfo[EnvTransfer]; ok {
			msg.WithEnvTransfer(string(envs))
		}
	}
	// set call type
	msg.WithCallType(codec.RequestType(req.GetCallType()))
	attachment.SetServerRequestAttachment(msg, attm)
}

// Encode implements codec.Codec.
// It encodes the rspBody to binary data and returns it to client.
func (s *ServerCodec) Encode(msg codec.Msg, rspBody []byte) ([]byte, error) {
	frameHead := loadOrStoreDefaultUnaryFrameHead(msg)
	if frameHead.isStream() {
		return s.streamCodec.Encode(msg, rspBody)
	}
	if !frameHead.isUnary() {
		return nil, errUnknownFrameType
	}

	rspProtocol := getAndInitResponseProtocol(msg)

	attm, err := io.ReadAll(attachment.GetServerResponseAttachment(msg))
	if err != nil {
		return nil, fmt.Errorf("encoding attachment : %w", err)
	}
	rspProtocol.AttachmentSize = uint32(len(attm))

	rspHead, err := proto.Marshal(rspProtocol)
	if err != nil {
		return nil, err
	}

	rspBuf, err := frameHead.construct(rspHead, rspBody, attm)
	if errors.Is(err, errHeadOverflowsUint16) {
		return handleEncodeErr(rspProtocol, frameHead, rspBody, err)
	}
	var frameTooLargeErr *errFrameTooLarge
	if errors.As(err, &frameTooLargeErr) || errors.Is(err, errHeadOverflowsUint32) {
		// If frame len is larger than DefaultMaxFrameSize or overflows uint32, set rspBody nil.
		return handleEncodeErr(rspProtocol, frameHead, nil, err)
	}
	return rspBuf, err
}

// getAndInitResponseProtocol returns rsp head from msg and initialize the rsp with msg.
// If rsp head is not found from msg, a new rsp head will be created and initialized.
func getAndInitResponseProtocol(msg codec.Msg) *trpcpb.ResponseProtocol {
	rsp, ok := msg.ServerRspHead().(*trpcpb.ResponseProtocol)
	if !ok {
		if req, ok := msg.ServerReqHead().(*trpcpb.RequestProtocol); ok {
			rsp = newResponseProtocol(req)
		} else {
			rsp = &trpcpb.ResponseProtocol{}
		}
	}

	// update serialization and compression type
	rsp.ContentType = uint32(msg.SerializationType())
	rsp.ContentEncoding = uint32(msg.CompressType())

	// convert error returned by server handler to ret code in response protocol head
	if err := msg.ServerRspErr(); err != nil {
		rsp.ErrorMsg = []byte(err.Msg)
		if err.Type == errs.ErrorTypeFramework {
			rsp.Ret = int32(err.Code)
		} else {
			rsp.FuncRet = int32(err.Code)
		}
	}

	if len(msg.ServerMetaData()) > 0 {
		if rsp.TransInfo == nil {
			rsp.TransInfo = make(map[string][]byte)
		}
		for k, v := range msg.ServerMetaData() {
			rsp.TransInfo[k] = v
		}
	}

	return rsp
}

func newResponseProtocol(req *trpcpb.RequestProtocol) *trpcpb.ResponseProtocol {
	return &trpcpb.ResponseProtocol{
		Version:         uint32(trpcpb.TrpcProtoVersion_TRPC_PROTO_V1),
		CallType:        req.CallType,
		RequestId:       req.RequestId,
		MessageType:     req.MessageType,
		ContentType:     req.ContentType,
		ContentEncoding: req.ContentEncoding,
	}
}

// handleEncodeErr handles encode err and returns RetServerEncodeFail.
func handleEncodeErr(
	rsp *trpcpb.ResponseProtocol,
	frameHead *FrameHead,
	rspBody []byte,
	encodeErr error,
) ([]byte, error) {
	// discard all TransInfo and return RetServerEncodeFail
	// cover the original no matter what
	rsp.TransInfo = nil
	rsp.Ret = int32(errs.RetServerEncodeFail)
	rsp.ErrorMsg = []byte(encodeErr.Error())
	rspHead, err := proto.Marshal(rsp)
	if err != nil {
		return nil, err
	}
	// if error still occurs, response will be discarded.
	// client will be notified as conn closed
	return frameHead.construct(rspHead, rspBody, nil)
}

// ClientCodec is an implementation of codec.Codec.
// Used for trpc clientside codec.
type ClientCodec struct {
	streamCodec   *ClientStreamCodec
	defaultCaller string // trpc.app.server.service
	requestID     uint32 // global unique request id
}

// Encode implements codec.Codec.
// It encodes reqBody into binary data. New msg will be cloned by client stub.
func (c *ClientCodec) Encode(msg codec.Msg, reqBody []byte) (reqBuf []byte, err error) {
	frameHead := loadOrStoreDefaultUnaryFrameHead(msg)
	if frameHead.isStream() {
		return c.streamCodec.Encode(msg, reqBody)
	}
	if !frameHead.isUnary() {
		return nil, errUnknownFrameType
	}

	// create a new framehead without modifying the original one
	// to avoid overwriting the requestID of the original framehead.
	frameHead = newDefaultUnaryFrameHead()
	req, err := loadOrStoreDefaultRequestProtocol(msg)
	if err != nil {
		return nil, err
	}

	// request id atomically increases by 1, ensuring that each request id is unique.
	requestID := atomic.AddUint32(&c.requestID, 1)
	frameHead.upgradeProtocol(curProtocolVersion, requestID)
	msg.WithRequestID(requestID)

	attm, err := io.ReadAll(attachment.GetClientRequestAttachment(msg))
	if err != nil {
		return nil, fmt.Errorf("encoding attachment : %w", err)
	}
	req.AttachmentSize = uint32(len(attm))

	updateRequestProtocol(req, updateCallerServiceName(msg, c.defaultCaller))

	reqHead, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	return frameHead.construct(reqHead, reqBody, attm)
}

// loadOrStoreDefaultRequestProtocol loads the existing RequestProtocol from msg if present.
// Otherwise, it stores default UnaryRequestProtocol created to msg and returns the default RequestProtocol.
func loadOrStoreDefaultRequestProtocol(msg codec.Msg) (*trpcpb.RequestProtocol, error) {
	if req := msg.ClientReqHead(); req != nil {
		// client req head not being nil means it's created on purpose and set to
		// record request protocol head
		req, ok := req.(*trpcpb.RequestProtocol)
		if !ok {
			return nil, errors.New("client encode req head type invalid, must be trpc request protocol head")
		}
		return req, nil
	}

	req := newDefaultUnaryRequestProtocol()
	msg.WithClientReqHead(req)
	return req, nil
}

func newDefaultUnaryRequestProtocol() *trpcpb.RequestProtocol {
	return &trpcpb.RequestProtocol{
		Version:  uint32(trpcpb.TrpcProtoVersion_TRPC_PROTO_V1),
		CallType: uint32(trpcpb.TrpcCallType_TRPC_UNARY_CALL),
	}
}

// update updates CallerServiceName of msg with name
func updateCallerServiceName(msg codec.Msg, name string) codec.Msg {
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(name)
	}
	return msg
}

// update updates req with requestID and  msg.
func updateRequestProtocol(req *trpcpb.RequestProtocol, msg codec.Msg) {
	req.RequestId = msg.RequestID()
	req.Caller = []byte(msg.CallerServiceName())
	// set callee service name
	req.Callee = []byte(msg.CalleeServiceName())
	// set backend rpc name
	req.Func = []byte(msg.ClientRPCName())
	// set backend serialization type
	req.ContentType = uint32(msg.SerializationType())
	// set backend compression type
	req.ContentEncoding = uint32(msg.CompressType())
	// set rest timeout for downstream
	req.Timeout = uint32(msg.RequestTimeout() / time.Millisecond)
	// set dyeing info
	if msg.Dyeing() {
		req.MessageType = req.MessageType | uint32(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)
	}
	// set client transinfo
	req.TransInfo = setClientTransInfo(msg, req.TransInfo)
	// set call type
	req.CallType = uint32(msg.CallType())
}

// setClientTransInfo sets client TransInfo.
func setClientTransInfo(msg codec.Msg, trans map[string][]byte) map[string][]byte {
	// set MetaData
	if len(msg.ClientMetaData()) > 0 {
		if trans == nil {
			trans = make(map[string][]byte)
		}
		for k, v := range msg.ClientMetaData() {
			trans[k] = v
		}
	}
	if len(msg.DyeingKey()) > 0 {
		if trans == nil {
			trans = make(map[string][]byte)
		}
		trans[DyeingKey] = []byte(msg.DyeingKey())
	}
	if len(msg.EnvTransfer()) > 0 {
		if trans == nil {
			trans = make(map[string][]byte)
		}
		trans[EnvTransfer] = []byte(msg.EnvTransfer())
	} else {
		// if msg.EnvTransfer() empty, transmitted env info in req.TransInfo should be cleared
		if _, ok := trans[EnvTransfer]; ok {
			trans[EnvTransfer] = nil
		}
	}
	return trans
}

// Decode implements codec.Codec.
// It decodes rspBuf into rspBody.
func (c *ClientCodec) Decode(msg codec.Msg, rspBuf []byte) (rspBody []byte, err error) {
	if len(rspBuf) < int(frameHeadLen) {
		return nil, errors.New("client decode rsp buf len invalid")
	}
	frameHead := newDefaultUnaryFrameHead()
	frameHead.extract(rspBuf)
	msg.WithFrameHead(frameHead)
	if frameHead.TotalLen != uint32(len(rspBuf)) {
		return nil, fmt.Errorf("total len %d is not actual buf len %d", frameHead.TotalLen, len(rspBuf))
	}
	if trpcpb.TrpcDataFrameType(frameHead.FrameType) != trpcpb.TrpcDataFrameType_TRPC_UNARY_FRAME {
		return c.streamCodec.Decode(msg, rspBuf)
	}
	if frameHead.HeaderLen == 0 {
		return nil, errors.New("client decode pb head len empty")
	}

	responseProtocolBegin := uint32(frameHeadLen)
	responseProtocolEnd := responseProtocolBegin + uint32(frameHead.HeaderLen)
	if responseProtocolEnd > uint32(len(rspBuf)) {
		return nil, errors.New("client decode pb head len invalid")
	}
	rsp, err := loadOrStoreResponseHead(msg)
	if err != nil {
		return nil, err
	}
	if err := proto.Unmarshal(rspBuf[responseProtocolBegin:responseProtocolEnd], rsp); err != nil {
		return nil, err
	}

	attachmentBegin := frameHead.TotalLen - rsp.AttachmentSize
	if s := uint32(len(rspBuf)) - attachmentBegin; rsp.AttachmentSize != s {
		return nil, fmt.Errorf("decoding attachment:(%d) len of attachment"+
			"isn't equal to expected AttachmentSize(%d)", s, rsp.AttachmentSize)
	}
	if err := updateMsg(msg, frameHead, rsp, rspBuf[attachmentBegin:]); err != nil {
		return nil, err
	}

	bodyBegin, bodyEnd := responseProtocolEnd, attachmentBegin
	return rspBuf[bodyBegin:bodyEnd], nil
}

func loadOrStoreResponseHead(msg codec.Msg) (*trpcpb.ResponseProtocol, error) {
	// client rsp head being nil means no need to record backend response protocol head
	// most of the time, response head is not set and should be created here.
	rsp := msg.ClientRspHead()
	if rsp == nil {
		rsp := &trpcpb.ResponseProtocol{}
		msg.WithClientRspHead(rsp)
		return rsp, nil
	}

	// client rsp head not being nil means it's created on purpose and set to
	// record response protocol head
	{
		rsp, ok := rsp.(*trpcpb.ResponseProtocol)
		if !ok {
			return nil, errors.New("client decode rsp head type invalid, must be trpc response protocol head")
		}
		return rsp, nil
	}
}

// loadOrStoreDefaultUnaryFrameHead loads the existing frameHead from msg if present.
// Otherwise, it stores default Unary FrameHead to msg, and returns the default Unary FrameHead.
func loadOrStoreDefaultUnaryFrameHead(msg codec.Msg) *FrameHead {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok {
		frameHead = newDefaultUnaryFrameHead()
		msg.WithFrameHead(frameHead)
	}
	return frameHead
}

func copyRspHead(dst, src *trpcpb.ResponseProtocol) {
	dst.Version = src.Version
	dst.CallType = src.CallType
	dst.RequestId = src.RequestId
	dst.Ret = src.Ret
	dst.FuncRet = src.FuncRet
	dst.ErrorMsg = src.ErrorMsg
	dst.MessageType = src.MessageType
	dst.TransInfo = src.TransInfo
	dst.ContentType = src.ContentType
	dst.ContentEncoding = src.ContentEncoding
}

func updateMsg(msg codec.Msg, frameHead *FrameHead, rsp *trpcpb.ResponseProtocol, attm []byte) error {
	msg.WithFrameHead(frameHead)
	msg.WithCompressType(int(rsp.GetContentEncoding()))
	msg.WithSerializationType(int(rsp.GetContentType()))

	// reset client metadata if new transinfo is returned with response
	if len(rsp.TransInfo) > 0 {
		md := msg.ClientMetaData()
		if len(md) == 0 {
			md = codec.MetaData{}
		}
		for k, v := range rsp.TransInfo {
			md[k] = v
		}
		msg.WithClientMetaData(md)
	}

	// if retcode is not 0, a converted error should be returned
	if rsp.GetRet() != 0 {
		err := &errs.Error{
			Type: errs.ErrorTypeCalleeFramework,
			Code: trpcpb.TrpcRetCode(rsp.GetRet()),
			Desc: ProtocolName,
			Msg:  string(rsp.GetErrorMsg()),
		}
		msg.WithClientRspErr(err)
	} else if rsp.GetFuncRet() != 0 {
		msg.WithClientRspErr(errs.New(int(rsp.GetFuncRet()), string(rsp.GetErrorMsg())))
	}

	// error should be returned immediately for request id mismatch
	req, err := loadOrStoreDefaultRequestProtocol(msg)
	if err == nil && rsp.RequestId != req.RequestId {
		return fmt.Errorf("rsp request_id %d different from req request_id %d", rsp.RequestId, req.RequestId)
	}

	// handle protocol upgrading
	frameHead.upgradeProtocol(curProtocolVersion, rsp.RequestId)
	msg.WithRequestID(rsp.RequestId)

	attachment.SetClientResponseAttachment(msg, attm)
	return nil
}
