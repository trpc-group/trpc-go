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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
	"trpc.group/trpc-go/trpc-go/transport"
	frouter "github.com/fasthttp/router"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// ------------------------------------- old stub -----------------------------------------//

type GreeterService interface {
	SayHello(ctx context.Context, req *helloworld.HelloRequest, rsp *helloworld.HelloReply) (err error)
}

func GreeterService_SayHello_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (
	rspBody interface{}, err error) {
	req := &helloworld.HelloRequest{}
	rsp := &helloworld.HelloReply{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, reqBody interface{}, rspBody interface{}) error {
		return svr.(GreeterService).SayHello(ctx, reqBody.(*helloworld.HelloRequest), rspBody.(*helloworld.HelloReply))
	}

	err = filters.Handle(ctx, req, rsp, handleFunc)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

var GreeterServer_ServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.examples.restful.helloworld.Greeter",
	HandlerType: (*GreeterService)(nil),
	Methods: []server.Method{
		{
			Name: "/trpc.examples.restful.helloworld.Greeter/SayHello",
			Func: GreeterService_SayHello_Handler,
			Bindings: []*restful.Binding{
				{
					Name:   "/trpc.examples.restful.helloworld.Greeter/SayHello",
					Input:  func() restful.ProtoMessage { return new(helloworld.HelloRequest) },
					Output: func() restful.ProtoMessage { return new(helloworld.HelloReply) },
					Handler: func(svc interface{}, ctx context.Context, reqBody, respbody interface{}) error {
						return svc.(GreeterService).SayHello(ctx, reqBody.(*helloworld.HelloRequest), respbody.(*helloworld.HelloReply))
					},
					HTTPMethod:   http.MethodGet,
					Pattern:      restful.Enforce("/v2/bar/{name}"),
					Body:         nil,
					ResponseBody: nil,
				},
			},
		},
	},
}

func RegisterGreeterService(s server.Service, svr GreeterService) {
	if err := s.Register(&GreeterServer_ServiceDesc, svr); err != nil {
		panic(fmt.Sprintf("Greeter register error: %v", err))
	}
}

// ------------------------------------------------------------------------------------------//

type greeter struct{}

func (s *greeter) SayHello(ctx context.Context, req *helloworld.HelloRequest, rsp *helloworld.HelloReply) error {
	rsp.Message = req.Name
	return nil
}

func TestPreviousVersionStub(t *testing.T) {
	var serverFilter filter.ServerFilter = func(ctx context.Context, req interface{},
		next filter.ServerHandleFunc) (rsp interface{}, err error) {
		helloReq, ok := req.(*helloworld.HelloRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}
		if helloReq.Name != "world" {
			return nil, errors.New("wrong name")
		}
		resp, err := next(ctx, req)
		if err != nil {
			return nil, err
		}
		helloResp, ok := resp.(*helloworld.HelloReply)
		if !ok {
			return nil, errors.New("invalid response")
		}
		helloResp.Message += "a"
		return helloResp, nil
	}
	filter.Register("restful.oldversion.stub", serverFilter, nil)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	// service registration
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.test.helloworld.GreeterPreviousVersionStub"),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithFilter(filter.GetServer("restful.oldversion.stub")),
	)
	s.AddService("trpc.test.helloworld.GreeterPreviousVersionStub", service)
	RegisterGreeterService(s, &greeter{})

	// start server
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// create restful request
	req, err := http.NewRequest(http.MethodGet, addr+"/v2/bar/world", nil)
	require.Nil(t, err)

	// send restful request
	cli := http.Client{}
	resp1, err := cli.Do(req)
	require.Nil(t, err)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusOK, resp1.StatusCode)
	bodyBytes1, err := io.ReadAll(resp1.Body)
	require.Nil(t, err)
	type responseBody struct {
		Message string `json:"message"`
	}
	respBody := &responseBody{}
	json.Unmarshal(bodyBytes1, respBody)
	require.Equal(t, "worlda", respBody.Message)

	resp2, err := cli.Do(req)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, resp2.StatusCode, http.StatusOK)
	bodyBytes2, err := io.ReadAll(resp2.Body)
	require.Nil(t, err)
	json.Unmarshal(bodyBytes2, respBody)
	require.Equal(t, "worlda", respBody.Message)
}

