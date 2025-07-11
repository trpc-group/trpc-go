//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package filter implements client/server filter(interceptor) chains.
//
// Signatures of filters have been refactored after v0.9.0.
// There remains lots of dirty codes to keep backward compatibility.
package filter

import (
	"context"
	"sync"

	"trpc.group/trpc-go/trpc-go/rpcz"
)

// ClientHandleFunc defines the client side filter(interceptor) function type.
type ClientHandleFunc func(ctx context.Context, req, rsp interface{}) error

// ServerHandleFunc defines the server side filter(interceptor) function type.
type ServerHandleFunc func(ctx context.Context, req interface{}) (rsp interface{}, err error)

// ClientFilter is the client side filter(interceptor) type. They are chained to process request.
type ClientFilter func(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error

// ServerFilter is the server side filter(interceptor) type. They are chained to process request.
type ServerFilter func(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error)

// NoopServerFilter is the noop implementation of ServerFilter.
func NoopServerFilter(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error) {
	return next(ctx, req)
}

// NoopClientFilter is the noop implementation of ClientFilter.
func NoopClientFilter(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error {
	return next(ctx, req, rsp)
}

// EmptyChain is an empty chain.
var EmptyChain = ClientChain{}

// ClientChain chains client side filters.
type ClientChain []ClientFilter

// Filter invokes every client side filters in the chain.
func (c ClientChain) Filter(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error {
	nextF := func(ctx context.Context, req, rsp interface{}) error {
		_, end, ctx := rpcz.NewSpanContext(ctx, "CallFunc")
		err := next(ctx, req, rsp)
		end.End()
		return err
	}

	names, ok := names(ctx)
	for i := len(c) - 1; i >= 0; i-- {
		curHandleFunc, curFilter, curI := nextF, c[i], i
		nextF = func(ctx context.Context, req, rsp interface{}) error {
			if ok {
				var ender rpcz.Ender
				_, ender, ctx = rpcz.NewSpanContext(ctx, name(names, curI))
				defer ender.End()
			}
			return curFilter(ctx, req, rsp, curHandleFunc)
		}
	}
	return nextF(ctx, req, rsp)
}

func names(ctx context.Context) ([]string, bool) {
	names, ok := rpcz.SpanFromContext(ctx).Attribute(rpcz.TRPCAttributeFilterNames)
	if !ok {
		return nil, false
	}
	ns, ok := names.([]string)
	return ns, ok
}

func name(names []string, index int) string {
	if index >= len(names) || index < 0 {
		const unknownName = "unknown"
		return unknownName
	}
	return names[index]
}

// ServerChain chains server side filters.
type ServerChain []ServerFilter

// Filter invokes every server side filters in the chain.
func (c ServerChain) Filter(ctx context.Context, req interface{}, next ServerHandleFunc) (interface{}, error) {
	nextF := func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		_, end, ctx := rpcz.NewSpanContext(ctx, "HandleFunc")
		rsp, err = next(ctx, req)
		end.End()
		return rsp, err
	}

	names, ok := names(ctx)
	for i := len(c) - 1; i >= 0; i-- {
		curHandleFunc, curFilter, curI := nextF, c[i], i
		nextF = func(ctx context.Context, req interface{}) (interface{}, error) {
			if ok {
				var ender rpcz.Ender
				_, ender, ctx = rpcz.NewSpanContext(ctx, name(names, curI))
				defer ender.End()
			}
			rsp, err := curFilter(ctx, req, curHandleFunc)
			return rsp, err
		}
	}
	return nextF(ctx, req)
}

var (
	lock          = sync.RWMutex{}
	serverFilters = make(map[string]ServerFilter)
	clientFilters = make(map[string]ClientFilter)
)

// Register registers server/client filters by name.
func Register(name string, s ServerFilter, c ClientFilter) {
	lock.Lock()
	defer lock.Unlock()
	serverFilters[name] = s
	clientFilters[name] = c
}

// GetServer gets the ServerFilter by name.
func GetServer(name string) ServerFilter {
	lock.RLock()
	f := serverFilters[name]
	lock.RUnlock()
	return f
}

// GetClient gets the ClientFilter by name.
func GetClient(name string) ClientFilter {
	lock.RLock()
	f := clientFilters[name]
	lock.RUnlock()
	return f
}
