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

// Package main is the main package.
package main

import (
	"context"
	"crypto/rand"
	"sync"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	pb "trpc.group/trpc-go/trpc-go/examples/features/rspobsoleted/proto"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
)

func main() {
	// Initialize sync.Pool.
	p := newPool()
	// Create a new trpc server.
	// Provide a server option to set the OnResponseObsoleted handler,
	// which will be called by the framework after each response is no longer
	// in use (typically after marshalling it into bytes).
	s := trpc.NewServer(server.WithOnResponseObsoleted(func(ctx context.Context, rsp interface{}) {
		p.putRsp(rsp.(*pb.Response))
		p.releaseResourceFromContext(ctx, resourceKey)
	}))

	// Register the current implementation into the service object.
	pb.RegisterRspObsoletedExampleService(s.Service("trpc.examples.rspobsoleted.RspObsoletedExample"), &rspObsoletedImpl{
		pool: p,
	})

	// Start the service and block here.
	if err := s.Serve(); err != nil {
		log.Fatalf("service serves error: %v", err)
	}
}

type pool struct {
	rspPool   *sync.Pool
	bytesPool *sync.Pool
}

func newPool() *pool {
	const defaultSize = 4096
	return &pool{
		rspPool: &sync.Pool{
			New: func() interface{} {
				return &pb.Response{}
			},
		},
		bytesPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, defaultSize)
			},
		},
	}
}

func (p *pool) getRsp() *pb.Response {
	return p.rspPool.Get().(*pb.Response)
}

func (p *pool) getBytes() []byte {
	return p.bytesPool.Get().([]byte)
}

func (p *pool) putRsp(rsp *pb.Response) {
	rsp.Msg = nil
	p.rspPool.Put(rsp)
}

func (p *pool) putBytes(bs []byte) {
	p.bytesPool.Put(bs)
}

const resourceKey = "resource"

func (p *pool) releaseResourceFromContext(ctx context.Context, key interface{}) {
	if bs, ok := codec.Message(ctx).CommonMeta()[key].([]byte); ok {
		p.putBytes(bs)
	} else {
		panic("not exist")
	}
}

func (p *pool) addResourceToContext(ctx context.Context, key, resource interface{}) {
	codec.Message(ctx).CommonMeta()[key] = resource
}

type rspObsoletedImpl struct {
	pool *pool
}

func (impl *rspObsoletedImpl) Hello(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	rsp := impl.pool.getRsp()
	bs := impl.pool.getBytes()
	// Below are some simulated operations that mimic real-world scenarios.
	bs = append(bs, req.Msg...)
	const randSize = 8
	bs = bs[len(bs) : len(bs)+randSize]
	n, err := rand.Read(bs[len(bs) : len(bs)+randSize])
	if err != nil {
		return nil, err
	}
	const offset = 4
	// Suppose rsp.Msg captures only a portion of the bytes retrieved from the pool,
	// which can happen in the real world if the user utilizes a special unmarshal method
	// that reuses the provided bytes to avoid allocation and copying.
	rsp.Msg = bs[offset : len(bs)+n]
	// Therefore the bs retrieved from the pool needs to be put back
	// inside the OnResponseObsoleted handler.
	impl.pool.addResourceToContext(ctx, resourceKey, bs)
	return rsp, nil
}
