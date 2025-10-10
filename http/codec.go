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

package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	ibytes "trpc.group/trpc-go/trpc-go/internal/bytes"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	"trpc.group/trpc-go/trpc-go/internal/http/fastop"
	"trpc.group/trpc-go/trpc-go/internal/protocol"

	"github.com/r3labs/sse/v2"
)

// Constants of header keys related to trpc.
const (
	TrpcVersion      = "trpc-version"
	TrpcCallType     = "trpc-call-type"
	TrpcMessageType  = "trpc-message-type"
	TrpcRequestID    = "trpc-request-id"
	TrpcTimeout      = "trpc-timeout"
	TrpcCaller       = "trpc-caller"
	TrpcCallerMethod = "trpc-caller-method"
	TrpcCallee       = "trpc-callee"
	TrpcTransInfo    = "trpc-trans-info"
	TrpcEnv          = "trpc-env"
	TrpcDyeingKey    = "trpc-dyeing-key"
	// TrpcErrorMessage used to pass error messages,
	// contains user code's error or frame errors (such as validation framework).
	TrpcErrorMessage = "trpc-error-msg"
	// TrpcFrameworkErrorCode used to pass the error code reported by framework.
	TrpcFrameworkErrorCode = "trpc-ret"
	// TrpcUserFuncErrorCode used to pass the error code reported by user.
	TrpcUserFuncErrorCode = "trpc-func-ret"
	// Connection is used to set whether connect mode is "Connection".
	Connection = "Connection"
)

var (
	canonicalContentType         = textproto.CanonicalMIMEHeaderKey("Content-Type")
	canonicalXContentTypeOptions = textproto.CanonicalMIMEHeaderKey("X-Content-Type-Options")
	canonicalContentEncoding     = textproto.CanonicalMIMEHeaderKey("Content-Encoding")
)

var (
	canonicalTrpcVersion            = textproto.CanonicalMIMEHeaderKey(TrpcVersion)
	canonicalTrpcCallType           = textproto.CanonicalMIMEHeaderKey(TrpcCallType)
	canonicalTrpcMessageType        = textproto.CanonicalMIMEHeaderKey(TrpcMessageType)
	canonicalTrpcRequestID          = textproto.CanonicalMIMEHeaderKey(TrpcRequestID)
	canonicalTrpcTimeout            = textproto.CanonicalMIMEHeaderKey(TrpcTimeout)
	canonicalTrpcCaller             = textproto.CanonicalMIMEHeaderKey(TrpcCaller)
	canonicalTrpcCallerMethod       = textproto.CanonicalMIMEHeaderKey(TrpcCallerMethod)
	canonicalTrpcCallee             = textproto.CanonicalMIMEHeaderKey(TrpcCallee)
	canonicalTrpcTransInfo          = textproto.CanonicalMIMEHeaderKey(TrpcTransInfo)
	canonicalTrpcErrorMessage       = textproto.CanonicalMIMEHeaderKey(TrpcErrorMessage)
	canonicalTrpcFrameworkErrorCode = textproto.CanonicalMIMEHeaderKey(TrpcFrameworkErrorCode)
	canonicalTrpcUserFuncErrorCode  = textproto.CanonicalMIMEHeaderKey(TrpcUserFuncErrorCode)
)

var contentTypeSerializationType = map[string]int{
	"application/json":                  codec.SerializationTypeJSON,
	"application/protobuf":              codec.SerializationTypePB,
	"application/x-protobuf":            codec.SerializationTypePB,
	"application/pb":                    codec.SerializationTypePB,
	"application/proto":                 codec.SerializationTypePB,
	"application/flatbuffer":            codec.SerializationTypeFlatBuffer,
	"application/octet-stream":          codec.SerializationTypeNoop,
	"application/x-www-form-urlencoded": codec.SerializationTypeForm,
	"application/xml":                   codec.SerializationTypeXML,
	"text/xml":                          codec.SerializationTypeTextXML,
	"multipart/form-data":               codec.SerializationTypeFormData,
}

var serializationTypeContentType = map[int]string{
	codec.SerializationTypeJSON:       "application/json",
	codec.SerializationTypePB:         "application/proto",
	codec.SerializationTypeFlatBuffer: "application/flatbuffer",
	codec.SerializationTypeNoop:       "application/octet-stream",
	codec.SerializationTypeForm:       "application/x-www-form-urlencoded",
	codec.SerializationTypeXML:        "application/xml",
	codec.SerializationTypeTextXML:    "text/xml",
	codec.SerializationTypeFormData:   "multipart/form-data",
}

