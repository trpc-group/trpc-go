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
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
	httpdata "trpc.group/trpc-go/trpc-go/test/testdata"
)

func (s *TestSuite) TestHTTPRuleOK() {
	for _, e := range allRESTfulServerEnv {
		s.Run(e.String(), func() { s.testHTTPRuleOK(e) })
	}
}
func (s *TestSuite) testHTTPRuleOK(e *restfulServerEnv) {
	s.startServer(
		&testRESTfulService{},
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
		server.WithServerAsync(e.async),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	s.Run("fill user name", func() {
		rsp, err := http.Post(
			s.unaryCallCustomURL(),
			"application/json",
			bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{
				FillUsername: true,
				Username:     validUserNameForAuth,
			})),
		)
		require.Condition(s.T(), func() bool {
			return err == nil && rsp.StatusCode == http.StatusOK
		})

		var r testpb.SimpleResponse
		mustUnmarshalProtoJSON(s.T(), rsp.Body, &r)
		if err := rsp.Body.Close(); err != nil {
			s.T().Log(err)
		}
		require.Equal(s.T(), validUserNameForAuth, r.Username)
	})
	s.Run("don't fill user name ", func() {
		rsp, err := http.Get(fmt.Sprintf("http://%v/UnaryCall/%s", s.listener.Addr(), validUserNameForAuth))
		require.Condition(s.T(), func() bool {
			return err == nil && rsp.StatusCode == http.StatusOK
		})

		var r testpb.SimpleResponse
		mustUnmarshalProtoJSON(s.T(), rsp.Body, &r)
		if err := rsp.Body.Close(); err != nil {
			s.T().Log(err)
		}
		require.Equal(s.T(), "", r.Username)
	})
}

func (s *TestSuite) TestContentTypeMultipartFormData() {
	for _, e := range allRESTfulServerEnv {
		s.Run(e.String(), func() { s.testContentTypeMultipartFormData(e) })
	}
}
func (s *TestSuite) testContentTypeMultipartFormData(e *restfulServerEnv) {
	l, err := net.Listen("tcp", defaultServerAddress)
	require.Nil(s.T(), err)
	s.listener = l
	svr := s.newRESTfulServer(
		&testRESTfulService{},
		server.WithListener(l),
		server.WithServerAsync(e.async),
	)
	serviceName := fmt.Sprintf("trpc.testing.end2end.TestRESTful%d", s.autoIncrID)
	router := restful.GetRouter(serviceName)
	processMultipartFormData := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := &thttp.Header{Request: r, Response: &httptest.ResponseRecorder{}}
			msg := codec.Message(thttp.WithHeader(context.Background(), header))
			in, err := thttp.DefaultServerCodec.Decode(msg, nil)
			require.Nil(s.T(), err)
			require.Equal(s.T(), httpdata.MultipartFormDataFirstPartNames, string(in))

			head := thttp.Head(msg.Context())
			file1, fileHeader1, err := head.Request.FormFile("file1")
			defer func() {
				require.Nil(s.T(), file1.Close())
			}()
			require.Nil(s.T(), err)
			require.Equal(s.T(), "text/plain", fileHeader1.Header.Get("Content-Type"))
			require.Equal(s.T(), "1.txt", fileHeader1.Filename)

			file2, fileHeader2, err := head.Request.FormFile("file2")
			require.Nil(s.T(), err)
			defer func() {
				require.Nil(s.T(), file2.Close())
			}()
			require.Equal(s.T(), "image/png", fileHeader2.Header.Get("Content-Type"))
			require.Equal(s.T(), "1px.png", fileHeader2.Filename)

			file3, fileHeader3, err := head.Request.FormFile("file3")
			require.Nil(s.T(), err)
			defer func() {
				require.Nil(s.T(), file3.Close())
			}()
			require.Equal(s.T(), "application/json", fileHeader3.Header.Get("Content-Type"))
			require.Equal(s.T(), "json.json", fileHeader3.Filename)

			w.Header().Add("Content-Type", r.Header.Get("Content-Type"))
		})
	}
	restful.RegisterRouter(serviceName, processMultipartFormData(router))

	go func(t *testing.T) {
		if err := svr.Serve(); err != nil {
			t.Log(err)
		}
	}(s.T())
	s.T().Cleanup(func() {
		if err != svr.Close(nil) {
			s.T().Log(err)
		}
	})

	r, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("http://%v/UnaryCall/%s", s.listener.Addr(), "TestContentTypeMultipartFormData"),
		bytes.NewReader([]byte("")),
	)
	r.Header.Add("Content-Type", httpdata.MultipartFormDataBoundary)
	r.Body = io.NopCloser(strings.NewReader(httpdata.MultipartFormDataBody))

	rsp, err := (&http.Client{}).Do(r)
	require.Condition(s.T(), func() bool {
		return err == nil && rsp.Header.Get("Content-Type") == httpdata.MultipartFormDataBoundary
	})

	bts, err := io.ReadAll(rsp.Body)
	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}
	require.Condition(s.T(), func() bool {
		return err == nil && len(bts) == 0
	})
}

