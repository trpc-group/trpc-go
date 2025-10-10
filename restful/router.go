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

package restful

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/internal/http/fastop"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/restful/dat"
)

// Router is restful router.
type Router struct {
	opts        *Options
	transcoders map[string][]*transcoder
}

// NewRouter creates a Router.
func NewRouter(opts ...Option) *Router {
	o := Options{
		ErrorHandler:                 DefaultErrorHandler,
		HeaderMatcher:                DefaultHeaderMatcher,
		RespSerializerGetter:         DefaultRespSerializerGetter,
		ResponseHandler:              DefaultResponseHandler,
		FastHTTPErrHandler:           DefaultFastHTTPErrorHandler,
		FastHTTPHeaderMatcher:        DefaultFastHTTPHeaderMatcher,
		FastHTTPRespSerializerGetter: DefaultFastHTTPRespSerializerGetter,
		FastHTTPRespHandler:          DefaultFastHTTPRespHandler,
		methods:                      make(map[string]*methodOptions),
	}
	for _, opt := range opts {
		opt(&o)
	}
	o.rebuildHeaderMatcher()

	return &Router{
		opts:        &o,
		transcoders: make(map[string][]*transcoder),
	}
}

var (
	routers    = make(map[string]http.Handler) // tRPC service name -> Router
	routerLock sync.RWMutex
)

var (
	fasthttpRouters    = make(map[string]fasthttp.RequestHandler) // tRPC service name -> Router
	fasthttpRouterLock sync.RWMutex
)

// RegisterRouter registers a Router which corresponds to a tRPC Service.
func RegisterRouter(name string, router http.Handler) {
	routerLock.Lock()
	routers[name] = router
	routerLock.Unlock()
}

// MustRegisterRouter registers a Router which corresponds to a tRPC Service.
// It will panic if the router has been registered.
//
// In most cases, the framework uses the init + RegisterRouter method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegisterRouter to forcibly register a component 'xxx', while the framework
// uses init + RegisterRouter to register another component 'yyy', conflicts may occur. If the init function
// for MustRegisterRouter is executed before the conflicting init function, MustRegisterRouter might not raise an
// error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegisterRouter and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegisterRouter(name string, router http.Handler) {
	if r := GetRouter(name); r != nil {
		panic("router already registered: " + name)
	}
	RegisterRouter(name, router)
}

// RegisterFasthttpRouter registers a fasthttp router which corresponds to a tRPC Service.
func RegisterFasthttpRouter(name string, router fasthttp.RequestHandler) {
	fasthttpRouterLock.Lock()
	fasthttpRouters[name] = router
	fasthttpRouterLock.Unlock()
}

// MustRegisterFasthttpRouter registers a fasthttp router which corresponds to a tRPC Service.
// It will panic if the router has been registered.
func MustRegisterFasthttpRouter(name string, router fasthttp.RequestHandler) {
	if r := GetFasthttpRouter(name); r != nil {
		panic("fasthttp router already registered: " + name)
	}
	RegisterFasthttpRouter(name, router)
}

// GetRouter returns a Router which corresponds to a tRPC Service.
func GetRouter(name string) http.Handler {
	routerLock.RLock()
	router := routers[name]
	routerLock.RUnlock()
	return router
}

// GetFasthttpRouter returns a fasthttp router which corresponds to a tRPC Service.
func GetFasthttpRouter(name string) fasthttp.RequestHandler {
	fasthttpRouterLock.RLock()
	router := fasthttpRouters[name]
	fasthttpRouterLock.RUnlock()
	return router
}

// ProtoMessage is alias of proto.Message.
type ProtoMessage proto.Message

// Initializer initializes a ProtoMessage.
type Initializer func() ProtoMessage

// BodyLocator locates which fields of the proto message would be
// populated according to HttpRule body.
type BodyLocator interface {
	Body() string
	Locate(ProtoMessage) interface{}
}

// ResponseBodyLocator locates which fields of the proto message would be marshaled
// according to HttpRule response_body.
type ResponseBodyLocator interface {
	ResponseBody() string
	Locate(ProtoMessage) interface{}
}

