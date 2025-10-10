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

package http_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"testing"

	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/server"
	helloworld "trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
)

func TestFastHTTPServerEncode(t *testing.T) {
	// Perform a test for missing requestCtx.
	ctx := context.Background()
	msg := codec.Message(ctx)
	_, err := thttp.DefaultFastHTTPServerCodec.Encode(msg, nil)
	require.NotNil(t, err)

	requestCtx := &fasthttp.RequestCtx{}
	req := &requestCtx.Request
	rsp := &requestCtx.Response
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	// Perform a test for requestCtx exists.
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	msg.WithCompressType(codec.CompressTypeGzip)
	_, err = thttp.DefaultFastHTTPServerCodec.Encode(msg, []byte("hello"))
	require.Equal(t, requestCtx.Response.Body(), []byte("hello"))
	require.Nil(t, err)

	// Perform a test for ErrHandler: frameError.
	req.Reset()
	rsp.Reset()
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	msg.WithServerRspErr(errs.NewFrameError(1, "frameError"))
	_, err = thttp.DefaultFastHTTPServerCodec.Encode(msg, []byte("write failed"))
	// NOTICE: After the server returns an error, even there is a response data,
	// it will be ignored and will not be processed or returned.
	require.Empty(t, rsp.Body())
	// NOTICE: err is expected to be nil
	require.Nil(t, err)

	// Perform a test for ErrHandler: userError.
	req.Reset()
	rsp.Reset()
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	msg.WithServerRspErr(errs.New(10086, "userError"))
	_, err = thttp.DefaultFastHTTPServerCodec.Encode(msg, []byte("write failed"))
	// NOTICE: After the server returns an error, even there is a response data,
	// it will be ignored and will not be processed or returned.
	require.Empty(t, rsp.Body())
	// NOTICE: err is expected to be nil.
	require.Nil(t, err)

	// Perform a test for RspHandler.
	req.Reset()
	rsp.Reset()
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	_, err = thttp.DefaultFastHTTPServerCodec.Encode(msg, []byte{123})
	require.Nil(t, err)

	// Perform a test for Multipart/Form-Data and MetaData.
	req.Reset()
	rsp.Reset()
	requestCtx.Response.Header.SetContentType("Multipart/Form-Data")
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	msg.WithServerMetaData(codec.MetaData{"a": []byte{1}})
	_, err = thttp.DefaultFastHTTPServerCodec.Encode(msg, []byte{123})
	require.Nil(t, err)

	// Perform a test for DisableEncodeTransInfoBase64.
	req.Reset()
	rsp.Reset()
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	msg.WithServerMetaData(codec.MetaData{"meta-key": []byte("meta-value")})
	sc := thttp.FastHTTPServerCodec{AutoReadBody: true, DisableEncodeTransInfoBase64: true}
	_, err = sc.Encode(msg, []byte{123})
	require.Nil(t, err)
	require.Contains(t, string(rsp.Header.Peek(thttp.TrpcTransInfo)), "meta-value")
}

func TestFastHTTPServerDecode(t *testing.T) {
	// Perform a test for missing requestCtx.
	ctx := context.Background()
	msg := codec.Message(ctx)
	_, err := thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.NotNil(t, err)

	// Perform a test for requestCtx exists.
	requestCtx := &fasthttp.RequestCtx{}
	req := &requestCtx.Request
	rsp := &requestCtx.Response
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	_, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.Nil(t, err)

	// Perform a test for getReqBody err: POSTOnly but PATCH.
	sc := thttp.FastHTTPServerCodec{AutoReadBody: true, POSTOnly: true}
	req.Reset()
	rsp.Reset()
	req.Header.SetMethod(fasthttp.MethodPatch)
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	_, err = sc.Decode(msg, nil)
	require.NotNil(t, err)

	// Perform a GET request.
	req.Reset()
	rsp.Reset()
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("www.qq.com/xyz=abc")
	msg.WithServerRspErr(errs.ErrServerNoFunc)
	_, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.Nil(t, err)

	// Perform a POST request.
	req.Reset()
	rsp.Reset()
	ctx = thttp.WithRequestCtx(ctx, requestCtx)
	msg = codec.Message(ctx)
	req.Header.SetMethod("POST")
	req.SetRequestURI("www.qq.com")
	req.SetBody([]byte("{xyz:\"abc\""))
	_, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.Nil(t, err)
}

