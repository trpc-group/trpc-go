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

package main

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata"
)

var (
	metaKey = "key"
	metaVal = []byte("value")
)

func init() {
	filter.Register("local_filter", filter.ServerFilter(
		func(
			ctx context.Context, req interface{}, next filter.ServerHandleFunc,
		) (rsp interface{}, err error) {
			log.Info("inside local filter")
			rsp, err = next(ctx, req)
			return
		}), nil)
	filter.Register("global_filter", filter.ServerFilter(
		func(
			ctx context.Context, req interface{}, next filter.ServerHandleFunc,
		) (rsp interface{}, err error) {
			log.Info("inside global filter")
			rsp, err = next(ctx, req)
			return
		}), nil)
}

func main() {
	s := trpc.NewServer()
	pb.RegisterGreeterService(s, &testServer{})
	go func() {
		time.Sleep(3 * time.Second)
		// The current example uses trpc_go.yaml to control the scope of the client.
		// Users can modify the client.service.scope to "local" or "remote" to see
		// performance comparisons.
		// It is also possible to use client options, such as
		//  pb.NewClientProxy(client.WithScope("local"))
		// or
		//  pb.NewClientProxy(client.WithScope("remote"))
		// or
		//  pb.NewClientProxy(client.WithScope("all"))
		// to switch between different scope to use.
		p := pb.NewGreeterClientProxy(client.WithFilter(
			func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
				msg := codec.Message(ctx)
				var m codec.MetaData
				m = msg.ClientMetaData()
				if m == nil {
					m = codec.MetaData{}
				}
				m[metaKey] = metaVal
				msg.WithClientMetaData(m)
				return next(ctx, req, rsp)
			}))
		ctx := trpc.BackgroundContext()
		tot := 300000
		start := time.Now()
		for i := 0; i < tot; i++ {
			// During calling, it is also possible to specify the client scope
			// by adding client.WithScope("local") options.
			_, err := p.SayHello(ctx, &pb.HelloRequest{
				// Use a large message to illustrate the performance boost by reducing the cost of serialization.
				Msg: `Four score and seven years ago our fathers brought forth on this continent, a new nation, 
conceived in Liberty, and dedicated to the proposition that all men are created equal.
Now we are engaged in a great civil war, testing whether that nation, or any nation so conceived and so dedicated, 
can long endure. We are met on a great battle-field of that war. We have come to dedicate a portion of that field, 
as a final resting place for those who here gave their lives that that nation might live. 
It is altogether fitting and proper that we should do this.
But, in a larger sense, we can not dedicate -- we can not consecrate -- we can not hallow -- this ground. 
The brave men, living and dead, who struggled here, have consecrated it, far above our poor power to add or detract. 
The world will little note, nor long remember what we say here, but it can never forget what they did here. 
It is for us the living, rather, to be dedicated here to the unfinished work which they who fought here have thus far 
so nobly advanced. It is rather for us to be here dedicated to the great task remaining before us -- that from these 
honored dead we take increased devotion to that cause for which they gave the last full measure of devotion -- that 
we here highly resolve that these dead shall not have died in vain -- that this nation, under God, shall have a new 
birth of freedom -- and that government of the people, by the people, for the people, shall not perish from the earth.
`,
			})
			if err != nil {
				log.Errorf("got error: %+v", err)
			}
		}
		elapsed := time.Since(start)
		log.Infof("QPS: %d, average cost: %.2fms", int(float64(tot)/elapsed.Seconds()),
			1000*elapsed.Seconds()/float64(tot))
	}()
	s.Serve()
}

type testServer struct{}

func (s *testServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	msg := codec.Message(ctx)
	m := msg.ServerMetaData()
	v, ok := m[metaKey]
	if ok {
		log.Infof("meta key %v exists, the value is %q", metaKey, v)
	} else {
		log.Infof("meta key %v does not exist", metaKey)
	}
	return &pb.HelloReply{Msg: req.Msg}, nil
}

func (s *testServer) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Msg: req.Msg}, nil
}
