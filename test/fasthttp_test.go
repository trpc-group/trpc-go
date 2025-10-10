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

package test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
	"trpc.group/trpc-go/trpc-go/transport"
)

func (s *TestSuite) TestFastHTTPCustomErrorHandler() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		oldErrHandler := thttp.DefaultFastHTTPServerCodec.ErrHandler
		thttp.DefaultFastHTTPServerCodec.ErrHandler = func(ctx *fasthttp.RequestCtx, e *errs.Error) {
			ctx.Response.Header.Set("Custom-Error", fmt.Sprintf(`{"ret-code":%d, "ret-msg":"%s"}`, e.Code, e.Msg))
		}
		defer func() {
			thttp.DefaultFastHTTPServerCodec.ErrHandler = oldErrHandler
		}()
		s.startServer(&testFastHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testCustomErrorHandler(e) })
		s.Run(e.String(), func() { s.testFastHTTPCustomErrorHandler(e) })
	}
}
func (s *TestSuite) testFastHTTPCustomErrorHandler(e *httpRPCEnv) {
	type customError struct {
		RetCode int    `json:"ret-code"`
		RetMsg  string `json:"ret-msg"`
	}
	fasthttpRspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(fasthttpRspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.ResponseType = testpb.PayloadType_RANDOM
	_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Nil(s.T(), err)
	ce := &customError{}
	require.Nil(s.T(), json.Unmarshal([]byte(fasthttpRspHead.Response.Header.Peek("custom-error")), ce))
	require.Equal(s.T(), retUnsupportedPayload, ce.RetCode)
}

func (s *TestSuite) TestFastHTTPClientReqAndRspHeader() {
	s.startServer(&testFastHTTPService{})

	s.Run("http", func() { s.testClientReqAndRspHeader() })
	s.Run("fasthttp", func() { s.testFastHTTPClientReqAndRspHeader() })
}
func (s *TestSuite) testFastHTTPClientReqAndRspHeader() {
	s.T().Run("ReqHead is not *FastHTTPClientReqHeader", func(t *testing.T) {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)

		_, err := s.newFastHTTPRPCClient(client.WithReqHead("string type")).UnaryCall(context.Background(), req)
		require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "fasthttp header must be type of *FastHTTPClientReqHeader")

		_, err = s.newFastHTTPRPCClient(client.WithReqHead(nil)).UnaryCall(context.Background(), req)
		require.Nil(t, err)
	})
	s.T().Run("RspHead is not *FastHTTPClientReqHeader", func(t *testing.T) {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)

		_, err := s.newFastHTTPRPCClient(client.WithRspHead("string type")).UnaryCall(context.Background(), req)
		require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "fasthttp header must be type of *FastHTTPClientRspHeader")

		_, err = s.newFastHTTPRPCClient(client.WithRspHead(nil)).UnaryCall(context.Background(), req)
		require.Nil(t, err)
	})
}

func (s *TestSuite) TestFastHTTPDefaultErrorHandler() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testFastHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testDefaultErrorHandler(e) })
		s.Run(e.String(), func() { s.testFastHTTPDefaultErrorHandler(e) })
	}
}
func (s *TestSuite) testFastHTTPDefaultErrorHandler(e *httpRPCEnv) {
	fasthttpRspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(fasthttpRspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}

	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.ResponseType = testpb.PayloadType_RANDOM

	_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)
	require.Equal(s.T(), retUnsupportedPayload, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "unsupported payload type")
	require.Equal(s.T(), fmt.Sprint(errs.Code(err)), string(fasthttpRspHead.Response.Header.Peek(thttp.TrpcUserFuncErrorCode)))
	require.Equal(s.T(), string(fasthttpRspHead.Response.Header.Peek("Trpc-Error-Msg")), errs.Msg(err), "full err: %+v", err)
	require.Equal(
		s.T(),
		fasthttp.StatusOK,
		fasthttpRspHead.Response.StatusCode(),
		"any framework error code not in thttp.ErrsToHTTPStatus map are converted to fasthttp.StatusOK",
	)
}

func (s *TestSuite) TestFastHTTPHandleErrServerNoResponse() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testFastHTTPService{
			TRPCService: TRPCService{
				UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
					return nil, errs.ErrServerNoResponse
				},
			},
		}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testHandleErrServerNoResponse() })
		s.Run(e.String(), func() { s.testFastHTTPHandleErrServerNoResponse() })
	}
}
func (s *TestSuite) testFastHTTPHandleErrServerNoResponse() {
	bs, err := proto.Marshal(s.defaultSimpleRequest)
	require.Nil(s.T(), err)

	fc := thttp.NewFastHTTPClient("fasthttp-client")
	fasthttpReq := fasthttp.AcquireRequest()
	fasthttpRsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fasthttpReq)
	defer fasthttp.ReleaseResponse(fasthttpRsp)

	fasthttpReq.SetRequestURI(s.unaryCallCustomURL())
	fasthttpReq.Header.SetContentType("application/pb")
	fasthttpReq.SetBody(bs)

	err = fc.Do(fasthttpReq, fasthttpRsp)
	require.Nil(s.T(), err)
	require.Equal(s.T(), fasthttp.StatusInternalServerError, fasthttpRsp.StatusCode())

	bs = fasthttpRsp.Body()
	require.Nil(s.T(), err)
	require.Containsf(s.T(), string(bs), "server handle error: type:framework, code:0, msg:server no response", "full err: %+v", err)
}

func (s *TestSuite) TestFastHTTPSendHTTPSRequestToHTTPServer() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testSendHTTPSRequestToHTTPServer(e) })
		s.Run(e.String(), func() { s.testFastHTTPSendHTTPSRequestToHTTPServer(e) })
	}
}
func (s *TestSuite) testFastHTTPSendHTTPSRequestToHTTPServer(e *httpRPCEnv) {
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: http.MethodPost, Scheme: "https"}),
		client.WithRspHead(&thttp.FastHTTPClientRspHeader{}),
		client.WithProtocol(protocol.FastHTTP),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	rsp, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)
	require.Nil(s.T(), rsp)
	require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
}

