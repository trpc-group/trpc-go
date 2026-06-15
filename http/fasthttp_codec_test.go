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

package http_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"testing"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func TestFastHTTPServerEncode(t *testing.T) {
	ctx := context.Background()
	msg := codec.Message(ctx)
	_, err := thttp.DefaultFastHTTPServerCodec.Encode(msg, nil)
	require.Error(t, err)

	requestCtx := &fasthttp.RequestCtx{}
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	msg.WithCompressType(codec.CompressTypeGzip)
	_, err = thttp.DefaultFastHTTPServerCodec.Encode(msg, []byte("hello"))
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), requestCtx.Response.Body())
	require.Equal(t, "gzip", string(requestCtx.Response.Header.Peek(fasthttp.HeaderContentEncoding)))

	requestCtx.Request.Reset()
	requestCtx.Response.Reset()
	msg = codec.Message(ctx)
	msg.WithServerRspErr(errs.NewFrameError(1, "frame\r\nerror"))
	_, err = thttp.DefaultFastHTTPServerCodec.Encode(msg, []byte("ignored"))
	require.NoError(t, err)
	require.Empty(t, requestCtx.Response.Body())
	require.Equal(t, "frame\\r\\nerror", string(requestCtx.Response.Header.Peek(thttp.TrpcErrorMessage)))

	requestCtx.Request.Reset()
	requestCtx.Response.Reset()
	msg = codec.Message(ctx)
	msg.WithServerRspErr(errs.New(10086, "user error"))
	_, err = thttp.DefaultFastHTTPServerCodec.Encode(msg, []byte("ignored"))
	require.NoError(t, err)
	require.Empty(t, requestCtx.Response.Body())
	require.Equal(t, "10086", string(requestCtx.Response.Header.Peek(thttp.TrpcUserFuncErrorCode)))

	requestCtx.Request.Reset()
	requestCtx.Response.Reset()
	requestCtx.Response.Header.SetContentType("multipart/form-data; boundary=test")
	msg = codec.Message(ctx)
	msg.WithServerMetaData(codec.MetaData{"meta-key": []byte("meta-value")})
	sc := thttp.FastHTTPServerCodec{
		AutoReadBody:                 true,
		RspHandler:                   thttp.DefaultFastHTTPServerCodec.RspHandler,
		DisableEncodeTransInfoBase64: true,
	}
	_, err = sc.Encode(msg, []byte("form"))
	require.NoError(t, err)
	require.Contains(t, string(requestCtx.Response.Header.Peek(thttp.TrpcTransInfo)), "meta-value")
	require.Contains(t, string(requestCtx.Response.Header.ContentType()), "application/json")
}

func TestFastHTTPServerDecode(t *testing.T) {
	ctx := context.Background()
	msg := codec.Message(ctx)
	_, err := thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.Error(t, err)

	requestCtx := &fasthttp.RequestCtx{}
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	_, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.NoError(t, err)

	sc := thttp.FastHTTPServerCodec{AutoReadBody: true, POSTOnly: true}
	requestCtx.Request.Reset()
	requestCtx.Request.Header.SetMethod(fasthttp.MethodPatch)
	msg = codec.Message(ctx)
	_, err = sc.Decode(msg, nil)
	require.Error(t, err)

	requestCtx.Request.Reset()
	requestCtx.Request.Header.SetMethod(fasthttp.MethodGet)
	requestCtx.Request.SetRequestURI("http://example.com/path?k=v")
	msg = codec.Message(ctx)
	body, err := thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Equal(t, "k=v", string(body))
	require.Equal(t, codec.SerializationTypeGet, msg.SerializationType())

	requestCtx.Request.Reset()
	requestCtx.Request.Header.SetMethod(fasthttp.MethodPost)
	requestCtx.Request.Header.SetContentType("application/json")
	requestCtx.Request.SetRequestURI("http://example.com/path")
	requestCtx.Request.SetBodyString(`{"k":"v"}`)
	msg = codec.Message(ctx)
	body, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Equal(t, `{"k":"v"}`, string(body))
	require.Equal(t, codec.SerializationTypeJSON, msg.SerializationType())
}

