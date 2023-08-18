// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package http_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	thttp "trpc.group/trpc-go/trpc-go/http"
)

func TestStdHTTPClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("unsupported method"))
			return
		}
		if _, err := io.Copy(w, r.Body); err != nil {
			w.Write([]byte(err.Error()))
		}
	}))
	defer ts.Close()

	body := []byte(`{"name": "xyz"}`)
	cli := thttp.NewStdHTTPClient("trpc.http.stdclient.test")

	rsp1, err1 := cli.Get(ts.URL)
	require.Nil(t, err1)
	require.Equal(t, http.StatusOK, rsp1.StatusCode)
	require.Equal(t, int64(0), rsp1.ContentLength)

	rsp2, err2 := cli.Post(ts.URL, "application/json", bytes.NewBuffer(body))
	require.Nil(t, err2)
	require.Equal(t, http.StatusOK, rsp2.StatusCode)

	rspBody2, err := io.ReadAll(rsp2.Body)
	defer rsp2.Body.Close()
	require.Nil(t, err)
	require.Equal(t, body, rspBody2)

	req, _ := http.NewRequest("PUT", ts.URL, bytes.NewBuffer(body))
	rsp3, err3 := cli.Do(req)
	require.Nil(t, err3)
	require.Equal(t, http.StatusInternalServerError, rsp3.StatusCode)

	rspBody3, err := io.ReadAll(rsp3.Body)
	defer rsp3.Body.Close()
	require.Nil(t, err)
	require.Equal(t, "unsupported method", string(rspBody3))
}
