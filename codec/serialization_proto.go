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
	"fmt"

	"github.com/golang/protobuf/proto"
)

func init() {
	RegisterSerializer(SerializationTypePB, &PBSerialization{})
}

// PBSerialization provides protobuf serialization mode.
type PBSerialization struct{}

// Unmarshal deserializes the in bytes into body.
func (s *PBSerialization) Unmarshal(in []byte, body interface{}) error {
	msg, ok := body.(proto.Message)
	if !ok {
		return fmt.Errorf("failed to unmarshal body: expected proto.Message, got %T", body)
	}
	return proto.Unmarshal(in, msg)
}

// Marshal returns the serialized bytes in protobuf protocol.
func (s *PBSerialization) Marshal(body interface{}) ([]byte, error) {
	msg, ok := body.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("failed to marshal body: expected proto.Message, got %T", body)
	}
	return proto.Marshal(msg)
}
