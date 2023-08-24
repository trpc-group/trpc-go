// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package restful

import (
	"context"
	"net/http"

	"github.com/valyala/fasthttp"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/restful/errors"
)

const (
	// MarshalErrorContent is the content of http response body indicating error marshaling failure.
	MarshalErrorContent = `{"code": 11, "message": "failed to marshal error"}`
)

// ErrorHandler handles tRPC errors.
type ErrorHandler func(context.Context, http.ResponseWriter, *http.Request, error)

// FastHTTPErrorHandler handles tRPC errors when fasthttp is used.
type FastHTTPErrorHandler func(context.Context, *fasthttp.RequestCtx, error)

// WithStatusCode is the error that corresponds to an HTTP status code.
type WithStatusCode struct {
	StatusCode int
	Err        error
}

// Error implements Go error.
func (w *WithStatusCode) Error() string {
	return w.Err.Error()
}

// Unwrap returns the wrapped error.
func (w *WithStatusCode) Unwrap() error {
	return w.Err
}

// tRPC error code => http status code
var httpStatusMap = map[trpcpb.TrpcRetCode]int{
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

// marshalError marshals an error.
func marshalError(err error, s Serializer) ([]byte, error) {
	// All Serializers for tRPC-Go RESTful are expected to marshal proto messages.
	// So it's better to convert a tRPC error to an *errors.Err.
	terr := &errors.Err{
		Code:    int32(errs.Code(err)),
		Message: errs.Msg(err),
	}

	return s.Marshal(terr)
}

// statusCodeFromError returns the status code from the error.
func statusCodeFromError(err error) int {
	statusCode := http.StatusInternalServerError

	if withStatusCode, ok := err.(*WithStatusCode); ok {
		statusCode = withStatusCode.StatusCode
	} else {
		if statusFromMap, ok := httpStatusMap[errs.Code(err)]; ok {
			statusCode = statusFromMap
		}
	}

	return statusCode
}

// DefaultErrorHandler is the default ErrorHandler.
var DefaultErrorHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	// get outbound Serializer
	_, s := serializerForTranscoding(r.Header[headerContentType],
		r.Header[headerAccept])
	w.Header().Set(headerContentType, s.ContentType())

	// marshal error
	buf, merr := marshalError(err, s)
	if merr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(MarshalErrorContent))
		return
	}
	// write response
	w.WriteHeader(statusCodeFromError(err))
	w.Write(buf)
}

// DefaultFastHTTPErrorHandler is the default FastHTTPErrorHandler.
var DefaultFastHTTPErrorHandler = func(ctx context.Context, requestCtx *fasthttp.RequestCtx, err error) {
	// get outbound Serializer
	_, s := serializerForTranscoding(
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerContentType))},
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerAccept))},
	)
	requestCtx.Response.Header.Set(headerContentType, s.ContentType())

	// marshal error
	buf, merr := marshalError(err, s)
	if merr != nil {
		requestCtx.Response.SetStatusCode(http.StatusInternalServerError)
		requestCtx.Write([]byte(MarshalErrorContent))
		return
	}
	// write response
	requestCtx.SetStatusCode(statusCodeFromError(err))
	requestCtx.Write(buf)
}
