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
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/server"

	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestCustomErrorHandler() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		oldErrHandler := thttp.DefaultServerCodec.ErrHandler
		thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
			w.Header().Set("Custom-Error", fmt.Sprintf(`{"ret-code": %d, "ret-msg": "%s"}`, e.Code, e.Msg))
		}
		defer func() {
			thttp.DefaultServerCodec.ErrHandler = oldErrHandler
		}()
		s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testCustomErrorHandler(e) })
		s.Run(e.String(), func() { s.testFastHTTPCustomErrorHandler(e) })
	}
}
func (s *TestSuite) testCustomErrorHandler(e *httpRPCEnv) {
	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.ResponseType = testpb.PayloadType_RANDOM
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Nil(s.T(), err)
	type customError struct {
		RetCode int    `json:"ret-code"`
		RetMsg  string `json:"ret-msg"`
	}
	ce := &customError{}
	require.Nil(s.T(), json.Unmarshal([]byte(rspHead.Response.Header.Get("Custom-Error")), ce))
	require.Equal(s.T(), retUnsupportedPayload, ce.RetCode)
}

func (s *TestSuite) TestClientReqAndRspHeader() {
	s.startServer(&testHTTPService{})
	s.T().Cleanup(func() { s.closeServer(nil) })

	s.Run("http", func() { s.testClientReqAndRspHeader() })
	s.Run("fasthttp", func() { s.testFastHTTPClientReqAndRspHeader() })
}
func (s *TestSuite) testClientReqAndRspHeader() {
	s.T().Run("ReqHead is not *http.ClientReqHeader", func(t *testing.T) {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := s.newHTTPRPCClient(client.WithReqHead("string type")).UnaryCall(context.Background(), req)
		require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "http header must be type of *http.ClientReqHeader")

		_, err = s.newHTTPRPCClient(client.WithReqHead(nil)).UnaryCall(context.Background(), req)
		require.Nil(t, err)
	})
	s.T().Run("RspHead is not *http.ClientRspHeader", func(t *testing.T) {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := s.newHTTPRPCClient(client.WithRspHead("string type")).UnaryCall(context.Background(), req)
		require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "http header must be type of *http.ClientRspHeader")

		_, err = s.newHTTPRPCClient(client.WithRspHead(nil)).UnaryCall(context.Background(), req)
		require.Nil(t, err)
	})
}

func (s *TestSuite) TestDefaultErrorHandler() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testDefaultErrorHandler(e) })
		s.Run(e.String(), func() { s.testFastHTTPDefaultErrorHandler(e) })
	}
}
func (s *TestSuite) testDefaultErrorHandler(e *httpRPCEnv) {
	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}

	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.ResponseType = testpb.PayloadType_RANDOM
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), retUnsupportedPayload, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "unsupported payload type")
	require.Equal(s.T(), fmt.Sprint(errs.Code(err)), rspHead.Response.Header.Get(thttp.TrpcUserFuncErrorCode))
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err), "full err: %+v", err)
	require.Equal(
		s.T(),
		http.StatusOK,
		rspHead.Response.StatusCode,
		"any framework error code not in thttp.ErrsToHTTPStatus map are converted to http.StatusOK",
	)
}

func (s *TestSuite) TestHandleErrServerNoResponse() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testHTTPService{TRPCService: TRPCService{UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
			return nil, errs.ErrServerNoResponse
		}}}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testHandleErrServerNoResponse() })
		s.Run(e.String(), func() { s.testFastHTTPHandleErrServerNoResponse() })
	}
}
func (s *TestSuite) testHandleErrServerNoResponse() {
	bts, err := proto.Marshal(s.defaultSimpleRequest)
	require.Nil(s.T(), err)

	c := thttp.NewStdHTTPClient("http-client")
	rsp, err := c.Post(s.unaryCallCustomURL(), "application/pb", bytes.NewReader(bts))
	require.Nil(s.T(), err)
	require.Equal(s.T(), http.StatusInternalServerError, rsp.StatusCode)

	bts, err = io.ReadAll(rsp.Body)
	require.Nil(s.T(), err)
	require.Containsf(s.T(), string(bts), "http server handle error: type:framework, code:0, msg:server no response", "full err: %+v", err)
}

func (s *TestSuite) TestSendHTTPSRequestToHTTPServer() {
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
func (s *TestSuite) testSendHTTPSRequestToHTTPServer(e *httpRPCEnv) {
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(&thttp.ClientRspHeader{}),
		client.WithProtocol(protocol.HTTPS),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	rsp, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)
	require.Nil(s.T(), rsp)
	s.T().Log(rsp, err)
	require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
}

