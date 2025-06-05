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
	"net/url"
	"strings"

	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterSerializer(&FormSerializer{})
}

// FormSerializer is used for Content-Type: application/x-www-form-urlencoded.
type FormSerializer struct {
	// If DiscardUnknown is set, unknown fields are ignored.
	DiscardUnknown bool
	// UnquoteString is used to unquote the string/bytes if the original data
	// type is string/bytes.
	UnquoteString bool
}

// Marshal implements Serializer.
// It does the same thing as the jsonpb marshaler's Marshal method.
func (f *FormSerializer) Marshal(v interface{}) ([]byte, error) {
	if f.UnquoteString {
		// If the type of v is string/bytes, the normal marshalling will cause
		// the string/bytes to be additionally quoted and the control characters
		// will be degraded to normal characters.
		// Therefore, we explicitly check the type for manual marshalling.
		if val, ok := v.(*[]byte); ok && val != nil {
			return *val, nil
		}
		if val, ok := v.(*string); ok && val != nil {
			return []byte(*val), nil
		}
		// Fall back to the normal cases.
	}
	msg, ok := v.(proto.Message)
	if !ok { // marshal a field of tRPC message
		return marshal(v)
	}
	// marshal tRPC message
	return Marshaller.Marshal(msg)
}

// Unmarshal implements Serializer
func (f *FormSerializer) Unmarshal(data []byte, v interface{}) error {
	if f.UnquoteString {
		if val, ok := v.(*[]byte); ok && val != nil {
			*val = data
			return nil
		}
		if val, ok := v.(*string); ok && val != nil {
			*val = string(data)
			return nil
		}
		// Fall back to the normal cases.
	}

	msg, ok := assertProtoMessage(v)
	if !ok {
		return errNotProtoMessageType
	}

	// get url.Values
	vs, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}
	// populate proto message
	for key, values := range vs {
		fieldPath := strings.Split(key, ".")
		if err := PopulateMessage(msg, fieldPath, values); err != nil {
			if !f.DiscardUnknown || !errors.Is(err, ErrTraverseNotFound) {
				return err
			}
		}
	}

	return nil
}

// Name implements Serializer.
func (*FormSerializer) Name() string {
	return "application/x-www-form-urlencoded"
}

// ContentType implements Serializer.
// Does the same thing as jsonpb marshaler.
func (*FormSerializer) ContentType() string {
	return "application/json"
}
