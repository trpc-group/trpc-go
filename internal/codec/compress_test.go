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

package codec_test

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
)

func TestIsValidCompressType(t *testing.T) {
	tests := []struct {
		name string
		arg  int
		want bool
	}{
		{"valid compress type that is defined in codec", codec.CompressTypeSnappy, true},
		{"valid compress type that isn't defined in codec", 10000, true},
		{"invalid compress type", -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := icodec.IsValidCompressType(tt.arg); got != tt.want {
				t.Errorf("IsValidCompressType() = %v, want %v", got, tt.want)
			}
		})
	}
}