func TestTRPCGlobalMessage(t *testing.T) {
	cfgPath := t.TempDir() + "/cfg.yaml"
	require.Nil(t, os.WriteFile(cfgPath, []byte(`
global:
  namespace: development
  env_name: environment
  container_name: container
  enable_set: Y
  full_set_name: full.set.name
server:
  service:
    - name: trpc.test.helloworld.Greeter
      protocol: restful
`), 0644))
	trpc.ServerConfigPath = cfgPath

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer l.Close()

	s := trpc.NewServer(server.WithRESTOptions(
		restful.WithFilterFunc(func() filter.ServerChain {
			return []filter.ServerFilter{
				func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
					msg := trpc.Message(ctx)
					require.Equal(t, "development", msg.Namespace())
					require.Equal(t, "environment", msg.EnvName())
					require.Equal(t, "container", msg.CalleeContainerName())
					require.Equal(t, "full.set.name", msg.SetName())
					return next(ctx, req)
				},
			}
		})),
		server.WithListener(l))
	RegisterGreeterService(s, &greeter{})
	go func() {
		fmt.Println(s.Serve())
	}()

	rsp, err := http.Get(fmt.Sprintf("http://%s/v2/bar/world", l.Addr().String()))
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHTTPOkWithDetailedError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer l.Close()
	s := server.New(
		server.WithListener(l),
		server.WithServiceName("trpc.test.helloworld.Greeter2"),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithRESTOptions(
			restful.WithErrorHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
				restful.DefaultErrorHandler(ctx, w, r, &restful.WithStatusCode{StatusCode: http.StatusOK, Err: err})
			})),
		server.WithFilter(func(
			ctx context.Context,
			req interface{},
			next filter.ServerHandleFunc,
		) (rsp interface{}, err error) {
			return nil, errs.New(errs.RetServerThrottled, "always throttled")
		}))
	RegisterGreeterService(s, &greeter{})
	go func() {
		fmt.Println(s.Serve())
	}()

	rsp, err := http.Get(fmt.Sprintf("http://%s/v2/bar/world", l.Addr().String()))
	require.Nil(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
	rspBody, err := io.ReadAll(rsp.Body)
	require.Nil(t, err)
	require.Contains(t, string(rspBody), strconv.Itoa(errs.RetServerThrottled))
	require.NotContains(t, string(rspBody), strconv.Itoa(errs.RetUnknown))
	require.Contains(t, string(rspBody), "always throttled")
}

func TestNoPanicOnFilterReturnsNil(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer l.Close()
	s := server.New(
		server.WithListener(l),
		server.WithServiceName("trpc.test.helloworld.Greeter3"),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithFilter(func(
			ctx context.Context, req interface{}, next filter.ServerHandleFunc,
		) (rsp interface{}, err error) {
			head := ctx.Value(thttp.ContextKeyHeader).(*thttp.Header)
			head.Response.Header().Add(t.Name(), t.Name())
			return nil, nil
		}))
	RegisterGreeterService(s, &greeter{})
	go func() {
		fmt.Println(s.Serve())
	}()

	rsp, err := http.Get(fmt.Sprintf("http://%s/v2/bar/world", l.Addr().String()))
	require.Nil(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
	require.Equal(t, t.Name(), rsp.Header.Get(t.Name()))
}

func TestTimeout(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:")
	require.Nil(t, err)
	defer l.Close()
	s := server.New(
		server.WithListener(l),
		server.WithServiceName(t.Name()),
		server.WithProtocol("restful"),
		server.WithTimeout(time.Second))
	RegisterGreeterService(s, &greeterAlwaysTimeout{})
	errCh := make(chan error)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(time.Millisecond * 100):
	}
	defer s.Close(nil)

	start := time.Now()
	rsp, err := http.Get(fmt.Sprintf("http://%s/v2/bar/world", l.Addr().String()))
	require.Nil(t, err)
	require.Equal(t, http.StatusGatewayTimeout, rsp.StatusCode)
	require.InDelta(t, time.Second, time.Since(start), float64(time.Millisecond*100))
}

