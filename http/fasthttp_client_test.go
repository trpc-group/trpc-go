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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestFastHTTPClientStdServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("unsupported method"))
			return
		}
		if _, err := io.Copy(w, r.Body); err != nil {
			w.Write([]byte(err.Error()))
		}
	}))
	defer ts.Close()

	fc := thttp.NewFastHTTPClient("trpc.fasthttp.client.test")

	// Perform a GET request.
	code1, rsp1, err1 := fc.Get(nil, ts.URL)
	require.Nil(t, err1)
	require.Equal(t, fasthttp.StatusOK, code1)
	require.Nil(t, rsp1)

	// Perform a POST request.
	body := []byte(`{"name": "xyz"}`)
	req2 := fasthttp.AcquireRequest()
	rsp2 := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req2)
	defer fasthttp.ReleaseResponse(rsp2)
	req2.Header.SetMethod(fasthttp.MethodPost)
	req2.Header.SetContentType("application/json")
	req2.Header.SetRequestURI(ts.URL)
	req2.SetBody(body)
	err2 := fc.Do(req2, rsp2)
	require.Nil(t, err2)
	require.Equal(t, fasthttp.StatusOK, rsp2.StatusCode())
	require.Equal(t, body, rsp2.Body())

	// Perform a PUT request.
	req3 := fasthttp.AcquireRequest()
	rsp3 := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req3)
	defer fasthttp.ReleaseResponse(rsp3)
	req3.Header.SetMethod(fasthttp.MethodPut)
	req3.SetRequestURI(ts.URL)
	err3 := fc.Do(req3, rsp3)
	require.Nil(t, err3)
	require.Equal(t, fasthttp.StatusInternalServerError, rsp3.StatusCode())
	require.Equal(t, "unsupported method", string(rsp3.Body()))
}

func TestFastHTTPClientFastHTTPServer(t *testing.T) {
	go fasthttp.ListenAndServe("127.0.0.1:8088", func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Method()) == fasthttp.MethodPut {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.WriteString("unsupported method")
		}
		ctx.Write(ctx.Request.Body())
	})
	time.Sleep(time.Second)

	cli := thttp.NewFastHTTPClient("trpc.fasthttp.client.test")

	// Perform a GET request.
	code1, rsp1, err1 := cli.Get(nil, "http://127.0.0.1:8088")
	require.Nil(t, err1)
	require.Equal(t, fasthttp.StatusOK, code1)
	require.Equal(t, 0, len(rsp1))

	// Perform a POST request.
	body := []byte(`{"name": "xyz"}`)
	req2 := fasthttp.AcquireRequest()
	rsp2 := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req2)
	defer fasthttp.ReleaseResponse(rsp2)
	req2.Header.SetMethod(fasthttp.MethodPost)
	req2.Header.SetContentType("application/json")
	req2.Header.SetRequestURI("http://127.0.0.1:8088")
	req2.SetBody(body)
	err2 := cli.Do(req2, rsp2)
	require.Nil(t, err2)
	require.Equal(t, fasthttp.StatusOK, rsp2.StatusCode())
	require.Equal(t, body, rsp2.Body())

	// Perform a PUT request.
	req3 := fasthttp.AcquireRequest()
	rsp3 := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req3)
	defer fasthttp.ReleaseResponse(rsp3)
	req3.Header.SetMethod(fasthttp.MethodPut)
	req3.SetRequestURI("http://127.0.0.1:8088")
	err3 := cli.Do(req3, rsp3)
	require.Nil(t, err3)
	require.Equal(t, fasthttp.StatusInternalServerError, rsp3.StatusCode())
	require.Equal(t, "unsupported method", string(rsp3.Body()))
}

func TestFastHTTPProxyStdServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("unsupported method"))
			return
		}
		if _, err := io.Copy(w, r.Body); err != nil {
			w.Write([]byte(err.Error()))
		}
	}))
	defer ts.Close()

	target := strings.Replace(ts.URL, "http", "ip", 1)
	fcp := thttp.NewFastHTTPClientProxy("trpc.fasthttp.client.test", client.WithTarget(target))

	reqBody := &codec.Body{}
	rspBody := &codec.Body{}

	// Perform a GET request.
	err := fcp.Get(context.Background(), "", rspBody)
	require.Nil(t, err)
	require.Nil(t, rspBody.Data)

	// Perform a POST request.
	reqBody.Data = []byte(`{"name": "xyz"}`)
	err = fcp.Post(context.Background(), "", reqBody, rspBody)
	require.Nil(t, err)
	require.Equal(t, reqBody.Data, rspBody.Data)

	// Perform a PUT request.
	rspBody.Data = []byte{}
	err = fcp.Put(context.Background(), "", reqBody, rspBody)
	require.NotNil(t, err)

	// Perform a PATCH request.
	reqBody.Data = []byte(`{"name": "xyz"}`)
	err = fcp.Patch(context.Background(), "", reqBody, rspBody)
	require.Nil(t, err)
	require.Equal(t, reqBody.Data, rspBody.Data)

	// Perform a DELETE request.
	reqBody.Data = []byte(`{"name": "xyz"}`)
	err = fcp.Delete(context.Background(), "", reqBody, rspBody)
	require.Nil(t, err)
	require.Equal(t, reqBody.Data, rspBody.Data)
}

func TestFastHTTPProxyFastHTTPServer(t *testing.T) {
	go fasthttp.ListenAndServe("127.0.0.1:8099", func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Method()) == fasthttp.MethodPut {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.WriteString("unsupported method")
			return
		}
		ctx.Write(ctx.Request.Body())
	})
	time.Sleep(time.Second)

	fcp := thttp.NewFastHTTPClientProxy("trpc.http.fastClient.test", client.WithTarget("ip://127.0.0.1:8099"))
	reqBody := &codec.Body{}
	rspBody := &codec.Body{}

	// Perform a GET request.
	err := fcp.Get(context.Background(), "", rspBody)
	require.Nil(t, err)
	require.Nil(t, rspBody.Data)

	// Perform a POST request.
	reqBody.Data = []byte(`{"name": "xyz"}`)
	err = fcp.Post(context.Background(), "", reqBody, rspBody)
	require.Nil(t, err)
	require.Equal(t, reqBody.Data, rspBody.Data)

	// Perform a PUT request.
	rspBody.Data = []byte{}
	err = fcp.Put(context.Background(), "", reqBody, rspBody)
	t.Log(string(rspBody.Data))
	require.NotNil(t, err, err)
}

func TestPassThrough(t *testing.T) {
	c := thttp.NewFastHTTPClient("trpc.http.fastClient.test", client.WithTarget("ip://1.1.1.1:12345"))
	_, _, err := c.Get(nil, "http://127.0.0.1:21932")
	require.Contains(t, err.Error(), "dial tcp4 127.0.0.1:21932: connect: connection refused")
}