func TestFastHTTPServerDecodeHeader(t *testing.T) {
	requestCtx := &fasthttp.RequestCtx{}
	ctx := thttp.WithRequestCtx(context.Background(), requestCtx)
	msg := codec.Message(ctx)

	req := &requestCtx.Request
	rsp := &requestCtx.Response
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("http://www.qq.com/trpc.http.test.helloworld/SayHello")
	req.Header.Add(fasthttp.HeaderContentEncoding, "gzip")
	req.Header.Add(fasthttp.HeaderContentType, "application/json")
	req.Header.Add(thttp.TrpcVersion, "1")
	req.Header.Add(thttp.TrpcCallType, "1")
	req.Header.Add(thttp.TrpcMessageType, "1")
	req.Header.Add(thttp.TrpcRequestID, "1")
	req.Header.Add(thttp.TrpcTimeout, "1000")
	req.Header.Add(thttp.TrpcCaller, "trpc.app.server.helloworld")
	req.Header.Add(thttp.TrpcCallee, "trpc.http.test.helloworld")
	req.Header.Add(thttp.TrpcTransInfo, `{"key1":"dmFsMQ==", "key2":"dmFsMg=="}`)

	_, err := thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.Nil(t, err)
	require.Equal(t, codec.CompressTypeGzip, msg.CompressType())
	require.Equal(t, codec.SerializationTypeJSON, msg.SerializationType())

	rh, ok := msg.ServerReqHead().(*trpc.RequestProtocol)
	require.True(t, ok)
	require.NotNil(t, req, "failed to decode get trpc req head")
	require.Equal(t, 1, int(rh.GetVersion()))
	require.Equal(t, 1, int(rh.GetCallType()))
	require.Equal(t, 1, int(rh.GetMessageType()))
	require.Equal(t, 1, int(rh.GetRequestId()))
	require.Equal(t, 1000, int(rh.GetTimeout()))
	require.Equal(t, "trpc.app.server.helloworld", string(rh.GetCaller()))
	require.Equal(t, "trpc.http.test.helloworld", string(rh.GetCallee()))
	require.Equal(t, "val1", string(rh.GetTransInfo()["key1"]))
	require.Equal(t, "val2", string(rh.GetTransInfo()["key2"]))

	// Perform a test for JSON unmarshal failed.
	req.Header.Set(thttp.TrpcTransInfo, `{"key1":"dmFsMQ==", "key2":"dmFsMg=="`)
	rsp.Reset()
	_, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	require.NotNil(t, err)

	// Perform a test for Base64 decode failed.
	req.Header.Set(thttp.TrpcTransInfo, fmt.Sprintf(`{"%s":"%s"}`, thttp.TrpcEnv, "Production"))
	rsp.Reset()
	_, err = thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	rh, _ = msg.ServerReqHead().(*trpc.RequestProtocol)
	require.Nil(t, err)
	require.Equal(t, "Production", string(rh.GetTransInfo()[thttp.TrpcEnv]))
}

func TestFastHTTPServerDecodeMultipartForm(t *testing.T) {
	requestCtx := &fasthttp.RequestCtx{}
	requestCtx.Request.Reset()
	requestCtx.Response.Reset()
	requestCtx.Request.Header.SetMethod(fasthttp.MethodPost)
	uri := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(uri)
	uri.SetScheme("http")
	uri.SetHost("test.com")
	uri.SetPath("/path")

	queryArgs := uri.QueryArgs()
	queryArgs.Set("queryArgs1", "queryVal1")
	queryArgs.Set("queryArgs2", "queryVal2")
	requestCtx.Request.SetURI(uri)

	postArgs := requestCtx.Request.PostArgs()
	postArgs.Set("postArgs1", "postVal1")
	postArgs.Set("postArgs2", "postVal2")

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("multipartParam1", "value1")
	writer.WriteField("multipartParam2", "value2")

	fileWriter, err := writer.CreateFormFile("file", "example.txt")
	if err != nil {
		panic("Failed to create form file: " + err.Error())
	}
	fileWriter.Write([]byte("This is the content of the file."))
	writer.Close()

	requestCtx.Request.Header.SetContentType(writer.FormDataContentType())
	requestCtx.Request.SetBody(buf.Bytes())

	ctx := thttp.WithRequestCtx(context.Background(), requestCtx)
	msg := codec.Message(ctx)
	bs, err := thttp.DefaultFastHTTPServerCodec.Decode(msg, nil)
	t.Log(string(bs))
	require.Nil(t, err)
}

