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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/hashicorp/go-multierror"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
)

// FastHTTPHeaderMatcher matches fasthttp request header to tRPC Stub Context.
type FastHTTPHeaderMatcher func(
	ctx context.Context,
	requestCtx *fasthttp.RequestCtx,
	serviceName, methodName string,
) (context.Context, error)

// DefaultFastHTTPHeaderMatcher is the default FastHTTPHeaderMatcher.
var DefaultFastHTTPHeaderMatcher = func(
	ctx context.Context,
	requestCtx *fasthttp.RequestCtx,
	serviceName, methodName string,
) (context.Context, error) {
	return withNewMessage(ctx, serviceName, methodName), nil
}

// FastHTTPRespHandler is the custom response handler when fasthttp is used.
type FastHTTPRespHandler func(
	ctx context.Context,
	requestCtx *fasthttp.RequestCtx,
	resp proto.Message,
	body []byte,
) error

// DefaultFastHTTPRespHandler is the default FastHTTPRespHandler.
func DefaultFastHTTPRespHandler(stubCtx context.Context, requestCtx *fasthttp.RequestCtx,
	protoResp proto.Message, body []byte) error {
	// compress
	writer := requestCtx.Response.BodyWriter()
	// fasthttp doesn't support getting multiple values of one key from http headers.
	// ctx.Request.Header.Peek is equivalent to req.Header.Get from Go net/http.

	if c := compressor([]string{string(requestCtx.Request.Header.Peek(headerAcceptEncoding))}); c != nil {
		writeCloser, err := c.Compress(writer)
		if err != nil {
			return err
		}
		defer writeCloser.Close()
		requestCtx.Response.Header.Set(headerContentEncoding, c.ContentEncoding())
		writer = writeCloser
	}

	sg, ok := fastHTTPRespSerializerGetterFromContext(stubCtx)
	if !ok {
		return errors.New("failed to get fastHTTPRespSerializerGetter")
	}
	s := sg(stubCtx, requestCtx)

	requestCtx.Response.Header.Set(headerContentType, s.ContentType())

	// set status code
	statusCode := GetStatusCodeOnSucceed(stubCtx)
	requestCtx.SetStatusCode(statusCode)

	// write body
	if statusCode != fasthttp.StatusNoContent && statusCode != fasthttp.StatusNotModified {
		writer.Write(body)
	}

	return nil
}

// HandleRequestCtx fasthttp handler
func (r *Router) HandleRequestCtx(requestCtx *fasthttp.RequestCtx) {
	newCtx := context.Background()
	var transcodeRequestErr *multierror.Error
	path := string(requestCtx.Path())
	for _, tr := range r.transcoders[string(requestCtx.Method())] {
		fieldValues, err := tr.pat.Match(path)
		if err != nil {
			log.Tracef("matching request URL.Path %s: %v", requestCtx.Path(), err)
			continue
		}

		stubCtx, err := r.opts.FastHTTPHeaderMatcher(newCtx, requestCtx,
			r.opts.ServiceName, tr.name)
		if err != nil {
			r.opts.FastHTTPErrHandler(stubCtx, requestCtx, errs.New(errs.RetServerDecodeFail, err.Error()))
			return
		}

		protoReq, err := tr.transcodeRequest(newFastHTTPRequestParams(requestCtx, fieldValues))
		if err != nil {
			transcodeRequestErr = multierror.Append(transcodeRequestErr, err)
			continue
		}

		protoResp, err := r.handle(stubCtx, tr, protoReq)
		if err != nil {
			r.opts.FastHTTPErrHandler(stubCtx, requestCtx, err)
			putBackCtxMessage(stubCtx)
			return
		}

		stubCtx = newContextWithFastHTTPRespSerializerGetter(stubCtx, r.opts.FastHTTPRespSerializerGetter)
		s := r.opts.FastHTTPRespSerializerGetter(stubCtx, requestCtx)
		body, err := tr.transcodeResponse(protoResp, s)
		if err != nil {
			r.opts.FastHTTPErrHandler(stubCtx, requestCtx,
				errs.Wrap(err, errs.RetServerEncodeFail, "transcoding response failed"))
			putBackCtxMessage(stubCtx)
			return
		}

		if err := r.opts.FastHTTPRespHandler(stubCtx, requestCtx, protoResp, body); err != nil {
			r.opts.FastHTTPErrHandler(stubCtx, requestCtx, errs.New(errs.RetServerEncodeFail, err.Error()))
		}
		putBackCtxMessage(stubCtx)
		return
	}
	if transcodeRequestErr != nil {
		r.opts.FastHTTPErrHandler(newCtx, requestCtx,
			errs.Newf(errs.RetServerDecodeFail, "transcoding request failed: %v", transcodeRequestErr))
		return
	}
	r.opts.FastHTTPErrHandler(newCtx, requestCtx, errs.New(errs.RetServerNoFunc,
		fmt.Sprintf("path `%s` failed to match any pattern", path)))
}

func newFastHTTPRequestParams(ctx *fasthttp.RequestCtx, fieldValues map[string]string) requestParams {
	form := make(url.Values)
	ctx.QueryArgs().VisitAll(func(key []byte, value []byte) {
		form.Add(string(key), string(value))
	})
	return requestParams{
		form:        form,
		compressor:  compressor([]string{string(ctx.Request.Header.Peek(headerContentEncoding))}),
		serializer:  requestSerializer([]string{string(ctx.Request.Header.Peek(headerContentType))}),
		fieldValues: fieldValues,
		body:        bytes.NewBuffer(ctx.PostBody()),
	}
}
