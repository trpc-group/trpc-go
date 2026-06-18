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

package restful_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
)

type filterCounts struct {
	total         int32
	pbRoute       int32
	customRoute   int32
	customRPCName atomic.Value
}

func (c *filterCounts) filter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (
	interface{}, error,
) {
	atomic.AddInt32(&c.total, 1)
	if _, ok := req.(*helloworld.HelloRequest); ok {
		atomic.AddInt32(&c.pbRoute, 1)
	}
	if req == nil {
		atomic.AddInt32(&c.customRoute, 1)
		c.customRPCName.Store(codec.Message(ctx).ServerRPCName())
	}
	return next(ctx, req)
}

func TestRegisterRouterDefaultBypassesCustomRouteFilters(t *testing.T) {
	addr, counts := startRESTfulMuxServer(t, false)

	body := doGET(t, addr+"/v2/bar/world", http.StatusOK)
	require.Equal(t, `{"message":"world"}`, body)
	require.Equal(t, int32(1), atomic.LoadInt32(&counts.total))
	require.Equal(t, int32(1), atomic.LoadInt32(&counts.pbRoute))

	body = doGET(t, addr+"/custom/ping", http.StatusOK)
	require.Equal(t, "pong", body)
	require.Equal(t, int32(1), atomic.LoadInt32(&counts.total))
	require.Equal(t, int32(0), atomic.LoadInt32(&counts.customRoute))
}

func TestWrapHandlerWithServerFilters(t *testing.T) {
	addr, counts := startRESTfulMuxServer(t, true)

	body := doGET(t, addr+"/v2/bar/world", http.StatusOK)
	require.Equal(t, `{"message":"world"}`, body)
	require.Equal(t, int32(1), atomic.LoadInt32(&counts.total))
	require.Equal(t, int32(1), atomic.LoadInt32(&counts.pbRoute))
	require.Equal(t, int32(0), atomic.LoadInt32(&counts.customRoute))

	body = doGET(t, addr+"/custom/ping", http.StatusOK)
	require.Equal(t, "pong", body)
	require.Equal(t, int32(2), atomic.LoadInt32(&counts.total))
	require.Equal(t, int32(1), atomic.LoadInt32(&counts.pbRoute))
	require.Equal(t, int32(1), atomic.LoadInt32(&counts.customRoute))
	require.Equal(t, "/custom/ping", counts.customRPCName.Load())
}

func TestWrapHandlerWithServerFiltersPanicsWithoutRouter(t *testing.T) {
	require.PanicsWithValue(t,
		fmt.Sprintf("restful: WrapHandlerWithServerFilters requires the original *restful.Router "+
			"to be registered before wrapping service %q; current router is %T", t.Name(), nil),
		func() {
			restful.WrapHandlerWithServerFilters(t.Name(), http.NewServeMux())
		},
	)
}

func startRESTfulMuxServer(t *testing.T, wrap bool) (string, *filterCounts) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	counts := &filterCounts{}
	serviceName := t.Name()
	s := server.New(
		server.WithListener(ln),
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithFilter(counts.filter),
	)
	RegisterGreeterService(s, &greeter{})

	router := restful.GetRouter(serviceName)
	require.NotNil(t, router)
	mux := http.NewServeMux()
	mux.Handle("/", router)
	mux.HandleFunc("/custom/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})

	handler := http.Handler(mux)
	if wrap {
		handler = restful.WrapHandlerWithServerFilters(serviceName, mux)
	}
	restful.RegisterRouter(serviceName, handler)

	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(200 * time.Millisecond):
	}
	t.Cleanup(func() {
		s.Close(nil)
		_ = ln.Close()
	})
	return "http://" + ln.Addr().String(), counts
}

func doGET(t *testing.T, url string, wantStatus int) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	rsp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, wantStatus, rsp.StatusCode)
	body, err := io.ReadAll(rsp.Body)
	require.NoError(t, err)
	return string(body)
}