func (s *TestSuite) TestFastHTTPStatusBadRequestDueToServerValidateFail() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testFastHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusBadRequestDueToServerValidateFail(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusBadRequestDueToServerValidateFail(e) })
	}
}
func (s *TestSuite) testFastHTTPStatusBadRequestDueToServerValidateFail(e *httpRPCEnv) {
	rspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.Username = "non-validate-name-?.@&*-_"
	_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerValidateFail, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusBadRequest, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), fmt.Sprint(errs.Code(err)), string(rspHead.Response.Header.Peek(thttp.TrpcUserFuncErrorCode)))
	require.Equal(s.T(), string(rspHead.Response.Header.Peek("Trpc-Error-Msg")), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusBadRequest, rspHead.Response.StatusCode())
}

func (s *TestSuite) TestFastHTTPStatusNotFoundDueToServerNoService() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		startServerWithoutAnyService := func(t *testing.T) {
			t.Helper()
			trpc.ServerConfigPath = "trpc_go_fasthttp_server.yaml"

			l, err := net.Listen("tcp", defaultServerAddress)
			if err != nil {
				t.Fatalf("net.Listen(%s) error", defaultServerAddress)
			}
			s.listener = l
			s.T().Logf("server address: %v", l.Addr())

			svr := trpc.NewServer(server.WithListener(s.listener), server.WithServerAsync(e.server.async))
			if svr == nil {
				t.Fatal("trpc.NewServer failed")
			}
			go svr.Serve()
			s.server = svr
		}
		startServerWithoutAnyService(s.T())
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusNotFoundDueToServerNoService(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusNotFoundDueToServerNoService(e) })
	}
}
func (s *TestSuite) testFastHTTPStatusNotFoundDueToServerNoService(e *httpRPCEnv) {
	rspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerNoFunc, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusNotFound, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), string(rspHead.Response.Header.Peek("Trpc-Error-Msg")), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusNotFound, rspHead.Response.StatusCode())
}

func (s *TestSuite) TestFastHTTPStatusNotFoundDueToServerNoFunc() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testFastHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusNotFoundDueToServerNoFunc(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusNotFoundDueToServerNoFunc(e) })
	}
}
func (s *TestSuite) testFastHTTPStatusNotFoundDueToServerNoFunc(e *httpRPCEnv) {
	rspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(rspHead),
		client.WithTarget(s.serverAddress() + "/NonexistentCall"),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerNoFunc, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusNotFound, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), string(rspHead.Response.Header.Peek("Trpc-Error-Msg")), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusNotFound, rspHead.Response.StatusCode())
}

func (s *TestSuite) TestFastHTTPStatusGatewayTimeoutDueToServerTimeout() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(
			&testFastHTTPService{},
			server.WithServerAsync(e.server.async),
			server.WithTimeout(50*time.Millisecond),
			server.WithFilter(
				func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
					return nil, errs.NewFrameError(errs.RetServerTimeout, "")
				},
			),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusGatewayTimeoutDueToServerTimeout(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusGatewayTimeoutDueToServerTimeout(e) })
	}
}
func (s *TestSuite) testFastHTTPStatusGatewayTimeoutDueToServerTimeout(e *httpRPCEnv) {
	RspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(RspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerTimeout, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusGatewayTimeout, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), string(RspHead.Response.Header.Peek("Trpc-Error-Msg")), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusGatewayTimeout, RspHead.Response.StatusCode())
}

func (s *TestSuite) TestFastHTTPStatusTooManyRequestsDueToServerOverload() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		const maxRequestQueueSize = 10
		requestQueue := make(chan interface{}, maxRequestQueueSize)
		defer func() {
			close(requestQueue)
		}()
		const limitedAccessUser = "LimitedAccessUser"

		s.startServer(
			&testFastHTTPService{},
			server.WithServerAsync(e.server.async),
			server.WithFilter(
				func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
					r, ok := req.(*testpb.SimpleRequest)
					if !ok {
						return next(ctx, req)
					}
					if r.Username == limitedAccessUser {
						select {
						case requestQueue <- req:
						default:
							return nil, errs.NewFrameError(errs.RetServerOverload, "requestQueue overflow!")
						}
					}
					return next(ctx, req)
				}),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusTooManyRequestsDueToServerOverload(e) })
		requestQueue = make(chan interface{}, maxRequestQueueSize)
		s.Run(e.String(), func() { s.testFastHTTPStatusTooManyRequestsDueToServerOverload(e) })
	}
}
func (s *TestSuite) testFastHTTPStatusTooManyRequestsDueToServerOverload(e *httpRPCEnv) {
	const maxRequestQueueSize = 10
	const limitedAccessUser = "LimitedAccessUser"

	sendFastHTTPRequest := func() (*thttp.FastHTTPClientRspHeader, error) {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		req.Username = limitedAccessUser
		rspHead := &thttp.FastHTTPClientRspHeader{}
		opts := []client.Option{
			client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
			client.WithRspHead(rspHead),
		}
		if e.client.disableConnectionPool {
			opts = append(opts, client.WithDisableConnectionPool())
		}
		_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)
		return rspHead, err
	}

	var g errgroup.Group
	for i := 0; i < maxRequestQueueSize; i++ {
		g.Go(func() error {
			_, err := sendFastHTTPRequest()
			return err
		})
	}
	require.Zero(s.T(), g.Wait())

	rspHead, err := sendFastHTTPRequest()
	require.Equal(s.T(), errs.RetServerOverload, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusTooManyRequests, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), string(rspHead.Response.Header.Peek("Trpc-Error-Msg")), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusTooManyRequests, rspHead.Response.StatusCode())
}

