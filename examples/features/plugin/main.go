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
    trpc "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/examples/features/common"
    pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
    s := trpc.NewServer()
    pb.RegisterGreeterService(s, &common.GreeterServerImpl{})
    s.Serve()
}