func (s *TestSuite) TestStatusBadRequestDueToServerValidateFail() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusBadRequestDueToServerValidateFail(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusBadRequestDueToServerValidateFail(e) })
	}
}
func (s *TestSuite) testStatusBadRequestDueToServerValidateFail(e *httpRPCEnv) {
	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.Username = "non-validate-name-?.@&*-_"
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerValidateFail, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusBadRequest, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), fmt.Sprint(errs.Code(err)), rspHead.Response.Header.Get(thttp.TrpcUserFuncErrorCode))
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusBadRequest, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusNotFoundDueToServerNoService() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		startServerWithoutAnyService := func(t *testing.T) {
			t.Helper()
			trpc.ServerConfigPath = "trpc_go_http_server.yaml"

			l, err := net.Listen("tcp", defaultServerAddress)
			if err != nil {
				t.Fatalf("net.Listen(%s) error", defaultServerAddress)
			}
			s.listener = l

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
func (s *TestSuite) testStatusNotFoundDueToServerNoService(e *httpRPCEnv) {
	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerNoFunc, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusNotFound, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusNotFound, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusNotFoundDueToServerNoFunc() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusNotFoundDueToServerNoFunc(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusNotFoundDueToServerNoFunc(e) })
	}
}
func (s *TestSuite) testStatusNotFoundDueToServerNoFunc(e *httpRPCEnv) {
	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(rspHead),
		client.WithTarget(s.serverAddress() + "/NonexistentCall"),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerNoFunc, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusNotFound, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusNotFound, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusGatewayTimeoutDueToServerTimeout() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(
			&testHTTPService{},
			server.WithServerAsync(e.server.async),
			server.WithTimeout(50*time.Millisecond),
			server.WithFilter(
				func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
					return nil, errs.NewFrameError(errs.RetServerTimeout, "")
				}),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusGatewayTimeoutDueToServerTimeout(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusGatewayTimeoutDueToServerTimeout(e) })
	}
}
func (s *TestSuite) testStatusGatewayTimeoutDueToServerTimeout(e *httpRPCEnv) {
	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerTimeout, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusGatewayTimeout, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusGatewayTimeout, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusTooManyRequestsDueToServerOverload() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		const maxRequestQueueSize = 10
		const limitedAccessUser = "LimitedAccessUser"
		requestQueue := make(chan interface{}, maxRequestQueueSize)
		defer func() {
			close(requestQueue)
		}()

		s.startServer(
			&testHTTPService{},
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
func (s *TestSuite) testStatusTooManyRequestsDueToServerOverload(e *httpRPCEnv) {
	const maxRequestQueueSize = 10
	const limitedAccessUser = "LimitedAccessUser"

	sendRequest := func() (*thttp.ClientRspHeader, error) {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		req.Username = limitedAccessUser
		rspHead := &thttp.ClientRspHeader{}
		opts := []client.Option{
			client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
			client.WithRspHead(rspHead),
		}
		if e.client.disableConnectionPool {
			opts = append(opts, client.WithDisableConnectionPool())
		}
		_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)
		return rspHead, err
	}

	var g errgroup.Group
	for i := 0; i < maxRequestQueueSize; i++ {
		g.Go(func() error {
			_, err := sendRequest()
			return err
		})
	}
	require.Zero(s.T(), g.Wait())

	rspHead, err := sendRequest()

	require.Equal(s.T(), errs.RetServerOverload, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusTooManyRequests, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusTooManyRequests, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusUnauthorizedDueToServerAuthFail() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusUnauthorizedDueToServerAuthFail(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusUnauthorizedDueToServerAuthFail(e) })
	}
}
func (s *TestSuite) testStatusUnauthorizedDueToServerAuthFail(e *httpRPCEnv) {
	var rspHead = &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.Username = "invalidUsername"
	req.FillUsername = true
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerAuthFail, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusUnauthorized, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusUnauthorized, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusInternalServerDueToServerReturnUnknown() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(
			&testHTTPService{},
			server.WithServerAsync(e.server.async),
			server.WithFilter(
				func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
					return nil, fmt.Errorf("unknown")
				}),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testStatusInternalServerDueToServerReturnUnknown(e) })
		s.Run(e.String(), func() { s.testFastHTTPStatusInternalServerDueToServerReturnUnknown(e) })
	}
}
func (s *TestSuite) testStatusInternalServerDueToServerReturnUnknown(e *httpRPCEnv) {
	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: http.MethodPost}),
		client.WithRspHead(rspHead),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetUnknown, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusInternalServerError, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Equal(s.T(), fmt.Sprint(errs.Code(err)), rspHead.Response.Header.Get(thttp.TrpcUserFuncErrorCode))
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusInternalServerError, rspHead.Response.StatusCode)
}

type customResponse struct {
	PayloadType int    `json:"payload-type"`
	PayloadBody []byte `json:"payload-body"`
	Username    string `json:"username"`
}

