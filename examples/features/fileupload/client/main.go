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

// Package main trpc-go supports stream RPC，with stream RPC,
// the client and server can establish a continuous connection to continuously send and receive data,
// thus allowing the server to provide continuous responses.
// this file is stream RPC client samples.
package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	pb "trpc.group/trpc-go/trpc-go/examples/features/stream/proto"
)

func main() {
	// Load configuration following the logic in trpc.NewServer.
	cfg, err := trpc.LoadConfig(trpc.ServerConfigPath)
	if err != nil {
		panic("load config fail: " + err.Error())
	}
	trpc.SetGlobalConfig(cfg)
	if err := trpc.Setup(cfg); err != nil {
		panic("setup plugin fail: " + err.Error())
	}
	callUploadFile()
}
func callUploadFile() {
	proxy := pb.NewTestStreamClientProxy(
		client.WithTarget("ip://127.0.0.1:8010"),
		client.WithProtocol("trpc"),
	)

	filePath := "./ITerm2_v3.4_icon.png"
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	stream, err := proxy.UploadFileStream(context.Background())
	if err != nil {
		log.Fatalf("could not upload file: %v", err)
	}

	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read file: %v", err)
		}

		err = stream.Send(&pb.UploadFileReq{
			Content:  buffer[:n],
			Filename: filepath.Base(filePath),
		})
		if err != nil {
			log.Fatalf("failed to send chunk: %v", err)
		}
	}

	status, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("failed to receive status: %v", err)
	}

	log.Printf("Upload status: %v, message: %s", status.GetSuccess(), status.GetMessage())
}
