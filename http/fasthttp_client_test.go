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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func TestFastHTTPClientStdServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("unsupported method"))
			return
		}
		_, _ = io.Copy(w, r.Body)
	}))
	defer ts.Close()

	fc := thttp.NewFastHTTPClient("trpc.fasthttp.client.test")

	status, body, err := fc.Get(nil, ts.URL)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, status)
	require.Empty(t, body)

	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.Header.SetRequestURI(ts.URL)
	req.SetBodyString(`{"name":"trpc"}`)
	require.NoError(t, fc.Do(req, rsp))
	require.Equal(t, fasthttp.StatusOK, rsp.StatusCode())
	require.Equal(t, `{"name":"trpc"}`, string(rsp.Body()))

	req.Reset()
	rsp.Reset()
	req.Header.SetMethod(fasthttp.MethodPut)
	req.SetRequestURI(ts.URL)
	require.NoError(t, fc.Do(req, rsp))
	require.Equal(t, fasthttp.StatusInternalServerError, rsp.StatusCode())
	require.Equal(t, "unsupported method", string(rsp.Body()))
}

func TestFastHTTPClientProxyStdServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("unsupported method"))
			return
		}
		_, _ = io.Copy(w, r.Body)
	}))
	defer ts.Close()

	target := strings.Replace(ts.URL, "http", "ip", 1)
	fcp := thttp.NewFastHTTPClientProxy("trpc.fasthttp.client.test", client.WithTarget(target))
	reqBody := &codec.Body{}
	rspBody := &codec.Body{}

	require.NoError(t, fcp.Get(context.Background(), "", rspBody))
	require.Nil(t, rspBody.Data)

	reqBody.Data = []byte(`{"name":"trpc"}`)
	require.NoError(t, fcp.Post(context.Background(), "", reqBody, rspBody))
	require.Equal(t, reqBody.Data, rspBody.Data)

	rspBody.Data = nil
	require.Error(t, fcp.Put(context.Background(), "", reqBody, rspBody))

	rspBody.Data = nil
	require.NoError(t, fcp.Patch(context.Background(), "", reqBody, rspBody))
	require.Equal(t, reqBody.Data, rspBody.Data)

	rspBody.Data = nil
	require.NoError(t, fcp.Delete(context.Background(), "", reqBody, rspBody))
	require.Equal(t, reqBody.Data, rspBody.Data)
}

func TestFastHTTPPassThroughError(t *testing.T) {
	c := thttp.NewFastHTTPClient("trpc.fasthttp.client.test", client.WithTarget("ip://127.0.0.1:1"))
	_, _, err := c.Get(nil, "http://127.0.0.1:1")
	require.Error(t, err)
}
