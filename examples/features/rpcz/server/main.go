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

// Package main this file is RPCZ server samples.
package main

import (
	"context"
	"flag"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	pb "trpc.group/trpc-go/trpc-go/examples/features/rpcz/proto"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
)

const attributeName = "SpecialAttribute"

var rpczType = flag.String("type", "Basic", "Input RPCZ Type")

func main() {
	// Create a service object, this step involves reading the configuration file
	s := trpc.NewServer()

	flag.Parse()

	if *rpczType == "" {
		// After reading the configuration file and before the service starts,
		// you can flexibly set RPCZ by configuring rpcz.GlobalRPCZ through the code.
		// At this time, the commit sampling logic requires implementing the ShouldRecord function.
		rpcz.NewRPCZ(&rpcz.Config{
			Fraction: 1.0,  // set the span fraction for RPCZ
			Capacity: 1000, // set the span capacity for RPCZ
			ShouldRecord: func(s rpcz.Span) bool {
				_, ok := s.Attribute(attributeName) // Only submit spans with the "SpecialAttribute" attribute.
				return ok
			},
			Exporter: nil,
		})
	}

	pb.RegisterRPCZService(s, &testRPCZAttributeImpl{})

	// service starts
	s.Serve()
}

type testRPCZAttributeImpl struct {
}

func (t *testRPCZAttributeImpl) Hello(ctx context.Context, req *pb.HelloReq) (*pb.HelloRsp, error) {
	switch *rpczType {
	case "Basic":
		return t.basicResult(ctx, req)
	case "Advanced":
		return t.advancedResult(ctx, req)
	case "Code":
		return t.codeResult(ctx, req)
	default:
		return nil, errs.New(111, "unknow rpcz type")
	}

}

func (t *testRPCZAttributeImpl) basicResult(ctx context.Context, req *pb.HelloReq) (*pb.HelloRsp, error) {
	rsp := &pb.HelloRsp{}
	log.Debugf("recv req:%s", req)
	rsp.Msg = "Hello " + req.GetMsg()
	return rsp, nil
}

func (t *testRPCZAttributeImpl) advancedResult(ctx context.Context, req *pb.HelloReq) (*pb.HelloRsp, error) {
	rsp := &pb.HelloRsp{}

	return rsp, errs.New(21, "21 error msg")
}

func (t *testRPCZAttributeImpl) codeResult(ctx context.Context, req *pb.HelloReq) (*pb.HelloRsp, error) {
	rsp := &pb.HelloRsp{}

	span := rpcz.SpanFromContext(ctx)
	span.SetAttribute(attributeName, 1)

	log.Debugf("recv req:%s", req)
	rsp.Msg = "Hello attribute rpcz: " + req.GetMsg()
	return rsp, nil
}
