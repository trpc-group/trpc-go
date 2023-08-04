package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var addr = "ip://127.0.0.1:8000"

func clientFilter(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
	msg := codec.Message(ctx)
	md := msg.ClientMetaData()
	if md == nil {
		md = codec.MetaData{}
	}
	md["timestamp2"] = []byte(time.Now().String())
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

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
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
		client.WithMetaData("timestamp1", []byte(time.Now().String())),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := clientProxy.SayHi(ctx, req)
	t.Log(rsp, err)
	assert.NotNil(t, err)
}
