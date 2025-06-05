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
	"encoding/json"
)

// JSONSerializer is a struct that implements the Serializer interface for
// handling JSON data in requests and responses.
// It uses "encoding/json", which has better performance than the default jsonpb
// serializer, but it cannot handle advanced features of protobuf, such as map, oneof, etc.
type JSONSerializer struct{}

// Marshal takes an interface{} value and converts it into a JSON-encoded byte slice.
// It implements the Marshal method of the Serializer interface.
// v: The value to be marshaled into JSON.
// Returns: A slice of bytes representing the JSON-encoded data and an error if any occurred during marshaling.
func (*JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal takes a JSON-encoded byte slice and decodes it into the specified interface{} value.
// It implements the Unmarshal method of the Serializer interface.
// data: The JSON-encoded data as a byte slice.
// v: A pointer to the value where the JSON data should be decoded.
// Returns: An error if any occurred during unmarshaling.
func (j *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Name returns the name identifier of the JSONSerializer.
// It implements the Name method of the Serializer interface.
// Returns: A string representing the name identifier of the serializer, which is "application/json".
func (*JSONSerializer) Name() string {
	return "application/json"
}

// ContentType returns the MIME content type that the JSONSerializer handles.
// It implements the ContentType method of the Serializer interface.
// Returns: A string representing the MIME content type, which is "application/json".
func (*JSONSerializer) ContentType() string {
	return "application/json"
}