func TestFastHTTPClientEncode(t *testing.T) {
	// Perform a test for both reqHeader and rspHeader are nil.
	_, msg := codec.WithNewMessage(context.Background())
	_, err := thttp.DefaultFastHTTPClientCodec.Encode(
		msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err)
	require.NotNil(t, msg.ClientReqHead())

	// Perform a test for normal case.
	_, msg = codec.WithNewMessage(context.Background())
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	ReqHeader := &thttp.FastHTTPClientReqHeader{Request: req}
	msg.WithClientReqHead(ReqHeader)
	_, err = thttp.DefaultFastHTTPClientCodec.Encode(
		msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err)
	require.NotNil(t, msg.ClientReqHead())

	// Perform a test for invalid type of reqHeader.
	_, msg = codec.WithNewMessage(context.Background())
	invalidReqHeader := &thttp.ClientReqHeader{}
	msg.WithClientReqHead(invalidReqHeader)
	_, err = thttp.DefaultFastHTTPClientCodec.Encode(msg, nil)
	require.NotNil(t, err)

	// Perform a test for invalid type of rspHeader.
	_, msg = codec.WithNewMessage(context.Background())
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	_, err = thttp.DefaultFastHTTPClientCodec.Encode(msg, nil)
	require.NotNil(t, err)

	// Perform a test for FastHTTPClientRspHeader With SSEHandler.
	_, msg = codec.WithNewMessage(context.Background())
	ReqHeader = &thttp.FastHTTPClientReqHeader{Request: req}
	msg.WithClientReqHead(ReqHeader)
	RspHeader := &thttp.FastHTTPClientRspHeader{SSEHandler: &NoopSSEHandler{}}
	msg.WithClientRspHead(RspHeader)
	_, err = thttp.DefaultFastHTTPClientCodec.Encode(
		msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err)
}

type ErrSSEHandler struct{}

type NoopSSEHandler struct{}

func (h *NoopSSEHandler) Handle(*sse.Event) error {
	return nil
}

func (h *ErrSSEHandler) Handle(*sse.Event) error {
	return errors.New("ErrSSEHandler")
}
func TestFastHTTPClientDecode(t *testing.T) {
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(rsp)

	// Perform a test for normal case.
	rsp.SetStatusCode(200)
	rsp.Header.SetContentEncoding("gzip")
	rsp.Header.SetContentType("application/json")
	rsp.Header.Add(thttp.TrpcTransInfo, `{"key1":"val1", "key2":"val2"}`)
	rsp.SetBodyString(respTests[1].Body)
	_, msg := codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})
	body, err := thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.Nil(t, err)
	require.NotNil(t, msg.ClientRspHead())
	require.Equal(t, respTests[1].Body, string(body), "body is error", string(body))
	require.Equal(t, codec.CompressTypeGzip, msg.CompressType())
}