func (s *TestSuite) TestCustomResponseHandler() {
	oldRspHandler := thttp.DefaultServerCodec.RspHandler
	thttp.DefaultServerCodec.RspHandler = func(w http.ResponseWriter, r *http.Request, rspBody []byte) error {
		require.NotEqual(s.T(), 0, len(rspBody))

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

		_, err = w.Write(bs)
		require.Nil(s.T(), err)

		return nil
	}
	defer func() {
		thttp.DefaultServerCodec.RspHandler = oldRspHandler
	}()
	s.startServer(&testHTTPService{})

	s.Run("http", func() { s.testCustomResponseHandler() })
	s.Run("fasthttp", func() { s.testFastHTTPCustomResponseHandler() })
}
func (s *TestSuite) testCustomResponseHandler() {
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.FillUsername = true
	req.Username = validUserNameForAuth
	bts, err := json.Marshal(&req)
	require.Nil(s.T(), err)

	rsp, err := http.Post(s.unaryCallCustomURL(), "application/json", bytes.NewReader(bts))

	require.Nil(s.T(), err)
	defer rsp.Body.Close()

	bts, err = io.ReadAll(rsp.Body)
	require.Nil(s.T(), err)

	ce := customResponse{}
	require.Nil(s.T(), json.Unmarshal(bts, &ce))
	require.Equal(s.T(), validUserNameForAuth, ce.Username)
	require.Equal(s.T(), int(req.ResponseType), ce.PayloadType)
}

func (s *TestSuite) TestCustomResponseHandlerResponseWriteError() {
	oldRspHandler := thttp.DefaultServerCodec.RspHandler
	thttp.DefaultServerCodec.RspHandler = func(w http.ResponseWriter, r *http.Request, rspBody []byte) error {
		return oldRspHandler(&testHTTPResponseWriter{ResponseWriter: w}, r, rspBody)
	}
	defer func() {
		thttp.DefaultServerCodec.RspHandler = oldRspHandler
	}()
	s.startServer(&testHTTPService{})

	s.Run("http", func() { s.testCustomResponseHandlerResponseWriteError() })
	s.Run("fasthttp", func() { s.testFastHTTPCustomResponseHandlerResponseWriteError() })
}
func (s *TestSuite) testCustomResponseHandlerResponseWriteError() {
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	rsp, err := http.Post(s.unaryCallCustomURL(), "application/json",
		bytes.NewReader(mustMarshalJSON(s.T(), &req)))
	require.Nil(s.T(), err)
	defer rsp.Body.Close()

	bts, err := io.ReadAll(rsp.Body)
	require.Nil(s.T(), err)

	ce := customResponse{}
	require.NotNil(s.T(), json.Unmarshal(bts, &ce),
		`ERROR log will occur with message like: "encode fail:http write response error"`)
}

type testHTTPResponseWriter struct {
	http.ResponseWriter
}

func (w testHTTPResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("writing failed")
}

func (s *TestSuite) TestStatusBadRequestDueToServerDecodeFail() {
	s.startServer(&testHTTPService{})

	s.Run("http", func() { s.testStatusBadRequestDueToServerDecodeFail() })
	s.Run("fasthttp", func() { s.testFastHTTPStatusBadRequestDueToServerDecodeFail() })
}
func (s *TestSuite) testStatusBadRequestDueToServerDecodeFail() {
	bts, err := json.Marshal(s.defaultSimpleRequest)
	require.Nil(s.T(), err)

	rsp, err := http.Post(s.unaryCallCustomURL(), "application/pb", bytes.NewReader(bts))
	require.Nil(s.T(), err)
	defer rsp.Body.Close()
	require.Equal(s.T(), http.StatusBadRequest, rsp.StatusCode)

	c := thttp.NewStdHTTPClient("http-client")
	rsp, err = c.Post(s.unaryCallCustomURL(), "application/pb", bytes.NewReader(bts))
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err), "full err: %+v", err)
	require.Equal(s.T(), http.StatusBadRequest, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
	require.Nil(s.T(), rsp)
}

