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

import "trpc.group/trpc-go/trpc-go/codec"

// IsValidCompressType checks whether t is a valid Compress type.
//
//go:inline
func IsValidCompressType(t int) bool {
	const minValidCompressType = codec.CompressTypeNoop
	return t >= minValidCompressType
}
