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

	"git.woa.com/jce/jce"
)

func init() {
	RegisterSerializer(SerializationTypeJCE, &JCESerialization{})
}

// JCESerialization provides jce serialization mode.
type JCESerialization struct{}

// Unmarshal deserializes in bytes into body, body should implement
// jce.Message interface.
func (j *JCESerialization) Unmarshal(in []byte, body interface{}) error {
	msg, ok := body.(jce.Message)
	if !ok {
		return fmt.Errorf("failed to unmarshal body: expected git.woa.com/jce/jce.Message, got %T."+
			"You may need to refer to issue https://git.woa.com/trpc-go/trpc-go/issues/897", body)
	}
	return jce.Unmarshal(in, msg)
}

// Marshal returns the bytes serialized in jce protocol.
func (j *JCESerialization) Marshal(body interface{}) ([]byte, error) {
	msg, ok := body.(jce.Message)
	if !ok {
		return nil, fmt.Errorf("failed to marshal body: expected git.woa.com/jce/jce.Message, got %T."+
			"You may need to refer to issue https://git.woa.com/trpc-go/trpc-go/issues/897", body)
	}
	return jce.Marshal(msg)
}