var contentEncodingCompressType = map[string]int{
	"gzip": codec.CompressTypeGzip,
}

var compressTypeContentEncoding = map[int]string{
	codec.CompressTypeGzip: "gzip",
}

// RegisterSerializer registers a new custom serialization method,
// such as RegisterSerializer("text/plain", 130, xxxSerializer).
func RegisterSerializer(httpContentType string, serializationType int, serializer codec.Serializer) {
	codec.RegisterSerializer(serializationType, serializer)
	RegisterContentType(httpContentType, serializationType)
}

// MustRegisterSerializer registers a new custom serialization method,
// such as MustRegisterSerializer("text/plain", 130, xxxSerializer).
// It will panic if the httpContentType or serializationType has been registered.
//
// In most cases, the framework uses the init + RegisterSerializer method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegisterSerializer to forcibly register a component 'xxx', while the framework
// uses init + RegisterSerializer to register another component 'yyy', conflicts may occur. If the init function
// for MustRegisterSerializer is executed before the conflicting init function, MustRegisterSerializer might not raise
// an error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegisterSerializer and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegisterSerializer(httpContentType string, serializationType int, serializer codec.Serializer) {
	codec.MustRegisterSerializer(serializationType, serializer)
	MustRegisterContentType(httpContentType, serializationType)
}

// RegisterContentType registers existing serialization method to
// contentTypeSerializationType and serializationTypeContentType.
func RegisterContentType(httpContentType string, serializationType int) {
	contentTypeSerializationType[httpContentType] = serializationType
	serializationTypeContentType[serializationType] = httpContentType
}

// MustRegisterContentType registers existing serialization method to
// contentTypeSerializationType and serializationTypeContentType.
// It will panic if the serializationType or httpContentType has been registered.
//
// In most cases, the framework uses the init + RegisterContentType method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegisterContentType to forcibly register a component 'xxx', while the framework
// uses init + RegisterContentType to register another component 'yyy', conflicts may occur. If the init function
// for MustRegisterContentType is executed before the conflicting init function, MustRegisterContentType might not
// raise an error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegisterContentType and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegisterContentType(httpContentType string, serializationType int) {
	if _, ok := contentTypeSerializationType[httpContentType]; ok {
		panic("content type already registered: " + httpContentType)
	}
	if _, ok := serializationTypeContentType[serializationType]; ok {
		panic("serialization type already registered: " + strconv.Itoa(serializationType))
	}
	RegisterContentType(httpContentType, serializationType)
}

// SetContentType sets one-way mapping relationship for compatibility
// with old framework services, allowing multiple http content type to
// map to the save trpc serialization type.
// Tell the framework which serialization method to use to parse this content-type.
// Such as, a non-standard http server returns content type seems to be "text/html",
// but it is actually "json" data. At this time, you can set it like this:
// SetContentType("text/html", codec.SerializationTypeJSON).
func SetContentType(httpContentType string, serializationType int) {
	contentTypeSerializationType[httpContentType] = serializationType
}

// RegisterContentEncoding registers an existing decompression method,
// such as RegisterContentEncoding("gzip", codec.CompressTypeGzip).
func RegisterContentEncoding(httpContentEncoding string, compressType int) {
	contentEncodingCompressType[httpContentEncoding] = compressType
	compressTypeContentEncoding[compressType] = httpContentEncoding
}

// MustRegisterContentEncoding registers an existing decompression method,
// such as MustRegisterContentEncoding("gzip", codec.CompressTypeGzip).
// It will panic if the httpContentEncoding or compressType has been registered.
//
// In most cases, the framework uses the init + RegisterContentEncoding method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegisterContentEncoding to forcibly register a component 'xxx', while the framework
// uses init + RegisterContentEncoding to register another component 'yyy', conflicts may occur. If the init function
// for MustRegisterContentEncoding is executed before the conflicting init function, MustRegisterContentEncoding might
// not raise an error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegisterContentEncoding and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegisterContentEncoding(httpContentEncoding string, compressType int) {
	if _, ok := contentEncodingCompressType[httpContentEncoding]; ok {
		panic("content encoding already registered: " + httpContentEncoding)
	}
	if _, ok := compressTypeContentEncoding[compressType]; ok {
		panic("compress type already registered: " + strconv.Itoa(compressType))
	}
	RegisterContentEncoding(httpContentEncoding, compressType)
}

// RegisterStatus registers trpc return code to http status.
func RegisterStatus(code int32, httpStatus int) {
	ErrsToHTTPStatus[code] = httpStatus
}