func (s *TestSuite) TestHTTP() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(&testHTTPService{
			TRPCService: TRPCService{
				UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
					header := thttp.Head(ctx).Request.Header
					if strings.Contains(header.Get("Content-Type"), "server-unsupported-content-type") {
						return nil, errs.New(http.StatusUnsupportedMediaType, "Unsupported Media Type")
					}
					if strings.Contains(header.Get("Content-Type"), "client-unsupported-content-type") {
						thttp.Response(ctx).Header().Set("Serialization-Type", fmt.Sprint(codec.SerializationTypeUnsupported))
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
func (s *TestSuite) testHTTP() {
	thttp.RegisterStatus(http.StatusUnsupportedMediaType, http.StatusUnsupportedMediaType)
	s.Run("AccessNonexistentResource", s.testHTTPAccessNonexistentResource)
	s.Run("SendSupportedContentType", s.testHTTPSendSupportedContentType)
	s.Run("ServerReceivedUnsupportedContentType", s.testHTTPServerReceivedUnsupportedContentType)
	s.Run("ClientReceivedUnsupportedContentType", s.testHTTPClientReceivedUnsupportedContentType)
	s.Run("EmptyBody", s.testHTTPEmptyBody)
	s.Run("PatchMethod", s.testHTTPPatchMethod)
}
func (s *TestSuite) testHTTPAccessNonexistentResource() {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodHead,
		http.MethodOptions,
	}
	incorrectURLs := []string{
		s.unaryCallDefaultURL() + "/incorrect",
		s.unaryCallCustomURL() + "/incorrect",
	}
	for _, m := range methods {
		for _, url := range incorrectURLs {
			req, err := http.NewRequest(m, url, nil)
			require.Nil(s.T(), err)

			doHTTPRequest := func() {
				rsp, err := http.DefaultClient.Do(req)
				require.Nil(s.T(), err)
				require.Equal(s.T(), http.StatusNotFound, rsp.StatusCode)
			}
			doHTTPRequest()

			doThttpRequest := func() {
				rsp, err := thttp.NewStdHTTPClient("http-client").Do(req)
				require.Equal(s.T(), http.StatusNotFound, thttp.ErrsToHTTPStatus[int32(errs.Code(err))])
				require.Nil(s.T(), rsp)
			}
			doThttpRequest()
		}
	}
}
func (s *TestSuite) testHTTPSendSupportedContentType() {
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
	for _, url := range urls {
		for contentType, serializationType := range contentTypeSerializationType {
			serializer := codec.GetSerializer(serializationType)
			bts, err := serializer.Marshal(s.defaultSimpleRequest)
			require.Nil(s.T(), err)
			body := bytes.NewReader(bts)

			doHTTPPost := func() {
				rsp, err := http.Post(url, contentType, body)
				require.Nil(s.T(), err)
				require.Equal(s.T(), http.StatusOK, rsp.StatusCode)
			}
			doHTTPPost()

			doTHTTPPost := func() {
				rsp, err := thttp.NewStdHTTPClient("http-client").Post(url, contentType, body)
				require.Nil(s.T(), err)
				require.Equal(s.T(), http.StatusOK, rsp.StatusCode)
			}
			doTHTTPPost()
		}
	}
}
func (s *TestSuite) testHTTPServerReceivedUnsupportedContentType() {
	contentTypes := []string{
		"server-unsupported-content-type-1",
		"server-unsupported-content-type-2",
	}
	urls := []string{
		s.unaryCallDefaultURL(),
		s.unaryCallCustomURL(),
	}
	body := bytes.NewReader([]byte(s.defaultSimpleRequest.String()))
	for _, url := range urls {
		for _, contentType := range contentTypes {
			doHTTPPost := func() {
				rsp, err := http.Post(url, contentType, body)
				require.Nil(s.T(), err)
				require.Equal(s.T(), http.StatusUnsupportedMediaType, rsp.StatusCode)
			}
			doHTTPPost()

			doTHTTPPost := func() {
				rsp, err := thttp.NewStdHTTPClient("http-client").Post(url, contentType, body)
				require.Nil(s.T(), err)
				require.Equal(s.T(), http.StatusUnsupportedMediaType, rsp.StatusCode)
			}
			doTHTTPPost()
		}
	}
}
func (s *TestSuite) testHTTPClientReceivedUnsupportedContentType() {
	contentTypes := []string{
		"client-unsupported-content-type-1",
		"client-unsupported-content-type-2",
	}
	urls := []string{
		s.unaryCallDefaultURL(),
		s.unaryCallCustomURL(),
	}
	body := bytes.NewReader([]byte(s.defaultSimpleRequest.String()))
	for _, url := range urls {
		for _, contentType := range contentTypes {
			doHTTPPost := func() {
				rsp, err := http.Post(url, contentType, body)
				require.Nil(s.T(), err)

				serializationType, err := strconv.ParseInt(rsp.Header.Get("Serialization-Type"), 10, 32)
				require.Nil(s.T(), err)
				require.Nil(s.T(), codec.GetSerializer(int(serializationType)))
			}
			doHTTPPost()

			doTHTTPPost := func() {
				rsp, err := thttp.NewStdHTTPClient("http-client").Post(url, contentType, body)
				require.Nil(s.T(), err)

				serializationType, err := strconv.ParseInt(rsp.Header.Get("Serialization-Type"), 10, 32)
				require.Nil(s.T(), err)
				require.Nil(s.T(), codec.GetSerializer(int(serializationType)))
			}
			doTHTTPPost()
		}
	}
}
func (s *TestSuite) testHTTPEmptyBody() {
	urls := []string{
		s.unaryCallDefaultURL(),
		s.unaryCallCustomURL(),
	}
	const contentType = "text"
	body := bytes.NewReader([]byte(""))
	for _, url := range urls {
		doHTTPPost := func() {
			rsp, err := http.Post(url, contentType, body)
			require.Nil(s.T(), err)
			require.Equal(s.T(), http.StatusOK, rsp.StatusCode)

			bts, err := io.ReadAll(rsp.Body)
			require.Nil(s.T(), err)
			require.Nil(s.T(), rsp.Body.Close())
			require.Contains(s.T(), string(bts), `"body":""`)
		}
		doHTTPPost()

		doTHTTPPost := func() {
			rsp, err := thttp.NewStdHTTPClient("http-client").Post(url, contentType, body)
			require.Nil(s.T(), err)
			require.Equal(s.T(), http.StatusOK, rsp.StatusCode)

			bts, err := io.ReadAll(rsp.Body)
			require.Nil(s.T(), err)
			require.Nil(s.T(), rsp.Body.Close())
			require.Contains(s.T(), string(bts), `"body":""`)
		}
		doTHTTPPost()
	}
}
func (s *TestSuite) testHTTPPatchMethod() {
	c := thttp.NewClientProxy(s.listener.Addr().String())
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	rsp := &testpb.SimpleResponse{}

	require.Nil(s.T(), c.Patch(trpc.BackgroundContext(), "/UnaryCall", req, rsp))
	require.Len(s.T(), rsp.Payload.GetBody(), int(req.ResponseSize))
}

func (s *TestSuite) TestHTTPSInsecureSkipVerify() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(
			&testHTTPService{},
			server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", ""),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(s.httpServerEnv.String(), s.testHTTPSInsecureSkipVerify)
		s.Run(s.httpServerEnv.String(), s.testFastHTTPSInsecureSkipVerify)
	}
}
func (s *TestSuite) testHTTPSInsecureSkipVerify() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
	)
	s.Run("connpoolDialOk", func() {
		c, err := connpool.Dial(&connpool.DialOptions{
			Network:     "tcp",
			LocalAddr:   "localhost:0",
			Address:     s.listener.Addr().String(),
			TLSCertFile: clientTLSCert,
			TLSKeyFile:  clientTLSKey,
		})
		require.Nil(s.T(), err)
		require.Nil(s.T(), c.Close())
	})
	s.Run("thttpRequestOk", func() {
		c1 := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, "none", ""),
		)
		rsp := &testpb.SimpleResponse{}
		require.Nil(s.T(), c1.Post(trpc.BackgroundContext(), "/UnaryCall", s.defaultSimpleRequest, rsp))
	})
	s.Run("httpRPCRequestOk", func() {
		c2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, "none", ""),
		)
		_, err := c2.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(time.Second))
		require.Nil(s.T(), err)
	})
	s.Run("netHTTPRequestOk", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)

		c3 := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{cert},
				},
			},
			Timeout: time.Second,
		}
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)
		_, err = c3.Post(
			s.unaryHTTPSCallCustomURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.Nil(s.T(), err)
	})
}