func TestRegisterRouterAddAdditionalPatternUsingServerMux(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.Nil(t, err)
	defer l.Close()
	serviceName := t.Name()
	s := server.New(
		server.WithListener(l),
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"))
	RegisterGreeterService(s, &greeter{})

	// 1. Get the old stdhttp router.
	r := restful.GetRouter(serviceName)
	// 2. Create a new stdhttp router.
	mux := http.NewServeMux()
	// 3. Pass the old stdhttp router as the "/*" for the new fasthttp router.
	mux.Handle("/", r)
	// 4. Register an additional pattern to the new stdhttp router.
	additionalPattern := "/path"
	dataForAdditionalPattern := []byte("data")
	mux.Handle(additionalPattern, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(dataForAdditionalPattern)
	}))
	// 5. Register the new stdhttp router to replace the original one.
	restful.RegisterRouter(serviceName, mux)

	errCh := make(chan error)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(time.Millisecond * 200):
	}
	defer s.Close(nil)

	req := "world"
	rsp, err := http.Get(fmt.Sprintf("http://%s/v2/bar/%s", l.Addr().String(), req))
	require.Nil(t, err)
	got, err := io.ReadAll(rsp.Body)
	require.Nil(t, err)
	require.Equal(t, "{\"message\":\"world\"}", string(got))

	rsp, err = http.Get(fmt.Sprintf("http://%s%s", l.Addr().String(), additionalPattern))
	require.Nil(t, err)
	got, err = io.ReadAll(rsp.Body)
	require.Nil(t, err)
	require.Equal(t, string(dataForAdditionalPattern), string(got))
}

func TestRegisterFasthttpRouterAddAdditionalPatternUsingServerMux(t *testing.T) {
	restfulProtocolBasedOnFasthttp := "restful_based_on_fasthttp"
	transport.RegisterServerTransport(restfulProtocolBasedOnFasthttp,
		thttp.NewRESTServerTransport(true))

	l, err := net.Listen("tcp", "127.0.0.1:")
	require.Nil(t, err)
	defer l.Close()
	serviceName := t.Name()
	s := server.New(
		server.WithListener(l),
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol(restfulProtocolBasedOnFasthttp))
	RegisterGreeterService(s, &greeter{})

	// 1. Get the old fasthttp router.
	r := restful.GetFasthttpRouter(serviceName)
	// 2. Create a new fasthttp router.
	fr := frouter.New()
	// 3. Pass the old fasthttp router as the "/*" for the new fasthttp router.
	fr.Handle(frouter.MethodWild, "/{filepath:*}", r)
	// 4. Register an additional pattern to the new fasthttp router.
	additionalPattern := "/path"
	dataForAdditionalPattern := []byte("data")
	fr.Handle(http.MethodGet, additionalPattern, func(ctx *fasthttp.RequestCtx) {
		ctx.Response.BodyWriter().Write(dataForAdditionalPattern)
	})
	// 5. Register the new fasthttp router to replace the original one.
	restful.RegisterFasthttpRouter(serviceName, fr.Handler)

	errCh := make(chan error)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(time.Millisecond * 200):
	}
	defer s.Close(nil)

	req := "world"
	rsp, err := http.Get(fmt.Sprintf("http://%s/v2/bar/%s", l.Addr().String(), req))
	require.Nil(t, err)
	got, err := io.ReadAll(rsp.Body)
	require.Nil(t, err)
	require.Equal(t, "{\"message\":\"world\"}", string(got))

	rsp, err = http.Get(fmt.Sprintf("http://%s%s", l.Addr().String(), additionalPattern))
	require.Nil(t, err)
	got, err = io.ReadAll(rsp.Body)
	require.Nil(t, err)
	require.Equal(t, string(dataForAdditionalPattern), string(got))
}

