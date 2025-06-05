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

package restful_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	hpb "trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
	"trpc.group/trpc-go/trpc-go/transport"
)

// greeterService is the helloworld service impl.
type greeterService struct{}

func (s *greeterService) SayHello(ctx context.Context, req *hpb.HelloRequest) (*hpb.HelloReply, error) {
	rsp := &hpb.HelloReply{}
	if req.Name != "xyz" {
		return nil, errors.New("test error")
	}
	rsp.Message = "test"
	return rsp, nil
}

func TestBasedOnFastHTTP(t *testing.T) {
	transport.RegisterServerTransport("restful_based_on_fasthttp",
		thttp.NewRestServerFastHTTPTransport(func() *fasthttp.Server {
			return &fasthttp.Server{}
		}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	// service registration
	s := &server.Server{}
	service := server.New(server.WithListener(ln),
		server.WithServiceName("trpc.test.helloworld.FastHTTP"+t.Name()),
		server.WithProtocol("restful_based_on_fasthttp"),
		server.WithRESTOptions(
			restful.WithFastHTTPHeaderMatcher(
				func(ctx context.Context, requestCtx *fasthttp.RequestCtx, serviceName string,
					methodName string) (context.Context, error) {
					return context.Background(), nil
				},
			),
			restful.WithFastHTTPRespHandler(
				func(
					ctx context.Context,
					requestCtx *fasthttp.RequestCtx,
					resp proto.Message,
					body []byte,
				) error {
					if string(requestCtx.Request.Header.Peek("Accept-Encoding")) != "gzip" {
						return errors.New("test error")
					}
					writeCloser, err := (&restful.GZIPCompressor{}).
						Compress(requestCtx.Response.BodyWriter())
					if err != nil {
						return err
					}
					defer writeCloser.Close()
					requestCtx.Response.Header.Set("Content-Encoding", "gzip")
					requestCtx.Response.Header.Set("Content-Type", "application/json")
					writeCloser.Write(body)
					return nil
				},
			),
		),
	)
	s.AddService("trpc.test.helloworld.FastHTTP"+t.Name(), service)
	hpb.RegisterGreeterService(s, &greeterService{})

	// start server
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()
	time.Sleep(100 * time.Millisecond)

	t.Run("send restful request ok", func(t *testing.T) {
		// create restful request
		data := `{"name": "xyz"}`
		buf := bytes.Buffer{}
		gBuf := gzip.NewWriter(&buf)
		_, err = gBuf.Write([]byte(data))
		require.Nil(t, err)
		gBuf.Close()
		req, err := http.NewRequest(http.MethodPost, addr+"/v1/foobar", &buf)
		require.Nil(t, err)
		req.Header.Add("Content-Type", "anything")
		req.Header.Add("Content-Encoding", "gzip")
		req.Header.Add("Accept-Encoding", "gzip")

		cli := http.Client{}
		resp, err := cli.Do(req)
		require.Nil(t, err)
		defer resp.Body.Close()
		require.Equal(t, resp.StatusCode, http.StatusOK)
		reader, err := gzip.NewReader(resp.Body)
		require.Nil(t, err)
		bodyBytes, err := io.ReadAll(reader)
		require.Nil(t, err)
		type responseBody struct {
			Message string `json:"message"`
		}
		respBody := &responseBody{}
		json.Unmarshal(bodyBytes, respBody)
		require.Equal(t, respBody.Message, "test")
	})
	t.Run("matching all by query params", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, addr+"/v2/bar?name=xyz", nil)
		require.Nil(t, err)
		rsp, err := http.DefaultClient.Do(req)
		require.Nil(t, err)
		defer rsp.Body.Close()
		require.Equal(t, rsp.StatusCode, http.StatusOK)
	})
	t.Run("matching request URL.Path failed", func(t *testing.T) {
		rsp, _ := http.Get(addr + "/v2/unknown")
		require.Equal(t, http.StatusNotFound, rsp.StatusCode)
		bts, err := io.ReadAll(rsp.Body)
		require.Nil(t, err)
		defer rsp.Body.Close()
		require.Contains(t, string(bts), "failed to match any pattern")
	})

	t.Run("transcoding request failed", func(t *testing.T) {
		rsp, _ := http.Get(addr + "/v3/qux/id")
		require.Equal(t, http.StatusBadRequest, rsp.StatusCode)
		bts, err := io.ReadAll(rsp.Body)
		require.Nil(t, err)
		defer rsp.Body.Close()
		require.Contains(t, string(bts), "transcoding request failed")
		// test response content-type
		require.Equal(t, rsp.Header.Get("Content-Type"), "application/json")
	})
	t.Run("server error", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, addr+"/v2/bar?name=anything", nil)
		require.Nil(t, err)
		rsp, err := http.DefaultClient.Do(req)
		require.Nil(t, err)
		defer rsp.Body.Close()
		require.Equal(t, rsp.StatusCode, http.StatusInternalServerError)
	})
	t.Run("err handler", func(t *testing.T) {
		data := `{"name": "abc"}`
		buf := bytes.Buffer{}
		gBuf := gzip.NewWriter(&buf)
		_, err = gBuf.Write([]byte(data))
		require.Nil(t, err)
		gBuf.Close()
		req, err := http.NewRequest(http.MethodPost, addr+"/v1/foobar", &buf)
		require.Nil(t, err)
		req.Header.Add("Content-Type", "anything")
		req.Header.Add("Content-Encoding", "gzip")
		req.Header.Add("Accept-Encoding", "gzip")
		c := http.Client{}
		rsp, err := c.Do(req)
		require.Nil(t, err)
		defer rsp.Body.Close()
		require.Equal(t, rsp.StatusCode, http.StatusInternalServerError)
		bts, err := io.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.Contains(t, string(bts), "test error")
	})
}

