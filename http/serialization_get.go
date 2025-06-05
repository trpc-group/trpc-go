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

package http

import (
	"errors"
	"net/url"

	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.RegisterSerializer(codec.SerializationTypeGet, NewGetSerialization(tag))
}

// NewGetSerialization initializes the get serialized object.
// In trpc-go before v0.16.0, the default behavior of `GetSerialization` is case-insensitive.
// In trpc-go between v0.16.0 and v0.18.1, it is case-sensitive.
// In trpc-go after v0.18.1, it is case-insensitive.
func NewGetSerialization(tag string) codec.Serializer {
	return NewGetSerializationWithCaseSensitive(tag, false)
}

// NewGetSerializationWithCaseSensitive initializes the get serialized object.
// After invoking this function, please invoke codec.RegisterSerializer() to
// Register the new Serialization.
//
// Example usage for using case-sensitive:
//
//	// New the GetSerialization's caseSensitive = true.
//	s := http.NewGetSerializationWithCaseSensitive("json", true)
//
//	// Remember to invoke codec.RegisterSerializer to register the new Serializer.
//	codec.RegisterSerializer(codec.SerializationTypeGet, s)
//
// Notice: By default, the GetSerialization is set to be case-insensitive,
// there is a drawback that it cannot unmarshal into nested structures.
// For more details, see https://git.woa.com/trpc-go/trpc-go/issues/865.
//
// In trpc-go before v0.16.0, the default behavior of `GetSerialization` is case-insensitive.
// In trpc-go between v0.16.0 and v0.18.1, it is case-sensitive.
// In trpc-go after v0.18.1, it is case-insensitive.
func NewGetSerializationWithCaseSensitive(tag string, caseSensitive bool) codec.Serializer {
	formSerializer := NewFormSerialization(tag)
	return &GetSerialization{
		formSerializer: formSerializer.(*FormSerialization),
		caseSensitive:  caseSensitive,
	}
}

// GetSerialization packages kv structure of the http get request.
// In trpc-go before v0.16.0, the default behavior of `GetSerialization` is case-insensitive.
// In trpc-go between v0.16.0 and v0.18.1, it is case-sensitive.
// In trpc-go after v0.18.1, it is case-insensitive.
// Notice: If GetSerialization is set to be case-insensitive (default),
// there is a drawback that it cannot unmarshal into nested structures.
// For more details, see https://git.woa.com/trpc-go/trpc-go/issues/865.
type GetSerialization struct {
	formSerializer *FormSerialization
	caseSensitive  bool
}

// Unmarshal unpacks kv structure.
// In trpc-go before v0.16.0, the default behavior of `GetSerialization` is case-insensitive.
// In trpc-go between v0.16.0 and v0.18.1, it is case-sensitive.
// In trpc-go after v0.18.1, it is case-insensitive, and user can
// use SetGetSerializationCaseSensitive(true) to accommodate scenarios that require case-sensitive.
func (s *GetSerialization) Unmarshal(in []byte, body interface{}) error {
	if s.caseSensitive {
		return s.formSerializer.Unmarshal(in, body)
	}
	values, err := url.ParseQuery(string(in))
	if err != nil {
		return err
	}
	return unmarshalValues(s.formSerializer.tagname, values, body)
}

// Marshal packages kv structure.
func (s *GetSerialization) Marshal(body interface{}) ([]byte, error) {
	jsonSerializer := codec.GetSerializer(codec.SerializationTypeJSON)
	if jsonSerializer == nil {
		return nil, errors.New("empty json serializer")
	}
	return jsonSerializer.Marshal(body)
}