// MustRegisterStatus registers trpc return code to http status.
// It will panic if the code has been registered.
//
// In most cases, the framework uses the init + RegisterStatus method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegisterStatus to forcibly register a component 'xxx', while the framework
// uses init + RegisterStatus to register another component 'yyy', conflicts may occur. If the init function
// for MustRegisterStatus is executed before the conflicting init function, MustRegisterStatus might not raise an
// error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegisterStatus and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegisterStatus(code int32, httpStatus int) {
	if _, ok := ErrsToHTTPStatus[code]; ok {
		panic("status already registered: " + strconv.Itoa(int(code)))
	}
	RegisterStatus(code, httpStatus)
}

func init() {
	codec.Register(protocol.HTTP, DefaultServerCodec, DefaultClientCodec)
	codec.Register(protocol.HTTPS, DefaultServerCodec, DefaultClientCodec)
	codec.Register(protocol.HTTP2, DefaultServerCodec, DefaultClientCodec)
	// Support no protocol file custom routing and feature isolation.
	codec.Register(protocol.HTTPNoProtocol, DefaultNoProtocolServerCodec, DefaultClientCodec)
	codec.Register(protocol.HTTPSNoProtocol, DefaultNoProtocolServerCodec, DefaultClientCodec)
	codec.Register(protocol.HTTP2NoProtocol, DefaultNoProtocolServerCodec, DefaultClientCodec)
}

var (
	// DefaultClientCodec is the default http client codec.
	DefaultClientCodec = &ClientCodec{
		ErrHandler: defaultDecodeErrHandler,
	}

	// DefaultServerCodec is the default http server codec.
	DefaultServerCodec = &ServerCodec{
		AutoGenTrpcHead:              true,
		ErrHandler:                   defaultErrHandler,
		RspHandler:                   defaultRspHandler,
		AutoReadBody:                 true,
		DisableEncodeTransInfoBase64: false,
	}

	// DefaultNoProtocolServerCodec is the default http no protocol server codec.
	DefaultNoProtocolServerCodec = &ServerCodec{
		AutoGenTrpcHead:              true,
		ErrHandler:                   defaultErrHandler,
		RspHandler:                   defaultRspHandler,
		AutoReadBody:                 false,
		DisableEncodeTransInfoBase64: false,
	}
)

// ErrEncodeMissingHeader defines error used for special handling
// in transport when ctx lost header information.
var ErrEncodeMissingHeader = errors.New("trpc/http: server encode missing http header in context")

// ServerCodec is the encoder/decoder for HTTP server.
type ServerCodec struct {
	// AutoGenTrpcHead converts trpc header automatically.
	// Auto conversion could be enabled by setting http.DefaultServerCodec.AutoGenTrpcHead with true.
	AutoGenTrpcHead bool

	// ErrHandler is error code handle function, which is filled into header by default.
	// Business can set this with http.DefaultServerCodec.ErrHandler = func(rsp, req, err) {}.
	ErrHandler ErrorHandler

	// RspHandler returns the data handle function. By default, data is returned directly.
	// Business can customize this method to shape returned data.
	// Business can set this with http.DefaultServerCodec.RspHandler = func(rsp, req, rspBody) {}.
	RspHandler ResponseHandler

	// AutoReadBody reads http request body automatically.
	AutoReadBody bool

	// CacheRequestBody determines whether to cache the request body bytes read from the client.
	// The default value is true if this boolean pointer is nil.
	// The reason for setting it to true for a nil pointer is to maintain backward compatibility.
	// When the flag is true, the entire request body will be cached inside the http.Head.ReqBody field,
	// which may lead to significant memory consumption when the payload is large.
	// To disable it, use:
	//
	//  import thttp "trpc.group/trpc-go/trpc-go/http"
	//  func init() {
	//  	cacheRequestBody := false
	//  	thttp.DefaultServerCodec.CacheRequestBody = &cacheRequestBody
	//  }
	//
	// Note: this flag affects the global http server codec for the HTTP RPC service.
	// If you want to control only some of the services, you may consider registering a new
	// http server codec for a different protocol name. However, I strongly advise against doing so, as the
	// process required for registering a new protocol name is rather complicated.
	CacheRequestBody *bool

	// DisableEncodeTransInfoBase64 indicates whether to disable encoding the transinfo value by base64.
	DisableEncodeTransInfoBase64 bool

	// POSTOnly determines whether to process only requests that use the HTTP POST method.
	// This is commonly used in an HTTP RPC server to allow only the HTTP POST method to be accepted,
	// instead of allowing both the POST and GET methods.
	POSTOnly bool
}