func TestFastHTTPPBSerialzerGetter(t *testing.T) {
	transport.RegisterServerTransport("restful_based_on_fasthttp",
		thttp.NewRestServerFastHTTPTransport(func() *fasthttp.Server {
			return &fasthttp.Server{}
		}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	addr := fmt.Sprintf("http://%s", ln.Addr())
	// service registration
	s := &server.Server{}
	service := server.New(server.WithListener(ln),
		server.WithServiceName("trpc.test.helloworld.FastHTTP"+t.Name()),
		server.WithProtocol("restful_based_on_fasthttp"),
		server.WithFilter(func(
			ctx context.Context, req interface{}, next filter.ServerHandleFunc,
		) (rsp interface{}, err error) {
			msg := trpc.Message(ctx)
			msg.WithSerializationType(codec.SerializationTypePB)
			return
		}),
		server.WithRESTOptions(
			restful.WithFastHTTPRespSerializerGetter(
				func(ctx context.Context, requestCtx *fasthttp.RequestCtx) restful.Serializer {
					// Users need to maintain the mapping between
					// msg.SerializationType() and the corresponding serializer.Name().
					// GetSerializer returns the serializer using serializer.Name().
					var serializationTypeContentType = map[int]string{
						// These values are all correct.
						codec.SerializationTypePB:   "application/octet-stream",
						codec.SerializationTypeJSON: "application/json",
						// codec.SerializationTypePB: ""application/protobuf",
						// codec.SerializationTypePB: "application/x-protobuf",
						// codec.SerializationTypePB: "application/pb",
						// codec.SerializationTypePB: "application/proto",
					}

					// Get serializer
					// Note: If users specify the response serializer using msg.SerializationType(),
					// the following behavior will occur:
					// Since the value of codec.SerializationTypePB is 0,
					// when the user does not set the SerializationType,
					// the &ProtoSerializer{} will be chosen as the default serializer.
					msg := trpc.Message(ctx)
					st := msg.SerializationType()
					s := restful.GetSerializer(serializationTypeContentType[st])

					// Note: When a serializer is not obtained,
					// it is recommended to use DefaultRespSerializerGetter as a fallback.
					// In most cases, the failure to obtain a serializer is due to the user not having registered the serializer.
					if s == nil {
						s = restful.DefaultFastHTTPRespSerializerGetter(ctx, requestCtx)
						log.Warnf("the serializer %s not found, get the serializer %s by default",
							serializationTypeContentType[st], s.Name())
					}
					return s
				},
			),
		),
	)
	s.AddService("trpc.test.helloworld.FastHTTP"+t.Name(), service)
	hpb.RegisterGreeterService(s, &greeterService{})

	// start server
	go func() {
		require.Nil(t, s.Serve())
	}()
	time.Sleep(100 * time.Millisecond)

	req0, err := http.NewRequest(http.MethodGet, addr+"/v2/bar?name=xyz", nil)
	require.Nil(t, err)
	rsp0, err := http.DefaultClient.Do(req0)
	require.Nil(t, err)
	defer rsp0.Body.Close()
	require.Equal(t, 1, len(rsp0.Header["Content-Type"]))
	require.Equal(t, "application/octet-stream", rsp0.Header["Content-Type"][0])

	// When an error occurs, the process will directly go through the FastHTTPErrorHandler without
	// passing through the FastHTTPSerializerGetter. Therefore, the serialization format will
	// default to application/json.
	// Note: The "default" here refers to the default serialization format,
	// whereas the "default" mentioned earlier refers to the zero value of SerializationType,
	// which is codec.SerializationTypePB.
	req1, err := http.NewRequest(http.MethodGet, addr+"/NONEXIST", nil)
	require.Nil(t, err)
	rsp1, err := http.DefaultClient.Do(req1)
	require.Nil(t, err)
	defer rsp1.Body.Close()
	require.Equal(t, 1, len(rsp1.Header["Content-Type"]))
	require.Equal(t, "application/json", rsp1.Header["Content-Type"][0])
}