func TestFastHTTPServerDecodeHeader(t *testing.T) {
	requestCtx := &fasthttp.RequestCtx{}
	ctx := thttp.WithRequestCtx(context.Background(), requestCtx)
	msg := codec.Message(ctx)

	req := &requestCtx.Request
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("http://example.com/trpc.http.test.helloworld/SayHello")
	req.Header.Add(fasthttp.HeaderContentEncoding, "gzip")
	req.Header.Add(fasthttp.HeaderContentType, "application/json")
	req.Header.Add(thttp.TrpcVersion, "1")
	req.Header.Add(thttp.TrpcCallType, "1")
	req.Header.Add(thttp.TrpcMessageType, "1")
	req.Header.Add(thttp.TrpcRequestID, "1")
	req.Header.Add(thttp.TrpcTimeout, "1000")
	req.Header.Add(thttp.TrpcCallerMethod, "CallerMethod")
	req.Header.Add(thttp.TrpcCaller, "trpc.app.server.caller")
	req.Header.Add(thttp.TrpcCallee, "trpc.http.test.helloworld")
	req.Header.Add(thttp.TrpcTransInfo, `{"key1":"dmFsMQ==", "key2":"dmFsMg=="}`)

	_, err := thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Equal(t, codec.CompressTypeGzip, msg.CompressType())
	require.Equal(t, codec.SerializationTypeJSON, msg.SerializationType())
	require.Equal(t, 1000*time.Millisecond, msg.RequestTimeout())
	require.Equal(t, "CallerMethod", msg.CallerMethod())

	rh, ok := msg.ServerReqHead().(*trpcpb.RequestProtocol)
	require.True(t, ok)
	require.Equal(t, uint32(1), rh.GetVersion())
	require.Equal(t, uint32(1), rh.GetCallType())
	require.Equal(t, uint32(1), rh.GetMessageType())
	require.Equal(t, uint32(1), rh.GetRequestId())
	require.Equal(t, uint32(1000), rh.GetTimeout())
	require.Equal(t, "trpc.app.server.caller", string(rh.GetCaller()))
	require.Equal(t, "trpc.http.test.helloworld", string(rh.GetCallee()))
	require.Equal(t, "val1", string(rh.GetTransInfo()["key1"]))
	require.Equal(t, "val2", string(rh.GetTransInfo()["key2"]))

	req.Header.Set(thttp.TrpcTransInfo, `{"key1":"dmFsMQ==", "key2":"dmFsMg=="`)
	_, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.Error(t, err)

	req.Header.Set(thttp.TrpcTransInfo, fmt.Sprintf(`{"%s":"%s"}`, thttp.TrpcEnv, "Production"))
	_, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.NoError(t, err)
	rh, ok = msg.ServerReqHead().(*trpcpb.RequestProtocol)
	require.True(t, ok)
	require.Equal(t, "Production", string(rh.GetTransInfo()[thttp.TrpcEnv]))
}

func TestFastHTTPServerDecodeMultipartForm(t *testing.T) {
	requestCtx := &fasthttp.RequestCtx{}
	requestCtx.Request.Header.SetMethod(fasthttp.MethodPost)

	uri := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(uri)
	uri.SetScheme("http")
	uri.SetHost("example.com")
	uri.SetPath("/path")
	uri.QueryArgs().Set("query", "value")
	requestCtx.Request.SetURI(uri)
	requestCtx.Request.PostArgs().Set("post", "value")

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	require.NoError(t, writer.WriteField("multipart", "value"))
	fileWriter, err := writer.CreateFormFile("file", "example.txt")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("file content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	requestCtx.Request.Header.SetContentType(writer.FormDataContentType())
	requestCtx.Request.SetBody(buf.Bytes())

	ctx := thttp.WithRequestCtx(context.Background(), requestCtx)
	msg := codec.Message(ctx)
	body, err := thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Contains(t, string(body), "query=value")
	require.Contains(t, string(body), "post=value")
	require.Contains(t, string(body), "multipart=value")
	require.Equal(t, codec.SerializationTypeFormData, msg.SerializationType())
}

func TestFastHTTPClientEncode(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	_, err := thttp.DefaultFastHTTPClientCodec.Encode(msg, []byte(`{"name":"trpc"}`))
	require.NoError(t, err)
	require.IsType(t, &thttp.FastHTTPClientReqHeader{}, msg.ClientReqHead())
	require.IsType(t, &thttp.FastHTTPClientRspHeader{}, msg.ClientRspHead())
	require.Equal(t, fasthttp.MethodPost, msg.ClientReqHead().(*thttp.FastHTTPClientReqHeader).Method)

	_, msg = codec.WithNewMessage(context.Background())
	_, err = thttp.DefaultFastHTTPClientCodec.Encode(msg, nil)
	require.NoError(t, err)
	require.Equal(t, fasthttp.MethodGet, msg.ClientReqHead().(*thttp.FastHTTPClientReqHeader).Method)

	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	_, err = thttp.DefaultFastHTTPClientCodec.Encode(msg, nil)
	require.Error(t, err)

	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.ClientRspHeader{})
	_, err = thttp.DefaultFastHTTPClientCodec.Encode(msg, nil)
	require.Error(t, err)
}

