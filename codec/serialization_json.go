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

package codec

import (
	jsoniter "github.com/json-iterator/go"
)

// JSONAPI is used by tRPC JSON serialization when the object does
// not conform to protobuf proto.Message interface.
//
// Deprecated: This global variable is exportable due to backward comparability issue but
// should not be modified. If users want to change the default behavior of
// internal JSON serialization, please use register your customized serializer
// function like:
//
//	codec.RegisterSerializer(codec.SerializationTypeJSON, yourOwnJSONSerializer)
var JSONAPI = jsoniter.ConfigCompatibleWithStandardLibrary

// JSONSerialization provides json serialization mode.
// golang official json package is implemented by reflection,
// and has very low performance. So trpc-go choose json-iterator package
// to implement json serialization.
type JSONSerialization struct{}

// Unmarshal deserializes the in bytes into body.
func (s *JSONSerialization) Unmarshal(in []byte, body interface{}) error {
	return JSONAPI.Unmarshal(in, body)
}

// Marshal returns the serialized bytes in json protocol.
func (s *JSONSerialization) Marshal(body interface{}) ([]byte, error) {
	return JSONAPI.Marshal(body)
}
