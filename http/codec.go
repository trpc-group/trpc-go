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

package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
)

// Constants of header keys related to trpc.
const (
	TrpcVersion     = "trpc-version"
	TrpcCallType    = "trpc-call-type"
	TrpcMessageType = "trpc-message-type"
	TrpcRequestID   = "trpc-request-id"
	TrpcTimeout     = "trpc-timeout"
	TrpcCaller      = "trpc-caller"
	TrpcCallee      = "trpc-callee"
	TrpcTransInfo   = "trpc-trans-info"
	TrpcEnv         = "trpc-env"
	TrpcDyeingKey   = "trpc-dyeing-key"
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

// RegisterContentType registers existing serialization method to
// contentTypeSerializationType and serializationTypeContentType.
func RegisterContentType(httpContentType string, serializationType int) {
	contentTypeSerializationType[httpContentType] = serializationType
	serializationTypeContentType[serializationType] = httpContentType
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

// RegisterStatus registers trpc return code to http status.
func RegisterStatus[T errs.ErrCode](code T, httpStatus int) {
	ErrsToHTTPStatus[trpcpb.TrpcRetCode(code)] = httpStatus
}

func init() {
	codec.Register("http", DefaultServerCodec, DefaultClientCodec)
	codec.Register("http2", DefaultServerCodec, DefaultClientCodec)
	// Support no protocol file custom routing and feature isolation.
	codec.Register("http_no_protocol", DefaultNoProtocolServerCodec, DefaultClientCodec)
	codec.Register("http2_no_protocol", DefaultNoProtocolServerCodec, DefaultClientCodec)
}

var (
	// DefaultClientCodec is the default http client codec.
	DefaultClientCodec = &ClientCodec{}

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

	// DisableEncodeTransInfoBase64 indicates whether to disable encoding the transinfo value by base64.
	DisableEncodeTransInfoBase64 bool
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
	Schema  string // Examples: HTTP, HTTPS.
	Method  string
	Host    string
	Request *http.Request
	Header  http.Header
	ReqBody io.Reader
}

// AddHeader adds http header.
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
}

// ErrsToHTTPStatus maps from framework errs retcode to http status code.
var ErrsToHTTPStatus = map[trpcpb.TrpcRetCode]int{
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

// setReqHeader sets request header.
func (sc *ServerCodec) setReqHeader(head *Header, msg codec.Msg) error {
	if !sc.AutoGenTrpcHead { // Auto generates trpc head.
		return nil
	}

	trpcReq := &trpcpb.RequestProtocol{}
	msg.WithServerReqHead(trpcReq)
	msg.WithServerRspHead(trpcReq)

	trpcReq.Func = []byte(msg.ServerRPCName())
	trpcReq.ContentType = uint32(msg.SerializationType())
	trpcReq.ContentEncoding = uint32(msg.CompressType())

	if v := head.Request.Header.Get(TrpcVersion); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.Version = uint32(i)
	}
	if v := head.Request.Header.Get(TrpcCallType); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.CallType = uint32(i)
	}
	if v := head.Request.Header.Get(TrpcMessageType); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.MessageType = uint32(i)
	}
	if v := head.Request.Header.Get(TrpcRequestID); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.RequestId = uint32(i)
	}
	if v := head.Request.Header.Get(TrpcTimeout); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.Timeout = uint32(i)
		msg.WithRequestTimeout(time.Millisecond * time.Duration(i))
	}
	if v := head.Request.Header.Get(TrpcCaller); v != "" {
		trpcReq.Caller = []byte(v)
		msg.WithCallerServiceName(v)
	}
	if v := head.Request.Header.Get(TrpcCallee); v != "" {
		trpcReq.Callee = []byte(v)
		msg.WithCalleeServiceName(v)
	}

	msg.WithDyeing((trpcReq.GetMessageType() & uint32(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)

	if v := head.Request.Header.Get(TrpcTransInfo); v != "" {
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

// getReqbody gets the body of request.
func (sc *ServerCodec) getReqbody(head *Header, msg codec.Msg) ([]byte, error) {
	msg.WithCalleeMethod(head.Request.URL.Path)
	msg.WithServerRPCName(head.Request.URL.Path)

	if !sc.AutoReadBody {
		return nil, nil
	}

	var reqBody []byte
	if head.Request.Method == http.MethodGet {
		msg.WithSerializationType(codec.SerializationTypeGet)
		reqBody = []byte(head.Request.URL.RawQuery)
	} else {
		var exist bool
		msg.WithSerializationType(codec.SerializationTypeJSON)
		ct := head.Request.Header.Get("Content-Type")
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
	head.ReqBody = reqBody
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("body readAll: %w", err)
	}
	// Reset body and allow multiple reads.
	// Refer to testcase: TestCoexistenceOfHTTPRPCAndNoProtocol.
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// updateMsg updates msg.
func (sc *ServerCodec) updateMsg(head *Header, msg codec.Msg) {
	ce := head.Request.Header.Get("Content-Encoding")
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

	reqBody, err := sc.getReqbody(head, msg)
	if err != nil {
		return nil, err
	}
	if err := sc.setReqHeader(head, msg); err != nil {
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

	w.Header().Add(TrpcErrorMessage, errMsg)
	if e.Type == errs.ErrorTypeFramework {
		w.Header().Add(TrpcFrameworkErrorCode, strconv.Itoa(int(e.Code)))
	} else {
		w.Header().Add(TrpcUserFuncErrorCode, strconv.Itoa(int(e.Code)))
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
	ctKey := "Content-Type"

	rsp.Header().Add("X-Content-Type-Options", "nosniff")
	ct := rsp.Header().Get(ctKey)
	if ct == "" {
		ct = req.Header.Get(ctKey)
		if req.Method == http.MethodGet || ct == "" {
			ct = "application/json"
		}
		rsp.Header().Add(ctKey, ct)
	}
	if strings.Contains(ct, serializationTypeContentType[codec.SerializationTypeFormData]) {
		formDataCt := getFormDataContentType()
		rsp.Header().Set(ctKey, formDataCt)
	}

	if len(msg.ServerMetaData()) > 0 {
		m := make(map[string]string)
		for k, v := range msg.ServerMetaData() {
			if sc.DisableEncodeTransInfoBase64 {
				m[k] = string(v)
				continue
			}
			m[k] = base64.StdEncoding.EncodeToString(v)
		}
		val, _ := codec.Marshal(codec.SerializationTypeJSON, m)
		rsp.Header().Set("trpc-trans-info", string(val))
	}

	// Return packet tells client to use which decompress method.
	if t := msg.CompressType(); icodec.IsValidCompressType(t) && t != codec.CompressTypeNoop {
		rsp.Header().Add("Content-Encoding", compressTypeContentEncoding[t])
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
type ClientCodec struct{}

// Encode sets metadata requested by http client.
// Client has been serialized and passed to reqBody with compress.
func (c *ClientCodec) Encode(msg codec.Msg, reqBody []byte) ([]byte, error) {
	var reqHeader *ClientReqHeader
	if msg.ClientReqHead() != nil { // User himself has set http client req header.
		httpReqHeader, ok := msg.ClientReqHead().(*ClientReqHeader)
		if !ok {
			return nil, errors.New("http header must be type of *http.ClientReqHeader")
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

	if msg.ClientRspHead() != nil { // User himself has set http client rsp header.
		_, ok := msg.ClientRspHead().(*ClientRspHeader)
		if !ok {
			return nil, errors.New("http header must be type of *http.ClientRspHeader")
		}
	} else {
		msg.WithClientRspHead(&ClientRspHeader{})
	}

	c.updateMsg(msg)
	return reqBody, nil
}

// Decode parses metadata in http client's response.
func (c *ClientCodec) Decode(msg codec.Msg, _ []byte) ([]byte, error) {
	rspHeader, ok := msg.ClientRspHead().(*ClientRspHeader)
	if !ok {
		return nil, errors.New("rsp header must be type of *http.ClientRspHeader")
	}

	var (
		body []byte
		err  error
	)
	rsp := rspHeader.Response
	if rsp.Body != nil && !rspHeader.ManualReadBody {
		defer rsp.Body.Close()
		if body, err = io.ReadAll(rsp.Body); err != nil {
			return nil, fmt.Errorf("readall http body fail: %w", err)
		}
		// Reset body and allow multiple read.
		rsp.Body.Close()
		rsp.Body = io.NopCloser(bytes.NewReader(body))
	}

	if val := rsp.Header.Get("Content-Encoding"); val != "" {
		msg.WithCompressType(contentEncodingCompressType[val])
	}
	ct := rsp.Header.Get("Content-Type")
	for contentType, serializationType := range contentTypeSerializationType {
		if strings.Contains(ct, contentType) {
			msg.WithSerializationType(serializationType)
			break
		}
	}
	if val := rsp.Header.Get(TrpcFrameworkErrorCode); val != "" {
		i, _ := strconv.Atoi(val)
		if i != 0 {
			e := &errs.Error{
				Type: errs.ErrorTypeCalleeFramework,
				Code: trpcpb.TrpcRetCode(i),
				Desc: "trpc",
				Msg:  rsp.Header.Get(TrpcErrorMessage),
			}
			msg.WithClientRspErr(e)
			return nil, nil
		}
	}
	if val := rsp.Header.Get(TrpcUserFuncErrorCode); val != "" {
		i, _ := strconv.Atoi(val)
		if i != 0 {
			msg.WithClientRspErr(errs.New(i, rsp.Header.Get(TrpcErrorMessage)))
			return nil, nil
		}
	}
	if rsp.StatusCode >= http.StatusMultipleChoices {
		e := &errs.Error{
			Type: errs.ErrorTypeBusiness,
			Code: trpcpb.TrpcRetCode(rsp.StatusCode),
			Desc: "http",
			Msg:  fmt.Sprintf("http client codec StatusCode: %s, body: %q", http.StatusText(rsp.StatusCode), body),
		}
		msg.WithClientRspErr(e)
		return nil, nil
	}
	return body, nil
}

// updateMsg updates msg.
func (c *ClientCodec) updateMsg(msg codec.Msg) {
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.http.%s.service", path.Base(os.Args[0])))
	}
}