// ContextKey defines context key of http.
type ContextKey string

const (
	// ContextKeyHeader key of http header
	ContextKeyHeader = ContextKey("TRPC_SERVER_HTTP_HEADER")
	// ParseMultipartFormMaxMemory maximum memory for parsing request body, default is 32M.
	ParseMultipartFormMaxMemory int64 = 32 << 20
)

// Header encapsulates http context.
type Header struct {
	// ReqBody caches the request body of the current client request.
	// This feature is enabled if thttp.DefaultServerCodec.CacheRequestBody is a nil pointer or &true.
	// Consider setting it to false if you find that large packets consume too much memory.
	ReqBody  []byte
	Request  *http.Request
	Response http.ResponseWriter
}

// ClientReqHeader encapsulates http client context.
// Setting ClientReqHeader is not allowed when NewClientProxy is waiting for the init of Client.
// Setting ClientReqHeader is needed for each RPC.
type ClientReqHeader struct {
	// Schema should be named as scheme according to https://www.rfc-editor.org/rfc/rfc3986#section-3.
	// Now that it has been exported, we can do nothing more than add a comment here.
	Schema string // Examples: HTTP, HTTPS.
	Method string
	// Host directly sets the final host field in the stdhttp.Request.
	// Use this field to set the host instead of using (*ClientReqHeader).AddHeader("Host", "xxx").
	Host    string
	Request *http.Request
	Header  http.Header
	ReqBody io.Reader
	// DecorateRequest will be called right before client.Do(request) to
	// allow users to make final custom modifications to the HTTP request.
	DecorateRequest func(*http.Request) *http.Request
}

// AddHeader adds http header.
// Note: Please use the (*ClientReqHeader).Host field to set the host instead of
// using (*ClientReqHeader).AddHeader("Host", "xxx").
func (h *ClientReqHeader) AddHeader(key string, value string) {
	if h.Header == nil {
		h.Header = make(http.Header)
	}
	h.Header.Add(key, value)
}

// ClientRspHeader encapsulates the context returned by http client request.
type ClientRspHeader struct {
	// ManualReadBody is used to control whether to read http response manually
	// (not read automatically by the framework).
	// Set it to true so that you can read data directly from Response.Body manually.
	// The default value is false.
	ManualReadBody bool
	Response       *http.Response

	// ResponseHandler is an interface that the framework will invoke
	// if SSECondition returns false OR SSEHandler is not defined.
	// If ResponseHandler is provided by the user, the framework will automatically
	// read the http response body and invoke the ResponseHandler for each response.
	ResponseHandler RspHandler

	// SSECondition is a function that users must implement to determine
	// whether to call server-sent event (SSE) message callbacks.
	// If SSECondition returns true AND SSEHandler is defined, the framework will
	// call the SSEHandler for each SSE event in sequence.
	SSECondition func(*http.Response) bool

	// SSEHandler is an interface that users must implement to handle
	// server-sent event (SSE) message callbacks.
	// When this field is provided by the user, the framework will automatically
	// add the following headers to the request, if they are not already present:
	//
	//  "Accept": "text/event-stream"
	//  "Connection": "keep-alive"
	//  "Cache-Control": "no-cache"
	//
	// Users MUST NOT set ManualReadBody to true.
	// The framework will automatically parse the HTTP response into SSE events
	// and invoke the SSEHandler for each SSE event in sequence.
	// If any SSEHandler returns an error, the process will be halted and the
	// error will be returned.
	// The parsing of SSE events will continue until an io.EOF is encountered
	// in the reading of the HTTP response body.
	SSEHandler SSEHandler
}

// RspHandler is an interface for users to implement common HTTP response callbacks.
type RspHandler interface {
	// Handle handles http response.
	// If the returned error is non-nil, the framework will abort the reading of
	// the HTTP connection.
	Handle(*http.Response) error
}

// SSEHandler is an interface for users to implement sse message callbacks.
type SSEHandler interface {
	// Handle handles sse event, if the returned error is non-nil,
	// the framework will abort the reading of the HTTP connection.
	Handle(*sse.Event) error
}

// ErrsToHTTPStatus maps from framework errs retcode to http status code.
var ErrsToHTTPStatus = map[int32]int{
	errs.RetServerDecodeFail:   http.StatusBadRequest,
	errs.RetServerEncodeFail:   http.StatusInternalServerError,
	errs.RetServerNoService:    http.StatusNotFound,
	errs.RetServerNoFunc:       http.StatusNotFound,
	errs.RetServerTimeout:      http.StatusGatewayTimeout,
	errs.RetServerOverload:     http.StatusTooManyRequests,
	errs.RetServerSystemErr:    http.StatusInternalServerError,
	errs.RetServerAuthFail:     http.StatusUnauthorized,
	errs.RetServerValidateFail: http.StatusBadRequest,
	errs.RetUnknown:            http.StatusInternalServerError,
}

