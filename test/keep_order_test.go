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
	"sync"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/test/protocols"
	"trpc.group/trpc-go/trpc-go/transport"
)

func (s *TestSuite) TestKeepOrderPreDecode() {
	metaDataKey := "keep_order_key"
	transports := [2]string{"default", "tnet"}
	for _, transportName := range transports {
		s.T().Run(fmt.Sprintf("pre-decode with transport %s", transportName), func(t *testing.T) {
			// keepOrderKey is the key for keep-order metadata.
			si := &keepOrderImpl{ids: make(map[string][]string)}
			s.startServer(&TRPCService{
				UnaryCallF: func(ctx context.Context, r *protocols.SimpleRequest) (*protocols.SimpleResponse, error) {
					return keepOrderUnaryCallFunc(t, r, si)
				},
			},
				server.WithKeepOrderPreDecodeExtractor(func(ctx context.Context, reqBody []byte) (string, bool) {
					// Implement keep-order logic for pre-decoding.
					msg := codec.Message(ctx)
					m := msg.ServerMetaData()
					if m == nil {
						t.Errorf("meta data is nil for %q\n", reqBody)
						return "", false
					}
					key, ok := m[metaDataKey]
					if !ok {
						t.Errorf("meta key %q does not exist for %q\n", metaDataKey, reqBody)
						return "", false
					}
					return string(key), true
				}),
				// transports registered in SetupSuite
				server.WithTransport((transport.GetServerTransport(transportName))))
			s.sendKeepOrderPreDecodeReq(t, metaDataKey)
		})
	}
}

func (s *TestSuite) TestKeepOrderPreUnmarshal() {
	transports := [2]string{"default", "tnet"}
	for _, transportName := range transports {
		s.T().Run(fmt.Sprintf("pre-unmarshal with transport %s", transportName), func(t *testing.T) {
			si := &keepOrderImpl{ids: make(map[string][]string)}
			s.startServer(&TRPCService{
				UnaryCallF: func(ctx context.Context, r *protocols.SimpleRequest) (*protocols.SimpleResponse, error) {
					return keepOrderUnaryCallFunc(t, r, si)
				},
			},
				server.WithKeepOrderPreUnmarshalExtractor(func(ctx context.Context, reqBody interface{}) (string, bool) {
					// Implement keep-order logic for pre-unmarshaling.
					r, ok := reqBody.(*protocols.SimpleRequest)
					if !ok {
						t.Errorf("invalid request type %T, want *proto.HelloReq", reqBody)
						return "", false
					}
					req := &keepOrderReq{}
					if err := json.Unmarshal(r.Payload.Body, req); err != nil {
						t.Errorf("json unmarshal error: %v", err)
						return "", false
					}
					return req.ID, true
				}),
				// transports registered in SetupSuite
				server.WithTransport(transport.GetServerTransport(transportName)))
			s.sendKeepOrderPreUnmarshalReq(t)
		})
	}
}

type keepOrderImpl struct {
	mu  sync.Mutex
	ids map[string][]string
}

type keepOrderReq struct {
	ID      string `json:"id"`
	Counter int32  `json:"counter"`
	Total   int32  `json:"total"`
}

func keepOrderUnaryCallFunc(t *testing.T, r *protocols.SimpleRequest, impl *keepOrderImpl) (*protocols.SimpleResponse, error) {
	req := &keepOrderReq{}
	if err := json.Unmarshal(r.Payload.Body, req); err != nil {
		return nil, err
	}
	time.Sleep(20 * time.Millisecond * time.Duration(req.Total-req.Counter))
	t.Logf("start process update request %+v", req)
	impl.mu.Lock()
	defer impl.mu.Unlock()
	ids := impl.ids[req.ID]
	ids = append(ids, strconv.Itoa(int(req.Counter)))
	impl.ids[req.ID] = ids
	rsp := &protocols.SimpleResponse{
		Payload: &protocols.Payload{
			Body: []byte(strings.Join(ids, " ")),
		},
	}
	if len(ids) == int(req.Total) {
		// Clear the key when full.
		delete(impl.ids, req.ID)
	}
	return rsp, nil
}

func (s *TestSuite) sendKeepOrderPreDecodeReq(t *testing.T, metaDataKey string) {
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	count := 10
	rsps := make([]<-chan *client.RspOrError[protocols.SimpleResponse], 0, count)
	for _, key := range keys {
		key := key
		proxy := s.newTRPCClient(
			client.WithMetaData(metaDataKey, []byte(key)),
			client.WithMultiplexedPool(multiplexed.New(multiplexed.WithConnectNumber(1))),
		)
		for i := 1; i <= count; i++ {
			i := i
			ech := make(chan error, 1)
			ctx := keeporder.NewContextWithClientInfo(trpc.BackgroundContext(), &keeporder.ClientInfo{
				SendError: ech,
			})
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			req := &keepOrderReq{
				ID:      key,
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

func (s *TestSuite) sendKeepOrderPreUnmarshalReq(t *testing.T) {
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	count := 10
	rsps := make([]<-chan *client.RspOrError[protocols.SimpleResponse], 0, count)
	for _, key := range keys {
		key := key
		proxy := s.newTRPCClient(
			client.WithMultiplexedPool(multiplexed.New(multiplexed.WithConnectNumber(1))),
		)
		for i := 1; i <= count; i++ {
			i := i
			ech := make(chan error, 1)
			ctx := keeporder.NewContextWithClientInfo(trpc.BackgroundContext(), &keeporder.ClientInfo{
				SendError: ech,
			})
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			req := &keepOrderReq{
				ID:      key,
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
