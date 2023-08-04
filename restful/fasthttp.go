package restful

import (
	"bytes"
	"context"
	"unsafe"

	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
	"trpc.group/trpc-go/trpc-go/errs"
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
	_, c := compressorForTranscoding(
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerContentEncoding))},
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerAcceptEncoding))},
	)
	if c != nil {
		writeCloser, err := c.Compress(writer)
		if err != nil {
			return err
		}
		defer writeCloser.Close()
		requestCtx.Response.Header.Set(headerContentEncoding, c.ContentEncoding())
		writer = writeCloser
	}

	// set response content-type
	_, s := serializerForTranscoding(
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerContentType))},
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerAccept))},
	)
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

// bytes2str is the high-performance way of converting []byte to string.
func bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// HandleRequestCtx fasthttp handler
func (r *Router) HandleRequestCtx(ctx *fasthttp.RequestCtx) {
	newCtx := context.Background()
	for _, tr := range r.transcoders[bytes2str(ctx.Method())] {
		fieldValues, err := tr.pat.Match(bytes2str(ctx.Path()))
		if err == nil {
			// header matching
			stubCtx, err := r.opts.FastHTTPHeaderMatcher(newCtx, ctx,
				r.opts.ServiceName, tr.name)
			if err != nil {
				r.opts.FastHTTPErrHandler(stubCtx, ctx, errs.New(errs.RetServerDecodeFail, err.Error()))
				return
			}

			// get inbound/outbound Compressor & Serializer
			reqCompressor, respCompressor := compressorForTranscoding(
				[]string{bytes2str(ctx.Request.Header.Peek(headerContentEncoding))},
				[]string{bytes2str(ctx.Request.Header.Peek(headerAcceptEncoding))},
			)
			reqSerializer, respSerializer := serializerForTranscoding(
				[]string{bytes2str(ctx.Request.Header.Peek(headerContentType))},
				[]string{bytes2str(ctx.Request.Header.Peek(headerAccept))},
			)

			// get query params
			form := make(map[string][]string)
			ctx.QueryArgs().VisitAll(func(key []byte, value []byte) {
				form[bytes2str(key)] = append(form[bytes2str(key)], bytes2str(value))
			})

			// set transcoding params
			params := paramsPool.Get().(*transcodeParams)
			params.reqCompressor = reqCompressor
			params.respCompressor = respCompressor
			params.reqSerializer = reqSerializer
			params.respSerializer = respSerializer
			params.body = bytes.NewBuffer(ctx.PostBody())
			params.fieldValues = fieldValues
			params.form = form

			// transcode
			resp, body, err := tr.transcode(stubCtx, params)
			if err != nil {
				r.opts.FastHTTPErrHandler(stubCtx, ctx, err)
				putBackCtxMessage(stubCtx)
				putBackParams(params)
				return
			}

			// response
			if err := r.opts.FastHTTPRespHandler(stubCtx, ctx, resp, body); err != nil {
				r.opts.FastHTTPErrHandler(stubCtx, ctx, errs.New(errs.RetServerEncodeFail, err.Error()))
			}
			putBackCtxMessage(stubCtx)
			putBackParams(params)
			return
		}
	}
	r.opts.FastHTTPErrHandler(newCtx, ctx, errs.New(errs.RetServerNoFunc, "failed to match any pattern"))
}
