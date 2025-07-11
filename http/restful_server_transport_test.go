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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	itls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestCompatibility(t *testing.T) {
	// Registers service.
	serviceName := "trpc.test.server.Greeter" + t.Name()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	url := "http://" + ln.Addr().String()
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName(serviceName),
		server.WithProtocol("restful"),
	)
	s.AddService(serviceName, service)
	helloworld.RegisterGreeterService(s, &greeterServerImpl{})

	go func() { require.Nil(t, s.Serve()) }()
	defer s.Close(nil)

	time.Sleep(100 * time.Millisecond)

	// Removes compatibility setting.
	restful.SetCtxForCompatibility(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			return ctx
		},
	)

	// Sends restful request.
	req1, err := http.NewRequest("POST", url+"/v1/foobar",
		bytes.NewBuffer([]byte(`{"name": "xyz"}`)))
	require.Nil(t, err)
	cli := http.Client{}
	resp1, err := cli.Do(req1)
	require.Nil(t, err)
	defer resp1.Body.Close()
	require.Equal(t, resp1.StatusCode, http.StatusInternalServerError)

	// Adds compatibility setting.
	restful.SetCtxForCompatibility(func(ctx context.Context, w http.ResponseWriter,
		r *http.Request) context.Context {
		return thttp.WithHeader(ctx, &thttp.Header{Response: w, Request: r})
	})

	// Sends restful request.
	req2, err := http.NewRequest("POST", url+"/v1/foobar",
		bytes.NewBuffer([]byte(`{"name": "xyz"}`)))
	require.Nil(t, err)
	resp2, err := cli.Do(req2)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, resp2.StatusCode, http.StatusOK)
}

func TestEnableTLS(t *testing.T) {
	// Registers service.
	s := &server.Server{}
	conf, err := itls.GetServerConfig("../testdata/ca.pem", "../testdata/server.crt", "../testdata/server.key")
	require.Nil(t, err, "%+v", err)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", conf)
	require.Nil(t, err)
	defer ln.Close()
	addr := strings.Split(ln.Addr().String(), ":")
	require.Equal(t, 2, len(addr))
	port := addr[1]
	// Must use localhost to replace 127.0.0.1, or else the following error will occur:
	// tls: failed to verify certificate: x509: cannot validate certificate for 127.0.0.1 because it doesn't contain any IP SANs.
	url := "https://localhost:" + port
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.test.helloworld.Greeter"),
		server.WithProtocol("restful"),
	)
	s.AddService("trpc.test.helloworld.Greeter", service)
	helloworld.RegisterGreeterService(s, &greeterServerImpl{})

	go func() { require.Nil(t, s.Serve()) }()
	defer s.Close(nil)

	time.Sleep(100 * time.Millisecond)

	// Sends https request.
	pool := x509.NewCertPool()
	ca, err := os.ReadFile("../testdata/ca.pem")
	require.Nil(t, err)
	pool.AppendCertsFromPEM(ca)
	cert, err := tls.LoadX509KeyPair("../testdata/client.crt", "../testdata/client.key")
	require.Nil(t, err)

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      pool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	req, err := http.NewRequest("POST", url+"/v1/foobar",
		bytes.NewBuffer([]byte(`{"name": "xyz"}`)))
	require.Nil(t, err)

	resp, err := cli.Do(req)
	require.Nil(t, err, "%+v", err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, http.StatusOK)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	type responseBody struct {
		Message string `json:"message"`
	}
	respBody := &responseBody{}
	json.Unmarshal(bodyBytes, respBody)
	require.Equal(t, respBody.Message, "test restful server transport")
}