func (s *TestSuite) TestFastHTTPStatusUnauthorizedDueToServerAuthFail() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testFastHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusUnauthorizedDueToServerAuthFail(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusUnauthorizedDueToServerAuthFail(e) })
	}
}
func (s *TestSuite) testFastHTTPStatusUnauthorizedDueToServerAuthFail(e *httpRPCEnv) {
	rspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}

	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.Username = "invalidUsername"
	req.FillUsername = true

	_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)
	require.Equal(s.T(), errs.RetServerAuthFail, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusUnauthorized, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), string(rspHead.Response.Header.Peek("Trpc-Error-Msg")), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusUnauthorized, rspHead.Response.StatusCode())
}

func (s *TestSuite) TestFastHTTPStatusInternalServerDueToServerReturnUnknown() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(
			&testFastHTTPService{},
			server.WithServerAsync(e.server.async),
			server.WithFilter(
				func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
					return nil, fmt.Errorf("unknown")
				},
			),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusInternalServerDueToServerReturnUnknown(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusInternalServerDueToServerReturnUnknown(e) })
	}
}
func (s *TestSuite) testFastHTTPStatusInternalServerDueToServerReturnUnknown(e *httpRPCEnv) {
	rspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	_, err := s.newFastHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetUnknown, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusInternalServerError, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), fmt.Sprint(errs.Code(err)), string(rspHead.Response.Header.Peek(thttp.TrpcUserFuncErrorCode)))
	require.Equal(s.T(), string(rspHead.Response.Header.Peek("Trpc-Error-Msg")), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusInternalServerError, rspHead.Response.StatusCode())
}

func (s *TestSuite) TestFastHTTPCustomResponseHandler() {
	oldRspHandler := thttp.DefaultFastHTTPServerCodec.RspHandler
	thttp.DefaultFastHTTPServerCodec.RspHandler = func(requestCtx *fasthttp.RequestCtx, rspBody []byte) error {
		require.NotEmpty(s.T(), rspBody)

		var rsp testpb.SimpleResponse
		err := json.Unmarshal(rspBody, &rsp)
		require.Nil(s.T(), err)

		pt := int(rsp.Payload.GetType())
		bs, err := json.Marshal(&customResponse{
			PayloadType: pt,
			PayloadBody: rsp.Payload.GetBody(),
			Username:    rsp.Username,
		})
		require.Nil(s.T(), err)

		_, err = requestCtx.Write(bs)
		require.Nil(s.T(), err)

		return nil
	}
	defer func() {
		thttp.DefaultFastHTTPServerCodec.RspHandler = oldRspHandler
	}()

	s.startServer(&testFastHTTPService{})

	s.Run("http", func() { s.testCustomResponseHandler() })
	s.Run("fasthttp", func() { s.testFastHTTPCustomResponseHandler() })
}
func (s *TestSuite) testFastHTTPCustomResponseHandler() {
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.FillUsername = true
	req.Username = validUserNameForAuth
	bts, err := json.Marshal(&req)
	require.Nil(s.T(), err)

	fasthttpReq := fasthttp.AcquireRequest()
	fasthttpRsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fasthttpReq)
	defer fasthttp.ReleaseResponse(fasthttpRsp)

	fasthttpReq.Header.SetMethod(fasthttp.MethodPost)
	fasthttpReq.SetRequestURI(s.unaryCallCustomURL())
	fasthttpReq.Header.SetContentType("application/json")
	fasthttpReq.SetBody(bts)
	err = fasthttp.Do(fasthttpReq, fasthttpRsp)
	require.Nil(s.T(), err)

	bts = fasthttpRsp.Body()
	ce := customResponse{}
	require.Nil(s.T(), json.Unmarshal(bts, &ce))
	require.Equal(s.T(), validUserNameForAuth, ce.Username)
	require.Equal(s.T(), int(req.ResponseType), ce.PayloadType)
}

func (s *TestSuite) TestFastHTTPCustomResponseHandlerResponseWriteError() {
	oldRspHandler := thttp.DefaultFastHTTPServerCodec.RspHandler
	thttp.DefaultFastHTTPServerCodec.RspHandler = func(requestCtx *fasthttp.RequestCtx, rspBody []byte) error {
		return errors.New("writing failed")
	}
	defer func() {
		thttp.DefaultFastHTTPServerCodec.RspHandler = oldRspHandler
	}()

	s.startServer(&testFastHTTPService{})

	s.Run("http", func() { s.testCustomResponseHandlerResponseWriteError() })
	s.Run("fasthttp", func() { s.testFastHTTPCustomResponseHandlerResponseWriteError() })
}
func (s *TestSuite) testFastHTTPCustomResponseHandlerResponseWriteError() {
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	bts := mustMarshalJSON(s.T(), &req)

	fasthttpReq := fasthttp.AcquireRequest()
	fasthttpRsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fasthttpReq)
	defer fasthttp.ReleaseResponse(fasthttpRsp)

	fasthttpReq.Header.SetMethod("post")
	fasthttpReq.SetRequestURI(s.unaryCallCustomURL())
	fasthttpReq.Header.SetContentType("application/json")
	fasthttpReq.SetBody(bts)
	err := fasthttp.Do(fasthttpReq, fasthttpRsp)
	require.Nil(s.T(), err)

	bts = fasthttpRsp.Body()
	require.Nil(s.T(), err)

	ce := customResponse{}
	require.NotNil(s.T(), json.Unmarshal(bts, &ce),
		`ERROR log will occur with message like: "encode fail:http write response error"`)
}

