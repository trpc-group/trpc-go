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

// Package codec provides some common codec-related functions.
package codec

import "trpc.group/trpc-go/trpc-go/codec"

// IsValidCompressType checks whether t is a valid Compress type.
func IsValidCompressType(t int) bool {
	const minValidCompressType = codec.CompressTypeNoop
	return t >= minValidCompressType
}
