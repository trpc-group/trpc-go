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
	"strconv"
	"strings"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/examples/features/keeporderclient/proto"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
)

func main() {
	// Load and setup client configuration.
	trpc.LoadGlobalConfig(trpc.ServerConfigPath)
	trpc.SetupClients(&trpc.GlobalConfig().Client)
	count := 10
	rsps := make([]<-chan *client.RspOrError[proto.UpdateRsp], 0, count)
	// Should specify multiplexed.WithConnectNumber(1) and use multiplexed mode.
	proxy := proto.NewPlayerClientProxy(
		client.WithMultiplexedPool(multiplexed.New(multiplexed.WithConnectNumber(1))))

	// Send multiple requests in order.
	for i := 1; i <= count; i++ {
		ctx := trpc.BackgroundContext()
		req := &proto.UpdateReq{
			Id:      "keeporder",
			Counter: int32(i),
			Total:   int32(count),
		}
		rspOrErrorCh, err := proxy.KeepOrderUpdate(ctx, req)
		if err != nil {
			log.Fatalf("client request failed: %+v", err)
		}
		rsps = append(rsps, rspOrErrorCh)
	}
	// Process multiple responses in order.
	results := make([]string, 0, len(rsps))
	for _, ch := range rsps {
		rspOrError := <-ch
		if rspOrError.Err != nil {
			log.Fatalf("client response failed: %+v", rspOrError.Err)
		}
		results = append(results, rspOrError.Rsp.State)
	}

	expects := make([]string, 0, len(results))
	expectSlice := make([][]string, count)
	for i := 1; i <= count; i++ {
		for j := 1; j <= i; j++ {
			expectSlice[i-1] = append(expectSlice[i-1], strconv.Itoa(j))
		}
		expect := strings.Join(expectSlice[i-1], " ")
		expects = append(expects, expect)
	}
	for i, expect := range expects {
		result := results[i]
		if result != expect {
			log.Errorf("[FAIL] count %d: expect %s, but got %s", i+1, expect, result)
		} else {
			log.Infof("[SUCCESS] count %d: expect %s, got %s", i+1, expect, result)
		}
	}
}