// HandleFunc is tRPC method handle function.
type HandleFunc func(svc interface{}, ctx context.Context, reqBody interface{}) (interface{}, error)

// Handler is tRPC method handle function.
// Deprecated
type Handler func(svc interface{}, ctx context.Context, reqBody, rspBody interface{}) error

// ExtractFilterFunc extracts tRPC service filter chain.
type ExtractFilterFunc func() filter.ServerChain

// Binding is the binding of tRPC method and HttpRule.
type Binding struct {
	Name         string
	Input        Initializer
	Output       Initializer
	Handler      Handler // Deprecated
	Filter       HandleFunc
	HTTPMethod   string
	Pattern      *Pattern
	Body         BodyLocator
	ResponseBody ResponseBodyLocator
}

// AddBinding creates a new Binding.
// Deprecated: use AddImplBinding instead.
func (r *Router) AddBinding(binding *Binding) error {
	return r.AddImplBinding(binding, r.opts.ServiceImpl)
}

// AddImplBinding creates a new binding with a specified service implementation.
func (r *Router) AddImplBinding(binding *Binding, serviceImpl interface{}) error {
	tr, err := r.newTranscoder(binding, serviceImpl)
	if err != nil {
		return fmt.Errorf("new transcoder during add impl binding: %w", err)
	}
	// add transcoder
	r.transcoders[binding.HTTPMethod] = append(r.transcoders[binding.HTTPMethod], tr)
	return nil
}

func (r *Router) newTranscoder(binding *Binding, serviceImpl interface{}) (*transcoder, error) {
	// for old stub compatibility
	// Deprecated
	if binding.Handler != nil && binding.Filter == nil {
		binding.Filter = convertToServerFilter(binding.Handler, binding.Output)
	}

	if binding.Output == nil {
		binding.Output = func() ProtoMessage { return &emptypb.Empty{} }
	}

	// create a transcoder
	tr := &transcoder{
		name:                 binding.Name,
		input:                binding.Input,
		output:               binding.Output,
		handler:              binding.Filter,
		httpMethod:           binding.HTTPMethod,
		pat:                  binding.Pattern,
		body:                 binding.Body,
		respBody:             binding.ResponseBody,
		router:               r,
		discardUnknownParams: r.opts.DiscardUnknownParams,
		serviceImpl:          serviceImpl,
	}

	// create a dat, filter all fields specified in HttpRule
	var fps [][]string
	if fromPat := binding.Pattern.FieldPaths(); fromPat != nil {
		fps = append(fps, fromPat...)
	}
	if binding.Body != nil {
		if fromBody := binding.Body.Body(); fromBody != "" && fromBody != "*" {
			fps = append(fps, strings.Split(fromBody, "."))
		}
	}
	if len(fps) > 0 {
		doubleArrayTrie, err := dat.Build(fps)
		if err != nil {
			return nil, fmt.Errorf("failed to build dat: %w", err)
		}
		tr.dat = doubleArrayTrie
	}
	return tr, nil
}

// Deprecated
func convertToServerFilter(h Handler, output Initializer) HandleFunc {
	return func(svc interface{}, ctx context.Context, reqBody interface{}) (interface{}, error) {
		rspBody := output()
		err := h(svc, ctx, reqBody, rspBody)
		return rspBody, err
	}
}

// ctxForCompatibility is used only for compatibility with thttp.
var ctxForCompatibility func(context.Context, http.ResponseWriter, *http.Request) context.Context

// SetCtxForCompatibility is used only for compatibility with thttp.
func SetCtxForCompatibility(f func(context.Context, http.ResponseWriter, *http.Request) context.Context) {
	ctxForCompatibility = f
}

// HeaderMatcher matches http request header to tRPC Stub Context.
type HeaderMatcher func(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	serviceName, methodName string,
) (context.Context, error)

// DefaultHeaderMatcher is the default HeaderMatcher.
var DefaultHeaderMatcher = func(
	ctx context.Context,
	w http.ResponseWriter,
	req *http.Request,
	serviceName, methodName string,
) (context.Context, error) {
	// Noted: it's better to do the same thing as withNewMessage.
	return withNewMessage(ctx, serviceName, methodName), nil
}

