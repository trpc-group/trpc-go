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
	"sync"
)

var (
	streamFilters = make(map[string]StreamFilter)
	lock          = sync.RWMutex{}
)

// ClientStream is the interface returned to users to call its methods.
type ClientStream interface {
	// RecvMsg receives messages.
	RecvMsg(m interface{}) error
	// SendMsg sends messages.
	SendMsg(m interface{}) error
	// CloseSend closes sender.
	// No more sending messages,
	// but it's still allowed to continue to receive messages.
	CloseSend() error
	// Context gets the Context.
	Context() context.Context
}

// ClientStreamDesc is the client stream description.
type ClientStreamDesc struct {
	// StreamName is the name of the stream, corresponding to Method of unary RPC.
	StreamName string
	// ClientStreams indicates whether it's client streaming.
	ClientStreams bool
	// ServerStreams indicates whether it's server streaming.
	ServerStreams bool
}

// StreamFilter is the client stream filter.
// StreamFilter processing happens before or after stream's establishing.
type StreamFilter func(ctx context.Context, desc *ClientStreamDesc, streamer Streamer) (ClientStream, error)

// Streamer is the wrapper filter function used to filter all methods of ClientStream.
type Streamer func(ctx context.Context, desc *ClientStreamDesc) (ClientStream, error)

// RegisterStreamFilter registers a StreamFilter by name.
func RegisterStreamFilter(name string, filter StreamFilter) {
	lock.Lock()
	streamFilters[name] = filter
	lock.Unlock()
}

// GetStreamFilter returns a StreamFilter by name.
func GetStreamFilter(name string) StreamFilter {
	lock.RLock()
	f := streamFilters[name]
	lock.RUnlock()
	return f
}

// StreamFilterChain client stream filters chain.
type StreamFilterChain []StreamFilter

// Filter implements StreamFilter for multi stream filters.
func (c StreamFilterChain) Filter(ctx context.Context,
	desc *ClientStreamDesc, streamer Streamer) (ClientStream, error) {
	for i := len(c) - 1; i >= 0; i-- {
		next, curFilter := streamer, c[i]
		streamer = func(ctx context.Context, desc *ClientStreamDesc) (ClientStream, error) {
			return curFilter(ctx, desc, next)
		}
	}
	return streamer(ctx, desc)
}
