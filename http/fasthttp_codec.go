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
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
)

func init() {
	codec.Register(protocol.FastHTTP, DefaultFastHTTPServerCodec, DefaultFastHTTPClientCodec)
	codec.Register(protocol.FastHTTPNoProtocol, DefaultFastHTTPNoProtocolServerCodec, DefaultFastHTTPClientCodec)
}

var (
	// DefaultFastHTTPClientCodec is the default fasthttp client side codec.
	DefaultFastHTTPClientCodec = &FastHTTPClientCodec{}

	// DefaultFastHTTPServerCodec is the default fasthttp server side codec.
	DefaultFastHTTPServerCodec = &FastHTTPServerCodec{
		AutoGenTrpcHead:              true,
		ErrHandler:                   defaultFastHTTPErrHandler,
		RspHandler:                   defaultFastHTTPRspHandler,
		AutoReadBody:                 true,
		DisableEncodeTransInfoBase64: false,
		POSTOnly:                     false,
	}

	// DefaultFastHTTPNoProtocolServerCodec is the default fasthttp_no_protocol server side codec.
	DefaultFastHTTPNoProtocolServerCodec = &FastHTTPServerCodec{
		AutoGenTrpcHead:              true,
		ErrHandler:                   defaultFastHTTPErrHandler,
		RspHandler:                   defaultFastHTTPRspHandler,
		AutoReadBody:                 false,
		DisableEncodeTransInfoBase64: false,
		POSTOnly:                     false,
	}
)

// ErrEncodeMissingRequestCtx defines error used for special handling
// in transport when ctx lost lost requestCtx information.
var ErrEncodeMissingRequestCtx = errors.New("trpc/fasthttp: server encode missing fasthttp requestCtx in context")

// FastHTTPClientReqHeader encapsulates fasthttp client context.
// Setting ClientReqHeader is not allowed when NewFastHTTPClientProxy is waiting for the init of Client.
// FastHTTPClientReqHeader is needed for each RPC.
type FastHTTPClientReqHeader struct {
	Request *fasthttp.Request
	Scheme  string // Examples: HTTP, HTTPS.
	Method  string
	// Host directly sets the final host field in the fasthttp.Request.
	Host string
	// DecorateRequest will be called right before client.DoRedirects(req, rsp, cnt) to
	// allow users to make final custom modifications to the fasthttp request.
	// Users can set the headers of req by configuring this field.
	DecorateRequest func(*fasthttp.Request) *fasthttp.Request
}

// FastHTTPRspHandler is an interface for users to implement fasthttp response callbacks.
type FastHTTPRspHandler interface {
	// Handle handles fasthttp response.
	// If the returned error is non-nil, the framework will
	// abort the reading of the fasthttp connection.
	Handle(*fasthttp.Response) error
}

// FastHTTPClientRspHeader encapsulates the context returned by fasthttp client response.
type FastHTTPClientRspHeader struct {
	Response *fasthttp.Response

	// ManualReadBody is used to control whether to read fasthttp response manually
	// (not read automatically by the framework).
	// Set it to true so that user can read data directly from Response.Body manually.
	// The default value is false.
	ManualReadBody bool

	// ResponseHandler is an interface that the framework will invoke
	// if SSECondition returns false OR SSEHandler is not defined.
	// If ResponseHandler is provided by the user, the framework will automatically
	// read the fasthttp response body and invoke the ResponseHandler for each response.
	ResponseHandler FastHTTPRspHandler

	// SSECondition is a function that users must implement to determine
	// whether to call server-sent event (SSE) message callbacks.
	// If SSECondition returns true AND SSEHandler is defined, the framework will
	// call the SSEHandler for each SSE event in sequence.
	SSECondition func(*fasthttp.Response) bool

	// SSEHandler is an interface that users must implement to handle
	// server-sent event (SSE) message callbacks.
	// When this field is provided by the user, the framework will automatically
	// add the following headers to the request, if they are not already present:
	//
	//  "Accept": "text/event-stream"
	//  "Connection": "keep-alive"
	//  "Cache-Control": "no-cache"
	//
	// The framework will automatically parse the fasthttp response into SSE events
	// and invoke the SSEHandler for each SSE event in sequence.
	// If any SSEHandler returns an error, the process will be halted and the
	// error will be returned.
	// The parsing of SSE events will continue until an io.EOF is encountered
	// in the reading of the fasthttp response body.
	SSEHandler SSEHandler
}

