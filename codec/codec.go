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
	"sync"
)

// RequestType is the type of client request, such as SendAndRecv and SendOnly.
type RequestType int

const (
	// SendAndRecv means send one request and receive one response.
	SendAndRecv RequestType = 0
	// SendOnly means only send request, no response.
	SendOnly RequestType = 1
)

// Codec defines the interface of business communication protocol,
// which contains head and body. It only parses the body in binary,
// and then the business body struct will be handled by serializer.
// In common, the body's protocol is pb, json, etc. Specially,
// we can register our own serializer to handle other body types.
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

// Register defines the logic of register a codec by name. It will be
// called by init function defined by third package. If there is no server codec,
// the second param serverCodec can be nil.
func Register(name string, serverCodec Codec, clientCodec Codec) {
	lock.Lock()
	serverCodecs[name] = serverCodec
	clientCodecs[name] = clientCodec
	lock.Unlock()
}

// MustRegister registers the codec by name. It will panic if the codec
// has been registered.
//
// In most cases, the framework uses the init + Register method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegister to forcibly register a component 'xxx', while the framework
// uses init + Register to register another component 'yyy', conflicts may occur. If the init function
// for MustRegister is executed before the conflicting init function, MustRegister might not raise an
// error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegister and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegister(name string, serverCodec Codec, clientCodec Codec) {
	client := GetClient(name)
	if client != nil {
		panic("client codec already registered: " + name)
	}
	server := GetServer(name)
	if server != nil {
		panic("server codec already registered: " + name)
	}
	Register(name, serverCodec, clientCodec)
}

// GetServer returns the server codec by name.
func GetServer(name string) Codec {
	lock.RLock()
	c := serverCodecs[name]
	lock.RUnlock()
	return c
}

// GetClient returns the client codec by name.
func GetClient(name string) Codec {
	lock.RLock()
	c := clientCodecs[name]
	lock.RUnlock()
	return c
}
