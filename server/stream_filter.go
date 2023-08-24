// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package server

import "sync"

var (
	streamFilters = make(map[string]StreamFilter)
	lock          = sync.RWMutex{}
)

// StreamServerInfo is stream information on server side.
type StreamServerInfo struct {
	// FullMethod is the full RPC method string, i.e., /package.service/method.
	FullMethod string
	// IsClientStream indicates whether the RPC is a client streaming RPC.
	IsClientStream bool
	// IsServerStream indicates whether the RPC is a server streaming RPC.
	IsServerStream bool
}

// StreamFilter is server stream filter.
type StreamFilter func(ss Stream, info *StreamServerInfo, handler StreamHandler) error

// StreamFilterChain  server stream filters chain.
type StreamFilterChain []StreamFilter

// Filter implements StreamFilter for multi stream filters.
func (c StreamFilterChain) Filter(ss Stream, info *StreamServerInfo, handler StreamHandler) error {
	for i := len(c) - 1; i >= 0; i-- {
		next, curFilter := handler, c[i]
		handler = func(ss Stream) error {
			return curFilter(ss, info, next)
		}
	}
	return handler(ss)
}

// RegisterStreamFilter registers server stream filter with name.
func RegisterStreamFilter(name string, filter StreamFilter) {
	lock.Lock()
	streamFilters[name] = filter
	lock.Unlock()
}

// GetStreamFilter gets server stream filter by name.
func GetStreamFilter(name string) StreamFilter {
	lock.RLock()
	f := streamFilters[name]
	lock.RUnlock()
	return f
}
