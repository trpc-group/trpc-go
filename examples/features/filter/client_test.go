package main

import (
	"context"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"

	_ "trpc.group/trpc-go/trpc-go"

	"github.com/stretchr/testify/assert"
)

var addr = "ip://127.0.0.1:8000"

func clientFilter(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
	msg := codec.Message(ctx)
	md := msg.ClientMetaData()
	if md == nil {
		md = codec.MetaData{}
	}
	md["test_filter"] = []byte("ok")
	msg.WithClientMetaData(md)
	return f(ctx, req, rsp)
}

func TestSayHello(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithTarget(addr),
		client.WithFilter(clientFilter),
	}

	clientProxy := helloworld.NewGreeterClientProxy(opts...)

	req := &helloworld.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHello(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}

func TestSayHi(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithTarget(addr),
	}

	clientProxy := helloworld.NewGreeterClientProxy(opts...)

	req := &helloworld.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHi(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}