func (s *TestSuite) TestHTTPSProtocolMisMatch() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(
			&testHTTPService{},
			server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", ""),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(s.httpServerEnv.String(), func() { s.testHTTPSProtocolMisMatch(false) })
		s.Run(s.httpServerEnv.String(), func() { s.testFastHTTPSProtocolMisMatch(false) })
	}
}
func (s *TestSuite) testHTTPSProtocolMisMatch(fastHTTPServer bool) {
	s.Run("thttpRequestFailed", func() {
		fc := thttp.NewStdHTTPClient(
			"fasthttp-client",
			client.WithProtocol(protocol.HTTP),
		)
		rsp, err := fc.Get(s.unaryCallCustomURL())

		if fastHTTPServer {
			require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
			require.Contains(s.T(), err.Error(), "EOF")
		} else {
			require.Nil(s.T(), err)
			defer rsp.Body.Close()

			require.Equal(s.T(), http.StatusBadRequest, rsp.StatusCode)
			bs, err := io.ReadAll(rsp.Body)
			require.Nil(s.T(), err)
			require.Equal(s.T(), []byte("Client sent an HTTP request to an HTTPS server.\n"), bs)
		}
	})
	s.Run("thttpRPCRequestFailed", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTP),
			client.WithTarget(s.serverAddress()),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))

		if fastHTTPServer {
			require.Nil(s.T(), rsp)
			require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
		} else {
			require.Nil(s.T(), rsp)
			require.NotNil(s.T(), err, "full err: %+v", err)
		}
	})
	s.Run("originHTTPRequestFailed", func() {
		rsp, err := http.Get(s.unaryCallCustomURL())

		if fastHTTPServer {
			require.Nil(s.T(), rsp)
			require.NotNil(s.T(), err)
			require.Contains(s.T(), err.Error(), "EOF")
		} else {
			require.Nil(s.T(), err)
			require.Equal(s.T(), fasthttp.StatusBadRequest, rsp.StatusCode)
			bs, err := io.ReadAll(rsp.Body)
			require.Nil(s.T(), err)
			require.Equal(s.T(), []byte("Client sent an HTTP request to an HTTPS server.\n"), bs)
		}
	})
}

