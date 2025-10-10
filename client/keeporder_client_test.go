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

package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
)

func TestKeepOrderClient(t *testing.T) {
	rsp := "hello world"
	cli := &testKeepOrderClient{
		wantRsp: rsp,
	}
	c := client.NewKeepOrderClient[testRsp](cli)
	ctx := context.Background()
	ch, err := c.KeepOrderInvoke(ctx, &testReq{})
	require.NoError(t, err)
	rspOrError := <-ch
	require.NoError(t, rspOrError.Err)
	require.NotNil(t, rspOrError.Rsp)
	require.EqualValues(t, rsp, rspOrError.Rsp.Message)
}

type testReq struct {
	Message string
}
type testRsp struct {
	Message string
}

type testKeepOrderClient struct {
	wantRsp string
}

func (c *testKeepOrderClient) Invoke(
	ctx context.Context,
	reqBody interface{},
	rspBody interface{},
	opt ...client.Option,
) error {
	info, ok := keeporder.ClientInfoFromContext(ctx)
	if !ok {
		return errors.New("client info not found")
	}
	info.SendError <- nil
	rsp, ok := rspBody.(*testRsp)
	if !ok {
		return errors.New("invalid response type")
	}
	rsp.Message = c.wantRsp
	return nil
}