func (s *TestSuite) TestStatusBadRequestDueToFastHTTPServerDecodeFail() {
	s.startServer(&testFastHTTPService{})

	s.Run("http", func() { s.testStatusBadRequestDueToServerDecodeFail() })
	s.Run("fasthttp", func() { s.testFastHTTPStatusBadRequestDueToServerDecodeFail() })
}
func (s *TestSuite) testFastHTTPStatusBadRequestDueToServerDecodeFail() {
	bts, err := json.Marshal(s.defaultSimpleRequest)
	require.Nil(s.T(), err)

	fasthttpReq := fasthttp.AcquireRequest()
	fasthttpRsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fasthttpReq)
	defer fasthttp.ReleaseResponse(fasthttpRsp)

	fasthttpReq.Header.SetMethod("post")
	fasthttpReq.SetRequestURI(s.unaryCallCustomURL())
	fasthttpReq.Header.SetContentType("application/pb")
	fasthttpReq.SetBody(bts)
	err = fasthttp.Do(fasthttpReq, fasthttpRsp)
	require.Nil(s.T(), err)
	require.Equal(s.T(), fasthttp.StatusBadRequest, fasthttpRsp.StatusCode())

	fasthttpRsp.Reset()
	fc := thttp.NewFastHTTPClient("fasthttp-client")
	err = fc.Do(fasthttpReq, fasthttpRsp)
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), fasthttp.StatusBadRequest, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Empty(s.T(), fasthttpRsp.Body())
}