func (s *TestSuite) TestHTTPSOneWayAuthentication() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(
			&testHTTPService{},
			server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", ""),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })
		s.Run(s.httpServerEnv.String(), s.testHTTPSOneWayAuthentication)
		s.Run(s.httpServerEnv.String(), s.testFastHTTPSOneWayAuthentication)
	}
}
func (s *TestSuite) testHTTPSOneWayAuthentication() {
	s.Run("Ok", s.testHTTPSOneWayOk)
	s.Run("ClientWithoutCertification", s.testHTTPSOneWayClientWithoutCA)
	s.Run("CertificationIsUnmatched", s.testHTTPSOneWayCAIsUnmatched)
	s.Run("InvalidClientTLSCert", s.testHTTPSOneWayInvalidClientTLSCert)
}
func (s *TestSuite) testHTTPSOneWayOk() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
		serverTLSCA   = "x509/server_ca_cert.pem"
		serverName    = "trpc.test.example.com"
	)
	s.Run("connpoolDialOk", func() {
		c, err := connpool.Dial(&connpool.DialOptions{
			Network:     "tcp",
			LocalAddr:   "localhost:0",
			Address:     s.listener.Addr().String(),
			TLSCertFile: clientTLSCert,
			TLSKeyFile:  clientTLSKey,
		})
		require.Nil(s.T(), err)
		require.Nil(s.T(), c.Close())
	})
	s.Run("thttpRequestOk", func() {
		c1 := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp := &testpb.SimpleResponse{}
		require.Nil(s.T(), c1.Post(trpc.BackgroundContext(), "/UnaryCall", req, rsp))
	})
	s.Run("httpRPCRequestOk", func() {
		c2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := c2.UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.Nil(s.T(), err)
	})
	s.Run("netHTTPRequestOk", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)

		b, err := os.ReadFile(serverTLSCA)
		require.Nil(s.T(), err)
		roots := x509.NewCertPool()
		require.True(s.T(), roots.AppendCertsFromPEM(b))

		c3 := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
					RootCAs:      roots,
					ServerName:   serverName,
				},
			},
			Timeout: time.Second,
		}
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)
		_, err = c3.Post(
			s.unaryHTTPSCallCustomURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.Nil(s.T(), err)
	})
}
func (s *TestSuite) testHTTPSOneWayClientWithoutCA() {
	// For explicit HTTPS, caFile must not be empty.
	// If it is, set it to "none" to use tlsConf.InsecureSkipVerify=true.
	s.Run("thttpRequestOK", func() {
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		rsp, err := thttp.NewStdHTTPClient(
			"http-client",
			client.WithProtocol(protocol.HTTPS),
		).Post(
			s.unaryHTTPSCallCustomURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.Nil(s.T(), err)
		require.Equal(s.T(), http.StatusOK, rsp.StatusCode)
	})
	// For explicit HTTPS, caFile must not be empty.
	// If it is, set it to "none" to use tlsConf.InsecureSkipVerify=true.
	s.Run("httpRPCRequestOK", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTPS),
			client.WithTarget(s.serverAddress()),
		).UnaryCall(trpc.BackgroundContext(), req)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), rsp)
	})
	s.Run("netHTTPRequestFailed", func() {
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)
		rsp, err := http.Post(
			s.unaryHTTPSCallCustomURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.NotNil(s.T(), err)
		require.Contains(s.T(), err.Error(), "x509")
		require.Nil(s.T(), rsp)
	})
}
func (s *TestSuite) testHTTPSOneWayCAIsUnmatched() {
	const (
		unmatchedClientTLSCert = "x509/client2_cert.pem"
		unmatchedClientTLSKey  = "x509/client1_key.pem"
		expectedErrorMsg       = "private key does not match public key"
	)
	_, err := tls.LoadX509KeyPair(unmatchedClientTLSCert, unmatchedClientTLSKey)
	require.NotNil(s.T(), err)
	require.Contains(s.T(), err.Error(), expectedErrorMsg)
	s.Run("connpoolDialFail", func() {
		_, err := connpool.Dial(&connpool.DialOptions{
			Network:     "tcp",
			Address:     s.listener.Addr().String(),
			CACertFile:  "root",
			TLSCertFile: unmatchedClientTLSCert,
			TLSKeyFile:  unmatchedClientTLSKey,
		})
		require.Equal(s.T(), errs.RetClientDecodeFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
	s.Run("thttpRequestFailed", func() {
		c1 := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(unmatchedClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp := &testpb.SimpleResponse{}
		err := c1.Post(trpc.BackgroundContext(), "/UnaryCall", req, rsp)
		require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
	s.Run("httpRPCRequestFailed", func() {
		c2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(unmatchedClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := c2.UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.NotNil(s.T(), err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
}
func (s *TestSuite) testHTTPSOneWayInvalidClientTLSCert() {
	const (
		invalidClientTLSCert  = "invalid file path"
		unmatchedClientTLSKey = "x509/client1_key.pem"
	)
	_, err := tls.LoadX509KeyPair(invalidClientTLSCert, unmatchedClientTLSKey)
	require.NotNil(s.T(), err)

	s.Run("connpoolDialFailed", func() {
		_, err := connpool.Dial(&connpool.DialOptions{
			Network:    "tcp",
			Address:    s.listener.Addr().String(),
			CACertFile: invalidClientTLSCert,
		})
		require.Equal(s.T(), errs.RetClientDecodeFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "client dial tls fail")
	})
	s.Run("thttpRequestFailed", func() {
		c1 := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(invalidClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		rsp := &testpb.SimpleResponse{}
		err := c1.Post(trpc.BackgroundContext(), "/UnaryCall", req, rsp)
		require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "getting standard http client failed")
	})
	s.Run("httpRPCRequestFailed", func() {
		c2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(invalidClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := c2.UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.Equal(s.T(), errs.RetClientConnectFail, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), errs.Msg(err), "getting standard http client failed")
	})
}

func (s *TestSuite) TestHTTPSTwoWayAuthentication() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(
			&testHTTPService{},
			server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", "x509/client_ca_cert.pem"),
		)
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(s.httpServerEnv.String(), s.testHTTPSTwoWayAuthentication)
		s.Run(s.httpServerEnv.String(), s.testFastHTTPSTwoWayAuthentication)
	}
}
func (s *TestSuite) testHTTPSTwoWayAuthentication() {
	s.Run("Ok", s.testHTTPSTwoWayOk)
	s.Run("CAIsUnmatched", s.testHTTPSTwoWayCAIsUnmatched)
	s.Run("ClientWithoutCA", s.testHTTPSTwoWayClientWithoutCA)
}
func (s *TestSuite) testHTTPSTwoWayOk() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
		serverTLSCA   = "x509/server_ca_cert.pem"
		serverName    = "trpc.test.example.com"
	)

	s.Run("thttpRequestOk", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		require.Nil(s.T(), thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", req, &testpb.SimpleResponse{}))
	})
	s.Run("httpRPCRequestOk", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.Nil(s.T(), err)
	})
	s.Run("netHTTPRequestOk", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)

		b, err := os.ReadFile(serverTLSCA)
		require.Nil(s.T(), err)

		roots := x509.NewCertPool()
		require.True(s.T(), roots.AppendCertsFromPEM(b))

		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		_, err = (&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
					ServerName:   serverName,
					RootCAs:      roots,
				},
			},
			Timeout: time.Second,
		}).Post(
			s.unaryHTTPSCallCustomURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.Nil(s.T(), err)
	})
}
func (s *TestSuite) testHTTPSTwoWayCAIsUnmatched() {
	const (
		clientTLSCert    = "x509/client1_cert.pem"
		clientTLSKey     = "x509/client1_key.pem"
		serverTLSCA      = "x509/server2_ca_cert.pem"
		serverName       = "trpc.test.example.com"
		expectedErrorMsg = "certificate signed by unknown authority"
	)

	s.Run("thttpRequestFailed", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		err := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", req, &testpb.SimpleResponse{})
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "full err: %+v", err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
	s.Run("httpRPCRequestFailed", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTP),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
	s.Run("netHTTPRequestFailed", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)

		b, err := os.ReadFile(serverTLSCA)
		require.Nil(s.T(), err)

		roots := x509.NewCertPool()
		require.True(s.T(), roots.AppendCertsFromPEM(b))

		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		_, err = (&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
					ServerName:   serverName,
					RootCAs:      roots,
				},
			},
			Timeout: time.Second,
		}).Post(
			s.unaryHTTPSCallDefaultURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
}
func (s *TestSuite) testHTTPSTwoWayClientWithoutCA() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
		serverName    = "trpc.test.example.com"
	)

	s.Run("thttpRequestFailed", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		err := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithProtocol(protocol.HTTPS),
			client.WithTLS("", clientTLSKey, "none", serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", req, &testpb.SimpleResponse{})
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err), "client didn't provide a certFile")
	})
	s.Run("httpRPCRequestFailed", func() {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol(protocol.HTTPS),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, "", serverName),
		).UnaryCall(trpc.BackgroundContext(), req, client.WithTimeout(time.Second))
		require.NotNil(s.T(), err)
	})
	s.Run("netHTTPRequestFailed", func() {
		cert, err := tls.LoadX509KeyPair(clientTLSCert, clientTLSKey)
		require.Nil(s.T(), err)
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)
		_, err = (&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{cert}, ServerName: serverName},
			},
		}).Post(
			s.unaryHTTPSCallDefaultURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.NotNil(s.T(), err, "certificate signed by unknown authority")
	})
}