// FastHTTPServerCodec is the encoder/decoder for fasthttp server.
type FastHTTPServerCodec struct {
	// AutoGenTrpcHead converts trpc header automatically.
	// Auto conversion could be enabled by setting AutoGenTrpcHead true.
	// DefaultFastHTTPServerCodec.AutoGenTrpcHead is true.
	// DefaultFastHTTPNoProtocolServerCodec.AutoGenTrpcHead is true.
	AutoGenTrpcHead bool

	// ErrHandler is error code handle function, which is filled into header by default.
	// Business can set this with ErrHandler = func(requestCtx, err) {}.
	ErrHandler FastHTTPErrorHandler

	// RspHandler returns the data handle function. By default, data is returned directly.
	// Business can set this with RspHandler = func(requestCtx, rspBody) {}
	// to shape returned data.
	RspHandler FastHTTPResponseHandler

	// AutoReadBody reads fasthttp request body automatically.
	// DefaultFastHTTPServerCodec.AutoReadBody is true.
	// DefaultFastHTTPNoProtocolServerCodec.AutoReadBody is false.
	AutoReadBody bool

	// DisableEncodeTransInfoBase64 indicates whether to disable encoding the transinfo value by base64.
	// DefaultFastHTTPServerCodec.DisableEncodeTransInfoBase64 is false.
	// DefaultFastHTTPNoProtocolServerCodec.DisableEncodeTransInfoBase64 is false.
	DisableEncodeTransInfoBase64 bool

	// POSTOnly determines whether to process only requests that use the POST method.
	// This is commonly used in an FastHTTP RPC server to allow only the POST method to be accepted,
	// instead of allowing both the POST and GET methods.
	// DefaultFastHTTPServerCodec.POSTOnly is false.
	// DefaultFastHTTPNoProtocolServerCodec.POSTOnly is false.
	POSTOnly bool
}

// FastHTTPErrorHandler handles error of fasthttp server's response.
// By default, the error code is placed in header,
// which can be replaced by a specific implementation of user.
type FastHTTPErrorHandler func(requestCtx *fasthttp.RequestCtx, e *errs.Error)

var defaultFastHTTPErrHandler = func(requestCtx *fasthttp.RequestCtx, e *errs.Error) {
	// Replace(-1) may be better than ReplaceAll.
	errMsg := strings.Replace(e.Msg, "\r", "\\r", -1)
	errMsg = strings.Replace(errMsg, "\n", "\\n", -1)

	requestCtx.Response.Header.Add(canonicalTrpcErrorMessage, errMsg)
	if e.Type == errs.ErrorTypeFramework {
		requestCtx.Response.Header.Add(canonicalTrpcFrameworkErrorCode, strconv.Itoa(int(e.Code)))
	} else {
		requestCtx.Response.Header.Add(canonicalTrpcUserFuncErrorCode, strconv.Itoa(int(e.Code)))
	}
	if code, ok := ErrsToHTTPStatus[e.Code]; ok {
		requestCtx.SetStatusCode(code)
	}
}

// FastHTTPResponseHandler handles data of fasthttp server's response.
// By default, the content is returned directly,
// which can be replaced by a specific implementation of user.
type FastHTTPResponseHandler func(requestCtx *fasthttp.RequestCtx, rspBody []byte) error

var defaultFastHTTPRspHandler = func(requestCtx *fasthttp.RequestCtx, rspBody []byte) error {
	if len(rspBody) != 0 {
		// SetBodyRaw sets response body, but without copying it.
		// From this point onward the body argument must not be changed.
		// User can define their own FastHTTPResponseHandler with SetBody() or SetBodyStream() or anything else.
		requestCtx.Response.SetBodyRaw(rspBody)
	}
	return nil
}

