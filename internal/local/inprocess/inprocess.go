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

// Package inprocess glues the calling of local clients to local servers.
package inprocess

import (
	"context"
	"errors"

	"trpc.group/trpc-go/trpc-go/codec"
	iserver "trpc.group/trpc-go/trpc-go/internal/local/server"
)

// Handle handles the incoming request, returns the response.
func Handle(ctx context.Context, serviceName string, req interface{}, opts Options) (interface{}, error) {
	if opts.Codec == nil {
		return nil, errors.New("inprocess handle requires a non-nil codec")
	}
	// Try to get service from the local server first.
	s, err := iserver.GetService(opts.Protocol, serviceName)
	if err != nil {
		return nil, err
	}
	originalMsg := codec.Message(ctx)
	// If the local calling fails, for "all" scope the client will fallback to try the "remote" scope.
	// So it is necessary to copy context and message here to avoid conflict afterwards.
	ctx, msg := codec.WithCloneContextAndMessage(ctx)
	inheritClientMetadata(originalMsg, msg)
	// 1. Use the client codec encode to fully mature the client context message (carrying the right fields
	// and metadata).
	reqBuf, err := opts.Codec.Encode(msg, nil)
	if err != nil {
		return nil, err
	}
	// 2. Use the server codec decode to indirectly convert the client context message to the server context message.
	// The client encode and server decode reuse the existing utilities to avoid maintaining extra logic for inprocess
	// calling.
	if err := s.PartialDecode(msg, reqBuf); err != nil {
		return nil, err
	}
	// 3. Finally call the server handler.
	return s.Handle(ctx, req)
}

func inheritClientMetadata(originalMsg, msg codec.Msg) {
	m := codec.MetaData{}
	for k, v := range originalMsg.ClientMetaData() {
		m[k] = v
	}
	msg.WithClientMetaData(m)
}
