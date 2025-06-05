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

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/test/protocols"
	"trpc.group/trpc-go/trpc-go/transport"
)

func (s *TestSuite) TestKeepOrderClient() {
	si := &keepOrderImpl{ids: make(map[string][]string)}
	old := s.tRPCEnv.server.async
	s.tRPCEnv.server.async = false
	defer func() {
		s.tRPCEnv.server.async = old
	}()
	s.startServer(&TRPCService{
		UnaryCallF: func(ctx context.Context, r *protocols.SimpleRequest) (*protocols.SimpleResponse, error) {
			req := &keepOrderReq{}
			if err := json.Unmarshal(r.Payload.Body, req); err != nil {
				return nil, err
			}
			s.T().Logf("start process update request %+v", req)
			si.mu.Lock()
			defer si.mu.Unlock()
			ids := si.ids[req.ID]
			ids = append(ids, strconv.Itoa(int(req.Counter)))
			si.ids[req.ID] = ids
			rsp := &protocols.SimpleResponse{
				Payload: &protocols.Payload{
					Body: []byte(strings.Join(ids, " ")),
				},
			}
			if len(ids) == int(req.Total) {
				// Clear the key when full.
				delete(si.ids, req.ID)
			}
			return rsp, nil
		},
	}, server.WithServerAsync(false))
	transports := [2]string{"default", "tnet"}
	for _, transportName := range transports {
		s.T().Run(fmt.Sprintf("client keep order with transport %s", transportName), func(t *testing.T) {
			s.sendKeepOrderReq(t, transportName)
		})
	}
}

func (s *TestSuite) sendKeepOrderReq(t *testing.T, transportName string) {
	count := 10
	rsps := make([]<-chan *client.RspOrError[protocols.SimpleResponse], 0, count)
	proxy := s.newTRPCClient(
		client.WithMultiplexedPool(multiplexed.New(multiplexed.WithConnectNumber(1))),
		client.WithTransport(transport.GetClientTransport(transportName)),
	)
	// Send multiple requests in order.
	for i := 1; i <= count; i++ {
		ctx := trpc.BackgroundContext()
		req := &keepOrderReq{
			ID:      "keeporder",
			Counter: int32(i),
			Total:   int32(count),
		}
		bs, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("json marshal failed: %+v", err)
		}
		rspOrErrorCh, err := proxy.KeepOrderUnaryCall(ctx, &protocols.SimpleRequest{
			Payload: &protocols.Payload{
				Body: bs,
			},
		}, client.WithTimeout(2*time.Second))
		if err != nil {
			t.Fatalf("client request failed: %+v", err)
		}
		rsps = append(rsps, rspOrErrorCh)
	}
	// Process multiple responses in order.
	results := make([]string, 0, len(rsps))
	for _, ch := range rsps {
		rspOrError := <-ch
		if rspOrError.Err != nil {
			t.Fatalf("client response failed: %+v", rspOrError.Err)
		}
		results = append(results, string(rspOrError.Rsp.Payload.Body))
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
			t.Errorf("[FAIL] count %d: expect %s, but got %s", i+1, expect, result)
		} else {
			t.Logf("[SUCCESS] count %d: expect %s, got %s", i+1, expect, result)
		}
	}
}