// handleContentTypeForCompatibility is used to address the inconsistency in the behavior
// of the ContentType header in the response (rsp) between fasthttp and net/http.
// For response headers, the ContentType logic differs between http and fasthttp,
// http defaults to returning "", while fasthttp defaults to return []byte("text/plain; charset=utf-8").
// Strangely, for request headers, both are consistent, returning "".
func handleContentTypeForCompatibility(req *fasthttp.Request, rsp *fasthttp.Response) {
	const defaultRspContentTypeForHTTP = ""
	const defaultRspContentTypeForFastHTTP = "text/plain; charset=utf-8"
	ct := string(rsp.Header.Peek(canonicalContentType))
	if ct == defaultRspContentTypeForFastHTTP {
		ct = string(req.Header.Peek(canonicalContentType))
		if string(req.Header.Method()) == fasthttp.MethodGet || ct == "" {
			ct = "application/json"
		}
		rsp.Header.Add(canonicalContentType, ct)
	}
	// The Content-Type header may contain additional information besides
	// the MIME type, such as character set encoding.
	// Direct comparison using equal may fail due to these additional details.
	if strings.Contains(ct, serializationTypeContentType[codec.SerializationTypeFormData]) {
		formDataCt := getFormDataContentType()
		rsp.Header.Set(canonicalContentType, formDataCt)
	}
}