func (s *TestSuite) TestFastHTTP() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(&testFastHTTPService{
			TRPCService: TRPCService{
				UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
					req := &thttp.RequestCtx(ctx).Request
					rsp := &thttp.RequestCtx(ctx).Response

					if strings.Contains(string(req.Header.ContentType()), "server-unsupported-content-type") {
						return nil, errs.New(fasthttp.StatusUnsupportedMediaType, "Unsupported Media Type")
					}
					if strings.Contains(string(req.Header.ContentType()), "client-unsupported-content-type") {
						rsp.Header.Add("Serialization-Type", fmt.Sprint(codec.SerializationTypeUnsupported))
					}

					payload, err := newPayload(in.GetResponseType(), in.GetResponseSize())
					if err != nil {
						return nil, err
					}
					return &testpb.SimpleResponse{Payload: payload}, nil
				},
			},
		})
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(s.httpServerEnv.String(), s.testHTTP)
		s.Run(s.httpServerEnv.String(), s.testFastHTTP)
	}
}
func (s *TestSuite) testFastHTTP() {
	thttp.RegisterStatus(fasthttp.StatusUnsupportedMediaType, fasthttp.StatusUnsupportedMediaType)

	s.Run("AccessNonexistentResource", s.testFastHTTPAccessNonexistentResource)
	s.Run("SendSupportedContentType", s.testFastHTTPSendSupportedContentType)
	s.Run("ServerReceivedUnsupportedContentType", s.testFastHTTPServerReceivedUnsupportedContentType)
	s.Run("ClientReceivedUnsupportedContentType", s.testFastHTTPClientReceivedUnsupportedContentType)
	s.Run("EmptyBody", s.testFastHTTPEmptyBody)
	s.Run("PatchMethod", s.testFastHTTPPatchMethod)
}
func (s *TestSuite) testFastHTTPAccessNonexistentResource() {
	methods := []string{
		fasthttp.MethodGet,
		fasthttp.MethodPost,
		fasthttp.MethodHead,
		fasthttp.MethodOptions,
	}
	incorrectURLs := []string{
		s.unaryCallDefaultURL() + "/incorrect",
		s.unaryCallCustomURL() + "/incorrect",
	}

	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	for _, m := range methods {
		for _, url := range incorrectURLs {
			doFastHTTPRequest := func() {
				req.Reset()
				rsp.Reset()
				req.SetRequestURI(url)
				req.Header.SetMethod(m)
				err := fasthttp.Do(req, rsp)
				require.Nil(s.T(), err)
				require.Equal(s.T(), fasthttp.StatusNotFound, rsp.StatusCode())
			}
			doFastHTTPRequest()

			dotFastHTTPRequest := func() {
				req.Reset()
				rsp.Reset()
				req.SetRequestURI(url)
				req.Header.SetMethod(m)
				err := thttp.NewFastHTTPClient("fasthttp-client").Do(req, rsp)
				require.Equal(s.T(), fasthttp.StatusNotFound, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
				require.Empty(s.T(), rsp.Body())
			}
			dotFastHTTPRequest()
		}
	}
}
func (s *TestSuite) testFastHTTPSendSupportedContentType() {
	contentTypeSerializationType := map[string]int{
		"application/json":       codec.SerializationTypeJSON,
		"application/protobuf":   codec.SerializationTypePB,
		"application/x-protobuf": codec.SerializationTypePB,
		"application/pb":         codec.SerializationTypePB,
		"application/proto":      codec.SerializationTypePB,
	}

	urls := []string{
		s.unaryCallDefaultURL(),
		s.unaryCallCustomURL(),
	}

	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	for _, url := range urls {
		for contentType, serializationType := range contentTypeSerializationType {
			serializer := codec.GetSerializer(serializationType)
			bts, err := serializer.Marshal(s.defaultSimpleRequest)
			require.Nil(s.T(), err)

			doFastHTTPPost := func() {
				req.Reset()
				rsp.Reset()
				req.SetRequestURI(url)
				req.Header.SetContentType(contentType)
				req.SetBody(bts)
				req.Header.SetMethod(fasthttp.MethodPost)
				err := fasthttp.Do(req, rsp)
				require.Nil(s.T(), err)
				require.Equal(s.T(), fasthttp.StatusOK, rsp.StatusCode())
			}
			doFastHTTPPost()

			dotFastHTTPPost := func() {
				req.Reset()
				rsp.Reset()
				req.SetRequestURI(url)
				req.Header.SetContentType(contentType)
				req.SetBody(bts)
				req.Header.SetMethod(fasthttp.MethodPost)
				err := thttp.NewFastHTTPClient("fasthttp-client").Do(req, rsp)
				require.Nil(s.T(), err)
				require.Equal(s.T(), fasthttp.StatusOK, rsp.StatusCode())
			}
			dotFastHTTPPost()
		}
	}
}
func (s *TestSuite) testFastHTTPServerReceivedUnsupportedContentType() {
	contentTypes := []string{
		"server-unsupported-content-type-1",
		"server-unsupported-content-type-2",
	}
	urls := []string{
		s.unaryCallDefaultURL(),
		s.unaryCallCustomURL(),
	}

	bts := []byte(s.defaultSimpleRequest.String())
	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	for _, url := range urls {
		for _, contentType := range contentTypes {

			doFastHTTPPost := func() {
				req.Reset()
				rsp.Reset()
				req.SetRequestURI(url)
				req.Header.SetContentType(contentType)
				req.SetBody(bts)
				req.Header.SetMethod(fasthttp.MethodPost)
				err := fasthttp.Do(req, rsp)
				require.Nil(s.T(), err)
				require.Equal(s.T(), fasthttp.StatusUnsupportedMediaType, rsp.StatusCode())
			}
			doFastHTTPPost()

			dotFastHTTPPost := func() {
				req.Reset()
				rsp.Reset()
				req.SetRequestURI(url)
				req.Header.SetContentType(contentType)
				req.SetBody(bts)
				req.Header.SetMethod(fasthttp.MethodPost)
				err := thttp.NewFastHTTPClient("fasthttp-client").Do(req, rsp)
				require.Nil(s.T(), err)
				require.Equal(s.T(), fasthttp.StatusUnsupportedMediaType, rsp.StatusCode())
			}
			dotFastHTTPPost()
		}
	}
}
func (s *TestSuite) testFastHTTPClientReceivedUnsupportedContentType() {
	contentTypes := []string{
		"client-unsupported-content-type-1",
		"client-unsupported-content-type-2",
	}
	urls := []string{
		s.unaryCallDefaultURL(),
		s.unaryCallCustomURL(),
	}

	bts := []byte(s.defaultSimpleRequest.String())
	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	for _, url := range urls {
		for _, contentType := range contentTypes {
			doFastHTTPPost := func() {
				req.Reset()
				rsp.Reset()
				req.SetRequestURI(url)
				req.Header.SetContentType(contentType)
				req.SetBody(bts)
				req.Header.SetMethod(fasthttp.MethodPost)
				err := fasthttp.Do(req, rsp)
				require.Nil(s.T(), err)

				serializationType, err := strconv.ParseInt(string(rsp.Header.Peek("Serialization-Type")), 10, 32)
				require.Nil(s.T(), err)
				require.Nil(s.T(), codec.GetSerializer(int(serializationType)))
			}
			doFastHTTPPost()

			dotFastHTTPPost := func() {
				req.Reset()
				rsp.Reset()
				req.SetRequestURI(url)
				req.Header.SetContentType(contentType)
				req.SetBody(bts)
				req.Header.SetMethod(fasthttp.MethodPost)
				err := thttp.NewFastHTTPClient("fasthttp-client").Do(req, rsp)
				require.Nil(s.T(), err)

				serializationType, err := strconv.ParseInt(string(rsp.Header.Peek("Serialization-Type")), 10, 32)
				require.Nil(s.T(), err)
				require.Nil(s.T(), codec.GetSerializer(int(serializationType)))
			}
			dotFastHTTPPost()
		}
	}
}
func (s *TestSuite) testFastHTTPEmptyBody() {
	urls := []string{
		s.unaryCallDefaultURL(),
		s.unaryCallCustomURL(),
	}
	const contentType = "text"
	bts := []byte{}
	req := fasthttp.AcquireRequest()
	rsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rsp)

	for _, url := range urls {
		doFastHTTPPost := func() {
			req.Reset()
			rsp.Reset()
			req.SetRequestURI(url)
			req.Header.SetContentType(contentType)
			req.SetBody(bts)
			req.Header.SetMethod(fasthttp.MethodPost)
			err := fasthttp.Do(req, rsp)
			require.Nil(s.T(), err)
			require.Equal(s.T(), fasthttp.StatusOK, rsp.StatusCode())
			require.Contains(s.T(), string(rsp.Body()), `"body":""`)
		}
		doFastHTTPPost()

		dotFastHTTPPost := func() {
			req.Reset()
			rsp.Reset()
			req.SetRequestURI(url)
			req.Header.SetContentType(contentType)
			req.SetBody(bts)
			req.Header.SetMethod(fasthttp.MethodPost)
			err := thttp.NewFastHTTPClient("fasthttp-client").Do(req, rsp)
			require.Nil(s.T(), err)
			require.Equal(s.T(), fasthttp.StatusOK, rsp.StatusCode())
			require.Contains(s.T(), string(rsp.Body()), `"body":""`)
		}
		dotFastHTTPPost()
	}
}
func (s *TestSuite) testFastHTTPPatchMethod() {
	fcp := thttp.NewFastHTTPClientProxy(s.listener.Addr().String())
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	rsp := &testpb.SimpleResponse{}

	require.Nil(s.T(), fcp.Patch(trpc.BackgroundContext(), "/UnaryCall", req, rsp))
	require.Len(s.T(), rsp.Payload.GetBody(), int(req.ResponseSize))
}

