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

package inprocess_test

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/local/inprocess"
	iserver "trpc.group/trpc-go/trpc-go/internal/local/server"
)

// TestHandleSuccess tests the successful handling of a request.
func TestHandleSuccess(t *testing.T) {
	ctx := context.Background()
	serviceName := "testService"
	iserver.Register(serviceName, "",
		func(ctx context.Context, f iserver.FilterFunc) (rspBody interface{}, err error) {
			return
		}, iserver.Options{
			ServerCodecGetter: func() codec.Codec {
				return &testCodec{}
			},
		})
	req := "request"
	opts := inprocess.Options{
		Codec: &testCodec{},
	}

	_, err := inprocess.Handle(ctx, serviceName, req, opts)
	if err != nil {
		t.Fatalf("Handle failed: %s", err)
	}
}

// TestHandleNilCodec tests the handling of a nil codec.
func TestHandleNilCodec(t *testing.T) {
	ctx := context.Background()
	serviceName := "testService"
	req := "request"
	opts := inprocess.Options{}

	_, err := inprocess.Handle(ctx, serviceName, req, opts)
	if err == nil {
		t.Fatal("Expected error for nil codec, got nil error")
	}
}

// TestHandleServiceNotFound tests the handling when the service is not found.
func TestHandleServiceNotFound(t *testing.T) {
	ctx := context.Background()
	serviceName := "nonExistentService"
	req := "request"
	opts := inprocess.Options{
		Codec: &testCodec{},
	}

	_, err := inprocess.Handle(ctx, serviceName, req, opts)
	if err == nil {
		t.Fatal("Expected error for non-existent service, got nil")
	}
}

// testCodec is a simple mock codec to use in tests.
type testCodec struct{}

func (c *testCodec) Encode(msg codec.Msg, body []byte) ([]byte, error) {
	return []byte("encoded"), nil
}

func (c *testCodec) Decode(msg codec.Msg, buffer []byte) ([]byte, error) {
	return buffer, nil
}