func TestFastHTTPClientErrDecode(t *testing.T) {
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(rsp)

	// Perform a test for ClientRspHead is nil.
	_, msg := codec.WithNewMessage(context.Background())
	_, err := thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NotNil(t, err)

	// Perform a test for mismatch type for FastHTTPClientRspHead.
	_, msg = codec.WithNewMessage(context.Background())
	// Perform a test for the case that wants Rsp but gets Req.
	msg.WithClientRspHead(&thttp.FastHTTPClientReqHeader{})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NotNil(t, err)

	// Perform a test for HandleSSE error.
	rsp.Reset()
	rsp.SetBody([]byte{1, 2, 3, 4, 5})
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp, SSEHandler: &ErrSSEHandler{}})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.NotNil(t, err)

	// Perform a test for Status Code >= fasthttp.StatusMultipleChoices.
	rsp.Reset()
	rsp.SetStatusCode(fasthttp.StatusMultipleChoices)
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.Nil(t, err)
	require.NotNil(t, msg.ClientRspErr())

	// Perform a test for ErrHandle: FrameworkError.
	rsp.Reset()
	rsp.Header.Add(thttp.TrpcFrameworkErrorCode, "1")
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.Nil(t, err)
	require.NotNil(t, msg.ClientRspErr())
	require.Equal(t, 1, errs.Code(msg.ClientRspErr()))

	// Perform a test for ErrHandle: UserFuncError.
	rsp.Reset()
	rsp.Header.Add(thttp.TrpcUserFuncErrorCode, "10086")
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithClientRspHead(&thttp.FastHTTPClientRspHeader{Response: rsp})
	_, err = thttp.DefaultFastHTTPClientCodec.Decode(msg, nil)
	require.Nil(t, err)
	require.NotNil(t, msg.ClientRspErr())
	require.Equal(t, 10086, errs.Code(msg.ClientRspErr()))
}

func TestServerCodecDecodeDisabledAuto(t *testing.T) {
	sc := thttp.FastHTTPServerCodec{
		AutoReadBody:    false,
		AutoGenTrpcHead: false,
	}

	// AutoReadBody == false, getReqBody will not be executed.
	// AutoGenTrpcHead == false, setReqHeader will not be executed.
	requestCtx := &fasthttp.RequestCtx{}
	ctx := thttp.WithRequestCtx(context.Background(), requestCtx)
	msg := codec.Message(ctx)
	bs, err := sc.Decode(msg, nil)
	require.Nil(t, bs)
	require.Nil(t, err)
}

func TestCoexistenceOfFastHTTPRPCAndNoProtocol(t *testing.T) {
	defer func() { thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0] }()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.test.hello.service" + t.Name()
	s := server.New(
		server.WithServiceName(serviceName),
		server.WithListener(ln),
		// Although the "fasthttp" protocol is represented as an FASTHTTP-RPC service and
		// the standard FASTHTTP service has its own protocol "fasthttp_no_protocol",
		// some users require that both protocols can coexist in the same service
		// (with the same ip and port).
		// This requires that the standard FASTHTTP handler function can still read the
		// request body, even if the `AutoReadBody` field in the default server
		// codec `DefaultFastHTTPServerCodec` for the `fasthttp` protocol is `true`.
		server.WithProtocol(protocol.FastHTTP),
	)
	thttp.FastHTTPHandleFunc("/", func(ctx *fasthttp.RequestCtx) {
		s := &codec.JSONPBSerialization{}
		body := ctx.Request.Body()
		req := &helloworld.HelloRequest{}
		if err := s.Unmarshal(body, req); err != nil {
			t.Log(err)
		}
		rsp := &helloworld.HelloReply{Message: req.Name}
		body, err = s.Marshal(rsp)
		if err != nil {
			t.Log(err)
		}
		ctx.Response.SetStatusCode(fasthttp.StatusOK)
		ctx.Write(body)
	})
	thttp.RegisterNoProtocolService(s)
	// Register protocol file service (HTTP RPC) implementation.
	helloworld.RegisterGreeterService(s, &greeterImpl{})

	// Start server.
	go s.Serve()

	ctx := context.Background()
	target := "ip://" + ln.Addr().String()

	// Send FastHTTP request.
	c := thttp.NewFastHTTPClientProxy(serviceName, client.WithTarget(target))
	msg := "hello"
	req := &helloworld.HelloRequest{Name: msg}
	rsp := &helloworld.HelloReply{}
	require.Nil(t, c.Post(ctx, "/", req, rsp,
		client.WithSerializationType(codec.SerializationTypeJSON)))
	require.Equal(t, msg, rsp.Message)

	// Send FASTHTTP-RPC request.
	proxy := helloworld.NewGreeterClientProxy(client.WithTarget(target), client.WithProtocol("http"))
	resp, err := proxy.SayHello(ctx, &helloworld.HelloRequest{Name: msg})
	require.Nil(t, err)
	require.Equal(t, msg, resp.Message)
}
