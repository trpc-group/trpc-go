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
	"testing"
)

// goos: linux
// goarch: amd64
// pkg: trpc.group/trpc-go/trpc-go/codec
// cpu: AMD EPYC 7K62 48-Core Processor
// benchmark_old-16  100000000  10.01 ns/op  0 B/op  0 allocs/op
// benchmark_new-16  191503724  6.261 ns/op  0 B/op  0 allocs/op
func BenchmarkSerializationSliceAndMap(b *testing.B) {
	oldRegisterSerializer(SerializationTypeNoop, &testNoopSerialization{})
	backup := GetSerializer(SerializationTypeNoop)
	defer func() {
		RegisterSerializer(SerializationTypeNoop, backup)
	}()
	RegisterSerializer(SerializationTypeNoop, &testNoopSerialization{})
	type message struct {
		Message string
	}
	req := &message{"hello"}
	b.Run("benchmark old", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bs, _ := oldMarshal(SerializationTypeNoop, req)
			oldUnmarshal(SerializationTypeNoop, bs, req)
		}
	})
	b.Run("benchmark new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bs, _ := Marshal(SerializationTypeNoop, req)
			Unmarshal(SerializationTypeNoop, bs, req)
		}
	})
}

func init() {
	oldRegisterSerializer(SerializationTypeFlatBuffer, &FBSerialization{})
	oldRegisterSerializer(SerializationTypeJSON, &JSONPBSerialization{})
	oldRegisterSerializer(SerializationTypeNoop, &NoopSerialization{})
	oldRegisterSerializer(SerializationTypePB, &PBSerialization{})
	oldRegisterSerializer(SerializationTypeXML, &XMLSerialization{})
	oldRegisterSerializer(SerializationTypeTextXML, &XMLSerialization{})
}

var oldSerializers = make(map[int]Serializer)

func oldRegisterSerializer(serializationType int, s Serializer) {
	oldSerializers[serializationType] = s
}

func oldGetSerializer(serializationType int) Serializer {
	return oldSerializers[serializationType]
}

func oldUnmarshal(serializationType int, in []byte, body interface{}) error {
	if body == nil {
		return nil
	}
	if len(in) == 0 {
		return nil
	}
	if serializationType == SerializationTypeUnsupported {
		return nil
	}

	s := oldGetSerializer(serializationType)
	if s == nil {
		return errors.New("serializer not registered")
	}
	return s.Unmarshal(in, body)
}

func oldMarshal(serializationType int, body interface{}) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	if serializationType == SerializationTypeUnsupported {
		return nil, nil
	}

	s := oldGetSerializer(serializationType)
	if s == nil {
		return nil, errors.New("serializer not registered")
	}
	return s.Marshal(body)
}

type testNoopSerialization struct{}

func (s *testNoopSerialization) Unmarshal(in []byte, body interface{}) error {
	return nil
}

func (s *testNoopSerialization) Marshal(body interface{}) ([]byte, error) {
	return nil, nil
}
