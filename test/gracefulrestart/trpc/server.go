// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package main is the main package.
package main

import (
	"context"
	"os"
	"strconv"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/test"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func main() {
	svr := trpc.NewServer()
	testpb.RegisterTestTRPCService(
		svr,
		&test.TRPCService{EmptyCallF: func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
			// Graceful restart will create a new process. We returns the current process ID to the client to
			// verify the status of the current service's restart process.
			trpc.SetMetaData(ctx, "server-pid", []byte(strconv.Itoa(os.Getpid())))
			return &testpb.Empty{}, nil
		}},
	)
	if err := svr.Serve(); err != nil {
		panic(err)
	}
}
