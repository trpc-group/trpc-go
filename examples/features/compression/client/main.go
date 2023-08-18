// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package main is the main package.
package main

import (
	"flag"

	"trpc.group/trpc-go/trpc-go"
	_ "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var compressTypeId = flag.String("type", "gzip", "Input Compress Type")

func main() {
	// init server
	_ = trpc.NewServer()

	// generate context
	ctx := trpc.BackgroundContext()

	// parses the command-line flags
	flag.Parse()

	// analyze compress type
	compressType := codec.CompressTypeNoop
	switch *compressTypeId {
	case "gzip":
		compressType = codec.CompressTypeGzip
	case "snappy":
		compressType = codec.CompressTypeSnappy
	case "zlib":
		compressType = codec.CompressTypeZlib
	case "streamSnappy":
		compressType = codec.CompressTypeStreamSnappy
	case "blockSnappy":
		compressType = codec.CompressTypeBlockSnappy
	default:
		log.Fatal("unknown compress type, please use gzip, snappy, zlib, streamSnappy or blockSnappy")
	}

	// log compress type
	log.Debugf("request with compressType : %v", *compressTypeId)

	// sets client options
	opts := []client.Option{
		client.WithCompressType(compressType),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)
	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}

	// send request
	rsp, err := clientProxy.SayHello(ctx, req)
	if err != nil {
		log.Error(err)
	}

	// log reply
	log.Infof("reply is: %+v", rsp)
}
