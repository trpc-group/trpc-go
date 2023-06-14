// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

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

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
)

// ------------------------------------- old stub -----------------------------------------//

type GreeterService interface {
	SayHello(ctx context.Context, req *helloworld.HelloRequest) (rsp *helloworld.HelloReply, err error)
}

func GreeterService_SayHello_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (
	rspBody interface{}, err error) {
	req := &helloworld.HelloRequest{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, reqbody interface{}) (rspbody interface{}, err error) {
		return svr.(GreeterService).SayHello(ctx, reqbody.(*helloworld.HelloRequest))
	}

	rsp, err := filters.Filter(ctx, req, handleFunc)
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
					Filter: func(svc interface{}, ctx context.Context, reqBody interface{}) (interface{}, error) {
						return svc.(GreeterService).SayHello(ctx, reqBody.(*helloworld.HelloRequest))
					},
					HTTPMethod:   "GET",
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
		panic(fmt.Sprintf("Greeter register error:%v", err))
	}
}

// ------------------------------------------------------------------------------------------//

type greeter struct{}

func (s *greeter) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	rsp := &helloworld.HelloReply{}
	rsp.Message = req.Name
	return rsp, nil
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

	// service registration
	s := &server.Server{}
	service := server.New(
		server.WithAddress("127.0.0.1:32781"),
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
	req, err := http.NewRequest("GET", "http://127.0.0.1:32781/v2/bar/world", nil)
	require.Nil(t, err)

	// send restful request
	cli := http.Client{}
	resp1, err := cli.Do(req)
	require.Nil(t, err)
	defer resp1.Body.Close()
	require.Equal(t, resp1.StatusCode, http.StatusOK)
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
	require.Contains(t, string(rspBody), strconv.Itoa(int(errs.RetServerThrottled)))
	require.NotContains(t, string(rspBody), strconv.Itoa(int(errs.RetUnknown)))
	require.Contains(t, string(rspBody), "always throttled")
}

func TestNoPanicOnFilterReturnsNil(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
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
	s := server.New(
		server.WithListener(l),
		server.WithServiceName(t.Name()),
		server.WithProtocol("restful"),
		server.WithTimeout(time.Millisecond*100))
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
	require.InDelta(t, time.Millisecond*100, time.Since(start), float64(time.Millisecond*30))
}

type greeterAlwaysTimeout struct{}

func (*greeterAlwaysTimeout) SayHello(ctx context.Context, req *helloworld.HelloRequest, rsp *helloworld.HelloReply) error {
	<-ctx.Done()
	return errs.NewFrameError(errs.RetServerTimeout, "ctx timeout")
}
