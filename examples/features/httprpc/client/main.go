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
	"bytes"
	"io"
	"net/http"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/examples/features/httprpc/proto/echo"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	c := pb.NewEchoClientProxy()
	rsp, err := c.UnaryEcho(trpc.BackgroundContext(), &pb.EchoRequest{Message: "hello"},
		client.WithTarget("ip://127.0.0.1:8090"),
		client.WithProtocol("http"))
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("response code: %d, response message: %s", rsp.Code, rsp.Message)

	resp, err := http.Post("http://127.0.0.1:8090/trpc.examples.echo.Echo/UnaryEcho",
		"application/json", bytes.NewReader([]byte(`{"json_message":"hello"}`)),
	)
	if err != nil {
		log.Error(err)
		return
	}
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}
	log.Infof("response: %v", string(bts))
}
