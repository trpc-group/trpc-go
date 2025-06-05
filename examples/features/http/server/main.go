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

// Package main is the server main package for http demo.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

// handle is a function that processes HTTP requests.
// Its implementation is consistent with the standard HTTP library.
func handle(w http.ResponseWriter, r *http.Request) error {
	_, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		return err
	}
	// Finally, use 'w' to send the response.
	w.Header().Set("Content-type", "application/text")
	w.Header().Set("reply", "response head")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("response body"))
	return nil
}

func main() {
	filter.Register("simple", ServerFilter, nil)
	// Init server.
	s := trpc.NewServer()

	// Register the handle function for the "/v1/hello" endpoint.
	thttp.HandleFunc("/v1/hello", handle)

	// When registering the NoProtocolService, the parameter passed must match the service name in the configuration: s.Service("trpc.app.server.stdhttp").
	thttp.RegisterNoProtocolService(s.Service("trpc.app.server.stdhttp"))

	// Start serving and listening.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}

func ServerFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
	msg := codec.Message(ctx)
	rsp, err = next(ctx, req)
	log.Info(msg.ClientReqHead())
	log.Info(msg.ClientRspHead())
	log.Info(msg.ServerReqHead())
	log.Info(msg.ServerRspHead())
	return rsp, err
}