// Head gets the corresponding http header from context.
func Head(ctx context.Context) *Header {
	if ret, ok := ctx.Value(ContextKeyHeader).(*Header); ok {
		return ret
	}
	return nil
}

// Request gets the corresponding http request from context.
func Request(ctx context.Context) *http.Request {
	head := Head(ctx)
	if head == nil {
		return nil
	}
	return head.Request
}

// Response gets the corresponding http response from context.
func Response(ctx context.Context) http.ResponseWriter {
	head := Head(ctx)
	if head == nil {
		return nil
	}
	return head.Response
}

// WithHeader sets http header in context.
func WithHeader(ctx context.Context, value *Header) context.Context {
	return context.WithValue(ctx, ContextKeyHeader, value)
}

// setReqHeaderAndUpdateMsg sets request header.
func (sc *ServerCodec) setReqHeaderAndUpdateMsg(head *Header, msg codec.Msg) error {
	if !sc.AutoGenTrpcHead { // Auto generates trpc head.
		return nil
	}

	trpcReq := &trpc.RequestProtocol{}
	msg.WithServerReqHead(trpcReq)
	msg.WithServerRspHead(trpcReq)

	trpcReq.Func = []byte(msg.ServerRPCName())
	trpcReq.ContentType = uint32(msg.SerializationType())
	trpcReq.ContentEncoding = uint32(msg.CompressType())

	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcVersion); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.Version = uint32(i)
	}
	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcCallType); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.CallType = uint32(i)
	}
	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcMessageType); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.MessageType = uint32(i)
	}
	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcRequestID); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.RequestId = uint32(i)
	}
	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcTimeout); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.Timeout = uint32(i)
		msg.WithRequestTimeout(time.Millisecond * time.Duration(i))
	}
	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcCaller); v != "" {
		trpcReq.Caller = []byte(v)
		msg.WithCallerServiceName(v)
	}

	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcCallerMethod); v != "" {
		msg.WithCallerMethod(v)
	}
	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcCallee); v != "" {
		trpcReq.Callee = []byte(v)
		msg.WithCalleeServiceName(v)
	}

	msg.WithDyeing((trpcReq.GetMessageType() & uint32(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)

	if v := fastop.CanonicalHeaderGet(head.Request.Header, canonicalTrpcTransInfo); v != "" {
		transInfo, err := unmarshalTransInfo(msg, v)
		if err != nil {
			return err
		}
		trpcReq.TransInfo = transInfo
	}
	return nil
}

func unmarshalTransInfo(msg codec.Msg, v string) (map[string][]byte, error) {
	m := make(map[string]string)
	if err := codec.Unmarshal(codec.SerializationTypeJSON, []byte(v), &m); err != nil {
		return nil, err
	}
	transInfo := make(map[string][]byte)
	// Since the http header can only transmit plaintext, but trpc transinfo is binary stream,
	// so it needs to be protected with base64.
	for k, v := range m {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			decoded = []byte(v)
		}
		transInfo[k] = decoded
		if k == TrpcEnv {
			msg.WithEnvTransfer(string(decoded))
		}
		if k == TrpcDyeingKey {
			msg.WithDyeingKey(string(decoded))
		}
	}
	msg.WithServerMetaData(transInfo)
	return transInfo, nil
}

// getReqBody gets the body of request.
func (sc *ServerCodec) getReqBody(head *Header, msg codec.Msg) ([]byte, error) {
	msg.WithCalleeMethod(head.Request.URL.Path)
	msg.WithServerRPCName(head.Request.URL.Path)

	if !sc.AutoReadBody {
		return nil, nil
	}

	if head.Request.Method != http.MethodPost && sc.POSTOnly {
		return nil, fmt.Errorf("server codec only allows POST method request, the current method is %s",
			head.Request.Method)
	}

	var reqBody []byte
	if head.Request.Method == http.MethodGet {
		msg.WithSerializationType(codec.SerializationTypeGet)
		reqBody = []byte(head.Request.URL.RawQuery)
	} else {
		var exist bool
		msg.WithSerializationType(codec.SerializationTypeJSON)
		ct := fastop.CanonicalHeaderGet(head.Request.Header, canonicalContentType)
		for contentType, serializationType := range contentTypeSerializationType {
			if strings.Contains(ct, contentType) {
				msg.WithSerializationType(serializationType)
				exist = true
				break
			}
		}
		if exist {
			var err error
			reqBody, err = getBody(ct, head.Request)
			if err != nil {
				return nil, err
			}
		}
	}
	if sc.CacheRequestBody == nil || *sc.CacheRequestBody {
		head.ReqBody = reqBody
	}
	return reqBody, nil
}