func TestMethodTimeout(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.Nil(t, err)
	defer l.Close()
	s := server.New(
		server.WithListener(l),
		server.WithServiceName(t.Name()),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithTimeout(time.Millisecond*100),
		server.WithMethodTimeout("/trpc.examples.restful.helloworld.Greeter/SayHello", time.Millisecond*200))
	RegisterGreeterService(s, &greeterAlwaysTimeout{})
	errCh := make(chan error)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(time.Millisecond * 200):
	}
	defer s.Close(nil)

	start := time.Now()
	rsp, err := http.Get(fmt.Sprintf("http://%s/v2/bar/world", l.Addr().String()))
	require.Nil(t, err)
	require.Equal(t, http.StatusGatewayTimeout, rsp.StatusCode)
	require.InDelta(t, time.Millisecond*200, time.Since(start), float64(time.Millisecond*30))
}

type greeterAlwaysTimeout struct{}

func (*greeterAlwaysTimeout) SayHello(ctx context.Context, req *helloworld.HelloRequest, rsp *helloworld.HelloReply) error {
	<-ctx.Done()
	return errs.NewFrameError(errs.RetServerTimeout, "ctx timeout")
}

func TestRegisterRouter(t *testing.T) {
	t.Run("router not registered", func(t *testing.T) {
		require.NotPanics(t, func() {
			restful.MustRegisterRouter("testRegisterRouter", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		})
	})
	t.Run("router already registered", func(t *testing.T) {
		require.Panics(t, func() {
			restful.MustRegisterRouter("testRegisterRouter", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		})
	})
}

func TestRegisterFasthttpRouter(t *testing.T) {
	t.Run("router not registered", func(t *testing.T) {
		require.NotPanics(t, func() {
			restful.MustRegisterFasthttpRouter("testRegisterRouter", func(ctx *fasthttp.RequestCtx) {})
		})
	})
	t.Run("router already registered", func(t *testing.T) {
		require.Panics(t, func() {
			restful.MustRegisterFasthttpRouter("testRegisterRouter", func(ctx *fasthttp.RequestCtx) {})
		})
	})
}

func TestPBSerializerGetter(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer l.Close()
	s := server.New(
		server.WithListener(l),
		server.WithServiceName("trpc.test.helloworld.Greeter4"),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithFilter(func(
			ctx context.Context, req interface{}, next filter.ServerHandleFunc,
		) (rsp interface{}, err error) {
			msg := trpc.Message(ctx)
			msg.WithSerializationType(codec.SerializationTypePB)
			return
		}),
		server.WithRESTOptions(
			restful.WithRespSerializerGetter(
				func(ctx context.Context, r *http.Request) restful.Serializer {
					// Users need to maintain the mapping between
					// msg.SerializationType() and the corresponding serializer.Name().
					// GetSerializer returns the serializer using serializer.Name().
					var serializationTypeContentType = map[int]string{
						// These values are all correct.
						codec.SerializationTypePB: "application/octet-stream",
						// codec.SerializationTypeJSON: "application/json",
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
						s = restful.DefaultRespSerializerGetter(ctx, r)
						log.Warnf("the serializer %s not found, get the serializer %s by default",
							serializationTypeContentType[st], s.Name())
					}
					return s
				},
			),
		),
	)
	RegisterGreeterService(s, &greeter{})
	go func() {
		fmt.Println(s.Serve())
	}()

	rsp0, err := http.Get(fmt.Sprintf("http://%s/v2/bar/world", l.Addr()))
	require.Nil(t, err)
	defer rsp0.Body.Close()
	require.Equal(t, 1, len(rsp0.Header["Content-Type"]))
	require.Equal(t, "application/octet-stream", rsp0.Header["Content-Type"][0])

	// When an error occurs, the process will directly go through the ErrorHandler without
	// passing through the RespSerializerGetter. Therefore, the serialization format
	// will default to application/json.
	// Note: The "default" here refers to the default serialization format,
	// whereas the "default" mentioned earlier refers to the zero value of SerializationType,
	// which is codec.SerializationTypePB.
	rsp1, err := http.Get(fmt.Sprintf("http://%s/NONEXIST", l.Addr()))
	require.Nil(t, err)
	defer rsp1.Body.Close()
	require.Equal(t, 1, len(rsp1.Header["Content-Type"]))
	require.Equal(t, "application/json", rsp1.Header["Content-Type"][0])
}
