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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/server"

	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestCustomErrorHandler() {
	for _, e := range allHTTPRPCEnvs {
		s.Run(e.String(), func() { s.testCustomErrorHandler(e) })
	}
}
func (s *TestSuite) testCustomErrorHandler(e *httpRPCEnv) {
	oldErrHandler := thttp.DefaultServerCodec.ErrHandler
	thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
		w.Header().Set("custom-error", fmt.Sprintf(`{"ret-code":%d, "ret-msg":"%s"}`, e.Code, e.Msg))
	}
	defer func() {
		thttp.DefaultServerCodec.ErrHandler = oldErrHandler
	}()
	s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))

	s.T().Cleanup(func() { s.closeServer(nil) })

	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(rspHead),
		client.WithMultiplexed(e.client.multiplexed),
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
	require.Nil(s.T(), json.Unmarshal([]byte(rspHead.Response.Header.Get("custom-error")), ce))
	require.Equal(s.T(), retUnsupportedPayload, ce.RetCode)
}

func (s *TestSuite) TestDefaultErrorHandler() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testDefaultErrorHandler(e) })
	}
}
func (s *TestSuite) testDefaultErrorHandler(e *httpRPCEnv) {
	s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))

	s.T().Cleanup(func() { s.closeServer(nil) })

	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(rspHead),
		client.WithMultiplexed(e.client.multiplexed),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}

	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.ResponseType = testpb.PayloadType_RANDOM
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.EqualValues(s.T(), retUnsupportedPayload, errs.Code(err))
	require.Contains(s.T(), errs.Msg(err), "unsupported payload type")
	require.Equal(s.T(), fmt.Sprint(errs.Code(err)), rspHead.Response.Header.Get(thttp.TrpcUserFuncErrorCode))
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err))
	require.Equal(
		s.T(),
		http.StatusOK,
		rspHead.Response.StatusCode,
		"any framework error code not in thttp.ErrsToHTTPStatus map are converted to ttp.StatusOK",
	)

}

func (s *TestSuite) TestSendHTTPSRequestToHTTPServer() {
	for _, e := range allHTTPRPCEnvs {
		s.Run(e.String(), func() { s.testSendHTTPSRequestToHTTPServer(e) })
	}
}
func (s *TestSuite) testSendHTTPSRequestToHTTPServer(e *httpRPCEnv) {
	s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))

	s.T().Cleanup(func() { s.closeServer(nil) })

	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(&thttp.ClientRspHeader{}),
		client.WithProtocol("https"),
		client.WithMultiplexed(e.client.multiplexed),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)

	require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err))
	require.Contains(s.T(), errs.Msg(err), "codec empty")
}

