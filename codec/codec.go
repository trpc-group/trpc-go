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

// Package codec defines the business communication protocol of
// packing and unpacking.
package codec

import (
	"errors"
	"fmt"
	"sync"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

// RequestType is the type of client request, such as SendAndRecv，SendOnly.
type RequestType int

const (
	// SendAndRecv means send one request and receive one response.
	SendAndRecv = RequestType(trpcpb.TrpcCallType_TRPC_UNARY_CALL)
	// SendOnly means only send request, no response.
	SendOnly = RequestType(trpcpb.TrpcCallType_TRPC_ONEWAY_CALL)
)

// Codec defines the interface of business communication protocol,
// which contains head and body. It only parses the body in binary,
// and then the business body struct will be handled by serializer.
// In common, the body's protocol is pb, json, etc. Specially,
// we can register our own serializer to handle other body type.
type Codec interface {
	// Encode pack the body into binary buffer.
	// client: Encode(msg, reqBody)(request-buffer, err)
	// server: Encode(msg, rspBody)(response-buffer, err)
	Encode(message Msg, body []byte) (buffer []byte, err error)

	// Decode unpack the body from binary buffer
	// server: Decode(msg, request-buffer)(reqBody, err)
	// client: Decode(msg, response-buffer)(rspBody, err)
	Decode(message Msg, buffer []byte) (body []byte, err error)
}

var (
	clientCodecs = make(map[string]Codec)
	serverCodecs = make(map[string]Codec)
	lock         sync.RWMutex
)

var ErrCodecAlreadyRegistered = errors.New("codec already registered")

var ErrCodecNotFound = errors.New("codec not found")

// Register defines the logic of register a codec by name. It will be
// called by init function defined by third package. If there is no server codec,
// the second param serverCodec can be nil.
func Register(name string, serverCodec Codec, clientCodec Codec) error {
	lock.Lock()
	defer lock.Unlock()
	
	if _, serverExists := serverCodecs[name]; serverExists {
		return fmt.Errorf("%w: server codec with name '%s'", ErrCodecAlreadyRegistered, name)
	}
	if _, clientExists := clientCodecs[name]; clientExists {
		return fmt.Errorf("%w: client codec with name '%s'", ErrCodecAlreadyRegistered, name)
	}
	
	serverCodecs[name] = serverCodec
	clientCodecs[name] = clientCodec
	return nil
}

// GetServer returns the server codec by name.
func GetServer(name string) (Codec, error) {
	lock.RLock()
	c, exists := serverCodecs[name]
	lock.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("%w: server codec with name '%s'", ErrCodecNotFound, name)
	}
	return c, nil
}

// GetClient returns the client codec by name.
func GetClient(name string) (Codec, error) {
	lock.RLock()
	c, exists := clientCodecs[name]
	lock.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("%w: client codec with name '%s'", ErrCodecNotFound, name)
	}
	return c, nil
}

// RegisterCompatible is a backward compatible version of Register that ignores errors.
func RegisterCompatible(name string, serverCodec Codec, clientCodec Codec) {
	_ = Register(name, serverCodec, clientCodec)
}

// GetServerCompatible is a backward compatible version of GetServer that ignores errors.
func GetServerCompatible(name string) Codec {
	c, _ := GetServer(name)
	return c
}

// GetClientCompatible is a backward compatible version of GetClient that ignores errors.
func GetClientCompatible(name string) Codec {
	c, _ := GetClient(name)
	return c
}