func (s *TestSuite) TestPassthroughForClientInvocation() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))
		s.T().Cleanup(func() { s.closeServer(nil) })

		s.Run(e.String(), func() { s.testPassthroughForClientInvocation() })
		s.Run(e.String(), func() { s.testFastHTTPPassthroughForClientInvocation() })
	}
}
func (s *TestSuite) testPassthroughForClientInvocation() {
	c := thttp.NewStdHTTPClient("http-client", client.WithTarget(s.serverAddress()+"10086"))
	rsp, err := c.Get(s.unaryCallCustomURL())
	require.Nil(s.T(), err)
	require.NotNil(s.T(), rsp)
	require.Equal(s.T(), http.StatusOK, rsp.StatusCode)
}

func (s *TestSuite) TestSendHTTPSRaw() {
	go http.ListenAndServeTLS("127.0.0.1:8081", "x509/server1_cert.pem", "x509/server1_key.pem", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	time.Sleep(time.Second)

	code, body, err := fasthttp.Get(nil, "http://127.0.0.1:8081")
	require.Nil(s.T(), err)
	require.Equal(s.T(), http.StatusBadRequest, code)
	require.Equal(s.T(), []byte("Client sent an HTTP request to an HTTPS server.\n"), body)

	rsp, err := http.Get("http://127.0.0.1:8081")
	require.Nil(s.T(), err)
	require.Equal(s.T(), http.StatusBadRequest, rsp.StatusCode)
	bs, err := io.ReadAll(rsp.Body)
	require.Nil(s.T(), err)
	defer rsp.Body.Close()
	require.Equal(s.T(), []byte("Client sent an HTTP request to an HTTPS server.\n"), bs)
}

const (
	compressTypeGzip = "gzip" // gzip compression
	compressTypeNoop = "noop" // noop compression
)

// Test that the http client sends messages in different compression formats
// and the server replies with messages in different compression formats.
func (s *TestSuite) TestHTTPClientAndServerCompressType() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.startServer(&testHTTPService{
			TRPCService: TRPCService{
				UnaryCallF: func(ctx context.Context, in *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
					header := thttp.Head(ctx).Request.Header
					// Test server using gzip compression
					if header.Get("Server-Compress-Type") == compressTypeGzip {
						thttp.Response(ctx).Header().Set("Content-Encoding", "gzip")
						// Compress messages using gzip
						var buf bytes.Buffer
						gz := gzip.NewWriter(&buf)
						_, err := gz.Write([]byte("test-CompressTypeGzip"))
						if err != nil {
							return nil, err
						}
						if err := gz.Close(); err != nil {
							return nil, err
						}
						// Write Message
						thttp.Response(ctx).Write(buf.Bytes())
						return nil, nil
					}
					// Test server uses no compression
					if header.Get("Server-Compress-Type") == compressTypeNoop {
						// Write Message
						thttp.Response(ctx).Write([]byte("test-CompressTypeNoop"))
						return nil, nil
					}
					return nil, nil
				},
			},
		})
		s.T().Cleanup(func() { s.closeServer(nil) })
		s.Run(s.httpServerEnv.String(), s.testHTTPClientAndServerCompressType)
	}
}

