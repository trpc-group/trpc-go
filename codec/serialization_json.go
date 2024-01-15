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

package codec

import (
	"encoding/json"

	jsoniter "github.com/json-iterator/go"
)

// JSONAPI is json packing and unpacking object, users can change
// the internal parameter.
var JSONAPI JSONSerializer = jsoniter.ConfigCompatibleWithStandardLibrary

// JSONSerializer is json packing and unpacking object interface
type JSONSerializer interface {
	Unmarshal([]byte, interface{}) error
	Marshal(interface{}) ([]byte, error)
}

// StandardJSONSerializer is a JSONSerializer using standard encoding/json.
type StandardJSONSerializer struct{}

// Unmarshal deserializes the in bytes into body.
func (StandardJSONSerializer) Unmarshal(in []byte, body interface{}) error {
	return json.Unmarshal(in, body)
}

// Marshal returns the serialized bytes in json protocol.
func (StandardJSONSerializer) Marshal(body interface{}) ([]byte, error) {
	return json.Marshal(body)
}

// JSONSerialization provides json serialization mode.
type JSONSerialization struct{}

// Unmarshal deserializes the in bytes into body.
func (s *JSONSerialization) Unmarshal(in []byte, body interface{}) error {
	return JSONAPI.Unmarshal(in, body)
}

// Marshal returns the serialized bytes in json protocol.
func (s *JSONSerialization) Marshal(body interface{}) ([]byte, error) {
	return JSONAPI.Marshal(body)
}