func (s *TestSuite) TestFastHTTPSInsecureSkipVerify() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(
			&testFastHTTPService{},
			server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", ""),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(s.httpServerEnv.String(), s.testHTTPSInsecureSkipVerify)
		s.Run(s.httpServerEnv.String(), s.testFastHTTPSInsecureSkipVerify)
	}
}
func (s *TestSuite) testFastHTTPSInsecureSkipVerify() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
	)
	s.Run("tFastHTTPRequestOk", func() {
		c1 := thttp.NewFastHTTPClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, "none", ""),
		)
		rsp := &testpb.SimpleResponse{}
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		require.Nil(s.T(), c1.Post(trpc.BackgroundContext(), "/UnaryCall", req, rsp))
	})
	s.Run("FastHTTPRPCRequestOk", func() {
		c2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, "none", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := c2.UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.Nil(s.T(), err)
	})
	s.Run("originFastHTTPRequestOk", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)

		c3 := fasthttp.Client{
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cert},
			},
		}
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)

		req.SetRequestURI(s.unaryHTTPSCallCustomURL())
		req.Header.SetContentType("application/json")
		req.Header.SetMethod(fasthttp.MethodPost)
		req.SetBody(bts)
		err = c3.Do(req, rsp)
		require.Nil(s.T(), err)
	})
}

func (s *TestSuite) TestFastHTTPSProtocolMisMatch() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(
			&testFastHTTPService{},
			server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", ""),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(s.httpServerEnv.String(), func() { s.testHTTPSProtocolMisMatch(true) })
		s.Run(s.httpServerEnv.String(), func() { s.testFastHTTPSProtocolMisMatch(true) })
	}
}
func (s *TestSuite) testFastHTTPSProtocolMisMatch(fastHTTPServer bool) {
	s.Run("tfasthttpRequestFailed", func() {
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		fc := thttp.NewFastHTTPClient(
			"fasthttp-client",
			client.WithProtocol(protocol.FastHTTP),
		)

		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)
		req.SetRequestURI(s.unaryCallCustomURL())
		req.Header.SetContentType("application/json")
		req.SetBody(bts)
		err = fc.Do(req, rsp)
		if fastHTTPServer {
			require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
			require.Contains(s.T(), err.Error(),
				"the server closed connection before returning the first response byte.")
		} else {
			require.Nil(s.T(), err)
			require.Equal(s.T(), fasthttp.StatusBadRequest, rsp.StatusCode())
			require.Equal(s.T(), []byte("Client sent an HTTP request to an HTTPS server.\n"), rsp.Body())
		}
	})
	s.Run("fasthttpRPCRequestFailed", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTarget(s.serverAddress()),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		if fastHTTPServer {
			require.Nil(s.T(), rsp)
			require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
			require.Contains(s.T(), err.Error(),
				"the server closed connection before returning the first response byte.")
		} else {
			require.Nil(s.T(), rsp)
			require.NotNil(s.T(), err)
			require.Contains(s.T(), err.Error(),
				"Client sent an HTTP request to an HTTPS server.")
		}
	})
	s.Run("originFastHTTPRequestFailed", func() {
		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)
		req.SetRequestURI(s.unaryCallCustomURL())
		err := fasthttp.Do(req, rsp)

		if fastHTTPServer {
			require.NotNil(s.T(), err)
			require.Contains(s.T(), err.Error(),
				"the server closed connection before returning the first response byte.")
			require.Equal(s.T(), http.StatusOK, rsp.StatusCode())
		} else {
			require.Nil(s.T(), err)
			require.Equal(s.T(), fasthttp.StatusBadRequest, rsp.StatusCode())
			require.Equal(s.T(), []byte("Client sent an HTTP request to an HTTPS server.\n"), rsp.Body())
		}
	})
}