func (s *TestSuite) testHTTPClientAndServerCompressType() {
	serverCompressType := []string{
		compressTypeGzip,
		compressTypeNoop,
	}
	clientCompressType := []int{
		codec.CompressTypeGzip,
		codec.CompressTypeNoop,
	}

	for _, sct := range serverCompressType {
		for _, cct := range clientCompressType {
			doHTTPRequest := func() {
				data := "Hello, I am http client!"
				if sct == compressTypeGzip {
					// Compress messages using gzip
					var gzipBuffer bytes.Buffer
					gzipWriter := gzip.NewWriter(&gzipBuffer)
					_, err := gzipWriter.Write([]byte(data))
					require.Nil(s.T(), err)
					err = gzipWriter.Close()
					require.Nil(s.T(), err)

					// Constructing a request
					req, err := http.NewRequest("POST", s.unaryCallCustomURL(), &gzipBuffer)
					require.Nil(s.T(), err)
					req.Header.Set("Content-Encoding", sct)
					req.Header.Set("Server-Compress-Type", sct)

					// Sending post request using net/http package
					c := &http.Client{}
					resp, err := c.Do(req)
					require.Nil(s.T(), err)
					require.Equal(s.T(), http.StatusOK, resp.StatusCode)
				}
				if sct == compressTypeNoop {
					// Constructing a request
					req, err := http.NewRequest("POST", s.unaryCallCustomURL(), strings.NewReader(data))
					require.Nil(s.T(), err)
					req.Header.Set("Server-Compress-Type", sct)

					// Sending post request using net/http package
					c := &http.Client{}
					resp, err := c.Do(req)
					require.Nil(s.T(), err)
					require.Equal(s.T(), http.StatusOK, resp.StatusCode)
				}
			}
			doHTTPRequest()

			doTHTTPPost := func() {
				req := &codec.Body{Data: []byte("Hello, I am thttp client!")}
				parsedURL, err := url.Parse(s.unaryCallCustomURL())
				require.Nil(s.T(), err)

				// Create a ClientReqHeader with the specified HTTP method (POST)
				reqHeader := &thttp.ClientReqHeader{
					Method: http.MethodPost,
				}

				// Add a custom "Server-Compress-Type" header to the HTTP request header
				reqHeader.AddHeader("Server-Compress-Type", sct)

				// Create a ClientProxy, set the protocol to HTTP, and use Noop serialization.
				httpCli := thttp.NewClientProxy("trpc.app.server.stdhttp",
					client.WithSerializationType(codec.SerializationTypeNoop),
					client.WithTarget("ip://"+parsedURL.Host),
					client.WithCompressType(cct),
					client.WithReqHead(reqHeader),
				)
				rsp := &codec.Body{}
				err = httpCli.Post(context.Background(), parsedURL.Path, req, rsp)
				require.Nil(s.T(), err)
			}
			doTHTTPPost()
		}
	}
}
