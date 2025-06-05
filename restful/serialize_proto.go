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
	protoSerializerNames := []string{
		"application/octet-stream",
		"application/protobuf",
		"application/x-protobuf",
		"application/pb",
		"application/proto"}
	for _, name := range protoSerializerNames {
		RegisterSerializer(&ProtoSerializer{protoSerializerName: name})
	}
}

var (
	errNotProtoMessageType = errors.New("type is not proto.Message")
)

// By default, ProtoSerializer.Name() and ProtoSerializer.ContentType() return defaultProtoSerializerName.
const defaultProtoSerializerName = "application/octet-stream"

// ProtoSerializer is used for content-Type: application/octet-stream,
// application/protobuf, application/x-protobuf, application/pb, application/proto.
type ProtoSerializer struct {
	protoSerializerName string
}

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
func (ps *ProtoSerializer) Name() string {
	if ps.protoSerializerName == "" {
		ps.protoSerializerName = defaultProtoSerializerName
	}
	return ps.protoSerializerName
}

// ContentType implements Serializer.
func (ps *ProtoSerializer) ContentType() string {
	if ps.protoSerializerName == "" {
		ps.protoSerializerName = defaultProtoSerializerName
	}
	return ps.protoSerializerName
}