func TestReplaceRouter(t *testing.T) {
	st := thttp.NewRESTServerTransport(true, transport.WithReusePort(true))
	restful.RegisterRouter("replacing", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	restful.RegisterRouter("no_replacing", restful.NewRouter())
	err := st.ListenAndServe(context.Background(), transport.WithServiceName("replacing"))
	require.NotNil(t, err)
	err = st.ListenAndServe(context.Background(), transport.WithServiceName("no_replacing"))
	require.Nil(t, err)
}

var (
	headerMatcherTransInfo, _ = json.Marshal(map[string]string{
		"kfuin": base64.StdEncoding.EncodeToString([]byte("3009025887")),
	})
)

func TestDefaultRESTHeaderMatcher(t *testing.T) {
	bgctx := trpc.BackgroundContext()
	req := http.Request{Header: make(http.Header)}
	req.Header.Set(thttp.TrpcCaller, "TestDefaultHeaderMatcher")
	req.Header.Set(thttp.TrpcTransInfo, string(headerMatcherTransInfo))
	req.Header.Set(thttp.TrpcTimeout, "2000")
	req.Header.Set(thttp.TrpcMessageType, "1")
	ctx, err := thttp.DefaultRESTHeaderMatcher(bgctx, nil, &req, "UTService", "UTMethod")
	require.Nil(t, err)
	msg := codec.Message(ctx)
	require.Equal(t, "UTService", msg.CalleeServiceName())
	require.Equal(t, "UTMethod", msg.ServerRPCName())
	require.Equal(t, "TestDefaultHeaderMatcher", msg.CallerServiceName())
	require.Equal(t, time.Duration(2000*time.Millisecond), msg.RequestTimeout())
	require.Equal(t, "3009025887", string(trpc.GetMetaData(ctx, "kfuin")))
	require.Equal(t, true, msg.Dyeing())

	req.Header.Set(thttp.TrpcTransInfo, "")
	req.Header.Set(thttp.TrpcMessageType, "0")
	ctx, err = thttp.DefaultRESTHeaderMatcher(bgctx, nil, &req, "UTService", "UTMethod")
	require.Nil(t, err)
	msg = codec.Message(ctx)
	require.Equal(t, "", string(trpc.GetMetaData(ctx, "kfuin")))
	require.Equal(t, false, msg.Dyeing())
}

func TestDefaultRESTFastHTTPHeaderMatcher(t *testing.T) {
	bgctx := trpc.BackgroundContext()
	req := fasthttp.RequestCtx{}
	req.Request.Header.Set(thttp.TrpcCaller, "TestDefaultHeaderMatcher")
	req.Request.Header.Set(thttp.TrpcTransInfo, string(headerMatcherTransInfo))
	req.Request.Header.Set(thttp.TrpcTimeout, "2000")
	req.Request.Header.Set(thttp.TrpcMessageType, "1")
	ctx, err := thttp.DefaultRESTFastHTTPHeaderMatcher(bgctx, &req, "UTService", "UTMethod")
	require.Nil(t, err)
	msg := codec.Message(ctx)
	require.Equal(t, "UTService", msg.CalleeServiceName())
	require.Equal(t, "UTMethod", msg.ServerRPCName())
	require.Equal(t, "TestDefaultHeaderMatcher", msg.CallerServiceName())
	require.Equal(t, time.Duration(2000*time.Millisecond), msg.RequestTimeout())
	require.Equal(t, "3009025887", string(trpc.GetMetaData(ctx, "kfuin")))
	require.Equal(t, true, msg.Dyeing())

	req = fasthttp.RequestCtx{}
	req.Request.Header.Set(thttp.TrpcTransInfo, "xyz")
	_, err = thttp.DefaultRESTFastHTTPHeaderMatcher(bgctx, &req, "UTService", "UTMethod")
	require.NotNil(t, err)
}

func TestPassListenerUseTLS(t *testing.T) {
	// Registers service.
	serviceName := "trpc.test.helloworld.Greeter" + t.Name()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := strings.Split(ln.Addr().String(), ":")
	require.Equal(t, 2, len(addr))
	port := addr[1]
	// Must use localhost to replace 127.0.0.1, or else the following error will occur:
	// tls: failed to verify certificate: x509: cannot validate certificate for 127.0.0.1 because it doesn't contain any IP SANs.
	url := "https://localhost:" + port
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName(serviceName),
		server.WithProtocol("restful"),
		server.WithTLS("../testdata/server.crt", "../testdata/server.key", "../testdata/ca.pem"),
	)
	s.AddService(serviceName, service)
	helloworld.RegisterGreeterService(s, &greeterServerImpl{})

	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()
	defer s.Close(nil)

	time.Sleep(100 * time.Millisecond)

	// Sends https request.
	pool := x509.NewCertPool()
	ca, err := os.ReadFile("../testdata/ca.pem")
	require.Nil(t, err)
	pool.AppendCertsFromPEM(ca)
	cert, err := tls.LoadX509KeyPair("../testdata/client.crt", "../testdata/client.key")
	require.Nil(t, err)

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      pool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	req, err := http.NewRequest("POST", url+"/v1/foobar",
		bytes.NewBuffer([]byte(`{"name": "xyz"}`)))
	require.Nil(t, err)

	resp, err := cli.Do(req)
	require.Nil(t, err, "err: %+v", err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, http.StatusOK)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	type responseBody struct {
		Message string `json:"message"`
	}
	respBody := &responseBody{}
	json.Unmarshal(bodyBytes, respBody)
	require.Equal(t, respBody.Message, "test restful server transport")
}

func TestListenAndServeInvalidAddrErr(t *testing.T) {
	serviceName := "trpc.test.helloworld.Greeter" + t.Name()
	s := &server.Server{}
	invalidAddr := "888.888.888.888:88888"
	service := server.New(
		server.WithAddress(invalidAddr),
		server.WithServiceName(serviceName),
		server.WithProtocol("restful"),
	)
	s.AddService(serviceName, service)
	require.NotNil(t, s.Serve())
}

type greeterServerImpl struct{}

func (s *greeterServerImpl) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	rsp := &helloworld.HelloReply{}
	if thttp.Head(ctx) == nil {
		return nil, errors.New("test error")
	}
	rsp.Message = "test restful server transport"
	return rsp, nil
}