func (s *TestSuite) TestDefaultHeaderMatcher() {
	for _, e := range allRESTfulServerEnv {
		s.Run(e.String(), func() { s.testDefaultHeaderMatcher(e) })
	}
}
func (s *TestSuite) testDefaultHeaderMatcher(e *restfulServerEnv) {
	type contextMessage struct {
		ServerRPCName     string `json:"server-rpc-name"`
		SerializationType int    `json:"serialization-type"`
	}

	s.startServer(&testRESTfulService{
		UnaryCallF: func(ctx context.Context, req *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
			msg := trpc.Message(ctx)
			bs, err := json.Marshal(&contextMessage{
				ServerRPCName:     msg.ServerRPCName(),
				SerializationType: msg.SerializationType(),
			})
			if err != nil {
				return nil, err
			}

			return &testpb.SimpleResponse{
				Username: req.GetUsername(),
				Payload: &testpb.Payload{
					Type: testpb.PayloadType_COMPRESSIBLE,
					Body: bs,
				},
			}, nil
		},
	}, server.WithServerAsync(e.async), server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)))
	s.T().Cleanup(func() { s.closeServer(nil) })

	rsp, err := http.Post(
		s.unaryCallCustomURL(),
		"application/json",
		bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{})),
	)
	require.Condition(s.T(), func() bool {
		return err == nil && rsp.StatusCode == http.StatusOK
	})

	var sr testpb.SimpleResponse
	mustUnmarshalProtoJSON(s.T(), rsp.Body, &sr)
	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}

	cm := contextMessage{}
	mustUnmarshalJSON(s.T(), sr.Payload.Body, &cm)
	require.Equal(
		s.T(),
		contextMessage{
			ServerRPCName:     "/trpc.testing.end2end.TestRESTful/UnaryCall",
			SerializationType: codec.SerializationTypePB,
		},
		cm,
	)
}

func (s *TestSuite) TestCustomHeaderMatcher() {
	for _, e := range allRESTfulServerEnv {
		if e.basedOnFastHTTP {
			continue
		}
		s.Run(e.String(), func() { s.testCustomHeaderMatcher(e) })
	}
}
func (s *TestSuite) testCustomHeaderMatcher(e *restfulServerEnv) {
	headerMatcher := func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
		serviceName, methodName string,
	) (context.Context, error) {
		ctx, msg := codec.WithNewMessage(ctx)
		msg.WithCallerApp("end2end-testing/restful-test")
		msg.WithEnvName("test")
		return ctx, nil
	}
	s.startServer(
		&testRESTfulService{
			UnaryCallF: func(ctx context.Context, req *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
				if app := trpc.Message(ctx).CallerApp(); app != "end2end-testing/restful-test" {
					return nil, fmt.Errorf("caller app %v isn't matched", app)
				}
				return &testpb.SimpleResponse{}, nil
			},
		},
		server.WithRESTOptions(restful.WithHeaderMatcher(headerMatcher)),
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	resp, err := http.Post(
		s.unaryCallCustomURL(),
		"application/json",
		bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{})),
	)
	require.Condition(s.T(), func() bool {
		s.T().Log(resp.StatusCode)
		return err == nil && resp.StatusCode == http.StatusOK
	})
}

func (s *TestSuite) TestRESTfulCustomResponseHandler() {
	for _, e := range allRESTfulServerEnv {
		if e.basedOnFastHTTP {
			continue
		}
		s.Run(e.String(), func() { s.testRESTfulCustomResponseHandler(e) })
	}
}
func (s *TestSuite) testRESTfulCustomResponseHandler(e *restfulServerEnv) {
	s.startServer(&testRESTfulService{},
		server.WithRESTOptions(restful.WithResponseHandler(func(
			ctx context.Context,
			w http.ResponseWriter,
			r *http.Request,
			resp proto.Message,
			body []byte,
		) error {
			if _, ok := resp.(*testpb.SimpleResponse); !ok {
				return fmt.Errorf("the type of resp %v isn't *testpb.SimpleResponse", resp)
			}
			http.SetCookie(
				w,
				&http.Cookie{
					Name:    "server-cookie",
					Value:   "server-cookie-value",
					Expires: time.Now().AddDate(0, 0, 1),
				},
			)
			return nil
		})),
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	rsp, err := http.Post(
		s.unaryCallCustomURL(),
		"application/json",
		bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{})),
	)
	require.Nil(s.T(), err)
	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}
	require.Condition(s.T(), func() bool {
		c := rsp.Cookies()
		return len(c) == 1 && c[0].Name == "server-cookie"
	})

}

