// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package codec

import (
	"errors"

	flatbuffers "github.com/google/flatbuffers/go"
)

func init() {
	RegisterSerializer(SerializationTypeFlatBuffer, &FBSerialization{})
}

// FBSerialization provides the flatbuffers serialization mode.
// Flatbuffers official url: https://google.github.io/flatbuffers
type FBSerialization struct{}

// Unmarshal deserializes the in bytes into body param, body
// should implement flatbuffersInit interface.
func (*FBSerialization) Unmarshal(in []byte, body interface{}) error {
	body, ok := body.(flatbuffersInit)
	if !ok {
		return errors.New("unmarshal fail: body does not implement flatbufferInit interface")
	}
	body.(flatbuffersInit).Init(in, flatbuffers.GetUOffsetT(in))
	return nil
}

// Marshal returns the serialized bytes, body should be a flatbuffers.Builder.
func (*FBSerialization) Marshal(body interface{}) ([]byte, error) {
	builder, ok := body.(*flatbuffers.Builder)
	if !ok {
		return nil, errors.New("marshal fail: body not *flatbuffers.Builder")
	}
	return builder.FinishedBytes(), nil
}

type flatbuffersInit interface {
	Init(data []byte, i flatbuffers.UOffsetT)
}
