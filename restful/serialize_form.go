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
}

// Marshal implements Serializer.
// It does the same thing as the jsonpb marshaler's Marshal method.
func (*FormSerializer) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok { // marshal a field of tRPC message
		return marshal(v)
	}
	// marshal tRPC message
	return Marshaller.Marshal(msg)
}

// Unmarshal implements Serializer
func (f *FormSerializer) Unmarshal(data []byte, v interface{}) error {
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