func (s *TestSuite) TestStatusBadRequestDueToServerValidateFail() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testStatusBadRequestDueToServerValidateFail(e) })
	}
}
func (s *TestSuite) testStatusBadRequestDueToServerValidateFail(e *httpRPCEnv) {
	s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))

	s.T().Cleanup(func() { s.closeServer(nil) })

	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(rspHead),
		client.WithMultiplexed(e.client.multiplexed),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.Username = "non-validate-name-?.@&*-_"
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerValidateFail, errs.Code(err))
	require.Equal(s.T(), http.StatusBadRequest, thttp.ErrsToHTTPStatus[errs.Code(err)])
	log.Debug(errs.Code(err))
	require.Equal(s.T(), fmt.Sprint(errs.Code(err).Number()), rspHead.Response.Header.Get(thttp.TrpcUserFuncErrorCode))
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err))
	require.Equal(s.T(), http.StatusBadRequest, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusNotFoundDueToServerNoService() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testStatusNotFoundDueToServerNoService(e) })
	}
}
func (s *TestSuite) testStatusNotFoundDueToServerNoService(e *httpRPCEnv) {
	startServerWithoutAnyService := func(t *testing.T) {
		t.Helper()
		trpc.ServerConfigPath = "trpc_go_http_server.yaml"

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

	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(rspHead),
		client.WithMultiplexed(e.client.multiplexed),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)

	require.Equal(s.T(), errs.RetServerNoFunc, errs.Code(err))
	require.Equal(s.T(), http.StatusNotFound, thttp.ErrsToHTTPStatus[errs.Code(err)])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err))
	require.Equal(s.T(), http.StatusNotFound, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusNotFoundDueToServerNoFunc() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testStatusNotFoundDueToServerNoFunc(e) })
	}
}
func (s *TestSuite) testStatusNotFoundDueToServerNoFunc(e *httpRPCEnv) {
	s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))

	s.T().Cleanup(func() { s.closeServer(nil) })

	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(rspHead),
		client.WithTarget(s.serverAddress() + "/NonexistentCall"),
		client.WithMultiplexed(e.client.multiplexed),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)

	require.Equal(s.T(), errs.RetServerNoFunc, errs.Code(err))
	require.Equal(s.T(), http.StatusNotFound, thttp.ErrsToHTTPStatus[errs.Code(err)])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err))
	require.Equal(s.T(), http.StatusNotFound, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusGatewayTimeoutDueToServerTimeout() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testStatusGatewayTimeoutDueToServerTimeout(e) })
	}
}
func (s *TestSuite) testStatusGatewayTimeoutDueToServerTimeout(e *httpRPCEnv) {
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

	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(rspHead),
		client.WithMultiplexed(e.client.multiplexed),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)

	require.Equal(s.T(), errs.RetServerTimeout, errs.Code(err))
	require.Equal(s.T(), http.StatusGatewayTimeout, thttp.ErrsToHTTPStatus[errs.Code(err)])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err))
	require.Equal(s.T(), http.StatusGatewayTimeout, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusTooManyRequestsDueToServerOverload() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testStatusTooManyRequestsDueToServerOverload(e) })
	}
}
func (s *TestSuite) testStatusTooManyRequestsDueToServerOverload(e *httpRPCEnv) {
	const maxRequestQueueSize = 10
	requestQueue := make(chan interface{}, maxRequestQueueSize)
	defer func() {
		close(requestQueue)
	}()
	const limitedAccessUser = "LimitedAccessUser"
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

	sendRequest := func() (*thttp.ClientRspHeader, error) {
		req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
		req.Username = limitedAccessUser
		rspHead := &thttp.ClientRspHeader{}
		opts := []client.Option{
			client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
			client.WithRspHead(rspHead),
			client.WithMultiplexed(e.client.multiplexed),
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

	require.Equal(s.T(), errs.RetServerOverload, errs.Code(err))
	require.Equal(s.T(), http.StatusTooManyRequests, thttp.ErrsToHTTPStatus[errs.Code(err)])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err))
	require.Equal(s.T(), http.StatusTooManyRequests, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusUnauthorizedDueToServerAuthFail() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testStatusUnauthorizedDueToServerAuthFail(e) })
	}
}
func (s *TestSuite) testStatusUnauthorizedDueToServerAuthFail(e *httpRPCEnv) {
	s.startServer(&testHTTPService{}, server.WithServerAsync(e.server.async))

	s.T().Cleanup(func() { s.closeServer(nil) })

	var rspHead = &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(rspHead),
		client.WithMultiplexed(e.client.multiplexed),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	req := proto.Clone(s.defaultSimpleRequest).(*testpb.SimpleRequest)
	req.Username = "invalidUsername"
	req.FillUsername = true
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), req)

	require.Equal(s.T(), errs.RetServerAuthFail, errs.Code(err))
	require.Equal(s.T(), http.StatusUnauthorized, thttp.ErrsToHTTPStatus[errs.Code(err)])
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err))
	require.Equal(s.T(), http.StatusUnauthorized, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestStatusInternalServerDueToServerReturnUnknown() {
	for _, e := range allHTTPRPCEnvs {
		if e.client.multiplexed {
			continue
		}
		s.Run(e.String(), func() { s.testStatusInternalServerDueToServerReturnUnknown(e) })
	}
}
func (s *TestSuite) testStatusInternalServerDueToServerReturnUnknown(e *httpRPCEnv) {
	s.startServer(
		&testHTTPService{},
		server.WithServerAsync(e.server.async),
		server.WithFilter(
			func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
				return nil, fmt.Errorf("unknown")
			}),
	)

	s.T().Cleanup(func() { s.closeServer(nil) })

	rspHead := &thttp.ClientRspHeader{}
	opts := []client.Option{
		client.WithReqHead(&thttp.ClientReqHeader{Method: "post"}),
		client.WithRspHead(rspHead),
		client.WithMultiplexed(e.client.multiplexed),
	}
	if e.client.disableConnectionPool {
		opts = append(opts, client.WithDisableConnectionPool())
	}
	_, err := s.newHTTPRPCClient(opts...).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest)

	require.Equal(s.T(), errs.RetUnknown, errs.Code(err))
	require.Equal(s.T(), http.StatusInternalServerError, thttp.ErrsToHTTPStatus[errs.Code(err)])
	require.EqualValues(s.T(), fmt.Sprint(errs.Code(err).Number()), rspHead.Response.Header.Get(thttp.TrpcUserFuncErrorCode))
	require.Equal(s.T(), rspHead.Response.Header.Get("Trpc-Error-Msg"), errs.Msg(err))
	require.Equal(s.T(), http.StatusInternalServerError, rspHead.Response.StatusCode)
}

