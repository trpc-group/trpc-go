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

// Package main is the server main package for admin demo.
package main

import (
	"fmt"
	"net/http"

	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// testCmds defines a custom admin command
func testCmds(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("test cmds"))
}

// init registers routes for custom admin commands.
func init() {
	// Register custom handler.
	admin.HandleFunc("/testCmds", testCmds)
}

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register service.
	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})

	// Serve and listen.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
