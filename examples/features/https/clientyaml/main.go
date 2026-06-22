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

package main

import (
	"context"
	"flag"
	"net/http"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Parse config path from command line since the default config path is parsed in trpc.NewServer.
	// We don't want to use new server here since it is just a client.
	configPath := flag.String("conf", "./trpc_go.yaml", "config path")
	flag.Parse()

	trpc.ServerConfigPath = *configPath
	// Load YAML config and setup plugins.
	if err := trpc.LoadGlobalConfig(trpc.ServerConfigPath); err != nil {
		log.Errorf("load config error: %v", err)
		return
	}
	if err := trpc.Setup(trpc.GlobalConfig()); err != nil {
		log.Errorf("setup error: %v", err)
		return
	}

	// Iterate configured client services and invoke each one once.
	cfg := trpc.GlobalConfig()
	for _, s := range cfg.Client.Service {
		// Choose the service name from YAML.
		name := s.ServiceName
		if name == "" {
			name = s.Callee
		}
		if name == "" {
			continue
		}

		// Create HTTP client proxy relying solely on YAML.
		httpCli := thttp.NewClientProxy(name,
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		)

		// Prepare headers and bodies.
		reqHeader := &thttp.ClientReqHeader{Method: http.MethodPost}
		reqHeader.AddHeader("request", "yaml-only")
		rspHead := &thttp.ClientRspHeader{}
		req := &codec.Body{Data: []byte("Hello from YAML-only client.")}
		rsp := &codec.Body{}

		// Invoke.
		log.Infof("=== Invoke service: %s ===", name)
		if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
			client.WithReqHead(reqHeader),
			client.WithRspHead(rspHead),
		); err != nil {
			log.Errorf("call %s failed: %v", name, err)
			continue
		}

		replyHead := ""
		if rspHead.Response != nil {
			replyHead = rspHead.Response.Header.Get("reply")
		}
		log.Infof("service %s ok, data: %q, reply head: %q", name, string(rsp.Data), replyHead)
	}
}