// getBody gets the body of request.
func getBody(contentType string, r *http.Request) ([]byte, error) {
	if strings.Contains(contentType, serializationTypeContentType[codec.SerializationTypeFormData]) {
		if r.Form == nil {
			if err := r.ParseMultipartForm(ParseMultipartFormMaxMemory); err != nil {
				return nil, fmt.Errorf("parse multipart form: %w", err)
			}
		}
		return []byte(r.Form.Encode()), nil
	}
	buf := ibytes.GetNopCloserBuffer()
	_, err := io.Copy(buf, r.Body)
	if err != nil {
		return nil, fmt.Errorf("copy from req.Body to buffer err: %w", err)
	}
	// Reset body and allow multiple reads.
	// Refer to test case: TestCoexistenceOfHTTPRPCAndNoProtocol.
	r.Body.Close()
	r.Body = buf
	return buf.Bytes(), nil
}

// updateMsg updates msg.
func (sc *ServerCodec) updateMsg(head *Header, msg codec.Msg) {
	ce := fastop.CanonicalHeaderGet(head.Request.Header, canonicalContentEncoding)
	if ce != "" {
		msg.WithCompressType(contentEncodingCompressType[ce])
	}

	// Update upstream service name.
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName("trpc.http.upserver.upservice")
	}

	// Update current service name.
	if msg.CalleeServiceName() == "" {
		msg.WithCalleeServiceName(fmt.Sprintf("trpc.http.%s.service", path.Base(os.Args[0])))
	}
}

// Decode decodes http header.
// http server transport has filled all the data of request into ctx,
// and reqBuf here is empty.
func (sc *ServerCodec) Decode(msg codec.Msg, _ []byte) ([]byte, error) {
	head := Head(msg.Context())
	if head == nil {
		return nil, errors.New("server decode missing http header in context")
	}

	reqBody, err := sc.getReqBody(head, msg)
	if err != nil {
		return nil, err
	}
	if err := sc.setReqHeaderAndUpdateMsg(head, msg); err != nil {
		return nil, err
	}

	sc.updateMsg(head, msg)
	return reqBody, nil
}

// ErrorHandler handles error of http server's response.
// By default, the error code is placed in header,
// which can be replaced by a specific implementation of user.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, e *errs.Error)

var defaultErrHandler = func(w http.ResponseWriter, _ *http.Request, e *errs.Error) {
	errMsg := strings.Replace(e.Msg, "\r", "\\r", -1)
	errMsg = strings.Replace(errMsg, "\n", "\\n", -1)

	fastop.CanonicalHeaderAdd(w.Header(), canonicalTrpcErrorMessage, errMsg)
	if e.Type == errs.ErrorTypeFramework {
		fastop.CanonicalHeaderAdd(w.Header(), canonicalTrpcFrameworkErrorCode, strconv.Itoa(int(e.Code)))
	} else {
		fastop.CanonicalHeaderAdd(w.Header(), canonicalTrpcUserFuncErrorCode, strconv.Itoa(int(e.Code)))
	}

	if code, ok := ErrsToHTTPStatus[e.Code]; ok {
		w.WriteHeader(code)
	}
}

// ResponseHandler handles data of http server's response.
// By default, the content is returned directly,
// which can replaced by a specific implementation of user.
type ResponseHandler func(w http.ResponseWriter, r *http.Request, rspBody []byte) error

var defaultRspHandler = func(w http.ResponseWriter, _ *http.Request, rspBody []byte) error {
	if len(rspBody) == 0 {
		return nil
	}
	if _, err := w.Write(rspBody); err != nil {
		return fmt.Errorf("http write response error: %s", err.Error())
	}
	return nil
}