func (s *TestSuite) TestFastHTTPSOneWayAuthentication() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(
			&testFastHTTPService{},
			server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", ""),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(s.httpServerEnv.String(), s.testHTTPSOneWayAuthentication)
		s.Run(s.httpServerEnv.String(), s.testFastHTTPSOneWayAuthentication)
	}
}
func (s *TestSuite) testFastHTTPSOneWayAuthentication() {
	s.Run("Ok", s.testFastHTTPSOneWayOk)
	s.Run("ClientWithoutCertification", s.testFastHTTPSOneWayClientWithoutCA)
	s.Run("CertificationIsUnmatched", s.testFastHTTPSOneWayCAIsUnmatched)
	s.Run("InvalidClientTLSCert", s.testFastHTTPSOneWayInvalidClientTLSCert)
}
func (s *TestSuite) testFastHTTPSOneWayOk() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
		serverTLSCA   = "x509/server_ca_cert.pem"
		serverName    = "trpc.test.example.com"
	)

	s.Run("fastHTTPClientProxyRequestOk", func() {
		fcp1 := thttp.NewFastHTTPClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp := &testpb.SimpleResponse{}
		require.Nil(s.T(), fcp1.Post(trpc.BackgroundContext(), "/UnaryCall", req, rsp))
	})

	s.Run("fastHTTPRPCRequestOk", func() {
		fcp2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp, err := fcp2.UnaryCall(trpc.BackgroundContext(), req)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), rsp)
	})

	s.Run("fastHTTPClientRequestOk", func() {
		fc := thttp.NewFastHTTPClient("fasthttp-client",
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		)

		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)

		req.SetRequestURI(s.unaryHTTPSCallCustomURL())
		req.Header.SetContentType("application/json")
		req.Header.SetMethod(fasthttp.MethodPost)
		req.SetBody(bts)
		err = fc.Do(req, rsp)
		require.Nil(s.T(), err)
	})

	s.Run("originFastHTTPRequestOK", func() {
		cert, err := tls.LoadX509KeyPair("x509/client1_cert.pem", "x509/client1_key.pem")
		require.Nil(s.T(), err)

		b, err := os.ReadFile(serverTLSCA)
		require.Nil(s.T(), err)
		roots := x509.NewCertPool()
		require.True(s.T(), roots.AppendCertsFromPEM(b))

		ofc := fasthttp.Client{TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{cert},
			RootCAs:            roots,
			ServerName:         serverName,
		}}

		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)
		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)

		req.SetRequestURI(s.unaryHTTPSCallCustomURL())
		req.Header.SetContentType("application/json")
		req.Header.SetMethod(fasthttp.MethodPost)
		req.SetBody(bts)
		err = ofc.Do(req, rsp)
		require.Nil(s.T(), err)
	})
}
func (s *TestSuite) testFastHTTPSOneWayClientWithoutCA() {
	s.Run("tfasthttpRequestOK", func() {
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		fc := thttp.NewFastHTTPClient(
			"fasthttp-client",
			client.WithProtocol(protocol.FastHTTP),
			client.WithTransport(thttp.NewFastHTTPClientTransport()),
			client.WithTLS("", "", "none", ""),
		)

		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)
		req.SetRequestURI(s.unaryHTTPSCallCustomURL())
		req.Header.SetContentType("application/json")
		req.SetBody(bts)
		err = fc.Do(req, rsp)

		require.Nil(s.T(), err)
		require.NotNil(s.T(), rsp)
	})

	s.Run("fasthttpRPCRequestOK", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTransport(thttp.NewFastHTTPClientTransport()),
			client.WithTLS("", "", "none", ""),
			client.WithTarget(s.serverAddress()),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))

		require.Nil(s.T(), err)
		require.NotNil(s.T(), rsp)
	})

	s.Run("originFastHTTPRequestFailed", func() {
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		ofc := fasthttp.Client{}
		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)
		req.SetRequestURI(s.unaryHTTPSCallCustomURL())
		req.Header.SetContentType("application/json")
		req.SetBody(bts)
		err = ofc.Do(req, rsp)

		require.NotNil(s.T(), err)
		require.Contains(s.T(), err.Error(), "x509")
		require.Equal(s.T(), fasthttp.StatusOK, rsp.StatusCode())
	})
}
func (s *TestSuite) testFastHTTPSOneWayCAIsUnmatched() {
	const (
		unmatchedClientTLSCert = "x509/client2_cert.pem"
		unmatchedClientTLSKey  = "x509/client1_key.pem"
		expectedErrorMsg       = "private key does not match public key"
	)

	s.Run("tFastHTTPRequestFailed", func() {
		c1 := thttp.NewFastHTTPClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(unmatchedClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp := &testpb.SimpleResponse{}
		err := c1.Post(trpc.BackgroundContext(), "/UnaryCall", req, rsp)
		require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})

	s.Run("FastHTTPRPCRequestFailed", func() {
		c2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(unmatchedClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := c2.UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.NotNil(s.T(), err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
}
func (s *TestSuite) testFastHTTPSOneWayInvalidClientTLSCert() {
	const (
		invalidClientTLSCert  = "invalid file path"
		unmatchedClientTLSKey = "x509/client1_key.pem"
	)

	s.Run("tFastHTTPRequestFailed", func() {
		c1 := thttp.NewFastHTTPClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(invalidClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp := &testpb.SimpleResponse{}
		err := c1.Post(trpc.BackgroundContext(), "/UnaryCall", req, rsp)
		require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "client load cert file error")
		require.Contains(s.T(), errs.Msg(err), "open invalid file path: no such file or directory")
	})

	s.Run("FastHTTPRPCRequestFailed", func() {
		c2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(invalidClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := c2.UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "full err: %+v", err)
		s.T().Log(errs.Msg(err))
		require.Contains(s.T(), errs.Msg(err), "client load cert file error")
		require.Contains(s.T(), errs.Msg(err), "open invalid file path: no such file or directory")
	})
}

func (s *TestSuite) TestFastHTTPSTwoWayAuthentication() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(
			&testFastHTTPService{},
			server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", "x509/client_ca_cert.pem"),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(s.httpServerEnv.String(), s.testHTTPSTwoWayAuthentication)
		s.Run(s.httpServerEnv.String(), s.testFastHTTPSTwoWayAuthentication)
	}
}
func (s *TestSuite) testFastHTTPSTwoWayAuthentication() {
	s.Run("Ok", s.testFastHTTPSTwoWayOk)
	s.Run("CAIsUnmatched", s.testFastHTTPSTwoWayCAIsUnmatched)
	s.Run("ClientWithoutCA", s.testFastHTTPSTwoWayClientWithoutCA)
}
func (s *TestSuite) testFastHTTPSTwoWayOk() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
		serverTLSCA   = "x509/server_ca_cert.pem"
		serverName    = "trpc.test.example.com"
	)

	s.Run("tFastHTTPRequestOk", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		require.Nil(s.T(), thttp.NewFastHTTPClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", req, &testpb.SimpleResponse{}))
	})

	s.Run("fastHTTPRPCRequestOk", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.Nil(s.T(), err)
	})

	s.Run("originFastHTTPRequestOk", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)

		b, err := os.ReadFile(serverTLSCA)
		require.Nil(s.T(), err)

		roots := x509.NewCertPool()
		require.True(s.T(), roots.AppendCertsFromPEM(b))

		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		ofc := fasthttp.Client{
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				ServerName:   serverName,
				RootCAs:      roots,
			},
		}

		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)

		req.SetRequestURI(s.unaryHTTPSCallCustomURL())
		req.Header.SetContentType("application/json")
		req.Header.SetMethod(fasthttp.MethodPost)
		req.SetBody(bts)
		err = ofc.Do(req, rsp)

		require.Nil(s.T(), err)
	})
}
func (s *TestSuite) testFastHTTPSTwoWayCAIsUnmatched() {
	const (
		clientTLSCert    = "x509/client1_cert.pem"
		clientTLSKey     = "x509/client1_key.pem"
		serverTLSCA2     = "x509/server2_ca_cert.pem"
		serverName       = "trpc.test.example.com"
		expectedErrorMsg = "certificate signed by unknown authority"
	)

	s.Run("tFastHTTPRequestFailed", func() {
		transport.RegisterClientTransport(protocol.FastHTTP, thttp.NewFastHTTPClientTransport())
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		err := thttp.NewFastHTTPClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA2, serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", req, &testpb.SimpleResponse{})
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})

	s.Run("FastHTTPRPCRequestFailed", func() {
		transport.RegisterClientTransport(protocol.FastHTTP, thttp.NewFastHTTPClientTransport())
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA2, serverName),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})

	s.Run("originFastHTTPRequestFailed", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)

		b, err := os.ReadFile(serverTLSCA2)
		require.Nil(s.T(), err)

		roots := x509.NewCertPool()
		require.True(s.T(), roots.AppendCertsFromPEM(b))

		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		ofc := fasthttp.Client{
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				ServerName:   serverName,
				RootCAs:      roots,
			},
		}
		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)

		req.SetRequestURI(s.unaryHTTPSCallCustomURL())
		req.Header.SetContentType("application/json")
		req.Header.SetMethod(fasthttp.MethodPost)
		req.SetBody(bts)
		err = ofc.Do(req, rsp)

		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
}
func (s *TestSuite) testFastHTTPSTwoWayClientWithoutCA() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
		serverName    = "trpc.test.example.com"
	)

	s.Run("tFastHTTPRequestFailed", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		err := thttp.NewFastHTTPClientProxy(
			s.listener.Addr().String(),
			client.WithTLS("", clientTLSKey, "none", serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", req, &testpb.SimpleResponse{})
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "client didn't provide a certFile")
	})
	s.Run("fastHTTPRPCRequestFailed", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.FastHTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, "", serverName),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.NotNil(s.T(), err)
	})
	s.Run("originFastHTTPRequestFailed", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		ofc := fasthttp.Client{
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				ServerName:   serverName,
			},
		}
		req := fasthttp.AcquireRequest()
		rsp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(rsp)

		req.SetRequestURI(s.unaryHTTPSCallCustomURL())
		req.Header.SetContentType("application/json")
		req.Header.SetMethod(fasthttp.MethodPost)
		req.SetBody(bts)
		err = ofc.Do(req, rsp)

		require.NotNil(s.T(), err, "certificate signed by unknown authority")
	})
}

