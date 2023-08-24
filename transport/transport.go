// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package transport is the network transport layer. It is only used for basic binary data network
// communication without any business logic.
// By default, there is only one pluggable ServerTransport and ClientTransport.
package transport

import (
	"context"
	"net"
	"reflect"
	"sync"

	"trpc.group/trpc-go/trpc-go/codec"
)

var (
	svrTrans    = make(map[string]ServerTransport)
	muxSvrTrans = sync.RWMutex{}

	clientTrans    = make(map[string]ClientTransport)
	muxClientTrans = sync.RWMutex{}
)

// FramerBuilder is the alias of codec.FramerBuilder.
type FramerBuilder = codec.FramerBuilder

// Framer is the alias of codec.Framer.
type Framer = codec.Framer

// contextKey is the context key.
type contextKey struct {
	name string
}

var (
	// LocalAddrContextKey is the local address context key.
	LocalAddrContextKey = &contextKey{"local-addr"}

	// RemoteAddrContextKey is the remote address context key.
	RemoteAddrContextKey = &contextKey{"remote-addr"}
)

// RemoteAddrFromContext gets remote address from context.
func RemoteAddrFromContext(ctx context.Context) net.Addr {
	addr, ok := ctx.Value(RemoteAddrContextKey).(net.Addr)
	if !ok {
		return nil
	}
	return addr
}

// ServerTransport defines the server transport layer interface.
type ServerTransport interface {
	ListenAndServe(ctx context.Context, opts ...ListenServeOption) error
}

// ClientTransport defines the client transport layer interface.
type ClientTransport interface {
	RoundTrip(ctx context.Context, req []byte, opts ...RoundTripOption) (rsp []byte, err error)
}

// Handler is the process function when server transport receive a package.
type Handler interface {
	Handle(ctx context.Context, req []byte) (rsp []byte, err error)
}

// CloseHandler handles the logic after connection closed.
type CloseHandler interface {
	HandleClose(ctx context.Context) error
}

var framerBuilders = make(map[string]codec.FramerBuilder)

// RegisterFramerBuilder register a codec.FramerBuilder.
func RegisterFramerBuilder(name string, fb codec.FramerBuilder) {
	fbv := reflect.ValueOf(fb)
	if fb == nil || fbv.Kind() == reflect.Ptr && fbv.IsNil() {
		panic("transport: register framerBuilders nil")
	}
	if name == "" {
		panic("transport: register framerBuilders name empty")
	}
	framerBuilders[name] = fb
}

// RegisterServerTransport register a ServerTransport.
func RegisterServerTransport(name string, t ServerTransport) {
	tv := reflect.ValueOf(t)
	if t == nil || tv.Kind() == reflect.Ptr && tv.IsNil() {
		panic("transport: register nil server transport")
	}
	if name == "" {
		panic("transport: register empty name of server transport")
	}
	muxSvrTrans.Lock()
	svrTrans[name] = t
	muxSvrTrans.Unlock()
}

// GetServerTransport gets the ServerTransport.
func GetServerTransport(name string) ServerTransport {
	muxSvrTrans.RLock()
	t := svrTrans[name]
	muxSvrTrans.RUnlock()
	return t
}

// RegisterClientTransport register a ClientTransport.
func RegisterClientTransport(name string, t ClientTransport) {
	tv := reflect.ValueOf(t)
	if t == nil || tv.Kind() == reflect.Ptr && tv.IsNil() {
		panic("transport: register nil client transport")
	}
	if name == "" {
		panic("transport: register empty name of client transport")
	}
	muxClientTrans.Lock()
	clientTrans[name] = t
	muxClientTrans.Unlock()
}

// GetClientTransport gets the ClientTransport.
func GetClientTransport(name string) ClientTransport {
	muxClientTrans.RLock()
	t := clientTrans[name]
	muxClientTrans.RUnlock()
	return t
}

// GetFramerBuilder gets the FramerBuilder by name.
func GetFramerBuilder(name string) codec.FramerBuilder {
	return framerBuilders[name]
}

// ListenAndServe wraps and starts the default server transport.
func ListenAndServe(opts ...ListenServeOption) error {
	return DefaultServerTransport.ListenAndServe(context.Background(), opts...)
}

// RoundTrip wraps and starts the default client transport.
func RoundTrip(ctx context.Context, req []byte, opts ...RoundTripOption) ([]byte, error) {
	return DefaultClientTransport.RoundTrip(ctx, req, opts...)
}