// withNewMessage create a new codec.Msg, put it into ctx,
// and set target service name and method name.
func withNewMessage(ctx context.Context, serviceName, methodName string) context.Context {
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithServerRPCName(methodName)
	msg.WithCalleeServiceName(serviceName)
	msg.WithSerializationType(codec.SerializationTypePB)
	return ctx
}

// CustomResponseHandler is the custom response handler.
type CustomResponseHandler func(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	resp proto.Message,
	body []byte,
) error

var httpStatusKey = "t-http-status"

// SetStatusCodeOnSucceed sets status code on succeed, should be 2XX.
// It's not supposed to call this function but use WithStatusCode in restful/errors.go
// to set status code on error.
func SetStatusCodeOnSucceed(ctx context.Context, code int) {
	msg := codec.Message(ctx)
	metadata := msg.ServerMetaData()
	if metadata == nil {
		metadata = codec.MetaData{}
	}
	metadata[httpStatusKey] = []byte(strconv.Itoa(code))
	msg.WithServerMetaData(metadata)
}

// GetStatusCodeOnSucceed returns status code on succeed.
// SetStatusCodeOnSucceed must be called first in tRPC method.
func GetStatusCodeOnSucceed(ctx context.Context) int {
	if metadata := codec.Message(ctx).ServerMetaData(); metadata != nil {
		if buf, ok := metadata[httpStatusKey]; ok {
			if code, err := strconv.Atoi(string(buf)); err == nil {
				return code
			}
		}
	}
	return http.StatusOK
}

// DefaultResponseHandler is the default CustomResponseHandler.
var DefaultResponseHandler = func(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	resp proto.Message,
	body []byte,
) error {
	// compress
	var writer io.Writer = w

	if c := compressor(r.Header[headerAcceptEncoding]); c != nil {
		writeCloser, err := c.Compress(w)
		if err != nil {
			return fmt.Errorf("failed to compress resp body: %w", err)
		}
		defer writeCloser.Close()
		fastop.CanonicalHeaderSet(w.Header(), headerContentEncoding, c.ContentEncoding())
		writer = writeCloser
	}

	sg, ok := respSerializerGetterFromContext(ctx)
	if !ok {
		return errors.New("failed to get SerializerGetter")
	}
	s := sg(ctx, r)

	fastop.CanonicalHeaderSet(w.Header(), headerContentType, s.ContentType())

	// set status code
	statusCode := GetStatusCodeOnSucceed(ctx)
	w.WriteHeader(statusCode)

	// response body
	if statusCode != http.StatusNoContent && statusCode != http.StatusNotModified {
		writer.Write(body)
	}

	return nil
}

// putBackCtxMessage calls codec.PutBackMessage to put a codec.Msg back to pool,
// if the codec.Msg has been put into ctx.
func putBackCtxMessage(ctx context.Context) {
	if msg, ok := ctx.Value(codec.ContextKeyMessage).(codec.Msg); ok {
		codec.PutBackMessage(msg)
	}
}

type transcodeError struct {
	err     error
	details string
}

func (e *transcodeError) Error() string { return e.err.Error() + ": " + e.details }

var (
	errHeaderMatcher    = errors.New("header matcher failed")
	errNotFind          = errors.New("not find")
	errTranscodeRequest = errors.New("transcode request failed")
)

