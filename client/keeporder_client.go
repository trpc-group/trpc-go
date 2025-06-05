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

package client

import (
	"context"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
	"github.com/panjf2000/ants/v2"
)

// KeepOrderClient writes the request synchronously (to keep order) and returns a channel that is expected
// to pass back response in the future.
type KeepOrderClient[RspType any] interface {
	KeepOrderInvoke(
		ctx context.Context,
		reqBody interface{},
		opt ...Option,
	) (
		<-chan *RspOrError[RspType],
		error,
	)
}

// RspOrError contains response or error.
type RspOrError[RspType any] struct {
	Rsp *RspType
	Err error
}

type keepOrderClient[RspType any] struct {
	cli Client
}

// NewKeepOrderClient returns a new keep-order client.
func NewKeepOrderClient[RspType any](
	cli Client,
) KeepOrderClient[RspType] {
	return &keepOrderClient[RspType]{cli: cli}
}

func (c *keepOrderClient[RspType]) KeepOrderInvoke(
	ctx context.Context,
	reqBody interface{},
	opt ...Option,
) (<-chan *RspOrError[RspType], error) {
	ch := make(chan *RspOrError[RspType], 1)
	ech := make(chan error, 1)
	ctx = keeporder.NewContextWithClientInfo(ctx, &keeporder.ClientInfo{
		SendError: ech,
	})
	ants.Submit(func() {
		var rsp RspType
		err := c.cli.Invoke(ctx, reqBody, &rsp, opt...)
		select {
		case ech <- err: // If the error is generated before transport write, this case will be executed.
		default:
		}
		ch <- &RspOrError[RspType]{Rsp: &rsp, Err: err}
		// Instead of putting back the message inside the stub code, we put back the message
		// in this asynchronous procedure after the response has returned.
		codec.PutBackMessage(codec.Message(ctx))
	})
	// This channel has data when:
	// 1. The error happens before the client transport can write the request.
	// 2. The client transport finishes the write and returns an error (could be a nil error).
	// And the above two cases contains all the scenarios, which make sure that the
	// following statement will not block forever.
	// In this way, it is guaranteed that the request will be send synchronously (to keep order) while
	// the response is returned asynchronously (to allow users to keep on sending other requests).
	err := <-ech
	return ch, err
}
