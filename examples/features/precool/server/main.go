//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package main is the server main package for precool demo.
package main

import (
	"fmt"
	"time"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	"trpc.group/trpc-go/trpc-go/precool"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

type precoolChecker struct {
	dbReady     bool
	cacheReady  bool
	configReady bool
}

func (pc *precoolChecker) CheckPrecool() precool.Status {
	if !pc.dbReady {
		return precool.Failure
	}
	if !pc.cacheReady || !pc.configReady {
		return precool.Ongoing
	}
	return precool.Success
}

func main() {
	s := trpc.NewServer()
	checker := &precoolChecker{}

	if err := s.RegisterServicePrecool("trpc.examples.precool.Precool", checker.CheckPrecool); err != nil {
		panic("register precool strategy: " + err.Error())
	}

	go func() {
		fmt.Println("Starting precool process...")

		time.Sleep(2 * time.Second)
		checker.dbReady = true
		fmt.Println("Database connection ready")

		time.Sleep(3 * time.Second)
		checker.cacheReady = true
		fmt.Println("Cache warmup completed")

		time.Sleep(1 * time.Second)
		checker.configReady = true
		fmt.Println("Configuration loaded")
		fmt.Println("Precool process completed successfully")
	}()

	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})

	fmt.Println("Server starting with precool detection enabled...")
	fmt.Println("Check process status: curl http://127.0.0.1:11014/cmds/is_precool/")
	fmt.Println("Check service status: curl http://127.0.0.1:11014/cmds/is_precool/trpc.examples.precool.Precool")

	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
