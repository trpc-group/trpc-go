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
	"net/http"
	"time"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
)

// Options are restful router options.
type Options struct {
	namespace   string // global namespace
	environment string // global environment
	container   string // global container name
	set         string // global set name

	ServiceName           string                // tRPC service name
	ServiceImpl           interface{}           // tRPC service impl
	FilterFunc            ExtractFilterFunc     // extract tRPC service filter chain
	ErrorHandler          ErrorHandler          // error handler
	HeaderMatcher         HeaderMatcher         // header matcher
	ResponseHandler       CustomResponseHandler // custom response handler
	FastHTTPErrHandler    FastHTTPErrorHandler  // fasthttp error handler
	FastHTTPHeaderMatcher FastHTTPHeaderMatcher // fasthttp header matcher
	FastHTTPRespHandler   FastHTTPRespHandler   // fasthttp custom response handler
	DiscardUnknownParams  bool                  // ignore unknown query params
	Timeout               time.Duration         // timeout
}

// Option sets restful router options.
type Option func(*Options)

func (o *Options) rebuildHeaderMatcher() {
	headerMatcher := o.HeaderMatcher
	o.HeaderMatcher = func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
		serviceName, methodName string,
	) (context.Context, error) {
		ctx, err := headerMatcher(ctx, w, r, serviceName, methodName)
		if err != nil {
			return nil, err
		}
		return withGlobalMsg(ctx, o), nil
	}

	fastHTTPHeaderMatcher := o.FastHTTPHeaderMatcher
	o.FastHTTPHeaderMatcher = func(
		ctx context.Context,
		requestCtx *fasthttp.RequestCtx,
		serviceName, methodName string,
	) (context.Context, error) {
		ctx, err := fastHTTPHeaderMatcher(ctx, requestCtx, serviceName, methodName)
		if err != nil {
			return nil, err
		}
		return withGlobalMsg(ctx, o), nil
	}
}

// WithNamespace returns an Option that set namespace.
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.namespace = namespace
	}
}

// WithEnvironment returns an Option that sets environment name.
func WithEnvironment(env string) Option {
	return func(o *Options) {
		o.environment = env
	}
}

// WithContainer returns an Option that sets container name.
func WithContainer(container string) Option {
	return func(o *Options) {
		o.container = container
	}
}

// WithSet returns an Option that sets set name.
func WithSet(set string) Option {
	return func(o *Options) {
		o.set = set
	}
}

// WithServiceName returns an Option that sets tRPC service name for the restful router.
func WithServiceName(name string) Option {
	return func(o *Options) {
		o.ServiceName = name
	}
}

// WithFilterFunc returns an Option that sets tRPC service filter chain extracting function
// for the restful router.
func WithFilterFunc(f ExtractFilterFunc) Option {
	return func(o *Options) {
		if getFilters := o.FilterFunc; getFilters != nil {
			o.FilterFunc = func() filter.ServerChain {
				return append(getFilters(), f()...)
			}
		} else {
			o.FilterFunc = f
		}
	}
}

// WithErrorHandler returns an Option that sets error handler for the restful router.
func WithErrorHandler(errorHandler ErrorHandler) Option {
	return func(o *Options) {
		o.ErrorHandler = errorHandler
	}
}

// WithHeaderMatcher returns an Option that sets header matcher for the restful router.
func WithHeaderMatcher(m HeaderMatcher) Option {
	return func(o *Options) {
		o.HeaderMatcher = m
	}
}

// WithResponseHandler returns an Option that sets custom response handler for
// the restful router.
func WithResponseHandler(h CustomResponseHandler) Option {
	return func(o *Options) {
		o.ResponseHandler = h
	}
}

// WithFastHTTPErrorHandler returns an Option that sets fasthttp error handler
// for the restful router.
func WithFastHTTPErrorHandler(errHandler FastHTTPErrorHandler) Option {
	return func(o *Options) {
		o.FastHTTPErrHandler = errHandler
	}
}

// WithFastHTTPHeaderMatcher returns an Option that sets fasthttp header matcher
// for the restful router.
func WithFastHTTPHeaderMatcher(m FastHTTPHeaderMatcher) Option {
	return func(o *Options) {
		o.FastHTTPHeaderMatcher = m
	}
}

// WithFastHTTPRespHandler returns an Option that sets fasthttp custom response
// handler for the restful router.
func WithFastHTTPRespHandler(h FastHTTPRespHandler) Option {
	return func(o *Options) {
		o.FastHTTPRespHandler = h
	}
}

// WithDiscardUnknownParams returns an Option that sets whether to ignore unknown query params
// for the restful router.
func WithDiscardUnknownParams(i bool) Option {
	return func(o *Options) {
		o.DiscardUnknownParams = i
	}
}

// WithTimeout returns an Option that sets timeout for the restful router.
func WithTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// withGlobalMsg sets tRPC yaml global fields to ctx message.
func withGlobalMsg(ctx context.Context, o *Options) context.Context {
	ctx, msg := codec.EnsureMessage(ctx)
	msg.WithNamespace(o.namespace)
	msg.WithEnvName(o.environment)
	msg.WithCalleeContainerName(o.container)
	msg.WithSetName(o.set)
	return ctx
}
