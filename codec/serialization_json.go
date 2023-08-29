// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package codec

import (
	jsoniter "github.com/json-iterator/go"
)

// JSONAPI is json packing and unpacking object, users can change
// the internal parameter.
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
