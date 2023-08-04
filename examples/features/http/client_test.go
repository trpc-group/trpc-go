package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var addr = "ip://127.0.0.1:8080"

func TestGreeterSayHello(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithTarget(addr),
		client.WithProtocol("http"),
		client.WithTLS("../../../testdata/client.crt", "../../../testdata/client.key", "none", ""),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHello(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}

func TestHttp2GreeterSayHello(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithTarget(addr),
		client.WithProtocol("http2"),
		client.WithTLS("../../../testdata/client.crt", "../../../testdata/client.key", "none", ""),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHello(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}

func TestGreeterSayHi(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithTarget(addr),
		client.WithProtocol("http"),
		client.WithTLS("../../../testdata/client.crt", "../../../testdata/client.key", "none", ""),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHi(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}