// Encode sets http header.
// The buffer of the returned packet has been written to the response writer in header,
// no need to return rspBuf.
func (sc *ServerCodec) Encode(msg codec.Msg, rspBody []byte) (b []byte, err error) {
	head := Head(msg.Context())
	if head == nil {
		return nil, ErrEncodeMissingHeader
	}
	req := head.Request
	rsp := head.Response
	defer func() {
		if buf, ok := req.Body.(*ibytes.NopCloserBuffer); ok && buf != nil {
			ibytes.PutNopCloserBuffer(buf)
		}
	}()

	fastop.CanonicalHeaderAdd(rsp.Header(), canonicalXContentTypeOptions, "nosniff")
	ct := fastop.CanonicalHeaderGet(rsp.Header(), canonicalContentType)
	if ct == "" {
		ct = fastop.CanonicalHeaderGet(req.Header, canonicalContentType)
		if req.Method == http.MethodGet || ct == "" {
			ct = "application/json"
		}
		fastop.CanonicalHeaderAdd(rsp.Header(), canonicalContentType, ct)
	}
	if strings.Contains(ct, serializationTypeContentType[codec.SerializationTypeFormData]) {
		formDataCt := getFormDataContentType()
		fastop.CanonicalHeaderSet(rsp.Header(), canonicalContentType, formDataCt)
	}

	if len(msg.ServerMetaData()) > 0 {
		m := make(map[string]string)
		if sc.DisableEncodeTransInfoBase64 {
			for k, v := range msg.ServerMetaData() {
				m[k] = string(v)
			}
		} else {
			for k, v := range msg.ServerMetaData() {
				m[k] = base64.StdEncoding.EncodeToString(v)
			}
		}
		val, _ := codec.Marshal(codec.SerializationTypeJSON, m)
		fastop.CanonicalHeaderSet(rsp.Header(), canonicalTrpcTransInfo, string(val))
	}

	// Return packet tells client to use which decompress method.
	if t := msg.CompressType(); icodec.IsValidCompressType(t) && t != codec.CompressTypeNoop {
		fastop.CanonicalHeaderAdd(rsp.Header(), canonicalContentEncoding, compressTypeContentEncoding[t])
	}

	// 1. Handle exceptions first, as long as server returns an error,
	// the returned data will no longer be processed.
	if e := msg.ServerRspErr(); e != nil {
		if sc.ErrHandler != nil {
			sc.ErrHandler(rsp, req, e)
		}
		return
	}
	// 2. process returned data under normal case.
	if sc.RspHandler != nil {
		if err := sc.RspHandler(rsp, req, rspBody); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// ClientCodec decodes http client request.
type ClientCodec struct {
	// ErrHandler is error code handle function, which is filled into header by default.
	// Business can set this with thttp.DefaultClientCodec.ErrHandler = func(rsp, msg, body) ([]byte, error) {}.
	ErrHandler DecodeErrorHandler
}

// Encode sets metadata requested by http client.
// Client has been serialized and passed to reqBody with compress.
func (c *ClientCodec) Encode(msg codec.Msg, reqBody []byte) ([]byte, error) {
	var reqHeader *ClientReqHeader
	if h := msg.ClientReqHead(); h != nil { // User himself has set http client req header.
		httpReqHeader, ok := h.(*ClientReqHeader)
		if !ok {
			return nil, fmt.Errorf("http header must be type of *http.ClientReqHeader, current type: %T", h)
		}
		reqHeader = httpReqHeader
	} else {
		reqHeader = &ClientReqHeader{}
		msg.WithClientReqHead(reqHeader)
	}

	if reqHeader.Method == "" {
		if len(reqBody) == 0 {
			reqHeader.Method = http.MethodGet
		} else {
			reqHeader.Method = http.MethodPost
		}
	}

	var rspHeader *ClientRspHeader
	if h := msg.ClientRspHead(); h != nil { // User himself has set http client rsp header.
		header, ok := h.(*ClientRspHeader)
		if !ok {
			return nil, fmt.Errorf("http header must be type of *http.ClientRspHeader, current type: %T", h)
		}
		rspHeader = header
	} else {
		rspHeader = &ClientRspHeader{}
		msg.WithClientRspHead(rspHeader)
	}

	tryFillSSEHeaders(reqHeader, rspHeader)

	c.updateMsg(msg)
	return reqBody, nil
}

func tryFillSSEHeaders(reqHeader *ClientReqHeader, rspHeader *ClientRspHeader) {
	if rspHeader.SSEHandler == nil {
		// User has not set sse handler, do nothing.
		return
	}
	tryAddHeader(reqHeader, "Accept", "text/event-stream")
	tryAddHeader(reqHeader, "Connection", "keep-alive")
	tryAddHeader(reqHeader, "Cache-Control", "no-cache")
}

func tryAddHeader(reqHeader *ClientReqHeader, key, val string) {
	if reqHeader.Header.Get(key) != "" {
		return
	}
	reqHeader.AddHeader(key, val)
}

// The default SSECondition always returns true.
var defaultSSECondition = func(*http.Response) bool {
	return true
}

// handleResponseBody process response body with different response types.
func handleResponseBody(rspHeader *ClientRspHeader) ([]byte, error) {
	rsp := rspHeader.Response
	if rsp.Body == nil || rspHeader.ManualReadBody {
		return nil, nil
	}
	defer rsp.Body.Close()

	// If SSECondition is not implemented, set a default one.
	if rspHeader.SSECondition == nil {
		rspHeader.SSECondition = defaultSSECondition
	}

	// If SSECondition returns true and SSEHandler is implemented, process with it.
	if rspHeader.SSECondition(rsp) && rspHeader.SSEHandler != nil {
		// Handle SSE response with SSEHandler.
		if err := handleSSE(rsp.Body, rspHeader.SSEHandler); err != nil {
			return nil, fmt.Errorf("handle sse error: %w", err)
		}
		return nil, nil
	}

	// Else if ResponseHandler is implemented, process with it.
	if rspHeader.ResponseHandler != nil {
		// Handle normal response with ResponseHandler.
		if err := rspHeader.ResponseHandler.Handle(rsp); err != nil {
			return nil, fmt.Errorf("handle response error: %w", err)
		}
		return nil, nil
	}

	// Default behavior: read all the body.
	var (
		body []byte
		err  error
	)
	if body, err = io.ReadAll(rsp.Body); err != nil {
		return nil, fmt.Errorf("read all http body fail: %w", err)
	}
	// Reset body and allow multiple read.
	rsp.Body.Close()
	rsp.Body = io.NopCloser(bytes.NewReader(body))

	return body, nil
}

// DecodeErrorHandler is used to handle error in ClientCodec.Decode()
type DecodeErrorHandler func(rsp *http.Response, msg codec.Msg, body []byte) ([]byte, error)

var defaultDecodeErrHandler = func(rsp *http.Response, msg codec.Msg, body []byte) ([]byte, error) {
	if val := fastop.CanonicalHeaderGet(rsp.Header, canonicalTrpcFrameworkErrorCode); val != "" {
		i, _ := strconv.Atoi(val)
		if i != 0 {
			msg.WithClientRspErr(
				errs.NewCalleeFrameError(i, fastop.CanonicalHeaderGet(rsp.Header, canonicalTrpcErrorMessage)))
			return nil, nil
		}
	}
	if val := fastop.CanonicalHeaderGet(rsp.Header, canonicalTrpcUserFuncErrorCode); val != "" {
		i, _ := strconv.Atoi(val)
		if i != 0 {
			msg.WithClientRspErr(
				errs.New(i, fastop.CanonicalHeaderGet(rsp.Header, canonicalTrpcErrorMessage)))
			return nil, nil
		}
	}
	if rsp.StatusCode >= http.StatusMultipleChoices {
		msg.WithClientRspErr(errs.New(
			rsp.StatusCode,
			fmt.Sprintf("http client codec StatusCode: %s, body: %q", http.StatusText(rsp.StatusCode), body)))
		return nil, nil
	}
	return body, nil
}

// Decode parses metadata in http client's response.
func (c *ClientCodec) Decode(msg codec.Msg, _ []byte) ([]byte, error) {
	rspHeader, ok := msg.ClientRspHead().(*ClientRspHeader)
	if !ok {
		return nil, errors.New("rsp header must be type of *http.ClientRspHeader")
	}

	body, err := handleResponseBody(rspHeader)
	if err != nil {
		return nil, fmt.Errorf("handle response body: %w", err)
	}

	rsp := rspHeader.Response
	if val := fastop.CanonicalHeaderGet(rsp.Header, canonicalContentEncoding); val != "" {
		msg.WithCompressType(contentEncodingCompressType[val])
	} else {
		msg.WithCompressType(codec.CompressTypeNoop)
	}
	ct := fastop.CanonicalHeaderGet(rsp.Header, canonicalContentType)
	for contentType, serializationType := range contentTypeSerializationType {
		if strings.Contains(ct, contentType) {
			msg.WithSerializationType(serializationType)
			break
		}
	}
	if c.ErrHandler != nil {
		return c.ErrHandler(rsp, msg, body)
	}
	return defaultDecodeErrHandler(rsp, msg, body)
}

// updateMsg updates msg.
func (c *ClientCodec) updateMsg(msg codec.Msg) {
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.http.%s.service", path.Base(os.Args[0])))
	}
}