func (s *TestSuite) TestRESTfulDefaultErrorHandler() {
	for _, e := range allRESTfulServerEnv {
		s.Run(e.String(), func() { s.testRESTfulDefaultErrorHandler(e) })
	}
}
func (s *TestSuite) testRESTfulDefaultErrorHandler(e *restfulServerEnv) {
	s.startServer(
		&testRESTfulService{},
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	rsp, err := http.Post(
		s.unaryCallCustomURL(),
		"application/json",
		bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{ResponseType: testpb.PayloadType_RANDOM})),
	)
	require.Nil(s.T(), err)
	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}
	require.Equal(
		s.T(),
		http.StatusInternalServerError,
		rsp.StatusCode,
		"any error code not in restful.httpStatusMap are converted to http.StatusInternalServerError",
	)
}

func (s *TestSuite) TestWithStatusCodeOption() {
	for _, e := range allRESTfulServerEnv {
		s.Run(e.String(), func() { s.testWithStatusCodeOption(e) })
	}
}
func (s *TestSuite) testWithStatusCodeOption(e *restfulServerEnv) {
	s.startServer(
		&testRESTfulService{
			UnaryCallF: func(ctx context.Context, req *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
				time.Sleep(time.Second)
				return nil, &restful.WithStatusCode{
					StatusCode: http.StatusRequestTimeout,
					Err:        fmt.Errorf("test error"),
				}
			},
		},
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	rsp, err := http.Post(
		s.unaryCallCustomURL(),
		"application/json",
		bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{})),
	)
	require.Condition(s.T(), func() bool {
		return err == nil && rsp.StatusCode == http.StatusRequestTimeout
	})

	bts, err := io.ReadAll(rsp.Body)
	require.Condition(s.T(), func() bool {
		return err == nil && bytes.Contains(bts, []byte(`"message":"test error"`))
	})
	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}
}

func (s *TestSuite) TestRESTfulCustomErrorHandler() {
	for _, e := range allRESTfulServerEnv {
		if e.basedOnFastHTTP {
			continue
		}
		s.Run(e.String(), func() { s.testRESTfulCustomErrorHandler(e) })
	}
}
func (s *TestSuite) testRESTfulCustomErrorHandler(e *restfulServerEnv) {
	errorHandler := func(_ context.Context, w http.ResponseWriter, _ *http.Request, e error) {

		if _, err := w.Write([]byte(
			fmt.Sprintf(`{"ret-code":%d, "ret-msg":"%s"}`, errs.Code(e), errs.Msg(e))),
		); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	s.startServer(
		&testRESTfulService{},
		server.WithRESTOptions(restful.WithErrorHandler(errorHandler)),
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	rsp, err := http.Post(
		s.unaryCallCustomURL(),
		"application/json",
		bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{ResponseType: testpb.PayloadType_RANDOM})),
	)
	require.Nil(s.T(), err)

	bts, err := io.ReadAll(rsp.Body)
	require.Nil(s.T(), err)
	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}

	ret := struct {
		Code int    `json:"ret-code"`
		Msg  string `json:"ret-msg"`
	}{}
	mustUnmarshalJSON(s.T(), bts, &ret)
	require.Equal(s.T(), retUnsupportedPayload, ret.Code)
}

func (s *TestSuite) TestRESTfulServerReceivedUnsupportedContentType() {
	for _, e := range allRESTfulServerEnv {
		if e.basedOnFastHTTP {
			continue
		}
		s.Run(e.String(), func() { s.testRESTfulServerReceivedUnsupportedContentType(e) })
	}
}
func (s *TestSuite) testRESTfulServerReceivedUnsupportedContentType(e *restfulServerEnv) {
	const serverUnsupportedContentType = "server-unsupported-content-type"
	errorHandler := func(_ context.Context, w http.ResponseWriter, r *http.Request, e error) {
		if strings.Contains(r.Header.Get("Content-Type"), serverUnsupportedContentType) {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			e = errs.Wrap(e, http.StatusUnsupportedMediaType, "Unsupported Media Type")
		}
		if _, err := fmt.Fprintf(w, `{"ret-code":%d, "ret-msg":"%s"}`, errs.Code(e), errs.Msg(e)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	s.startServer(
		&testRESTfulService{},
		server.WithRESTOptions(restful.WithErrorHandler(errorHandler)),
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	rsp, err := http.Post(
		s.unaryCallCustomURL(),
		serverUnsupportedContentType,
		bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{ResponseType: testpb.PayloadType_RANDOM})),
	)
	require.Condition(s.T(), func() bool {
		return err == nil && rsp.StatusCode == http.StatusUnsupportedMediaType
	})

	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}
}

