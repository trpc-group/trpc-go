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

package restful

import (
	"errors"
	"reflect"

	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterSerializer(&ProtoSerializer{})
}

var (
	errNotProtoMessageType = errors.New("type is not proto.Message")
)

// ProtoSerializer is used for content-Type: application/octet-stream.
type ProtoSerializer struct{}

// Marshal implements Serializer.
func (*ProtoSerializer) Marshal(v interface{}) ([]byte, error) {
	msg, ok := assertProtoMessage(v)
	if !ok {
		return nil, errNotProtoMessageType
	}
	return proto.Marshal(msg)
}

// Unmarshal implements Serializer.
func (*ProtoSerializer) Unmarshal(data []byte, v interface{}) error {
	msg, ok := assertProtoMessage(v)
	if !ok {
		return errNotProtoMessageType
	}
	return proto.Unmarshal(data, msg)
}

// assertProtoMessage asserts the type of the input is proto.Message
// or a chain of pointers to proto.Message.
func assertProtoMessage(v interface{}) (proto.Message, bool) {
	msg, ok := v.(proto.Message)
	if ok {
		return msg, true
	}
	// proto reflection
	rv := reflect.ValueOf(v)
	// get the value
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() { // if the pointer points to nil，New an object
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		// if the type is proto message，return it
		if msg, ok := rv.Interface().(proto.Message); ok {
			return msg, true
		}
		rv = rv.Elem()
	}
	return nil, false
}

// Name implements Serializer.
func (*ProtoSerializer) Name() string {
	return "application/octet-stream"
}

// ContentType implements Serializer.
func (*ProtoSerializer) ContentType() string {
	return "application/octet-stream"
}