func (s *TestSuite) TestFastHTTPPassthroughForClientInvocation() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testFastHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testPassthroughForClientInvocation() })
		s.Run(e.String(), func() { s.testFastHTTPPassthroughForClientInvocation() })
	}
}
func (s *TestSuite) testFastHTTPPassthroughForClientInvocation() {
	c := thttp.NewFastHTTPClient("fasthttp-client", client.WithTarget(s.serverAddress()+"10086"))
	code, b, err := c.Get(nil, s.unaryCallCustomURL())
	require.Nil(s.T(), err)
	require.NotNil(s.T(), b)
	require.Equal(s.T(), http.StatusOK, code)
}

func (s *TestSuite) TestFastHTTPSRaw() {
	go fasthttp.ListenAndServeTLS("127.0.0.1:8080", "x509/server1_cert.pem", "x509/server1_key.pem", func(ctx *fasthttp.RequestCtx) {})
	time.Sleep(time.Second)

	fasthttpReq := fasthttp.AcquireRequest()
	fasthttpRsp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fasthttpReq)
	defer fasthttp.ReleaseResponse(fasthttpRsp)
	fasthttpReq.SetRequestURI("http://127.0.0.1:8080")
	err := fasthttp.Do(fasthttpReq, fasthttpRsp)
	require.Contains(s.T(), err.Error(), "the server closed connection before returning the first response byte.")
	require.Equal(s.T(), fasthttp.StatusOK, fasthttpRsp.StatusCode())

	rsp, err := http.Get("http://127.0.0.1:8080")
	_, ok := err.(*url.Error)
	require.True(s.T(), ok)
	require.Nil(s.T(), rsp)
}

// TestFastHTTPWithoutPreConfiguredListener verifies that FastHTTP server works correctly
// when started without a pre-configured listener. This test specifically covers the case
// where an internal graceful.Listener is created, which is different from the default
// *net.Listener used in other tests.
func (s *TestSuite) TestFastHTTPWithoutPreConfiguredListener() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testFastHTTPWithoutPreConfiguredListener(e) })
	}
}

func (s *TestSuite) testFastHTTPWithoutPreConfiguredListener(e *httpRPCEnv) {
	// Get available server address.
	ln, err := net.Listen("tcp", defaultServerAddress)
	require.Nil(s.T(), err)
	actualAddr := ln.Addr().String()
	ln.Close()

	// Initialize server without listener.
	svr := &server.Server{}
	svr.AddService(
		fasthttpServiceName,
		server.New(
			server.WithServiceName(fasthttpServiceName),
			server.WithProtocol(protocol.FastHTTP),
			server.WithAddress(actualAddr), // Only specify the address, do not pass in the listener.
			server.WithServerAsync(e.server.async)),
	)
	testpb.RegisterTestHTTPService(svr.Service(fasthttpServiceName), &testFastHTTPService{})
	s.server = svr
	go svr.Serve()

	// Wait for server to start.
	time.Sleep(100 * time.Millisecond)

	// Prepare client request.
	rspHead := &thttp.FastHTTPClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.FastHTTPClientReqHeader{Method: fasthttp.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}

	// Send request and verify response.
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	proxy := testpb.NewTestHTTPClientProxy(append([]client.Option{
		client.WithProtocol(protocol.FastHTTP),
		client.WithTarget(fmt.Sprintf("%s://%v", "ip", actualAddr)),
		client.WithTimeout(time.Second)}, opts...)...)
	_, err = proxy.UnaryCall(trpc.BackgroundContext(), req)
	require.Nil(s.T(), err)
	require.Equal(s.T(), fasthttp.StatusOK, rspHead.Response.StatusCode())
}