func (s *TestSuite) TestRESTfulClientReceivedUnsupportedContentType() {
	for _, e := range allRESTfulServerEnv {
		if e.basedOnFastHTTP {
			continue
		}
		s.Run(e.String(), func() { s.testRESTfulClientReceivedUnsupportedContentType(e) })
	}
}
func (s *TestSuite) testRESTfulClientReceivedUnsupportedContentType(e *restfulServerEnv) {
	const clientUnsupportedContentType = "client-unsupported-content-type"
	errorHandler := func(_ context.Context, w http.ResponseWriter, r *http.Request, e error) {
		w.Header().Set("Content-Type", clientUnsupportedContentType)
		if _, err := fmt.Fprintf(w, `{"ret-code":%d, "ret-msg":"%s"}`, errs.Code(e), errs.Msg(e)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	s.startServer(
		&testRESTfulService{},
		server.WithRESTOptions(restful.WithErrorHandler(errorHandler)),
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	rsp, err := http.Post(
		s.unaryCallCustomURL(),
		"application/json",
		bytes.NewReader(mustMarshalJSON(s.T(), &testpb.SimpleRequest{ResponseType: testpb.PayloadType_RANDOM})),
	)
	require.Nil(s.T(), err)
	require.Equal(s.T(), rsp.Header.Get("Content-Type"), clientUnsupportedContentType)

	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}
}

func (s *TestSuite) TestRESTfulEmptyBody() {
	for _, e := range allRESTfulServerEnv {
		s.Run(e.String(), func() { s.testRESTfulEmptyBody(e) })
	}
}
func (s *TestSuite) testRESTfulEmptyBody(e *restfulServerEnv) {
	s.startServer(
		&testRESTfulService{},
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	rsp, err := http.Post(s.unaryCallCustomURL(), "text", bytes.NewReader([]byte("")))
	require.Nil(s.T(), err)
	require.Equal(s.T(), http.StatusOK, rsp.StatusCode)

	bts, err := io.ReadAll(rsp.Body)
	require.Nil(s.T(), err)
	require.Contains(s.T(), string(bts), `"body":""`)

	if err := rsp.Body.Close(); err != nil {
		s.T().Log(err)
	}
}

func (s *TestSuite) TestRESTfulAccessNonexistentResource() {
	for _, e := range allRESTfulServerEnv {
		s.Run(e.String(), func() { s.testRESTfulAccessNonexistentResource(e) })
	}
}
func (s *TestSuite) testRESTfulAccessNonexistentResource(e *restfulServerEnv) {
	s.startServer(
		&testRESTfulService{},
		server.WithServerAsync(e.async),
		server.WithTransport(thttp.NewRESTServerTransport(e.basedOnFastHTTP)),
	)
	s.T().Cleanup(func() { s.closeServer(nil) })

	tests := []struct {
		name   string
		method string
	}{
		{http.MethodGet, http.MethodGet},
		{http.MethodPost, http.MethodPost},
		{http.MethodHead, http.MethodHead},
		{http.MethodOptions, http.MethodOptions},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, s.unaryCallCustomURL()+"/invalid-path/incorrect.resource", nil)
			require.Nil(t, err)

			rsp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			require.Equal(t, http.StatusNotFound, rsp.StatusCode)
		})
	}
}

func mustMarshalJSON(t *testing.T, any interface{}) []byte {
	t.Helper()

	bts, err := json.Marshal(any)
	require.Nil(t, err)
	return bts
}

func mustUnmarshalProtoJSON(t *testing.T, r io.Reader, m proto.Message) {
	t.Helper()

	bts, err := io.ReadAll(r)
	require.Nil(t, err)
	require.Nil(t, protojson.Unmarshal(bts, m))
}

func mustUnmarshalJSON(t *testing.T, data []byte, any interface{}) {
	t.Helper()

	require.Nil(t, json.Unmarshal(data, any))
}