func (s *TestSuite) TestCustomResponseHandler() {
	type customResponse struct {
		PayloadType int    `json:"payload-type"`
		PayloadBody []byte `json:"payload-body"`
		Username    string `json:"username"`
	}

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
func (s *TestSuite) TestStatusBadRequestDueToServerDecodeFail() {
	s.startServer(&testHTTPService{})

	bts, err := json.Marshal(s.defaultSimpleRequest)
	require.Nil(s.T(), err)

	rsp, err := http.Post(s.unaryCallCustomURL(), "application/pb", bytes.NewReader(bts))
	require.Nil(s.T(), err)
	defer rsp.Body.Close()
	require.Equal(s.T(), http.StatusBadRequest, rsp.StatusCode)

	c := thttp.NewStdHTTPClient("http-client")
	rsp, err = c.Post(s.unaryCallCustomURL(), "application/pb", bytes.NewReader(bts))
	require.Equal(s.T(), errs.RetServerDecodeFail, errs.Code(err))
	require.Equal(s.T(), http.StatusBadRequest, thttp.ErrsToHTTPStatus[errs.Code(err)])
	require.Nil(s.T(), rsp)
}

func (s *TestSuite) TestHTTP() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.Run(s.httpServerEnv.String(), s.testHTTP)
	}
}
func (s *TestSuite) testHTTP() {
	thttp.RegisterStatus(http.StatusUnsupportedMediaType, http.StatusUnsupportedMediaType)
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
	s.Run("AccessNonexistentResource", s.testHTTPAccessNonexistentResource)
	s.Run("SendSupportedContentType", s.testHTTPSendSupportedContentType)
	s.Run("ServerReceivedUnsupportedContentType", s.testHTTPServerReceivedUnsupportedContentType)
	s.Run("ClientReceivedUnsupportedContentType", s.testHTTPClientReceivedUnsupportedContentType)
	s.Run("EmptyBody", s.testHTTPEmptyBody)
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
				require.Equal(s.T(), http.StatusNotFound, thttp.ErrsToHTTPStatus[errs.Code(err)])
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

			bts, _ := io.ReadAll(rsp.Body)
			require.Nil(s.T(), rsp.Body.Close())
			require.Contains(s.T(), string(bts), `"body":""`)
		}
		doHTTPPost()

		doTHTTPPost := func() {
			rsp, err := thttp.NewStdHTTPClient("http-client").Post(url, contentType, body)
			require.Nil(s.T(), err)
			require.Equal(s.T(), http.StatusOK, rsp.StatusCode)

			bts, _ := io.ReadAll(rsp.Body)
			require.Nil(s.T(), rsp.Body.Close())
			require.Contains(s.T(), string(bts), `"body":""`)
		}
		doTHTTPPost()
	}
}