func TestFastHTTPClientDecode(t *testing.T) {
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(rsp)
	rsp.SetStatusCode(fasthttp.StatusOK)
	rsp.Header.SetContentEncoding("gzip")
	rsp.Header.SetContentType("application/json")
	rsp.SetBodyString(`{"msg":"ok"}`)

	_, msg := codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})
	body, err := thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Equal(t, `{"msg":"ok"}`, string(body))
	require.Equal(t, codec.CompressTypeGzip, msg.CompressType())
	require.Equal(t, codec.SerializationTypeJSON, msg.SerializationType())
}

func TestFastHTTPClientDecodeErrors(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	_, err := thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.Error(t, err)

	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientReqHeader{})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.Error(t, err)

	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(rsp)

	rsp.Reset()
	rsp.SetStatusCode(fasthttp.StatusMultipleChoices)
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.NotNil(t, msg.ClientRspErr())

	rsp.Reset()
	rsp.Header.Add(thttp.TrpcFrameworkErrorCode, "1")
	rsp.Header.Add(thttp.TrpcErrorMessage, "frame error")
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Equal(t, trpcpb.TrpcRetCode(1), errs.Code(msg.ClientRspErr()))

	rsp.Reset()
	rsp.Header.Add(thttp.TrpcUserFuncErrorCode, "10086")
	rsp.Header.Add(thttp.TrpcErrorMessage, "user error")
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Equal(t, trpcpb.TrpcRetCode(10086), errs.Code(msg.ClientRspErr()))
}

func TestFastHTTPClientDecodeHandlers(t *testing.T) {
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(rsp)

	rsp.SetBodyString("data: hello\n\n")
	_, msg := codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp, SSEHandler: errSSEHandler{}})
	_, err := thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.Error(t, err)

	rsp.Reset()
	rsp.SetBodyString("body")
	_, msg = codec.WithNewMessage(context.Background())
	handled := false
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{
		Response: rsp,
		ResponseHandler: fastHTTPRspHandlerFunc(func(*fasthttp.Response) error {
			handled = true
			return nil
		}),
	})
	body, err := thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Nil(t, body)
	require.True(t, handled)

	rsp.Reset()
	rsp.SetBodyString("manual")
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp, ManualReadBody: true})
	body, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Nil(t, body)
}

func TestFastHTTPServerDecodeDisabledAuto(t *testing.T) {
	sc := thttp.FastHTTPServerCodec{
		AutoReadBody:    false,
		AutoGenTrpcHead: false,
	}
	requestCtx := &fasthttp.RequestCtx{}
	ctx := thttp.WithRequestCtx(context.Background(), requestCtx)
	msg := codec.Message(ctx)
	body, err := sc.Decode(msg, nil)
	require.NoError(t, err)
	require.Nil(t, body)
}

type errSSEHandler struct{}

func (errSSEHandler) Handle(*sse.Event) error {
	return errors.New("sse error")
}

type fastHTTPRspHandlerFunc func(*fasthttp.Response) error

func (f fastHTTPRspHandlerFunc) Handle(rsp *fasthttp.Response) error {
	return f(rsp)
}
