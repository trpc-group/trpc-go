package main

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/examples/features/httprpc/proto/echo"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// pbClientProxy invokes.
	c := pb.NewEchoClientProxy()
	pbRsp, err := c.UnaryEcho(trpc.BackgroundContext(), &pb.EchoRequest{Message: "hello"},
		client.WithTarget("ip://127.0.0.1:8091"),
		client.WithProtocol(protocol.FastHTTP))
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("response code: %d, response message: %s", pbRsp.Code, pbRsp.Message)

	// stdhttp invokes.
	stdhttpRsp, err := http.Post("http://127.0.0.1:8091/trpc.examples.echo.Echo/UnaryEcho",
		"application/json", bytes.NewReader([]byte(`{"json_message":"hello"}`)),
	)
	if err != nil {
		log.Error(err)
		return
	}
	bs, err := io.ReadAll(stdhttpRsp.Body)
	if err != nil {
		log.Error(err)
	}
	log.Infof("response: %v", string(bs))

	// fasthttpClient invokes.
	fc := thttp.NewFastHTTPClient("1")
	body := []byte(`{"json_message":"hello"}`)
	fasthttpReq := fasthttp.AcquireRequest()
	fasthttpRsp := fasthttp.AcquireResponse()
	// After invocation, remember to release the req and rsp.
	defer fasthttp.ReleaseRequest(fasthttpReq)
	defer fasthttp.ReleaseResponse(fasthttpRsp)

	fasthttpReq.Header.SetMethod("POST")
	fasthttpReq.Header.SetContentType("application/json")
	fasthttpReq.Header.SetRequestURI("http://127.0.0.1:8091/trpc.examples.echo.Echo/UnaryEcho")
	fasthttpReq.SetBody(body)
	if err = fc.Do(fasthttpReq, fasthttpRsp); err != nil {
		log.Error(err)
	}
	log.Info("response:", string(fasthttpRsp.Body()))

	// fasthttpClientProxy invokes.
	pbRsp = &pb.EchoResponse{}
	fcp := thttp.NewFastHTTPClientProxy("2", client.WithTarget("ip://127.0.0.1:8091"))
	if err = fcp.Post(context.Background(),
		"/trpc.examples.echo.Echo/UnaryEcho",
		&pb.EchoRequest{Message: "hello"}, pbRsp,
	); err != nil {
		log.Error(err)
	}
	log.Infof("response code: %d, response message: %s", pbRsp.Code, pbRsp.Message)
}
