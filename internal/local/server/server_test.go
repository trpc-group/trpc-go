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

package server_test

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/internal/local/server"
	"github.com/stretchr/testify/require"
)

// TestRegisterAndGetService tests the registration and retrieval of a service.
func TestRegisterAndGetService(t *testing.T) {
	s := server.NewLocalServer()
	serviceName := "testService"
	rpcName := "testRPC"
	opts := server.Options{
		ServerCodecGetter: func() codec.Codec {
			return &testCodec{}
		},
		Filters: filter.ServerChain{},
	}

	s.Register(serviceName, rpcName, func(ctx context.Context, f server.FilterFunc) (rspBody interface{}, err error) {
		return
	}, opts)

	retrievedService, err := s.GetService("", serviceName)
	if err != nil {
		t.Fatalf("Failed to get service: %s", err)
	}
	ctx, msg := codec.EnsureMessage(context.Background())
	msg.WithServerRPCName(rpcName)
	_, err = retrievedService.Handle(ctx, nil)
	require.NoError(t, err)
}

// TestServiceNotFound tests the error handling when a service is not found.
func TestServiceNotFound(t *testing.T) {
	s := server.NewLocalServer()
	serviceName := "nonExistentService"

	_, err := s.GetService("", serviceName)
	if err == nil {
		t.Fatal("Expected error for non-existent service, got nil")
	}
}

// TestHandleRequest tests the handling of a request.
func TestHandleRequest(t *testing.T) {
	serviceName := "testService"
	rpcName := "testRPC"
	opts := server.Options{
		ServerCodecGetter: func() codec.Codec {
			return &testCodec{}
		},
		Filters: filter.ServerChain{},
	}

	server.Register(serviceName, rpcName, func(ctx context.Context, f server.FilterFunc) (rspBody interface{}, err error) {
		req := ""
		ch, err := f(&req)
		if err != nil {
			return
		}
		return ch.Filter(ctx, req, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
			return req, nil
		})
	}, opts)
	service, err := server.GetService("", serviceName)
	require.NoError(t, err)

	ctx, msg := codec.EnsureMessage(context.Background())
	service.PartialDecode(msg, nil)
	msg.WithServerRPCName(rpcName)
	req := "request"
	resp, err := service.Handle(ctx, req)
	if err != nil {
		t.Fatalf("Handle failed: %s", err)
	}
	if resp != req {
		t.Errorf("Expected response 'request', got '%v'", resp)
	}
}

func TestHandlerNotFound(t *testing.T) {
	serviceName := "testService"
	rpcName := "testRPC"
	opts := server.Options{
		ServerCodecGetter: func() codec.Codec {
			return &testCodec{}
		},
		Filters: filter.ServerChain{},
	}

	server.Register(serviceName, rpcName, func(ctx context.Context, f server.FilterFunc) (rspBody interface{}, err error) {
		req := ""
		ch, err := f(&req)
		if err != nil {
			return
		}
		return ch.Filter(ctx, req, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
			return req, nil
		})
	}, opts)
	service, err := server.GetService("", serviceName)
	require.NoError(t, err)

	ctx, msg := codec.EnsureMessage(context.Background())
	service.PartialDecode(msg, nil)
	msg.WithServerRPCName(rpcName + "_wrong_suffix")
	req := "request"
	_, err = service.Handle(ctx, req)
	require.Error(t, err)
}

// testCodec is a simple mock codec to use in tests.
type testCodec struct{}

func (c *testCodec) Encode(msg codec.Msg, body []byte) ([]byte, error) {
	return []byte("encoded"), nil
}

func (c *testCodec) Decode(msg codec.Msg, buffer []byte) ([]byte, error) {
	return buffer, nil
}
