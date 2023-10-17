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
	"errors"
)

// Serializer defines body serialization interface.
type Serializer interface {
	// Unmarshal deserialize the in bytes into body
	Unmarshal(in []byte, body interface{}) error

	// Marshal returns the bytes serialized from body.
	Marshal(body interface{}) (out []byte, err error)
}

// SerializationType defines the code of different serializers, such as
// protobuf, json, http-get-query and http-get-restful.
//
//   - code 0-127 is used for common modes in all language versions of trpc.
//   - code 128-999 is used for modes in any language specific version of trpc.
//   - code 1000+ is used for customized occasions in which conflicts should
//     be avoided.
const (
	// SerializationTypePB is protobuf serialization code.
	SerializationTypePB = 0
	// 1 is reserved by Tencent for internal usage.
	_ = 1
	// SerializationTypeJSON is json serialization code.
	SerializationTypeJSON = 2
	// SerializationTypeFlatBuffer is flatbuffer serialization code.
	SerializationTypeFlatBuffer = 3
	// SerializationTypeNoop is bytes empty serialization code.
	SerializationTypeNoop = 4
	// SerializationTypeXML is xml serialization code (application/xml for http).
	SerializationTypeXML = 5
	// SerializationTypeTextXML is xml serialization code (text/xml for http).
	SerializationTypeTextXML = 6

	// SerializationTypeUnsupported is unsupported serialization code.
	SerializationTypeUnsupported = 128
	// SerializationTypeForm is used to handle form request.
	SerializationTypeForm = 129
	// SerializationTypeGet is used to handle http get request.
	SerializationTypeGet = 130
	// SerializationTypeFormData is used to handle form data.
	SerializationTypeFormData = 131
)

var serializers = make(map[int]Serializer)

// RegisterSerializer registers serializer, will be called by init function
// in third package.
func RegisterSerializer(serializationType int, s Serializer) {
	serializers[serializationType] = s
}

// GetSerializer returns the serializer defined by serialization code.
func GetSerializer(serializationType int) Serializer {
	return serializers[serializationType]
}

// Unmarshal deserializes the in bytes into body. The specific serialization
// mode is defined by serializationType code, protobuf is default mode.
func Unmarshal(serializationType int, in []byte, body interface{}) error {
	if body == nil {
		return nil
	}
	if len(in) == 0 {
		return nil
	}
	if serializationType == SerializationTypeUnsupported {
		return nil
	}

	s := GetSerializer(serializationType)
	if s == nil {
		return errors.New("serializer not registered")
	}
	return s.Unmarshal(in, body)
}

// Marshal returns the serialized bytes from body. The specific serialization
// mode is defined by serializationType code, protobuf is default mode.
func Marshal(serializationType int, body interface{}) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	if serializationType == SerializationTypeUnsupported {
		return nil, nil
	}

	s := GetSerializer(serializationType)
	if s == nil {
		return nil, errors.New("serializer not registered")
	}
	return s.Marshal(body)
}
