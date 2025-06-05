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

// Package filter implements client/server filter(interceptor) chains.
//
// Signatures of filters have been refactored after v0.9.0.
// There remains lots of dirty codes to keep backward compatibility.
package filter

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	irpcz "trpc.group/trpc-go/trpc-go/internal/rpcz"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
)

// HandleFunc defines the old client side filter(interceptor) function type.
// Deprecated: Use ClientHandleFunc instead.
type HandleFunc = ClientHandleFunc

// ClientHandleFunc defines the client side filter(interceptor) function type.
type ClientHandleFunc func(ctx context.Context, req, rsp interface{}) error

// ServerHandleFunc defines the server side filter(interceptor) function type.
type ServerHandleFunc func(ctx context.Context, req interface{}) (rsp interface{}, err error)

// oldServerHandleFunc is the same as ClientHandleFunc in old version. Please use ServerHandleFunc in the new version.
// Deprecated: Use ServerHandleFunc instead.
type oldServerHandleFunc = ClientHandleFunc

// Filter is the filter(interceptor) type. They are chained to process request.
// Deprecated: Use ClientFilter instead.
type Filter = ClientFilter

// ClientFilter is the client side filter(interceptor) type. They are chained to process request.
type ClientFilter func(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error

// ServerFilter is the server side filter(interceptor) type. They are chained to process request.
type ServerFilter func(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error)

// oldServerFilter is the same as ClientFilter in old version. Please use ServerFilter in the new version.
// Deprecated: Use ServerFilter instead.
type oldServerFilter = ClientFilter

// NoopServerFilter is the noop implementation of ServerFilter.
func NoopServerFilter(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error) {
	return next(ctx, req)
}

// NoopClientFilter is the noop implementation of ClientFilter.
func NoopClientFilter(ctx context.Context, req, rsp interface{}, next HandleFunc) error {
	return next(ctx, req, rsp)
}

// NoopFilter is an alias of NoopClientFilter.
// Deprecated: Use NoopClientFilter instead.
var NoopFilter = NoopClientFilter

// Chain chains filters.
// Deprecated: Use ClientChain instead.
type Chain = ClientChain

// EmptyChain is an empty chain.
var EmptyChain = Chain{}

// ClientChain chains client side filters.
type ClientChain []Filter

// Handle invokes every client side filters in the chain.
// Deprecated: Use Filter instead.
func (c ClientChain) Handle(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error {
	return c.Filter(ctx, req, rsp, next)
}

// Filter invokes every client side filters in the chain.
func (c ClientChain) Filter(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error {
	if rpczenable.Enabled {
		nextF := func(ctx context.Context, req, rsp interface{}) error {
			_, end, ctx := rpcz.NewSpanContext(ctx, "CallFunc")
			err := next(ctx, req, rsp)
			end.End()
			return err
		}

		names, ok := irpcz.FilterNames(ctx)
		for i := len(c) - 1; i >= 0; i-- {
			curHandleFunc, curFilter, curI := nextF, c[i], i
			nextF = func(ctx context.Context, req, rsp interface{}) error {
				if ok {
					var ender rpcz.Ender
					_, ender, ctx = rpcz.NewSpanContext(ctx, irpcz.FilterName(names, curI))
					defer ender.End()
				}
				return curFilter(ctx, req, rsp, curHandleFunc)
			}
		}
		return nextF(ctx, req, rsp)
	}
	for i := len(c) - 1; i >= 0; i-- {
		curHandleFunc, curFilter := next, c[i]
		next = func(ctx context.Context, req, rsp interface{}) error {
			return curFilter(ctx, req, rsp, curHandleFunc)
		}
	}
	return next(ctx, req, rsp)
}

// ServerChain chains server side filters.
type ServerChain []ServerFilter

// Handle invokes every server side filters in the chain.
// Deprecated: Use Filter instead.
func (c ServerChain) Handle(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error {
	nextServerHandler := func(ctx context.Context, reqBody interface{}) (interface{}, error) {
		if err := next(ctx, reqBody, rsp); err != nil {
			return nil, err
		}
		return rsp, nil
	}
	rspBody, err := c.Filter(ctx, req, nextServerHandler)
	if rspBody != rsp {
		// User returned a new rsp struct in server filter, we must copy it.
		if err := copyRsp(rsp, rspBody); err != nil {
			return errs.NewFrameError(errs.RetServerEncodeFail, err.Error())
		}
	}
	return err
}

// copier Through which users can customize the modified 'rsp'
// Deprecated: The old version uses Handle to implement the filter, and the new version uses Filter.
// The old version uses copier to copy the response, and the new version uses Filter.
type copier interface {
	// CopyTo encodes the request.
	CopyTo(dst interface{}) error
}

func copyRsp(dst, src interface{}) error {
	switch src := src.(type) {
	case proto.Message:
		if dstPb, ok := dst.(proto.Message); ok {
			data, err := proto.Marshal(src)
			if err != nil {
				return fmt.Errorf("proto marshal rsp fail: %v", err)
			}
			if err := proto.Unmarshal(data, dstPb); err != nil {
				return fmt.Errorf("proto unmarshal rsp fail: %v", err)
			}
		} else {
			return fmt.Errorf("server filter returns a pb rsp to none pb rsp")
		}
	case copier:
		return src.CopyTo(dst)
	default:
		// Use json to keep backward compatibility for non-pb scenarios.
		// There is still a problem if user specifies omitempty.
		// For example, filter want to reset code=0 in rsp. With omitempty enabled, code=0 is not copied back to rsp.
		// However, omitempty is usually generated by pb, which would go to previous branch.
		// For the very rare cases where users define their own omitempty, we leave it to themselves.
		serializer := &codec.JSONPBSerialization{}
		data, err := serializer.Marshal(src)
		if err != nil {
			return fmt.Errorf("json marshal rsp fail: %v", err)
		}
		if err := serializer.Unmarshal(data, dst); err != nil {
			return fmt.Errorf("json unmarshal rsp fail: %v", err)
		}
	}
	return nil
}

// Filter invokes every server side filters in the chain.
func (c ServerChain) Filter(ctx context.Context, req interface{}, next ServerHandleFunc) (interface{}, error) {
	if rpczenable.Enabled {
		nextF := func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
			_, end, ctx := rpcz.NewSpanContext(ctx, "HandleFunc")
			rsp, err = next(ctx, req)
			end.End()
			return rsp, err
		}

		names, ok := irpcz.FilterNames(ctx)
		for i := len(c) - 1; i >= 0; i-- {
			curHandleFunc, curFilter, curI := nextF, c[i], i
			nextF = func(ctx context.Context, req interface{}) (interface{}, error) {
				if ok {
					var ender rpcz.Ender
					_, ender, ctx = rpcz.NewSpanContext(ctx, irpcz.FilterName(names, curI))
					defer ender.End()
				}
				rsp, err := curFilter(ctx, req, curHandleFunc)
				return rsp, err
			}
		}
		return nextF(ctx, req)
	}
	for i := len(c) - 1; i >= 0; i-- {
		curHandleFunc, curFilter := next, c[i]
		next = func(ctx context.Context, req interface{}) (interface{}, error) {
			return curFilter(ctx, req, curHandleFunc)
		}
	}
	return next(ctx, req)
}

var (
	lock          = sync.RWMutex{}
	serverFilters = make(map[string]ServerFilter)
	clientFilters = make(map[string]ClientFilter)
)

// Register registers server/client filters by name.
// Currently, use interface as the signature of server filter to be compatible with ServerFilter and ClientFilter.
// Finally, the signature of server filter should be changed to ServerFilter.
func Register(name string, s interface{}, c ClientFilter) {
	lock.Lock()
	defer lock.Unlock()
	serverFilters[name] = ConvertToServerFilter(name, s)
	clientFilters[name] = c
}

// ConvertToServerFilter converts oldServerFilter to ServerFilter.
// Deprecated
func ConvertToServerFilter(name string, filter interface{}) ServerFilter {
	switch f := filter.(type) {
	case ServerFilter:
		return f
	case func(context.Context, interface{}, ServerHandleFunc) (interface{}, error):
		return f
	case oldServerFilter:
		return convertToServerFilter(name, f)
	case func(context.Context, interface{}, interface{}, oldServerHandleFunc) error:
		return convertToServerFilter(name, f)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("server filter type: %T not supported", filter))
	}
}

// Deprecated
func convertToServerFilter(name string, c ClientFilter) ServerFilter {
	log.Warnf(`filter: %s is too old, please change to new ServerFilter,
any question please refer to ChangeLog v0.9.0`, name)
	return func(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error) {
		clientHandleFunc := func(ctx context.Context, reqBody, rspBody interface{}) error {
			rsp, err = next(ctx, reqBody)
			return err
		}
		// The old server filter, aka ClientFilter, need a rsp interface.
		// Pass nil to forbid any access to rsp, which may cause the program panic.
		// We use this mechanism to force users to update their program.
		err = c(ctx, req, nil, clientHandleFunc)
		return rsp, err
	}
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
