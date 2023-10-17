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
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/internal/dat"
)

// Router is restful router.
type Router struct {
	opts        *Options
	transcoders map[string][]*transcoder
}

// NewRouter creates a Router.
func NewRouter(opts ...Option) *Router {
	o := Options{
		ErrorHandler:          DefaultErrorHandler,
		HeaderMatcher:         DefaultHeaderMatcher,
		ResponseHandler:       DefaultResponseHandler,
		FastHTTPErrHandler:    DefaultFastHTTPErrorHandler,
		FastHTTPHeaderMatcher: DefaultFastHTTPHeaderMatcher,
		FastHTTPRespHandler:   DefaultFastHTTPRespHandler,
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

// RegisterRouter registers a Router which corresponds to a tRPC Service.
func RegisterRouter(name string, router http.Handler) {
	routerLock.Lock()
	routers[name] = router
	routerLock.Unlock()
}

// GetRouter returns a Router which corresponds to a tRPC Service.
func GetRouter(name string) http.Handler {
	routerLock.RLock()
	router := routers[name]
	routerLock.RUnlock()
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

// ExtractFilterFunc extracts tRPC service filter chain.
type ExtractFilterFunc func() filter.ServerChain

// Binding is the binding of tRPC method and HttpRule.
type Binding struct {
	Name         string
	Input        Initializer
	Output       Initializer
	Filter       HandleFunc
	HTTPMethod   string
	Pattern      *Pattern
	Body         BodyLocator
	ResponseBody ResponseBodyLocator
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
	msg.WithCalleeMethod(methodName)
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
			if code, err := strconv.Atoi(bytes2str(buf)); err == nil {
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
	_, c := compressorForTranscoding(r.Header[headerContentEncoding],
		r.Header[headerAcceptEncoding])
	if c != nil {
		writeCloser, err := c.Compress(w)
		if err != nil {
			return fmt.Errorf("failed to compress resp body: %w", err)
		}
		defer writeCloser.Close()
		w.Header().Set(headerContentEncoding, c.ContentEncoding())
		writer = writeCloser
	}

	// set response content-type
	_, s := serializerForTranscoding(r.Header[headerContentType],
		r.Header[headerAccept])
	w.Header().Set(headerContentType, s.ContentType())

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

// ServeHTTP implements http.Handler.
// TODO: better routing handling.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := ctxForCompatibility(req.Context(), w, req)
	for _, tr := range r.transcoders[req.Method] {
		fieldValues, err := tr.pat.Match(req.URL.Path)
		if err == nil {
			r.handle(ctx, w, req, tr, fieldValues)
			return
		}
	}
	r.opts.ErrorHandler(ctx, w, req, errs.New(errs.RetServerNoFunc, "failed to match any pattern"))
}

func (r *Router) handle(
	ctx context.Context,
	w http.ResponseWriter,
	req *http.Request,
	tr *transcoder,
	fieldValues map[string]string,
) {
	modifiedCtx, err := r.opts.HeaderMatcher(ctx, w, req, r.opts.ServiceName, tr.name)
	if err != nil {
		r.opts.ErrorHandler(ctx, w, req, errs.New(errs.RetServerDecodeFail, err.Error()))
		return
	}
	ctx = modifiedCtx
	defer putBackCtxMessage(ctx)

	timeout := r.opts.Timeout
	requestTimeout := codec.Message(ctx).RequestTimeout()
	if requestTimeout > 0 && (requestTimeout < timeout || timeout == 0) {
		timeout = requestTimeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// get inbound/outbound Compressor and Serializer
	reqCompressor, respCompressor := compressorForTranscoding(req.Header[headerContentEncoding],
		req.Header[headerAcceptEncoding])
	reqSerializer, respSerializer := serializerForTranscoding(req.Header[headerContentType],
		req.Header[headerAccept])

	// set transcoder params
	params, _ := paramsPool.Get().(*transcodeParams)
	params.reqCompressor = reqCompressor
	params.respCompressor = respCompressor
	params.reqSerializer = reqSerializer
	params.respSerializer = respSerializer
	params.body = req.Body
	params.fieldValues = fieldValues
	params.form = req.URL.Query()
	defer putBackParams(params)

	// transcode
	resp, body, err := tr.transcode(ctx, params)
	if err != nil {
		r.opts.ErrorHandler(ctx, w, req, err)
		return
	}

	// custom response handling
	if err := r.opts.ResponseHandler(ctx, w, req, resp, body); err != nil {
		r.opts.ErrorHandler(ctx, w, req, errs.New(errs.RetServerEncodeFail, err.Error()))
	}
}
