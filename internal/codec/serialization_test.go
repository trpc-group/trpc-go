// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package codec_test

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
)

func TestIsValidSerializationType(t *testing.T) {
	tests := []struct {
		name string
		arg  int
		want bool
	}{
		{"valid serialization type that is defined in codec", codec.SerializationTypePB, true},
		{"valid serialization type that isn't defined in codec", 10000, true},
		{"invalid serialization type", -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := icodec.IsValidSerializationType(tt.arg); got != tt.want {
				t.Errorf("IsValidSerializationType() = %v, want %v", got, tt.want)
			}
		})
	}
}
