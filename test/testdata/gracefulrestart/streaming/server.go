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
	"io"
	"os"
	"strconv"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/test"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func main() {
	svr := trpc.NewServer()
	testpb.RegisterTestStreamingService(
		svr,
		&test.StreamingService{FullDuplexCallF: func(stream testpb.TestStreaming_FullDuplexCallServer) error {
			for {
				in, err := stream.Recv()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}
				for range in.GetResponseParameters() {
					if err := stream.Send(&testpb.StreamingOutputCallResponse{
						Payload: &testpb.Payload{
							Type: testpb.PayloadType_COMPRESSIBLE,
							Body: []byte(strconv.Itoa(os.Getpid())),
						},
					}); err != nil {
						return err
					}
				}
			}
		}},
	)
	if err := svr.Serve(); err != nil {
		panic(err)
	}
}
