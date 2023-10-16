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

package transport

import (
	"context"
	"reflect"
	"sync"
)

var (
	clientStreamTrans    = make(map[string]ClientStreamTransport)
	muxClientStreamTrans = sync.RWMutex{}

	serverStreamTrans    = make(map[string]ServerStreamTransport)
	muxServerStreamTrans = sync.RWMutex{}
)

// ClientStreamTransport is the client stream transport interface.
// It's compatible with common RPC transport.
type ClientStreamTransport interface {
	// Send sends stream messages.
	Send(ctx context.Context, req []byte, opts ...RoundTripOption) error
	// Recv receives stream messages.
	Recv(ctx context.Context, opts ...RoundTripOption) ([]byte, error)
	// Init inits the stream.
	Init(ctx context.Context, opts ...RoundTripOption) error
	// Close closes stream transport, return connection to the resource pool.
	Close(ctx context.Context)
}

// ServerStreamTransport is the server stream transport interface.
// It's compatible with common RPC transport.
type ServerStreamTransport interface {
	// ServerTransport is used to keep compatibility with common RPC transport.
	ServerTransport
	// Send sends messages.
	Send(ctx context.Context, req []byte) error
	// Close is called when server encounters an error and cleans up.
	Close(ctx context.Context)
}

// RegisterServerStreamTransport Registers a named ServerStreamTransport.
func RegisterServerStreamTransport(name string, t ServerStreamTransport) {
	tv := reflect.ValueOf(t)
	if t == nil || tv.Kind() == reflect.Ptr && tv.IsNil() {
		panic("transport: register nil server transport")
	}
	if name == "" {
		panic("transport: register empty name of server transport")
	}
	muxServerStreamTrans.Lock()
	serverStreamTrans[name] = t
	muxServerStreamTrans.Unlock()

}

// RegisterClientStreamTransport registers a named ClientStreamTransport.
func RegisterClientStreamTransport(name string, t ClientStreamTransport) {
	tv := reflect.ValueOf(t)
	if t == nil || tv.Kind() == reflect.Ptr && tv.IsNil() {
		panic("transport: register nil client transport")
	}
	if name == "" {
		panic("transport: register empty name of client transport")
	}
	muxClientStreamTrans.Lock()
	clientStreamTrans[name] = t
	muxClientStreamTrans.Unlock()
}

// GetClientStreamTransport returns ClientStreamTransport by name.
func GetClientStreamTransport(name string) ClientStreamTransport {
	muxClientStreamTrans.RLock()
	t := clientStreamTrans[name]
	muxClientStreamTrans.RUnlock()
	return t
}

// GetServerStreamTransport returns ServerStreamTransport by name.
func GetServerStreamTransport(name string) ServerStreamTransport {
	muxServerStreamTrans.RLock()
	t := serverStreamTrans[name]
	muxServerStreamTrans.RUnlock()
	return t
}