func (s *TestSuite) TestHTTPSOneWayAuthentication() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.Run(s.httpServerEnv.String(), s.testHTTPSOneWayAuthentication)
	}
}
func (s *TestSuite) testHTTPSOneWayAuthentication() {
	s.startServer(
		&testHTTPService{},
		server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", ""),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })
	s.Run("Ok", s.testHTTPSOneWayOk)
	s.Run("ClientWithoutCertification", s.testHTTPSOneWayClientWithoutCA)
	s.Run("CertificationIsUnmatched", s.testHTTPSOneWayCAIsUnmatched)
}
func (s *TestSuite) testHTTPSOneWayOk() {
	const (
		clientTLSCert = "x509/client1_cert.pem"
		clientTLSKey  = "x509/client1_key.pem"
	)

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
			client.WithProtocol("http"),
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
					Certificates: []tls.Certificate{cert},
				},
			},
			Timeout: time.Second,
		}
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)
		_, err = c3.Post(
			s.unaryCallCustomURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.Nil(s.T(), err)
	})
}
func (s *TestSuite) testHTTPSOneWayClientWithoutCA() {
	s.Run("thttpRequestFailed", func() {
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		_, err = thttp.NewStdHTTPClient(
			"http-client",
			client.WithProtocol("http"),
		).Post(
			s.unaryCallCustomURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.Equal(s.T(), errs.RetClientDecodeFail, errs.Code(err))
		require.Contains(s.T(), err.Error(), "readall http body fail")
	})
	s.Run("httpRPCRequestFailed", func() {
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol("http"),
			client.WithTarget(s.serverAddress()),
		).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(time.Second))
		require.NotNil(s.T(), err)
	})
	s.Run("netHTTPRequestFailed", func() {
		bts, err := json.Marshal(s.defaultSimpleRequest)
		require.Nil(s.T(), err)

		rsp, _ := (&http.Client{Timeout: time.Second}).Post(
			s.unaryCallCustomURL(),
			"application/json",
			bytes.NewReader(bts),
		)
		require.Equal(s.T(), http.StatusBadRequest, rsp.StatusCode)
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

	s.Run("thttpRequestFailed", func() {
		c1 := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(unmatchedClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		rsp := &testpb.SimpleResponse{}
		err := c1.Post(trpc.BackgroundContext(), "/UnaryCall", s.defaultSimpleRequest, rsp)
		require.Equal(s.T(), errs.RetClientDecodeFail, errs.Code(err))
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
	s.Run("httpRPCRequestFailed", func() {
		c2 := testpb.NewTestHTTPClientProxy(
			client.WithProtocol("http"),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(unmatchedClientTLSCert, unmatchedClientTLSKey, "root", ""),
		)
		_, err := c2.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(time.Second))
		require.NotNil(s.T(), err)
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
}

func (s *TestSuite) TestHTTPSTwoWayAuthentication() {
	for _, e := range allHTTPServerEnvs {
		s.httpServerEnv = e
		s.Run(s.httpServerEnv.String(), s.testHTTPSTwoWayAuthentication)
	}
}
func (s *TestSuite) testHTTPSTwoWayAuthentication() {
	s.startServer(
		&testHTTPService{},
		server.WithTLS("x509/server1_cert.pem", "x509/server1_key.pem", "x509/client_ca_cert.pem"),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })
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
		require.Nil(s.T(), thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", s.defaultSimpleRequest, &testpb.SimpleResponse{}))
	})
	s.Run("httpRPCRequestOk", func() {
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol("http"),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(time.Second))
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
			s.unaryCallCustomURL(),
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
		err := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", s.defaultSimpleRequest, &testpb.SimpleResponse{})
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err))
		require.Contains(s.T(), err.Error(), expectedErrorMsg)
	})
	s.Run("httpRPCRequestFailed", func() {
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol("http"),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, serverTLSCA, serverName),
		).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(time.Second))
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
			fmt.Sprintf("https://%v/UnaryCall", s.listener.Addr()),
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
		err := thttp.NewClientProxy(
			s.listener.Addr().String(),
			client.WithTLS(clientTLSCert, clientTLSKey, "none", serverName),
		).Post(trpc.BackgroundContext(), "/UnaryCall", s.defaultSimpleRequest, &testpb.SimpleResponse{})
		require.Equal(s.T(), errs.RetClientNetErr, errs.Code(err))
		require.Contains(s.T(), err.Error(), "tls: bad certificate")
	})
	s.Run("httpRPCRequestFailed", func() {
		_, err := testpb.NewTestHTTPClientProxy(
			client.WithProtocol("http"),
			client.WithTarget(s.serverAddress()),
			client.WithTLS(clientTLSCert, clientTLSKey, "", serverName),
		).UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(time.Second))
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
			fmt.Sprintf("https://%v/UnaryCall", s.listener.Addr()),
			"application/json",
			bytes.NewReader(bts),
		)
		require.NotNil(s.T(), err, "certificate signed by unknown authority")
	})
}
