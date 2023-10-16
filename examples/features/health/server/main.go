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
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/healthcheck"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
	s := trpc.NewServer()
	// get admin service
	admin, err := trpc.GetAdminService(s)
	if err != nil {
		panic(err)
	}

	// you have two service in the server
	// you can register health check for each service
	unregisterHealthCheckFoo, updateStatusFoo, err := admin.RegisterHealthCheck("foo")
	if err != nil {
		panic(err)
	}
	unregisterHealthCheckBar, updateStatusBar, err := admin.RegisterHealthCheck("bar")
	if err != nil {
		panic(err)
	}

	go func() {
		// simulate slow start
		// because we don't set status as Serving
		// the status of service will be set to NotServing
		fmt.Println("services are starting, the status is NotServing")
		time.Sleep(10 * time.Second)

		// update status for both service to Serving
		// then the status of service will be set to Serving
		updateStatusFoo(healthcheck.Serving)
		updateStatusBar(healthcheck.Serving)
		fmt.Println("services are ready, the status is Serving")

		// 30s later, if any service is not serving, the status of service will be set to NotServing
		time.Sleep(30 * time.Second)
		updateStatusFoo(healthcheck.NotServing)
		fmt.Println("service foo is down, the status is NotServing")

		// recover status to Serving after 30s
		// Since all services are serving, the status of service will be set to Serving again
		time.Sleep(30 * time.Second)
		updateStatusFoo(healthcheck.Serving)
		fmt.Println("service foo is up, the status is Serving again")

		// set bar as NotServing
		// Since bar is not serving, the status of service will be set to NotServing
		time.Sleep(30 * time.Second)
		updateStatusBar(healthcheck.NotServing)
		fmt.Println("service bar is down, the status is NotServing")

		// unregister health check for bar
		// the whole server status only relies on the status of foo
		// and foo is Serving, so the status of service will be set to Serving
		time.Sleep(30 * time.Second)
		unregisterHealthCheckBar()
		fmt.Println("the health check for service bar is unregistered, the status is Serving")

		// set foo as NotServing
		// Since foo is not serving, the status of service will be set to NotServing
		time.Sleep(30 * time.Second)
		updateStatusFoo(healthcheck.NotServing)
		fmt.Println("service foo is down, the status is NotServing")

		// unregister health check for foo
		// Since there's no health check, the status of service will be set to Serving
		time.Sleep(30 * time.Second)
		unregisterHealthCheckFoo()
		fmt.Println("the health check for service foo is unregistered, " +
			"there's no health check for the server, the status is Serving by default")
	}()

	pb.RegisterGreeterService(s, &greeterServerImpl{})
	s.Serve()
}

// greeterServerImpl  implements service.
type greeterServerImpl struct {
}

// SayHello implements `SayHello` method.
func (t *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{}, nil
}

// SayHi implements `SayHello` method.
func (t *greeterServerImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{}, nil
}