func (r *Router) findTranscoderAndTranscodeRequest(ctx context.Context, w http.ResponseWriter, req *http.Request, path string) (
	*transcoder, ProtoMessage, context.Context, *transcodeError) {
	var transcodeRequestErr *multierror.Error
	for _, tr := range r.transcoders[req.Method] {
		fieldValues, err := tr.pat.Match(path)
		if err != nil {
			log.Tracef("matching request path %v: %v", path, err)
			continue
		}

		stubCtx, err := r.opts.HeaderMatcher(ctx, w, req, r.opts.ServiceName, tr.name)
		if err != nil {
			return nil, nil, nil, &transcodeError{
				err: errHeaderMatcher,
				details: fmt.Sprintf("path: %s, serviceName: %s, methodName: %s, error: %v",
					path, r.opts.ServiceName, tr.name, err),
			}
		}

		protoReq, err := tr.transcodeRequest(newHTTPRequestParams(req, fieldValues))
		if err != nil {
			putBackCtxMessage(stubCtx)
			transcodeRequestErr = multierror.Append(transcodeRequestErr, err)
			continue
		}
		return tr, protoReq, stubCtx, nil
	}
	if transcodeRequestErr != nil {
		return nil, nil, nil, &transcodeError{
			err:     errTranscodeRequest,
			details: "path: " + path + transcodeRequestErr.Error(),
		}
	}
	return nil, nil, nil, &transcodeError{err: errNotFind, details: "path: " + path}
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := ctxForCompatibility(req.Context(), w, req)
	tr, protoReq, stubCtx, transcodeErr := r.findTranscoderAndTranscodeRequest(ctx, w, req, req.URL.Path)
	if transcodeErr != nil {
		if req.URL.RawPath != "" {
			tr, protoReq, stubCtx, transcodeErr = r.findTranscoderAndTranscodeRequest(ctx, w, req, req.URL.RawPath)
		}
	}
	if transcodeErr != nil {
		switch transcodeErr.err {
		case errNotFind:
			r.opts.ErrorHandler(ctx, w, req, errs.New(errs.RetServerNoFunc,
				fmt.Sprintf("failed to match any pattern, details: %s", transcodeErr.details)))
		case errHeaderMatcher:
			r.opts.ErrorHandler(ctx, w, req, errs.New(errs.RetServerDecodeFail, transcodeErr.Error()))
		case errTranscodeRequest:
			r.opts.ErrorHandler(ctx, w, req,
				errs.Newf(errs.RetServerDecodeFail, "transcoding request failed: %v", transcodeErr))
		default:
		}
		return
	}

	protoResp, err := r.handle(stubCtx, tr, protoReq)
	if err != nil {
		r.opts.ErrorHandler(stubCtx, w, req, err)
		putBackCtxMessage(stubCtx)
		return
	}

	stubCtx = newContextWithRespSerializerGetter(stubCtx, r.opts.RespSerializerGetter)
	s := r.opts.RespSerializerGetter(stubCtx, req)
	body, err := tr.transcodeResponse(protoResp, s)
	if err != nil {
		r.opts.ErrorHandler(stubCtx, w, req, errs.Wrap(err, errs.RetServerEncodeFail, "transcoding response failed"))
		putBackCtxMessage(stubCtx)
	}

	if err := r.opts.ResponseHandler(stubCtx, w, req, protoResp, body); err != nil {
		r.opts.ErrorHandler(stubCtx, w, req, errs.New(errs.RetServerEncodeFail, err.Error()))
	}
	putBackCtxMessage(stubCtx)
}

func (r *Router) handle(
	stubCtx context.Context,
	tr *transcoder,
	protoReq ProtoMessage,
) (proto.Message, error) {
	timeout := r.opts.Timeout
	if mo, ok := r.opts.methods[tr.name]; ok && mo.timeout != nil {
		timeout = *mo.timeout
	}
	requestTimeout := codec.Message(stubCtx).RequestTimeout()
	if !r.opts.disableRequestTimeout &&
		requestTimeout > 0 && (requestTimeout < timeout || timeout == 0) {
		timeout = requestTimeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		stubCtx, cancel = context.WithTimeout(stubCtx, timeout)
		defer cancel()
	}

	return tr.handle(stubCtx, protoReq)
}

func newHTTPRequestParams(req *http.Request, fieldValues map[string]string) requestParams {
	return requestParams{
		compressor:  compressor(req.Header[headerContentEncoding]),
		serializer:  requestSerializer(req.Header[headerContentType]),
		fieldValues: fieldValues,
		body:        req.Body,
		form:        req.URL.Query(),
	}
}