// Encode packs the body into binary buffer.
// It implements codec.Codec interface for FastHTTPServerCodec.
// server: Encode(msg, rspBody) (rspBuffer, err)
func (sc *FastHTTPServerCodec) Encode(msg codec.Msg, rspBody []byte) ([]byte, error) {
	requestCtx := RequestCtx(msg.Context())
	if requestCtx == nil {
		return nil, ErrEncodeMissingRequestCtx
	}

	req := &requestCtx.Request
	rsp := &requestCtx.Response

	// nosniff is a security-related response header used to prevent browsers from MIME type sniffing,
	// thereby reducing the risk of cross-site scripting attacks and content injection attacks.
	// By setting this header, the security of the application can be enhanced.
	rsp.Header.Add(canonicalXContentTypeOptions, "nosniff")

	// For response headers, the ContentType logic differs between http and fasthttp,
	// use handleContentTypeForCompatibility to handle difference.
	handleContentTypeForCompatibility(req, rsp)

	// Return packet tells client to use which decompress method.
	if t := msg.CompressType(); icodec.IsValidCompressType(t) && t != codec.CompressTypeNoop {
		rsp.Header.Set(canonicalContentEncoding, compressTypeContentEncoding[t])
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
		val, err := codec.Marshal(codec.SerializationTypeJSON, m)
		if err != nil {
			return nil, err
		}
		rsp.Header.SetBytesV(canonicalTrpcTransInfo, val)
	}

	// 1. Handle exceptions first, as long as server returns an error,
	// the returned data will no longer be processed.
	if e := msg.ServerRspErr(); e != nil {
		if sc.ErrHandler != nil {
			sc.ErrHandler(requestCtx, e)
		}
		return nil, nil
	}

	// 2. process returned data under normal case.
	if sc.RspHandler != nil {
		if err := sc.RspHandler(requestCtx, rspBody); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// Decode unpacks the body from binary buffer.
// It implements codec.Codec interface for FastHTTPServerCodec.
// server: Decode(msg, reqBuffer) (reqBody, err)
func (sc *FastHTTPServerCodec) Decode(msg codec.Msg, _ []byte) ([]byte, error) {
	requestCtx := RequestCtx(msg.Context())
	if requestCtx == nil {
		return nil, errors.New("server decode missing fasthttp requestCtx in context")
	}

	msg.WithCalleeMethod(string(requestCtx.Path()))
	msg.WithServerRPCName(string(requestCtx.Path()))

	reqBody, err := sc.getReqBody(requestCtx, msg)
	if err != nil {
		return nil, err
	}

	if err := sc.setReqHeader(requestCtx, msg); err != nil {
		return nil, err
	}

	sc.updateMsg(requestCtx, msg)
	return reqBody, nil
}

// getReqBody gets the body of request and update the msg.
func (sc *FastHTTPServerCodec) getReqBody(
	requestCtx *fasthttp.RequestCtx,
	msg codec.Msg,
) ([]byte, error) {
	if !sc.AutoReadBody {
		return nil, nil
	}

	if sc.POSTOnly && string(requestCtx.Method()) != fasthttp.MethodPost {
		return nil, fmt.Errorf("server codec only allows POST method request, the current method is %s",
			string(requestCtx.Method()))
	}

	// The reqBody for GET is the QueryArgs.
	if string(requestCtx.Method()) == fasthttp.MethodGet {
		msg.WithSerializationType(codec.SerializationTypeGet)
		return requestCtx.URI().QueryString(), nil
	}

	// SerializationType is JSON by default.
	msg.WithSerializationType(codec.SerializationTypeJSON)
	ct := string(requestCtx.Request.Header.Peek(canonicalContentType))
	for contentType, serializationType := range contentTypeSerializationType {
		if !strings.Contains(ct, contentType) {
			continue
		}
		msg.WithSerializationType(serializationType)
		return getBodyForFastHTTP(ct, requestCtx)
	}

	return nil, nil
}

// getBodyForFastHTTP handles FormData specially,
// while for others it directly returns requestCtx.Request.Body().
func getBodyForFastHTTP(ct string, requestCtx *fasthttp.RequestCtx) ([]byte, error) {
	if !strings.Contains(ct, serializationTypeContentType[codec.SerializationTypeFormData]) {
		return requestCtx.Request.Body(), nil
	}

	// Fail fast.
	multipartForm, err := requestCtx.MultipartForm()
	if err != nil {
		return nil, err
	}

	// Acquire Args is for simplicity rather than efficiency.
	// Directly call args.QueryString() instead of handling it manually.
	args := fasthttp.AcquireArgs()
	defer fasthttp.ReleaseArgs(args)

	requestCtx.QueryArgs().VisitAll(func(key, value []byte) {
		args.AddBytesKV(key, value)
	})

	requestCtx.PostArgs().VisitAll(func(key, value []byte) {
		args.AddBytesKV(key, value)
	})

	for k, vs := range multipartForm.Value {
		for _, v := range vs {
			args.Add(k, v)
		}
	}

	return args.QueryString(), nil
}

// setReqHeader sets ServerReqHead according to the relative trpc-field in requestCtx.
func (sc *FastHTTPServerCodec) setReqHeader(requestCtx *fasthttp.RequestCtx, msg codec.Msg) error {
	// Auto generates trpc head is disabled, just return nil.
	if !sc.AutoGenTrpcHead {
		return nil
	}

	trpcReq := &trpc.RequestProtocol{}
	msg.WithServerReqHead(trpcReq)
	msg.WithServerRspHead(trpcReq)

	trpcReq.Func = []byte(msg.ServerRPCName())
	trpcReq.ContentType = uint32(msg.SerializationType())
	trpcReq.ContentEncoding = uint32(msg.CompressType())

	req := &requestCtx.Request
	if v := string(req.Header.Peek(canonicalTrpcVersion)); v != "" {
		version, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		trpcReq.Version = uint32(version)
	}

	if v := string(req.Header.Peek(canonicalTrpcCallType)); v != "" {
		callType, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		trpcReq.CallType = uint32(callType)
	}

	if v := string(req.Header.Peek(canonicalTrpcMessageType)); v != "" {
		messageType, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		trpcReq.MessageType = uint32(messageType)
	}

	if v := string(req.Header.Peek(canonicalTrpcRequestID)); v != "" {
		requestId, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		trpcReq.RequestId = uint32(requestId)
	}

	if v := string(req.Header.Peek(canonicalTrpcTimeout)); v != "" {
		timeout, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		trpcReq.Timeout = uint32(timeout)
		msg.WithRequestTimeout(time.Millisecond * time.Duration(timeout))
	}

	if method := string(req.Header.Peek(canonicalTrpcCallerMethod)); method != "" {
		msg.WithCallerMethod(method)
	}

	if caller := req.Header.Peek(canonicalTrpcCaller); len(caller) != 0 {
		trpcReq.Caller = caller
		msg.WithCallerServiceName(string(caller))
	}

	if callee := req.Header.Peek(canonicalTrpcCallee); len(callee) != 0 {
		trpcReq.Callee = callee
		msg.WithCalleeServiceName(string(callee))
	}

	msg.WithDyeing((trpcReq.GetMessageType() & uint32(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)

	if v := string(req.Header.Peek(canonicalTrpcTransInfo)); v != "" {
		transInfo, err := unmarshalTransInfo(msg, v)
		if err != nil {
			return err
		}
		trpcReq.TransInfo = transInfo
	}
	return nil
}

// updateMsg updates msg according to requestCtx.
func (sc *FastHTTPServerCodec) updateMsg(requestCtx *fasthttp.RequestCtx, msg codec.Msg) {
	req := &requestCtx.Request
	if v := string(req.Header.Peek(canonicalContentEncoding)); v != "" {
		msg.WithCompressType(contentEncodingCompressType[v])
	}

	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName("trpc.fasthttp.upserver.upservice")
	}

	if msg.CalleeServiceName() == "" {
		msg.WithCalleeServiceName(fmt.Sprintf("trpc.fasthttp.%s.service", path.Base(os.Args[0])))
	}
}

// FastHTTPClientCodec is the fasthttp client side codec.
type FastHTTPClientCodec struct {
	// ErrHandler is error code handle function, which is filled into header by default. Business can
	// set this with thttp.DefaultFastHTTPClientCodec.ErrHandler = func(rsp, msg, body) ([]byte, error) {}.
	ErrHandler FastHTTPDecodeErrorHandler
}

// Encode sets metadata requested by fasthttp client and packs the body into binary buffer.
// Client has been serialized and passed to reqBody with compress.
// It implements codec.Codec interface for FastHTTPClientCodec.
// client: Encode(msg, reqBody)(request-buffer, err)
func (c *FastHTTPClientCodec) Encode(msg codec.Msg, reqBody []byte) ([]byte, error) {
	var reqHeader *FastHTTPClientReqHeader
	if h := msg.ClientReqHead(); h != nil {
		fastHTTPReqHeader, ok := h.(*FastHTTPClientReqHeader)
		if !ok {
			return nil, fmt.Errorf("fasthttp header must be type of *FastHTTPClientReqHeader, current type: %T", h)
		}
		reqHeader = fastHTTPReqHeader
	} else {
		reqHeader = &FastHTTPClientReqHeader{}
		msg.WithClientReqHead(reqHeader)
	}

	var rspHeader *FastHTTPClientRspHeader
	if h := msg.ClientRspHead(); h != nil {
		fastHTTPRspHeader, ok := h.(*FastHTTPClientRspHeader)
		if !ok {
			return nil, fmt.Errorf("fasthttp header must be type of *FastHTTPClientRspHeader, current type: %T", h)
		}
		rspHeader = fastHTTPRspHeader
	} else {
		rspHeader = &FastHTTPClientRspHeader{}
		msg.WithClientRspHead(rspHeader)
	}

	// Align with thttp.
	if reqHeader.Method == "" {
		if len(reqBody) == 0 {
			reqHeader.Method = fasthttp.MethodGet
		} else {
			reqHeader.Method = fasthttp.MethodPost
		}
	}

	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.fasthttp.%s.service", path.Base(os.Args[0])))
	}

	if rspHeader.SSEHandler == nil {
		return reqBody, nil
	}

	req := reqHeader.Request
	if len(req.Header.Peek(fasthttp.HeaderAccept)) == 0 {
		req.Header.Add(fasthttp.HeaderAccept, "text/event-stream")
	}

	if len(req.Header.Peek(fasthttp.HeaderConnection)) == 0 {
		req.Header.Add(fasthttp.HeaderConnection, "keep-alive")
	}

	if len(req.Header.Peek(fasthttp.HeaderCacheControl)) == 0 {
		req.Header.Add(fasthttp.HeaderCacheControl, "no-cache")
	}

	return reqBody, nil
}

// FastHTTPDecodeErrorHandler is used to handle error in FastHTTPClientCodec.Decode()
type FastHTTPDecodeErrorHandler func(rsp *fasthttp.Response, msg codec.Msg, body []byte) ([]byte, error)

var defaultFastHTTPDecodeErrHandler = func(rsp *fasthttp.Response, msg codec.Msg, body []byte) ([]byte, error) {
	if fec := string(rsp.Header.Peek(canonicalTrpcFrameworkErrorCode)); fec != "" {
		frameworkErrcode, err := strconv.Atoi(fec)
		if err != nil {
			return nil, err
		}
		if frameworkErrcode != 0 {
			msg.WithClientRspErr(
				errs.NewCalleeFrameError(
					frameworkErrcode,
					string(rsp.Header.Peek(canonicalTrpcErrorMessage)),
				),
			)
			return nil, nil
		}
	}

	if uec := string(rsp.Header.Peek(canonicalTrpcUserFuncErrorCode)); uec != "" {
		userFuncErrcode, err := strconv.Atoi(uec)
		if err != nil {
			return nil, err
		}
		if userFuncErrcode != 0 {
			msg.WithClientRspErr(
				errs.New(
					userFuncErrcode,
					string(rsp.Header.Peek(canonicalTrpcErrorMessage)),
				),
			)
			return nil, nil
		}
	}

	// If rsp.StatusCode() >= 300, tfasthttp will invoke msg.WithClientRspErr.
	// Align with thttp.
	if rsp.StatusCode() >= fasthttp.StatusMultipleChoices {
		msg.WithClientRspErr(
			errs.New(rsp.StatusCode(), fmt.Sprintf("fasthttp client codec StatusCode: %s, body: %q",
				fasthttp.StatusMessage(rsp.StatusCode()), rsp.Body()),
			),
		)
		return nil, nil
	}
	return body, nil
}

// Decode unpacks the body from binary buffer and parses metadata in fasthttp response.
// It implements codec.Codec interface for FastHTTPClientCodec.
// client: Decode(msg, rspBuffer) (rspBody, err)
func (c *FastHTTPClientCodec) Decode(msg codec.Msg, _ []byte) ([]byte, error) {
	rspHeader, ok := msg.ClientRspHead().(*FastHTTPClientRspHeader)
	if !ok {
		return nil, fmt.Errorf("fasthttp header must be type of *fasthttp.ClientRspHeader, current type: %T", rspHeader)
	}

	body, err := handleFastHTTPResponseBody(rspHeader)
	if err != nil {
		return nil, fmt.Errorf("handle response body: %w", err)
	}

	rsp := rspHeader.Response
	if v := string(rsp.Header.Peek(canonicalContentEncoding)); v != "" {
		msg.WithCompressType(contentEncodingCompressType[v])
	}

	if ct := string(rsp.Header.Peek(canonicalContentType)); ct != "" {
		for contentType, serializationType := range contentTypeSerializationType {
			if strings.Contains(ct, contentType) {
				msg.WithSerializationType(serializationType)
				break
			}
		}
	}
	if c.ErrHandler != nil {
		return c.ErrHandler(rsp, msg, body)
	}
	return defaultFastHTTPDecodeErrHandler(rsp, msg, body)
}

// The default FastHTTPSSECondition always returns true.
var defaultFastHTTPSSECondition = func(*fasthttp.Response) bool {
	return true
}

// handleFastHTTPResponseBody process response body with different response types.
func handleFastHTTPResponseBody(rspHeader *FastHTTPClientRspHeader) ([]byte, error) {
	rsp := rspHeader.Response
	// Judge for ManualReadBody.
	if len(rsp.Body()) == 0 || rspHeader.ManualReadBody {
		return nil, nil
	}

	// If SSECondition is not implemented, set a default one.
	if rspHeader.SSECondition == nil {
		rspHeader.SSECondition = defaultFastHTTPSSECondition
	}

	// If SSECondition returns true and SSEHandler is implemented, process with it.
	if rspHeader.SSECondition(rsp) && rspHeader.SSEHandler != nil {
		// Handle SSE response with SSEHandler.
		if err := handleSSE(bytes.NewReader(rsp.Body()), rspHeader.SSEHandler); err != nil {
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

	return rsp.Body(), nil
}

type requestCtxKey struct{}

// WithRequestCtx sets fasthttp requestCtx in context.
func WithRequestCtx(ctx context.Context, requestCtx *fasthttp.RequestCtx) context.Context {
	return context.WithValue(ctx, requestCtxKey{}, requestCtx)
}

// RequestCtx gets the corresponding fasthttp requestCtx from context.
func RequestCtx(ctx context.Context) *fasthttp.RequestCtx {
	if requestCtx, ok := ctx.Value(requestCtxKey{}).(*fasthttp.RequestCtx); ok {
		return requestCtx
	}
	return nil
}
